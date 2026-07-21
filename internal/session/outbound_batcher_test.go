package session

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

type failingSendTransport struct {
	*mockTransport
	err error
}

func (m *failingSendTransport) Send([]byte) error { return m.err }

// newBatcherTestSession creates a session with mock transport and test workers
// suitable for batcher testing. Returns the session, transport, and cleanup.
func newBatcherTestSession(t *testing.T) (*Session, *mockTransport, func()) {
	t.Helper()
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt)
	return s, mt, cleanup
}

func TestBatcher_DisabledByDefault(t *testing.T) {
	s, _, cleanup := newBatcherTestSession(t)
	defer cleanup()
	if s.outboundBatcher != nil {
		t.Fatal("batcher should be nil by default")
	}
}

func TestSessionSendHighPriorityBypassesBatcher(t *testing.T) {
	s, mt, cleanup := newBatcherTestSession(t)
	defer cleanup()

	s.EnableOutboundBatching(1<<20, 5*time.Second)
	defer s.CloseOutboundBatching()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	pingID := time.Now().UnixNano()
	sendDone := make(chan error, 1)
	go func() {
		_, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: pingID}, 5*time.Second)
		sendDone <- err
	}()

	select {
	case <-mt.sendCh:
	case <-time.After(time.Second):
		t.Fatal("high-priority RPC waited for the batching window")
	}

	snap := s.BatcherSnapshot()
	if snap.HighDepth != 0 || snap.MessagesPacked != 0 {
		t.Fatalf("high-priority RPC entered batcher: %+v", snap)
	}

	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.Pong{
		MsgID:  msgID,
		PingID: pingID,
	})
	if err := <-sendDone; err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestSessionSendLowPriorityUsesBatcher(t *testing.T) {
	s, mt, cleanup := newBatcherTestSession(t)
	defer cleanup()

	s.EnableOutboundBatching(1<<20, 5*time.Second)
	defer s.CloseOutboundBatching()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	sendDone := make(chan error, 1)
	go func() {
		_, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.UploadSaveFilePartRequest{
			FileID:   1,
			FilePart: 0,
			Bytes:    []byte{1},
		}, 10*time.Second)
		sendDone <- err
	}()

	deadline := time.Now().Add(time.Second)
	for s.BatcherSnapshot().LowDepth != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if snap := s.BatcherSnapshot(); snap.LowDepth != 1 {
		t.Fatalf("low-priority RPC did not enter batcher: %+v", snap)
	}
	select {
	case <-mt.sendCh:
		t.Fatal("low-priority RPC bypassed the batching window")
	default:
	}

	wantErr := errors.New("test complete")
	s.pending.Reject(msgID, wantErr)
	if err := <-sendDone; !errors.Is(err, wantErr) {
		t.Fatalf("Send error = %v, want %v", err, wantErr)
	}
}

func TestBatcher_LoneRPCFlushesAfterWindow(t *testing.T) {
	s, mt, cleanup := newBatcherTestSession(t)
	defer cleanup()

	s.EnableOutboundBatching(1<<20, time.Millisecond)
	defer s.CloseOutboundBatching()
	time.Sleep(50 * time.Millisecond) // let flush loop start

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	handle, err := s.outboundBatcher.Submit(
		context.Background(), msgID, uint32(seqNo),
		&tg.PingRequest{PingID: 42}, PriorityHigh, 5*time.Second,
	)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Wait for the flush to send.
	time.Sleep(100 * time.Millisecond)

	// The mock transport should have received exactly one encrypted payload.
	select {
	case <-mt.sendCh:
		// Good — data was sent.
	default:
		t.Fatal("no data sent for lone RPC")
	}

	// Cancel the pending handle since we won't provide a server response.
	s.pending.Cancel(msgID)
	_ = handle

	snap := s.outboundBatcher.Snapshot()
	if snap.ContainersSent != 1 {
		t.Errorf("ContainersSent = %d, want 1", snap.ContainersSent)
	}
	if snap.MessagesPacked != 1 {
		t.Errorf("MessagesPacked = %d, want 1", snap.MessagesPacked)
	}
}

