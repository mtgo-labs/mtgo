package session

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

func encodeGzipResult(t *testing.T, obj tg.TLObject) []byte {
	t.Helper()
	var buf bytes.Buffer
	tg.WriteInt(&buf, tg.GzipPackedID)
	if err := (&tg.GzipPacked{Data: obj}).Encode(&buf); err != nil {
		t.Fatalf("encode gzip result: %v", err)
	}
	return buf.Bytes()
}

func TestCheckRawRPCErrorRejectsMalformedError(t *testing.T) {
	var data bytes.Buffer
	tg.WriteInt(&data, tg.RPCErrorTypeID)
	if err := checkRawRPCError(data.Bytes()); err == nil {
		t.Fatal("malformed RPCError was accepted")
	}
}

func TestCheckRawRPCErrorDetectsGzipPackedError(t *testing.T) {
	err := checkRawRPCError(encodeGzipResult(t, &tg.RPCError{
		ErrorCode:    420,
		ErrorMessage: "FLOOD_WAIT_1",
	}))
	rpcErr, ok := tgerr.As(err)
	if !ok || rpcErr.Code != 420 || rpcErr.Type != "FLOOD_WAIT" {
		t.Fatalf("error = %T %v, want FLOOD_WAIT RPC error", err, err)
	}
}

func TestCheckRawRPCErrorPreservesSuccessfulGzipPayload(t *testing.T) {
	if err := checkRawRPCError(encodeGzipResult(t, &tg.Pong{})); err != nil {
		t.Fatalf("successful gzip payload: %v", err)
	}
}

func TestSessionSendRawCancellationDoesNotWriteDrop(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	done := make(chan error, 1)
	go func() {
		_, err := s.SendRaw(ctx, msgID, uint32(seqNo), encodeTLObject(t, &tg.PingRequest{PingID: 1}), 5*time.Second)
		done <- err
	}()

	<-mt.sendCh
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("SendRaw error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SendRaw did not return promptly after cancellation")
	}

	select {
	case <-mt.sendCh:
		t.Fatal("SendRaw cancellation wrote rpc_drop_answer")
	case <-time.After(20 * time.Millisecond):
	}
}
