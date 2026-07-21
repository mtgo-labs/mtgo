package session

import (
	"context"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

func init() {
	// Register decoder for InvokeAfterMsgRequest so unpackIncoming can decode
	// chain-wrapped messages in tests.
	tg.Registry[tg.InvokeAfterMsgTypeID] = func(r *tg.Reader) (tg.TLObject, error) {
		v := &tg.InvokeAfterMsgRequest{}
		msgID, err := r.ReadInt64()
		if err != nil {
			return nil, err
		}
		v.MsgID = msgID
		inner, err := tg.ReadTLObject(r)
		if err != nil {
			return nil, err
		}
		v.Query = inner
		return v, nil
	}
}

// makeRPCErrorResponse builds an encrypted RPCResult containing an RPCError
// with the given error code and message, addressed to the given reqMsgID.
func makeRPCErrorResponse(s *Session, reqMsgID int64, code int32, msg string) []byte {
	return makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.RPCResult{
		ReqMsgID: reqMsgID,
		Result: &tg.RPCError{
			ErrorCode:    code,
			ErrorMessage: msg,
		},
	})
}

// makeSuccessResponse builds an encrypted RPCResult containing a Pong reply.
func makeSuccessResponse(s *Session, reqMsgID, pingID int64) []byte {
	return makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.RPCResult{
		ReqMsgID: reqMsgID,
		Result:   &tg.Pong{MsgID: reqMsgID, PingID: pingID},
	})
}

// ---------------------------------------------------------------------------
// G12: Additional error recovery
// ---------------------------------------------------------------------------

func TestInvokeFloodWaitDoesNotRetry(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	done := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: 1}, 3, 5*time.Second)
		done <- struct {
			obj tg.TLObject
			err error
		}{obj: obj, err: err}
	}()

	first := <-mt.sendCh
	firstMsg := unpackIncoming(first, s)
	if firstMsg == nil {
		t.Fatal("first message did not decode")
	}
	mt.recvCh <- makeRPCErrorResponse(s, firstMsg.MsgID, 420, "FLOOD_WAIT_10")

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("Invoke error = %v", result.err)
		}
		rpcErr, ok := result.obj.(*tg.RPCError)
		if !ok || rpcErr.ErrorCode != 420 || rpcErr.ErrorMessage != "FLOOD_WAIT_10" {
			t.Fatalf("Invoke result = %T %#v, want FLOOD_WAIT", result.obj, result.obj)
		}
	case <-time.After(time.Second):
		t.Fatal("Invoke did not return flood wait promptly")
	}

	select {
	case <-mt.sendCh:
		t.Fatal("Invoke retried a flood-wait response")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestInvokeConnectionNotInitedRetries(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	pingID := time.Now().UnixNano()
	done := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 2, 5*time.Second)
		done <- struct {
			obj tg.TLObject
			err error
		}{obj, err}
	}()

	// First send → respond with CONNECTION_NOT_INITED.
	first := <-mt.sendCh
	firstMsg := unpackIncoming(first, s)
	if firstMsg == nil {
		t.Fatal("first message did not decode")
	}
	mt.recvCh <- makeRPCErrorResponse(s, firstMsg.MsgID, 400, "CONNECTION_NOT_INITED")

	// Second send → respond with success.
	second := <-mt.sendCh
	secondMsg := unpackIncoming(second, s)
	if secondMsg == nil {
		t.Fatal("second message did not decode")
	}
	mt.recvCh <- makeSuccessResponse(s, secondMsg.MsgID, pingID)

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("Invoke() error: %v", got.err)
		}
		if _, ok := got.obj.(*tg.Pong); !ok {
			t.Fatalf("Invoke() = %T, want *tg.Pong", got.obj)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Invoke() timed out")
	}
}