func TestBatcher_BurstCoalesced(t *testing.T) {
	s, mt, cleanup := newBatcherTestSession(t)
	defer cleanup()

	s.EnableOutboundBatching(1<<20, 5*time.Millisecond)
	defer s.CloseOutboundBatching()
	time.Sleep(50 * time.Millisecond)

	const n = 10
	for i := 0; i < n; i++ {
		msgID := s.msgFactory.AllocateMsgID()
		seqNo := s.msgFactory.AllocateSeqNo(true)
		_, err := s.outboundBatcher.Submit(
			context.Background(), msgID, uint32(seqNo),
			&tg.PingRequest{PingID: int64(i)}, PriorityHigh, 5*time.Second,
		)
		if err != nil {
			t.Fatalf("Submit %d: %v", i, err)
		}
	}

	// Wait for flush (coalesce window + processing).
	time.Sleep(200 * time.Millisecond)

	snap := s.outboundBatcher.Snapshot()
	if snap.MessagesPacked != n {
		t.Errorf("MessagesPacked = %d, want %d", snap.MessagesPacked, n)
	}
	// Coalesced: fewer containers than messages.
	if snap.ContainersSent >= n {
		t.Errorf("ContainersSent = %d, expected < %d (not coalesced)", snap.ContainersSent, n)
	}
	if snap.ContainersSent == 0 {
		t.Error("ContainersSent = 0, expected at least 1")
	}

	// Verify data was sent to transport.
	sentCount := 0
	for {
		select {
		case <-mt.sendCh:
			sentCount++
		default:
			goto done
		}
	}
done:
	if sentCount == 0 {
		t.Fatal("no data sent to transport")
	}
}

func TestBatcher_ResendsChildAsEncryptedMessage(t *testing.T) {
	s, mt, cleanup := newBatcherTestSession(t)
	defer cleanup()

	s.EnableOutboundBatching(1<<20, time.Millisecond)
	defer s.CloseOutboundBatching()

	msgID1 := s.msgFactory.AllocateMsgID()
	seqNo1 := uint32(s.msgFactory.AllocateSeqNo(true))
	msgID2 := s.msgFactory.AllocateMsgID()
	seqNo2 := uint32(s.msgFactory.AllocateSeqNo(true))
	if _, err := s.outboundBatcher.Submit(context.Background(), msgID1, seqNo1, &tg.PingRequest{PingID: 11}, PriorityHigh, 5*time.Second); err != nil {
		t.Fatalf("Submit first: %v", err)
	}
	if _, err := s.outboundBatcher.Submit(context.Background(), msgID2, seqNo2, &tg.PingRequest{PingID: 22}, PriorityHigh, 5*time.Second); err != nil {
		t.Fatalf("Submit second: %v", err)
	}

	select {
	case <-mt.sendCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for initial container")
	}

	var resendReq bytes.Buffer
	tg.WriteVectorLong(&resendReq, []int64{msgID1})
	s.handleRawMsgResendReq(resendReq.Bytes())

	select {
	case payload := <-mt.sendCh:
		msg, err := unpackTestOutgoing(s, payload)
		if err != nil {
			t.Fatalf("unpack resent message: %v", err)
		}
		if msg.MsgID != msgID1 || msg.SeqNo != seqNo1 {
			t.Fatalf("resent envelope = (%d, %d), want (%d, %d)", msg.MsgID, msg.SeqNo, msgID1, seqNo1)
		}
		ping, ok := msg.Body.(*tg.PingRequest)
		if !ok || ping.PingID != 11 {
			t.Fatalf("resent body = %#v, want PingRequest(11)", msg.Body)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for resent child")
	}

	s.pending.Cancel(msgID1)
	s.pending.Cancel(msgID2)
}

func TestBatcher_WriteFailureCompletesCaller(t *testing.T) {
	s := newSessionWithAuthKey(t)
	wantErr := errors.New("write failed")
	mt := &failingSendTransport{mockTransport: newMockTransport(), err: wantErr}
	s.SetTransport(mt)
	cleanup := startTestWorkers(s, mt.mockTransport)
	defer cleanup()

	s.EnableOutboundBatching(1<<20, time.Millisecond)
	defer s.CloseOutboundBatching()

	started := time.Now()
	msgID := s.msgFactory.AllocateMsgID()
	seqNo := uint32(s.msgFactory.AllocateSeqNo(true))
	_, err := s.Send(context.Background(), msgID, seqNo, &tg.UploadSaveFilePartRequest{
		FileID:   1,
		FilePart: 0,
		Bytes:    []byte{1},
	}, 2*time.Second)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Send error = %v, want wrapped %v", err, wantErr)
	}
	var deliveryErr *DeliveryError
	if !errors.As(err, &deliveryErr) || deliveryErr.State != DeliveryUnknown {
		t.Fatalf("Send error = %v, want unknown DeliveryError", err)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("write failure took %v; caller was not completed promptly", elapsed)
	}
}

