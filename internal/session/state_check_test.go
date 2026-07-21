package session

import (
	"testing"
	"time"
)

func TestPendingManagerOverduePending(t *testing.T) {
	pm := NewPendingManager()

	// Register a pending query.
	h1, err := pm.Register(100, false)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Simulate it being sent in the past by overwriting sentAt.
	h1.sentAt = time.Now().Add(-2 * time.Second).UnixNano()

	// Register another that was just sent (not overdue).
	h2, err := pm.Register(200, false)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	_ = h2

	// Raw queries use the same MTProto delivery reconciliation.
	h3, err := pm.Register(300, true)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	h3.sentAt = time.Now().Add(-2 * time.Second).UnixNano()

	// Both overdue decoded and raw queries should be returned.
	ids := pm.OverduePending(1 * time.Second)
	if len(ids) != 2 || !containsMsgID(ids, 100) || !containsMsgID(ids, 300) {
		t.Fatalf("OverduePending: got %v, want 100 and 300", ids)
	}
}

func containsMsgID(ids []int64, target int64) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func TestPendingManagerMarkAckedExcludesFromOverdue(t *testing.T) {
	pm := NewPendingManager()

	h, err := pm.Register(42, false)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	h.sentAt = time.Now().Add(-5 * time.Second).UnixNano()

	// Before ack: should be overdue.
	ids := pm.OverduePending(1 * time.Second)
	if len(ids) != 1 {
		t.Fatalf("before ack: OverduePending returned %d ids, want 1", len(ids))
	}

	// Mark as acked.
	pm.MarkAcked(42)

	// After ack: should not be overdue.
	ids = pm.OverduePending(1 * time.Second)
	if len(ids) != 0 {
		t.Fatalf("after ack: OverduePending returned %d ids, want 0", len(ids))
	}
}

func TestHandleStateInfoRejectsNotReceived(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.stateReqs = make(map[int64]*pendingStateReq)

	// Register a pending query.
	h, err := s.pending.Register(500, false)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	h.sentAt = time.Now().Add(-2 * time.Second).UnixNano()

	// Record a state request for msgID 500.
	stateReqMsgID := int64(999)
	s.stateReqs[stateReqMsgID] = &pendingStateReq{
		msgIDs: []int64{500},
		sentAt: time.Now(),
	}

	// Server responds: status byte 0x02 = not received.
	s.handleStateInfo(stateReqMsgID, string([]byte{0x02}))

	// The pending handle should be rejected.
	select {
	case <-h.Done():
		_, _, err := h.Result()
		if err == nil {
			t.Fatal("expected error from rejected handle")
		}
	case <-time.After(time.Second):
		t.Fatal("handle was not rejected after state info 'not received'")
	}

	// The state request should be consumed.
	s.stateReqMu.Lock()
	_, ok := s.stateReqs[stateReqMsgID]
	s.stateReqMu.Unlock()
	if ok {
		t.Fatal("state request should be removed after processing")
	}
}

func TestHandleStateInfoKeepsReceived(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.stateReqs = make(map[int64]*pendingStateReq)

	h, err := s.pending.Register(600, false)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	stateReqMsgID := int64(888)
	s.stateReqs[stateReqMsgID] = &pendingStateReq{
		msgIDs: []int64{600},
		sentAt: time.Now(),
	}

	// Server responds: status byte 0x44 = received with a generated response.
	s.handleStateInfo(stateReqMsgID, string([]byte{0x44}))

	// The pending handle should still be active (not completed).
	select {
	case <-h.Done():
		t.Fatal("handle should NOT be completed for status 'received'")
	default:
		// Good — still pending.
	}
	if !h.IsAcked() {
		t.Fatal("received handle should be marked acknowledged")
	}
}

func TestHandleStateInfoIgnoresUnknownReqMsgID(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.stateReqs = make(map[int64]*pendingStateReq)

	h, err := s.pending.Register(700, false)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Call with a req_msg_id we never sent.
	s.handleStateInfo(12345, string([]byte{0x01}))

	// Handle should still be active.
	select {
	case <-h.Done():
		t.Fatal("handle should not be affected by unknown state request")
	default:
	}
}

func TestHandleStateInfoMultipleMessages(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.stateReqs = make(map[int64]*pendingStateReq)

	h1, _ := s.pending.Register(101, false)
	h2, _ := s.pending.Register(102, false)
	h3, _ := s.pending.Register(103, false)

	stateReqMsgID := int64(777)
	s.stateReqs[stateReqMsgID] = &pendingStateReq{
		msgIDs: []int64{101, 102, 103},
		sentAt: time.Now(),
	}

	// Status bytes: 0x01 (unknown), 0x03 (not received), 0x04 (received).
	s.handleStateInfo(stateReqMsgID, string([]byte{0x01, 0x03, 0x04}))

	// h1 should remain pending because delivery is unknown (0x01).
	select {
	case <-h1.Done():
		t.Fatal("h1 should remain pending")
	default:
	}

	// h2 should be rejected (0x03 = definitely not received).
	select {
	case <-h2.Done():
	case <-time.After(time.Second):
		t.Fatal("h2 should be rejected")
	}

	// h3 should remain pending and be marked acknowledged (0x04).
	select {
	case <-h3.Done():
		t.Fatal("h3 should remain pending")
	default:
	}
	if !h3.IsAcked() {
		t.Fatal("h3 should be marked acknowledged")
	}
}

func TestExpireStateReqs(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.stateReqs = make(map[int64]*pendingStateReq)

	// Add an old state request (expired).
	s.stateReqs[1] = &pendingStateReq{
		msgIDs: []int64{1},
		sentAt: time.Now().Add(-5 * time.Second),
	}

	// Add a fresh one.
	s.stateReqs[2] = &pendingStateReq{
		msgIDs: []int64{2},
		sentAt: time.Now(),
	}

	s.expireStateReqs()

	s.stateReqMu.Lock()
	_, hasOld := s.stateReqs[1]
	_, hasFresh := s.stateReqs[2]
	s.stateReqMu.Unlock()

	if hasOld {
		t.Fatal("expired state request should be removed")
	}
	if !hasFresh {
		t.Fatal("fresh state request should remain")
	}
}