func TestInvokePersistentTimestampOutdatedRetries(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	pingID := time.Now().UnixNano()
	done := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 2, 10*time.Second)
		done <- struct {
			obj tg.TLObject
			err error
		}{obj, err}
	}()

	// First send → PERSISTENT_TIMESTAMP_OUTDATED.
	first := <-mt.sendCh
	firstMsg := unpackIncoming(first, s)
	mt.recvCh <- makeRPCErrorResponse(s, firstMsg.MsgID, 400, "PERSISTENT_TIMESTAMP_OUTDATED")

	// After the 1-second delay, second send → success.
	second := <-mt.sendCh
	secondMsg := unpackIncoming(second, s)
	mt.recvCh <- makeSuccessResponse(s, secondMsg.MsgID, pingID)

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("Invoke() error: %v", got.err)
		}
		if _, ok := got.obj.(*tg.Pong); !ok {
			t.Fatalf("Invoke() = %T, want *tg.Pong", got.obj)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Invoke() timed out")
	}
}

func TestInvokeAuthKeyPermEmptyNoPFSSurfaces(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	pingID := time.Now().UnixNano()
	done := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 2, 5*time.Second)
		done <- struct {
			obj tg.TLObject
			err error
		}{obj, err}
	}()

	// First send → AUTH_KEY_PERM_EMPTY. Without PFS, should be surfaced.
	first := <-mt.sendCh
	firstMsg := unpackIncoming(first, s)
	mt.recvCh <- makeRPCErrorResponse(s, firstMsg.MsgID, 401, "AUTH_KEY_PERM_EMPTY")

	select {
	case got := <-done:
		if got.err == nil {
			t.Fatal("expected error, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Invoke() timed out")
	}
}

func TestInvokeG12RetryLimit(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	pingID := time.Now().UnixNano()
	done := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 1, 5*time.Second)
		done <- struct {
			obj tg.TLObject
			err error
		}{obj, err}
	}()

	// Respond to each send with CONNECTION_NOT_INITED. After maxG12Retries
	// (2), the error should be surfaced rather than retried indefinitely.
	// With retries=1 and maxG12Retries=2, Invoke makes 3 attempts:
	//   attempt 0 → CNI (g12Retries=1, maxAttempts++)
	//   attempt 1 → CNI (g12Retries=2, maxAttempts++)
	//   attempt 2 → CNI (g12Retries=2 not < 2, surface 400 error)
	for range maxG12Retries + 2 {
		select {
		case sent := <-mt.sendCh:
			msg := unpackIncoming(sent, s)
			if msg == nil {
				t.Fatal("message did not decode")
			}
			mt.recvCh <- makeRPCErrorResponse(s, msg.MsgID, 400, "CONNECTION_NOT_INITED")
		case got := <-done:
			if got.err == nil {
				t.Fatal("expected error after G12 retries exhausted, got nil")
			}
			return
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for send")
		}
	}

	// All responses sent — Invoke should now return with the error.
	select {
	case got := <-done:
		if got.err == nil {
			t.Fatal("expected error after G12 retries exhausted, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Invoke() timed out")
	}
}

// ---------------------------------------------------------------------------
// G13: invokeAfterMsg chain support
// ---------------------------------------------------------------------------

func TestSetChainAndGetChainLastMsgID(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.chains = make(map[int64]int64)

	// Initially no chain entry.
	_, ok := s.ChainLastMsgID(42)
	if ok {
		t.Fatal("expected chain 42 to be absent")
	}

	s.SetChain(1000, 42)
	last, ok := s.ChainLastMsgID(42)
	if !ok {
		t.Fatal("expected chain 42 to exist after SetChain")
	}
	if last != 1000 {
		t.Fatalf("chain last = %d, want 1000", last)
	}

	s.SetChain(2000, 42)
	last, _ = s.ChainLastMsgID(42)
	if last != 2000 {
		t.Fatalf("chain last = %d, want 2000", last)
	}
}

func TestClearChain(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.chains = make(map[int64]int64)

	s.SetChain(1000, 7)
	s.ClearChain(7)

	_, ok := s.ChainLastMsgID(7)
	if ok {
		t.Fatal("expected chain 7 to be removed after ClearChain")
	}
}

