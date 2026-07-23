package session

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
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

func nextOutgoingRPCMessage(t *testing.T, s *Session, mt *mockTransport) *tg.MTProtoMessage {
	t.Helper()
	for {
		select {
		case data := <-mt.sendCh:
			message := unpackIncoming(data, s)
			if message == nil {
				t.Fatal("outgoing message did not decode")
			}
			if _, service := message.Body.(*tg.MsgsAck); service {
				continue
			}
			return message
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for outgoing RPC")
		}
	}
}

func outgoingConstructor(t *testing.T, s *Session, data []byte) (int64, uint32) {
	t.Helper()
	raw, decrypted, err := unpackIncomingMessageEnvelope(data, s.sessionIDBytes(), s.authKey, s.authKeyID)
	if decrypted != nil {
		defer crypto.ReleaseAESBuf(decrypted)
	}
	if err != nil {
		t.Fatalf("unpack outgoing envelope: %v", err)
	}
	if len(raw.BodyRaw) < 4 {
		t.Fatalf("outgoing body length = %d, want constructor", len(raw.BodyRaw))
	}
	return raw.MsgID, binary.LittleEndian.Uint32(raw.BodyRaw[:4])
}

func nextOutgoingConstructor(t *testing.T, s *Session, mt *mockTransport) (int64, uint32) {
	t.Helper()
	for {
		select {
		case data := <-mt.sendCh:
			msgID, constructor := outgoingConstructor(t, s, data)
			if constructor == tg.MsgsAckTypeID {
				continue
			}
			return msgID, constructor
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for outgoing RPC constructor")
		}
	}
}

type decodedBindEnvelope struct {
	msgID            int64
	nonce            int64
	expiresAt        int32
	encryptedMessage []byte
}

