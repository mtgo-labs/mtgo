package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
)

// unpackEnvelope decrypts a message and returns the envelope (msgID, seqNo)
// without TL-deserializing the body. Useful for inspecting sent messages whose
// body type is not in the decode Registry (e.g. RPCDropAnswerRequest).
func unpackEnvelope(data []byte, s *Session) *tg.MTProtoMessageRaw {
	msg, _, _ := crypto.UnpackEnvelope(data, s.sessionIDBytes(), s.authKey, s.authKeyID)
	return msg
}

// TestSessionDropRPCRejectsPendingHandle verifies that DropRPC sends
// rpc_drop_answer and rejects the pending handle for the target msgID.
func TestSessionDropRPCRejectsPendingHandle(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	// Register a pending handle as if an RPC was sent.
	targetMsgID := s.msgFactory.AllocateMsgID()
	_, err := s.pending.Register(targetMsgID, false)
	if err != nil {
		t.Fatalf("Register() error: %v", err)
	}

	dropDone := make(chan error, 1)
	go func() {
		dropDone <- s.DropRPC(context.Background(), targetMsgID)
	}()

	// The session sends the RPCDropAnswerRequest.
	sent := <-mt.sendCh
	sentEnv := unpackEnvelope(sent, s)
	if sentEnv == nil {
		t.Fatal("DropRPC did not send a message")
	}

	// Verify the body constructor ID is RPCDropAnswerRequest (0x58e4a740).
	if len(sentEnv.BodyRaw) < 4 {
		t.Fatalf("body too short: %d bytes", len(sentEnv.BodyRaw))
	}
	bodyID := uint32(sentEnv.BodyRaw[0]) | uint32(sentEnv.BodyRaw[1])<<8 |
		uint32(sentEnv.BodyRaw[2])<<16 | uint32(sentEnv.BodyRaw[3])<<24
	if bodyID != tg.RPCDropAnswerTypeID {
		t.Errorf("body constructor = 0x%x, want 0x%x (RPCDropAnswerRequest)", bodyID, tg.RPCDropAnswerTypeID)
	}
	// Verify the ReqMsgID field (int64 at offset 4).
	if len(sentEnv.BodyRaw) >= 12 {
		reqMsgID := int64(sentEnv.BodyRaw[4]) | int64(sentEnv.BodyRaw[5])<<8 |
			int64(sentEnv.BodyRaw[6])<<16 | int64(sentEnv.BodyRaw[7])<<24 |
			int64(sentEnv.BodyRaw[8])<<32 | int64(sentEnv.BodyRaw[9])<<40 |
			int64(sentEnv.BodyRaw[10])<<48 | int64(sentEnv.BodyRaw[11])<<56
		if reqMsgID != targetMsgID {
			t.Errorf("ReqMsgID = %d, want %d", reqMsgID, targetMsgID)
		}
	}

	// Server responds with RPCAnswerDropped.
	mt.recvCh <- makeEncryptedResponse(
		s, makeServerMsgID(),
		uint32(s.msgFactory.AllocateSeqNo(false)),
		&tg.RPCResult{ReqMsgID: sentEnv.MsgID, Result: &tg.RPCAnswerDropped{MsgID: targetMsgID, SeqNo: 1, Bytes: 4}},
	)

	// DropRPC should return nil.
	select {
	case err := <-dropDone:
		if err != nil {
			t.Fatalf("DropRPC() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("DropRPC() timed out")
	}

	// The pending handle for targetMsgID should be rejected (removed from map).
	if s.pending.Has(targetMsgID) {
		t.Error("pending handle still exists after DropRPC")
	}
}

// TestSessionDropRPCAnswerUnknown verifies that RPCAnswerUnknown is handled
// gracefully (the server has no record of the msgID).
func TestSessionDropRPCAnswerUnknown(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	dropDone := make(chan error, 1)
	go func() {
		dropDone <- s.DropRPC(context.Background(), 99999)
	}()

	// Read the RPCDropAnswerRequest.
	sent := <-mt.sendCh
	sentEnv := unpackEnvelope(sent, s)
	if sentEnv == nil {
		t.Fatal("DropRPC did not send a message")
	}

	// Server responds with RPCAnswerUnknown.
	mt.recvCh <- makeEncryptedResponse(
		s, makeServerMsgID(),
		uint32(s.msgFactory.AllocateSeqNo(false)),
		&tg.RPCResult{ReqMsgID: sentEnv.MsgID, Result: &tg.RPCAnswerUnknown{}},
	)

	select {
	case err := <-dropDone:
		if err != nil {
			t.Fatalf("DropRPC() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("DropRPC() timed out")
	}
}

// TestSessionDropRPCRejectsOriginalCaller verifies that when DropRPC is called
// while an Invoke is in-flight, the original Invoke caller receives
// ErrRPCDropped.
func TestSessionDropRPCRejectsOriginalCaller(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	// Start an Invoke that will hang (no server response).
	pingID := time.Now().UnixNano()
	invokeDone := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 1, 30*time.Second)
		invokeDone <- struct {
			obj tg.TLObject
			err error
		}{obj, err}
	}()

	// Wait for the Invoke's PingRequest to be sent so we can capture its msgID.
	sent := <-mt.sendCh
	sentEnv := unpackEnvelope(sent, s)
	if sentEnv == nil {
		t.Fatal("Invoke did not send a message")
	}
	targetMsgID := sentEnv.MsgID

	// Now drop the RPC.
	dropDone := make(chan error, 1)
	go func() {
		dropDone <- s.DropRPC(context.Background(), targetMsgID)
	}()

	// Read the RPCDropAnswerRequest sent by DropRPC.
	dropSent := <-mt.sendCh
	dropEnv := unpackEnvelope(dropSent, s)
	if dropEnv == nil {
		t.Fatal("DropRPC did not send a message")
	}

	// Server responds with RPCAnswerDroppedRunning.
	mt.recvCh <- makeEncryptedResponse(
		s, makeServerMsgID(),
		uint32(s.msgFactory.AllocateSeqNo(false)),
		&tg.RPCResult{ReqMsgID: dropEnv.MsgID, Result: &tg.RPCAnswerDroppedRunning{}},
	)

	// DropRPC completes.
	select {
	case err := <-dropDone:
		if err != nil {
			t.Fatalf("DropRPC() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("DropRPC() timed out")
	}

	// The original Invoke should receive ErrRPCDropped.
	select {
	case result := <-invokeDone:
		if result.err == nil {
			t.Fatal("Invoke() returned nil error, want ErrRPCDropped")
		}
		if !errors.Is(result.err, ErrRPCDropped) {
			t.Errorf("Invoke() error = %v, want errors.Is(err, ErrRPCDropped)", result.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Invoke() did not unblock after DropRPC")
	}
}