func TestWrapChainNoEntryReturnsOriginal(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.chains = make(map[int64]int64)

	query := &tg.PingRequest{PingID: 42}
	wrapped := s.wrapChain(99, query)
	if wrapped != query {
		t.Fatalf("expected original query for empty chain, got %T", wrapped)
	}
}

func TestWrapChainWrapsInInvokeAfterMsg(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.chains = make(map[int64]int64)

	s.SetChain(12345, 3)
	query := &tg.PingRequest{PingID: 99}
	wrapped := s.wrapChain(3, query)

	after, ok := wrapped.(*tg.InvokeAfterMsgRequest)
	if !ok {
		t.Fatalf("expected *tg.InvokeAfterMsgRequest, got %T", wrapped)
	}
	if after.MsgID != 12345 {
		t.Fatalf("after.MsgID = %d, want 12345", after.MsgID)
	}
	if _, ok := after.Query.(*tg.PingRequest); !ok {
		t.Fatalf("expected inner query *tg.PingRequest, got %T", after.Query)
	}
}

func TestWrapChainZeroMsgIDReturnsOriginal(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.chains = make(map[int64]int64)

	// Manually set a zero msg_id.
	s.chainMu.Lock()
	s.chains[5] = 0
	s.chainMu.Unlock()

	query := &tg.PingRequest{PingID: 1}
	wrapped := s.wrapChain(5, query)
	if wrapped != query {
		t.Fatalf("expected original query for zero msg_id, got %T", wrapped)
	}
}

func TestChainIDFromContext(t *testing.T) {
	_, ok := ChainIDFromContext(context.Background())
	if ok {
		t.Fatal("expected no chain ID from background context")
	}

	ctx := WithChainID(context.Background(), 42)
	id, ok := ChainIDFromContext(ctx)
	if !ok {
		t.Fatal("expected chain ID 42")
	}
	if id != 42 {
		t.Fatalf("chain ID = %d, want 42", id)
	}

	// Zero chain ID is a no-op.
	ctx0 := WithChainID(context.Background(), 0)
	_, ok0 := ChainIDFromContext(ctx0)
	if ok0 {
		t.Fatal("expected no chain ID for zero chain ID")
	}
}

func TestInvokeChainedUpdatesChainOnSuccess(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	pingID := time.Now().UnixNano()
	done := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.InvokeChained(context.Background(), 42, &tg.PingRequest{PingID: pingID}, 2, 5*time.Second)
		done <- struct {
			obj tg.TLObject
			err error
		}{obj, err}
	}()

	// First send: no prior chain entry, so query should NOT be wrapped.
	first := <-mt.sendCh
	firstMsg := unpackIncoming(first, s)
	if firstMsg == nil {
		t.Fatal("first message did not decode")
	}
	// Verify the body is a plain PingRequest, not InvokeAfterMsgRequest.
	if _, ok := firstMsg.Body.(*tg.PingRequest); !ok {
		t.Fatalf("first body = %T, want *tg.PingRequest (no chain yet)", firstMsg.Body)
	}

	mt.recvCh <- makeSuccessResponse(s, firstMsg.MsgID, pingID)

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("InvokeChained() error: %v", got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("InvokeChained() timed out")
	}

	// After success, chain 42 should point to firstMsg.MsgID.
	last, ok := s.ChainLastMsgID(42)
	if !ok {
		t.Fatal("expected chain 42 to exist after successful InvokeChained")
	}
	if last != firstMsg.MsgID {
		t.Fatalf("chain last = %d, want %d", last, firstMsg.MsgID)
	}
}