func decodeBindEnvelope(t *testing.T, s *Session, permKey, data []byte) decodedBindEnvelope {
	t.Helper()
	raw, decrypted, err := unpackIncomingMessageEnvelope(data, s.sessionIDBytes(), s.authKey, s.authKeyID)
	if decrypted != nil {
		defer crypto.ReleaseAESBuf(decrypted)
	}
	if err != nil {
		t.Fatalf("unpack bind envelope: %v", err)
	}
	r := tg.NewReader(raw.BodyRaw)
	defer tg.ReleaseReader(r)
	constructor, err := r.ReadUint32()
	if err != nil || constructor != tg.AuthBindTempAuthKeyTypeID {
		t.Fatalf("bind constructor = 0x%x, %v", constructor, err)
	}
	permKeyID, err := r.ReadInt64()
	if err != nil {
		t.Fatalf("read outer perm key ID: %v", err)
	}
	nonce, err := r.ReadInt64()
	if err != nil {
		t.Fatalf("read outer nonce: %v", err)
	}
	expiresAt, err := r.ReadInt32()
	if err != nil {
		t.Fatalf("read outer expiry: %v", err)
	}
	encryptedMessage, err := r.ReadBytes()
	if err != nil {
		t.Fatalf("read encrypted bind message: %v", err)
	}
	if len(encryptedMessage) < 40 || (len(encryptedMessage)-24)%16 != 0 {
		t.Fatalf("encrypted bind message length = %d", len(encryptedMessage))
	}
	if got := int64(binary.LittleEndian.Uint64(encryptedMessage[:8])); got != permKeyID {
		t.Fatalf("encrypted perm key ID = %d, want %d", got, permKeyID)
	}
	var msgKey [16]byte
	copy(msgKey[:], encryptedMessage[8:24])
	aesKey, aesIV := deriveMsgAESKeyIV(permKey, msgKey, 0)
	plain, err := crypto.IGEDecrypt(encryptedMessage[24:], aesKey[:], aesIV[:])
	if err != nil {
		t.Fatalf("decrypt bind message: %v", err)
	}
	defer crypto.ReleaseAESBuf(plain)
	if len(plain) < 32 {
		t.Fatalf("bind plaintext length = %d", len(plain))
	}
	bodyLen := int(binary.LittleEndian.Uint32(plain[28:32]))
	plainLen := 32 + bodyLen
	if bodyLen < 0 || plainLen > len(plain) {
		t.Fatalf("bind inner body length = %d, plaintext = %d", bodyLen, len(plain))
	}
	hash := sha1.Sum(plain[:plainLen])
	if !bytes.Equal(msgKey[:], hash[4:20]) {
		t.Fatal("bind plaintext message key mismatch")
	}
	innerMsgID := int64(binary.LittleEndian.Uint64(plain[16:24]))
	if innerMsgID != raw.MsgID {
		t.Fatalf("bind inner msg_id = %d, outer msg_id = %d", innerMsgID, raw.MsgID)
	}
	if seqNo := binary.LittleEndian.Uint32(plain[24:28]); seqNo != 0 {
		t.Fatalf("bind inner seq_no = %d, want 0", seqNo)
	}
	innerReader := tg.NewReader(plain[32:plainLen])
	defer tg.ReleaseReader(innerReader)
	innerObject, err := tg.ReadTLObject(innerReader)
	if err != nil {
		t.Fatalf("decode bind_auth_key_inner: %v", err)
	}
	inner, ok := innerObject.(*tg.BindAuthKeyInner)
	if !ok {
		t.Fatalf("bind inner object = %T", innerObject)
	}
	if inner.Nonce != nonce || inner.PermAuthKeyID != permKeyID || inner.ExpiresAt != expiresAt {
		t.Fatalf("bind inner fields = nonce %d, perm %d, expiry %d; outer = %d, %d, %d",
			inner.Nonce, inner.PermAuthKeyID, inner.ExpiresAt, nonce, permKeyID, expiresAt)
	}
	return decodedBindEnvelope{
		msgID:            raw.MsgID,
		nonce:            nonce,
		expiresAt:        expiresAt,
		encryptedMessage: encryptedMessage,
	}
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

func TestInvokeAuthKeyDuplicatedDoesNotRetry(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	done := make(chan error, 1)
	go func() {
		_, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: 1}, 3, 5*time.Second)
		done <- err
	}()
	first := <-mt.sendCh
	firstMsg := unpackIncoming(first, s)
	if firstMsg == nil {
		t.Fatal("first message did not decode")
	}
	mt.recvCh <- makeRPCErrorResponse(s, firstMsg.MsgID, 406, "AUTH_KEY_DUPLICATED")
	select {
	case err := <-done:
		if !tgerr.Is(err, tgerr.ErrAuthKeyDuplicated) {
			t.Fatalf("Invoke error = %v, want AUTH_KEY_DUPLICATED", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Invoke did not return AUTH_KEY_DUPLICATED promptly")
	}
	select {
	case <-mt.sendCh:
		t.Fatal("Invoke retried AUTH_KEY_DUPLICATED")
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

func TestInvokeAuthKeyPermEmptyOnBindDoesNotRecurse(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	pfs := NewTempKeyManager(2, false, make([]byte, 256), true, nil)
	pfs.tempKey = make([]byte, 256)
	pfs.issuedAt = time.Now()
	pfs.expiresAt = time.Now().Add(time.Hour)
	s.SetPFS(pfs)

	done := make(chan error, 1)
	go func() {
		_, err := s.Invoke(context.Background(), &tg.AuthBindTempAuthKeyRequest{
			PermAuthKeyID:    1,
			Nonce:            2,
			ExpiresAt:        int32(time.Now().Add(time.Hour).Unix()),
			EncryptedMessage: []byte{1},
		}, 2, 5*time.Second)
		done <- err
	}()
	first := <-mt.sendCh
	firstMsg, decrypted, err := unpackIncomingMessageEnvelope(first, s.sessionIDBytes(), s.authKey, s.authKeyID)
	if decrypted != nil {
		defer crypto.ReleaseAESBuf(decrypted)
	}
	if err != nil {
		t.Fatalf("unpack bind message: %v", err)
	}
	mt.recvCh <- makeRPCErrorResponse(s, firstMsg.MsgID, 401, "AUTH_KEY_PERM_EMPTY")

	select {
	case err := <-done:
		if !tgerr.Is(err, tgerr.ErrAuthKeyPermEmpty) {
			t.Fatalf("Invoke() = %v, want AUTH_KEY_PERM_EMPTY", err)
		}
	case <-time.After(time.Second):
		t.Fatal("bind recursively retried AUTH_KEY_PERM_EMPTY")
	}
	select {
	case <-mt.sendCh:
		t.Fatal("bind sent a recursive auth.bindTempAuthKey request")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestConcurrentAuthKeyPermEmptyUsesSingleRebindAndReinitializes(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	pfs := NewTempKeyManager(2, false, make([]byte, 256), true, nil)
	pfs.tempKey = make([]byte, 256)
	pfs.issuedAt = time.Now()
	pfs.expiresAt = time.Now().Add(time.Hour)
	pfs.bound = true
	pfs.bindEpoch = 1
	s.SetPFS(pfs)
	var initCalls atomic.Int32
	s.SetPFSInitConnection(func(context.Context) error {
		initCalls.Add(1)
		return nil
	})

	done := make(chan error, 2)
	for pingID := int64(1); pingID <= 2; pingID++ {
		pingID := pingID
		go func() {
			_, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 2, 5*time.Second)
			done <- err
		}()
	}

	requests := make([]*tg.MTProtoMessage, 0, 2)
	for range 2 {
		request := nextOutgoingRPCMessage(t, s, mt)
		if _, ok := request.Body.(*tg.PingRequest); !ok {
			t.Fatalf("initial request = %T, want *tg.PingRequest", request.Body)
		}
		requests = append(requests, request)
	}
	for _, request := range requests {
		mt.recvCh <- makeRPCErrorResponse(s, request.MsgID, 401, "AUTH_KEY_PERM_EMPTY")
	}

	bindMsgID, bindConstructor := nextOutgoingConstructor(t, s, mt)
	if bindConstructor != tg.AuthBindTempAuthKeyTypeID {
		t.Fatalf("recovery constructor = 0x%x, want auth.bindTempAuthKey", bindConstructor)
	}
	duplicateTimer := time.NewTimer(50 * time.Millisecond)
checkDuplicate:
	for {
		select {
		case duplicate := <-mt.sendCh:
			_, constructor := outgoingConstructor(t, s, duplicate)
			if constructor == tg.MsgsAckTypeID {
				continue
			}
			t.Fatalf("concurrent recovery sent duplicate constructor 0x%x before bind completed", constructor)
		case <-duplicateTimer.C:
			break checkDuplicate
		}
	}
	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.RPCResult{
		ReqMsgID: bindMsgID,
		Result:   &tg.BoolTrue{},
	})

	for range 2 {
		request := nextOutgoingRPCMessage(t, s, mt)
		ping, ok := request.Body.(*tg.PingRequest)
		if !ok {
			t.Fatalf("retried request = %T, want *tg.PingRequest", request.Body)
		}
		mt.recvCh <- makeSuccessResponse(s, request.MsgID, ping.PingID)
	}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("Invoke() after shared PFS rebind = %v", err)
		}
	}
	if got := initCalls.Load(); got != 1 {
		t.Fatalf("PFS initConnection calls = %d, want 1", got)
	}
	if pfs.NeedsInitConnection() {
		t.Fatal("PFS initConnection requirement remained pending")
	}
}

func TestBindTempAuthKeyInnerMessageIDMatchesOuterOnEveryAttempt(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	permKey := make([]byte, 256)
	tempKey := make([]byte, 256)
	for i := range permKey {
		permKey[i] = byte(i)
		tempKey[i] = byte(255 - i)
	}
	pfs := NewTempKeyManager(2, false, permKey, true, nil, time.Now())
	pfs.tempKey = tempKey
	pfs.issuedAt = time.Now()
	pfs.expiresAt = time.Now().Add(time.Hour)
	pfs.bindExpiresAt = 1_800_003_600
	s.SetPFS(pfs)

	bindDone := make(chan error, 1)
	go func() {
		bindDone <- pfs.Bind(context.Background(), s.SessionID(), s.Invoke)
	}()
	nextAttempt := func() decodedBindEnvelope {
		t.Helper()
		select {
		case data := <-mt.sendCh:
			return decodeBindEnvelope(t, s, permKey, data)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for auth.bindTempAuthKey")
			return decodedBindEnvelope{}
		}
	}

	first := nextAttempt()
	if first.expiresAt != pfs.bindExpiresAt {
		t.Fatalf("first bind expiry = %d, want %d", first.expiresAt, pfs.bindExpiresAt)
	}
	mt.recvCh <- makeRPCErrorResponse(s, first.msgID, 500, "INTERNAL")
	second := nextAttempt()
	if second.msgID == first.msgID {
		t.Fatal("bind retry reused the outer message ID")
	}
	if second.nonce == first.nonce {
		t.Fatal("bind retry reused the nonce")
	}
	if bytes.Equal(second.encryptedMessage, first.encryptedMessage) {
		t.Fatal("bind retry reused the encrypted payload")
	}
	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.RPCResult{
		ReqMsgID: second.msgID,
		Result:   &tg.BoolTrue{},
	})
	select {
	case err := <-bindDone:
		if err != nil {
			t.Fatalf("Bind() after retry = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Bind() did not complete")
	}
}

func TestBindTempAuthKeyRequiresBoolTrue(t *testing.T) {
	tests := []struct {
		name   string
		result tg.TLObject
	}{
		{name: "false", result: &tg.BoolFalse{}},
		{name: "decoded false", result: tg.TLBool(false)},
		{name: "unexpected", result: &tg.Pong{}},
		{name: "nil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewTempKeyManager(2, false, make([]byte, 256), true, nil)
			manager.mu.Lock()
			manager.tempKey = make([]byte, 256)
			manager.expiresAt = time.Now().Add(time.Hour)
			manager.bindExpiresAt = int32(time.Now().Add(time.Hour).Unix())
			manager.mu.Unlock()

			err := manager.Bind(context.Background(), 123, func(context.Context, tg.TLObject, int, time.Duration) (tg.TLObject, error) {
				return tt.result, nil
			})
			if err == nil {
				t.Fatal("Bind() succeeded without BoolTrue")
			}
			if manager.IsBound() {
				t.Fatal("failed bind marked key as bound")
			}
			if manager.NeedsInitConnection() {
				t.Fatal("failed bind requested initConnection")
			}
			if epoch := manager.BindEpoch(); epoch != 0 {
				t.Fatalf("bind epoch = %d, want 0", epoch)
			}
		})
	}
}

