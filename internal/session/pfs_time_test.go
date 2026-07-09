package session

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// encodeBadMsgNotification builds the raw TL body for a bad_msg_notification
// service message (constructor ID + bad_msg_id + bad_msg_seqno + error_code).
func encodeBadMsgNotification(badMsgID int64, seqno, errorCode int32) []byte {
	body := make([]byte, 20)
	binary.LittleEndian.PutUint32(body[0:4], tg.BadMsgNotificationTypeID)
	binary.LittleEndian.PutUint64(body[4:12], uint64(badMsgID))
	binary.LittleEndian.PutUint32(body[12:16], uint32(seqno))
	binary.LittleEndian.PutUint32(body[16:20], uint32(errorCode))
	return body
}

func TestBadMsgNotificationCode16AdjustsTimeAndRejects(t *testing.T) {
	s := newSessionWithAuthKey(t)

	// Use a server time well in the future so UpdateServerTime will advance.
	futureTime := time.Now().Add(2 * time.Hour)
	badMsgID := futureTime.Unix() << 32

	h, err := s.pending.Register(badMsgID, false)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	body := encodeBadMsgNotification(badMsgID, 0, 16)
	if !s.handleRawPacket(badMsgID, body) {
		t.Fatal("handleRawPacket returned false, want true")
	}

	<-h.Done()
	_, _, herr := h.Result()
	if herr == nil {
		t.Fatal("expected error from Reject, got nil")
	}

	// Verify server time was updated.
	newMsgID := s.msgFactory.AllocateMsgID()
	serverTime := newMsgID >> 32
	if serverTime < futureTime.Unix() {
		t.Fatalf("server time not advanced: got %d, want >= %d", serverTime, futureTime.Unix())
	}
}

func TestBadMsgNotificationCode17AdjustsTimeAndRejects(t *testing.T) {
	s := newSessionWithAuthKey(t)

	futureTime := time.Now().Add(2 * time.Hour)
	badMsgID := futureTime.Unix() << 32

	h, err := s.pending.Register(badMsgID, false)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	body := encodeBadMsgNotification(badMsgID, 0, 17)
	if !s.handleRawPacket(badMsgID, body) {
		t.Fatal("handleRawPacket returned false, want true")
	}

	<-h.Done()
	_, _, herr := h.Result()
	if herr == nil {
		t.Fatal("expected error from Reject, got nil")
	}

	// Verify server time was updated (forward, so UpdateServerTime advances).
	newMsgID := s.msgFactory.AllocateMsgID()
	serverTime := newMsgID >> 32
	if serverTime < futureTime.Unix() {
		t.Fatalf("server time not advanced: got %d, want >= %d", serverTime, futureTime.Unix())
	}
}

func TestBadMsgNotificationCode18ResolvesAsError(t *testing.T) {
	s := newSessionWithAuthKey(t)

	badMsgID := time.Now().Unix() << 32

	h, err := s.pending.Register(badMsgID, false)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	body := encodeBadMsgNotification(badMsgID, 0, 18)
	if !s.handleRawPacket(badMsgID, body) {
		t.Fatal("handleRawPacket returned false, want true")
	}

	<-h.Done()
	result, _, herr := h.Result()
	// For code 18, the handle should be Resolved with a BadMsgNotification (not rejected).
	if herr != nil {
		t.Fatalf("expected nil error for code 18, got %v", herr)
	}
	if result == nil {
		t.Fatal("expected non-nil result for code 18")
	}
	bad, ok := result.(*tg.BadMsgNotification)
	if !ok {
		t.Fatalf("expected *tg.BadMsgNotification, got %T", result)
	}
	if bad.ErrorCode != 18 {
		t.Fatalf("error code: got %d, want 18", bad.ErrorCode)
	}
	if bad.BadMsgID != badMsgID {
		t.Fatalf("bad msg id: got %d, want %d", bad.BadMsgID, badMsgID)
	}
}

func TestPFSRenewalLoopDisabledReturnsCtxErr(t *testing.T) {
	s := newSessionWithAuthKey(t)
	// PFS is not set — loop should block until context cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.pfsRenewalLoop(ctx)
	}()

	cancel()
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pfsRenewalLoop did not return after context cancellation")
	}
}

func TestPFSRenewalLoopDisabledExplicitNil(t *testing.T) {
	s := newSessionWithAuthKey(t)

	// Explicitly set a disabled PFS manager.
	mgr := NewTempKeyManager(2, false, make([]byte, 256), false, false, nil)
	s.SetPFS(mgr)

	if mgr.IsEnabled() {
		t.Fatal("PFS manager should be disabled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.pfsRenewalLoop(ctx)
	}()

	cancel()
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pfsRenewalLoop did not return after context cancellation")
	}
}
