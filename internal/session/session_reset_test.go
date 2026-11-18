package session

import (
	"context"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// waitSent receives an outbound frame from the mock transport or fails the
// test instead of blocking forever.
func waitSent(t *testing.T, mt *mockTransport) []byte {
	t.Helper()
	select {
	case b := <-mt.sendCh:
		return b
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for an outbound message")
		return nil
	}
}

// TestBadServerSaltOnPingSatisfiesWait covers the keep-alive ping being
// rejected with bad_server_salt: the salt is stored and the ping's wait is
// completed, so the ping loop does not run into its pong timeout and force a
// reconnect on every salt expiry.
func TestBadServerSaltOnPingSatisfiesWait(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	// Arm exactly what pingLoop arms: build (msg_id known), register, write.
	pingID := time.Now().UnixNano()
	pongCh := make(chan struct{})
	encrypted, msgID, err := s.buildServiceMessageID(&tg.PingDelayDisconnectRequest{
		PingID:          pingID,
		DisconnectDelay: 60,
	})
	if err != nil {
		t.Fatalf("buildServiceMessageID: %v", err)
	}
	s.pingMux.Lock()
	s.pingCbs[pingID] = pongCh
	s.curPingMsgID = msgID
	s.curPingID = pingID
	s.curPongCh = pongCh
	s.pingMux.Unlock()
	mt.sendCh <- encrypted

	newSalt := int64(0x5A17BEEF)
	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.BadServerSalt{
		BadMsgID:      msgID,
		BadMsgSeqno:   1,
		ErrorCode:     48,
		NewServerSalt: newSalt,
	})

	select {
	case <-pongCh:
	case <-time.After(10 * time.Second):
		t.Fatal("rejected ping wait never satisfied; ping loop would reconnect")
	}
	if got := s.saltMgr.Load(); got != newSalt {
		t.Fatalf("salt = %x, want %x", got, newSalt)
	}
	s.pingMux.Lock()
	_, stillRegistered := s.pingCbs[pingID]
	s.pingMux.Unlock()
	if stillRegistered {
		t.Fatal("ping callback not cleaned up after rejection")
	}
}

// TestSessionCode17RotatesSessionAndRetries covers bad_msg_notification code
// 17 (msg_id too high, local clock ahead): the time offset is corrected, the
// session id rotates so the retry's lower msg_ids are valid in a fresh
// session, and the request eventually succeeds.
func TestSessionCode17RotatesSessionAndRetries(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	oldSID := s.SessionID()
	oldOffset := s.msgFactory.ServerTimeOffset()

	invokeDone := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	pingID := time.Now().UnixNano()
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 3, 10*time.Second)
		invokeDone <- struct {
			obj tg.TLObject
			err error
		}{obj: obj, err: err}
	}()

	firstSent := waitSent(t, mt)
	firstMsg := unpackIncoming(firstSent, s)
	if firstMsg == nil {
		t.Fatal("first sent message did not decode")
	}

	// The server reports our msg_id as too high; its msg_id carries the real
	// server time, four minutes behind our skewed local clock (within the
	// inbound replay window so the read loop processes the frame).
	serverTime := time.Now().Add(-4 * time.Minute).Unix()
	notificationMsgID := serverTime<<32 | 1
	mt.recvCh <- makeEncryptedResponse(s, notificationMsgID, uint32(s.msgFactory.AllocateSeqNo(false)), &tg.BadMsgNotification{
		BadMsgID:    firstMsg.MsgID,
		BadMsgSeqno: 1,
		ErrorCode:   17,
	})

	secondSent := waitSent(t, mt)
	secondMsg := unpackIncoming(secondSent, s)
	if secondMsg == nil {
		t.Fatal("retry message did not decode")
	}
	if secondMsg.MsgID == firstMsg.MsgID {
		t.Fatal("retry reused the rejected msg_id")
	}
	if s.SessionID() == oldSID {
		t.Fatal("session id not rotated after code 17; retried msg_ids stay below the session high-water mark")
	}
	if got, want := s.msgFactory.ServerTimeOffset(), oldOffset-240; got > want+5 {
		t.Fatalf("server time offset = %d, want ~%d (corrected backward)", got, want)
	}

	// After the backward time correction the validator's reference clock is
	// the corrected server time; the response envelope must carry a matching
	// timestamp or it is dropped as "from the future".
	correctedMsgID := (time.Now().Unix()+s.msgFactory.ServerTimeOffset())<<32 | 3
	mt.recvCh <- makeEncryptedResponse(s, correctedMsgID, uint32(s.msgFactory.AllocateSeqNo(false)), &tg.Pong{
		MsgID:  secondMsg.MsgID,
		PingID: pingID,
	})

	select {
	case res := <-invokeDone:
		if res.err != nil {
			t.Fatalf("Invoke error: %v", res.err)
		}
		if _, ok := res.obj.(*tg.Pong); !ok {
			t.Fatalf("Invoke result = %T, want *tg.Pong", res.obj)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Invoke did not complete after session reset")
	}
}

// TestSessionInvokeRetriesConnectionLayerInvalid covers the G12 retry for
// CONNECTION_LAYER_INVALID: the query is re-sent instead of failing.
func TestSessionInvokeRetriesConnectionLayerInvalid(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	invokeDone := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	pingID := time.Now().UnixNano()
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 3, 10*time.Second)
		invokeDone <- struct {
			obj tg.TLObject
			err error
		}{obj: obj, err: err}
	}()

	firstMsg := unpackIncoming(waitSent(t, mt), s)
	if firstMsg == nil {
		t.Fatal("first sent message did not decode")
	}
	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.RPCResult{
		ReqMsgID: firstMsg.MsgID,
		Result: &tg.RPCError{
			ErrorCode:    400,
			ErrorMessage: "CONNECTION_LAYER_INVALID",
		},
	})

	secondMsg := unpackIncoming(waitSent(t, mt), s)
	if secondMsg == nil {
		t.Fatal("retry message did not decode")
	}
	// After the backward time correction the validator's reference clock is
	// the corrected server time; the response envelope must carry a matching
	// timestamp or it is dropped as "from the future".
	correctedMsgID := (time.Now().Unix()+s.msgFactory.ServerTimeOffset())<<32 | 3
	mt.recvCh <- makeEncryptedResponse(s, correctedMsgID, uint32(s.msgFactory.AllocateSeqNo(false)), &tg.Pong{
		MsgID:  secondMsg.MsgID,
		PingID: pingID,
	})

	select {
	case res := <-invokeDone:
		if res.err != nil {
			t.Fatalf("Invoke error: %v", res.err)
		}
		if _, ok := res.obj.(*tg.Pong); !ok {
			t.Fatalf("Invoke result = %T, want *tg.Pong", res.obj)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Invoke did not complete after CONNECTION_LAYER_INVALID retry")
	}
}