func newPFSRecoveryTestSession(t *testing.T) (*Session, *mockTransport, *TempKeyManager, func()) {
	t.Helper()
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)

	pfs := NewTempKeyManager(2, false, make([]byte, 256), true, nil, time.Now())
	pfs.tempKey = make([]byte, 256)
	pfs.issuedAt = time.Now()
	pfs.expiresAt = time.Now().Add(time.Hour)
	pfs.bound = true
	pfs.bindEpoch = 1
	s.SetPFS(pfs)
	return s, mt, pfs, cleanup
}

func startBlockedPFSWrite(t *testing.T, s *Session) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- s.writeEncrypted(context.Background(), []byte{1}, time.Second)
	}()
	select {
	case err := <-done:
		t.Fatalf("application write passed the PFS recovery gate: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	return done
}

func TestCancelledWaitResponseDoesNotBlockOnPFSRecovery(t *testing.T) {
	s := newSessionWithAuthKey(t)
	msgID := s.msgFactory.AllocateMsgID()
	handle, err := s.pending.Register(msgID, false)
	if err != nil {
		t.Fatalf("Register() = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s.pfsWriteMu.Lock()
	done := make(chan error, 1)
	go func() {
		_, err := s.waitResponse(ctx, handle, msgID, time.Second)
		done <- err
	}()

	select {
	case err := <-done:
		s.pfsWriteMu.Unlock()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waitResponse() = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		s.pfsWriteMu.Unlock()
		<-done
		t.Fatal("canceled waitResponse blocked behind PFS recovery")
	}
}

func assertPFSRecoveryFailureStopsWrites(t *testing.T, s *Session, mt *mockTransport, blockedWrite <-chan error) {
	t.Helper()
	if !s.stopping.Load() {
		t.Fatal("PFS recovery failure did not stop the session")
	}
	source, _, cause := s.ShutdownCause()
	if source != "pfsRecovery" || cause == nil {
		t.Fatalf("ShutdownCause() = (%q, %v), want pfsRecovery failure", source, cause)
	}
	select {
	case err := <-blockedWrite:
		if !errors.Is(err, ErrSessionClosed) {
			t.Fatalf("blocked write error = %v, want ErrSessionClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked write remained stuck after PFS recovery failed")
	}

	_, err := s.Send(
		context.Background(),
		s.msgFactory.AllocateMsgID(),
		uint32(s.msgFactory.AllocateSeqNo(true)),
		&tg.PingRequest{PingID: 99},
		time.Second,
	)
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Send() after PFS recovery failure = %v, want ErrSessionClosed", err)
	}
	select {
	case <-mt.sendCh:
		t.Fatal("session wrote to the transport after PFS recovery failure")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestPFSRebindFailureStopsWrites(t *testing.T) {
	s, mt, pfs, cleanup := newPFSRecoveryTestSession(t)
	defer cleanup()

	recoveryDone := make(chan error, 1)
	go func() {
		recoveryDone <- s.recoverPFSBinding(context.Background(), pfs, pfs.BindEpoch())
	}()
	bindMsgID, constructor := nextOutgoingConstructor(t, s, mt)
	if constructor != tg.AuthBindTempAuthKeyTypeID {
		t.Fatalf("recovery constructor = 0x%x, want auth.bindTempAuthKey", constructor)
	}
	blockedWrite := startBlockedPFSWrite(t, s)
	mt.recvCh <- makeRPCErrorResponse(s, bindMsgID, 401, "AUTH_KEY_PERM_EMPTY")

	select {
	case err := <-recoveryDone:
		if !tgerr.Is(err, tgerr.ErrAuthKeyPermEmpty) {
			t.Fatalf("recoverPFSBinding() = %v, want AUTH_KEY_PERM_EMPTY", err)
		}
	case <-time.After(time.Second):
		t.Fatal("PFS rebind failure did not return")
	}
	assertPFSRecoveryFailureStopsWrites(t, s, mt, blockedWrite)
}

func TestPFSInitConnectionFailureStopsWrites(t *testing.T) {
	s, mt, pfs, cleanup := newPFSRecoveryTestSession(t)
	defer cleanup()

	initErr := errors.New("initConnection failed")
	initEntered := make(chan struct{})
	releaseInit := make(chan struct{})
	s.SetPFSInitConnection(func(context.Context) error {
		close(initEntered)
		<-releaseInit
		return initErr
	})
	recoveryDone := make(chan error, 1)
	go func() {
		recoveryDone <- s.recoverPFSBinding(context.Background(), pfs, pfs.BindEpoch())
	}()
	bindMsgID, constructor := nextOutgoingConstructor(t, s, mt)
	if constructor != tg.AuthBindTempAuthKeyTypeID {
		t.Fatalf("recovery constructor = 0x%x, want auth.bindTempAuthKey", constructor)
	}
	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.RPCResult{
		ReqMsgID: bindMsgID,
		Result:   &tg.BoolTrue{},
	})
	select {
	case <-initEntered:
	case <-time.After(time.Second):
		t.Fatal("PFS initConnection callback was not called")
	}
	blockedWrite := startBlockedPFSWrite(t, s)
	close(releaseInit)

	select {
	case err := <-recoveryDone:
		if !errors.Is(err, initErr) {
			t.Fatalf("recoverPFSBinding() = %v, want initConnection failure", err)
		}
	case <-time.After(time.Second):
		t.Fatal("PFS initConnection failure did not return")
	}
	assertPFSRecoveryFailureStopsWrites(t, s, mt, blockedWrite)
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