func TestBatcher_PriorityOrdering(t *testing.T) {
	s, _, cleanup := newBatcherTestSession(t)
	defer cleanup()

	s.EnableOutboundBatching(1<<20, 5*time.Millisecond)
	defer s.CloseOutboundBatching()
	time.Sleep(50 * time.Millisecond)

	// Queue low-priority items first, then high-priority.
	for i := 0; i < 5; i++ {
		msgID := s.msgFactory.AllocateMsgID()
		seqNo := s.msgFactory.AllocateSeqNo(true)
		_, _ = s.outboundBatcher.Submit(
			context.Background(), msgID, uint32(seqNo),
			&tg.PingRequest{PingID: int64(100 + i)}, PriorityLow, 5*time.Second,
		)
	}
	for i := 0; i < 3; i++ {
		msgID := s.msgFactory.AllocateMsgID()
		seqNo := s.msgFactory.AllocateSeqNo(true)
		_, _ = s.outboundBatcher.Submit(
			context.Background(), msgID, uint32(seqNo),
			&tg.PingRequest{PingID: int64(i)}, PriorityHigh, 5*time.Second,
		)
	}

	time.Sleep(200 * time.Millisecond)

	snap := s.outboundBatcher.Snapshot()
	total := snap.MessagesPacked
	if total != 8 {
		t.Errorf("MessagesPacked = %d, want 8", total)
	}
}

func TestBatcher_NoLeak(t *testing.T) {
	before := runtime.NumGoroutine()

	s, _, cleanup := newBatcherTestSession(t)
	s.EnableOutboundBatching(1<<20, time.Millisecond)
	defer s.CloseOutboundBatching()
	time.Sleep(50 * time.Millisecond)

	// Submit a few items.
	for i := 0; i < 5; i++ {
		msgID := s.msgFactory.AllocateMsgID()
		seqNo := s.msgFactory.AllocateSeqNo(true)
		_, _ = s.outboundBatcher.Submit(
			context.Background(), msgID, uint32(seqNo),
			&tg.PingRequest{PingID: int64(i)}, PriorityHigh, 5*time.Second,
		)
	}
	time.Sleep(100 * time.Millisecond)

	s.CloseOutboundBatching()
	cleanup()

	time.Sleep(100 * time.Millisecond)
	after := runtime.NumGoroutine()
	if leaked := after - before; leaked > 0 {
		t.Fatalf("goroutine leak: before=%d after=%d (leaked %d)", before, after, leaked)
	}
}

func TestBatcher_ConcurrentSubmit(t *testing.T) {
	s, _, cleanup := newBatcherTestSession(t)
	defer cleanup()

	s.EnableOutboundBatching(1<<20, 5*time.Millisecond)
	defer s.CloseOutboundBatching()
	time.Sleep(50 * time.Millisecond)

	const goroutines = 10
	const perG = 5
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				msgID := s.msgFactory.AllocateMsgID()
				seqNo := s.msgFactory.AllocateSeqNo(true)
				_, _ = s.outboundBatcher.Submit(
					context.Background(), msgID, uint32(seqNo),
					&tg.PingRequest{PingID: 1}, PriorityHigh, 5*time.Second,
				)
			}
		}()
	}
	wg.Wait()

	time.Sleep(200 * time.Millisecond)

	snap := s.outboundBatcher.Snapshot()
	want := int64(goroutines * perG)
	if snap.MessagesPacked != want {
		t.Errorf("MessagesPacked = %d, want %d", snap.MessagesPacked, want)
	}
}