func TestInvokeChainedWrapsAfterPriorChainEntry(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	// Seed the chain with a prior msg_id.
	priorMsgID := int64(1234567890)
	s.SetChain(priorMsgID, 100)

	pingID := time.Now().UnixNano()
	done := make(chan error, 1)
	go func() {
		_, err := s.InvokeChained(context.Background(), 100, &tg.PingRequest{PingID: pingID}, 1, 5*time.Second)
		done <- err
	}()

	// The sent message should be wrapped in InvokeAfterMsgRequest.
	sent := <-mt.sendCh
	msg := unpackIncoming(sent, s)
	if msg == nil {
		t.Fatal("message did not decode")
	}

	after, ok := msg.Body.(*tg.InvokeAfterMsgRequest)
	if !ok {
		t.Fatalf("expected *tg.InvokeAfterMsgRequest, got %T", msg.Body)
	}
	if after.MsgID != priorMsgID {
		t.Fatalf("after.MsgID = %d, want %d", after.MsgID, priorMsgID)
	}

	// Verify the query inside the wrapper is the PingRequest.
	_, ok = after.Query.(*tg.PingRequest)
	if !ok {
		t.Fatalf("inner query = %T, want *tg.PingRequest", after.Query)
	}

	mt.recvCh <- makeSuccessResponse(s, msg.MsgID, pingID)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("InvokeChained() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("InvokeChained() timed out")
	}
}

func TestInvokeChainedMsgWaitFailedClearsChain(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	priorMsgID := int64(999999)
	s.SetChain(priorMsgID, 50)

	pingID := time.Now().UnixNano()
	done := make(chan error, 1)
	go func() {
		_, err := s.InvokeChained(context.Background(), 50, &tg.PingRequest{PingID: pingID}, 1, 5*time.Second)
		done <- err
	}()

	// First send should be wrapped (chain has prior entry).
	first := <-mt.sendCh
	firstMsg := unpackIncoming(first, s)
	if _, ok := firstMsg.Body.(*tg.InvokeAfterMsgRequest); !ok {
		t.Fatalf("first body = %T, want *tg.InvokeAfterMsgRequest", firstMsg.Body)
	}

	// Respond with MSG_WAIT_FAILED.
	mt.recvCh <- makeRPCErrorResponse(s, firstMsg.MsgID, 400, "MSG_WAIT_FAILED")

	// Second send should be unwrapped (chain cleared).
	second := <-mt.sendCh
	secondMsg := unpackIncoming(second, s)
	if _, ok := secondMsg.Body.(*tg.PingRequest); !ok {
		t.Fatalf("second body = %T, want *tg.PingRequest (chain cleared)", secondMsg.Body)
	}

	// Verify chain 50 is cleared.
	_, chainExists := s.ChainLastMsgID(50)
	if chainExists {
		t.Fatal("expected chain 50 to be cleared after MSG_WAIT_FAILED")
	}

	mt.recvCh <- makeSuccessResponse(s, secondMsg.MsgID, pingID)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("InvokeChained() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("InvokeChained() timed out")
	}

	// After success, chain 50 should be re-set with secondMsg.MsgID.
	last, ok := s.ChainLastMsgID(50)
	if !ok {
		t.Fatal("expected chain 50 to be re-established after successful retry")
	}
	if last != secondMsg.MsgID {
		t.Fatalf("chain last = %d, want %d", last, secondMsg.MsgID)
	}
}

func TestInvokeWithZeroChainIDIsUnwrapped(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	// Chain ID 0 should be a no-op — query sent unwrapped.
	pingID := time.Now().UnixNano()
	done := make(chan error, 1)
	go func() {
		_, err := s.InvokeChained(context.Background(), 0, &tg.PingRequest{PingID: pingID}, 1, 5*time.Second)
		done <- err
	}()

	sent := <-mt.sendCh
	msg := unpackIncoming(sent, s)
	if _, ok := msg.Body.(*tg.PingRequest); !ok {
		t.Fatalf("body = %T, want *tg.PingRequest (chain ID 0 = no wrap)", msg.Body)
	}

	mt.recvCh <- makeSuccessResponse(s, msg.MsgID, pingID)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("InvokeChained() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("InvokeChained() timed out")
	}
}
