package session

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
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

func TestBadMsgNotificationCode16AdjustsTimeAndResolves(t *testing.T) {
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
	result, _, herr := h.Result()
	if herr != nil {
		t.Fatalf("expected nil error for code 16, got %v", herr)
	}
	bad, ok := result.(*tg.BadMsgNotification)
	if !ok {
		t.Fatalf("expected *tg.BadMsgNotification, got %T", result)
	}
	if bad.ErrorCode != 16 {
		t.Fatalf("error code: got %d, want 16", bad.ErrorCode)
	}
	if bad.BadMsgID != badMsgID {
		t.Fatalf("bad msg id: got %d, want %d", bad.BadMsgID, badMsgID)
	}

	// Verify server time was updated.
	newMsgID := s.msgFactory.AllocateMsgID()
	serverTime := newMsgID >> 32
	if serverTime < futureTime.Unix() {
		t.Fatalf("server time not advanced: got %d, want >= %d", serverTime, futureTime.Unix())
	}
}

func TestBadMsgNotificationCode17AdjustsTimeAndResolves(t *testing.T) {
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
	result, _, herr := h.Result()
	if herr != nil {
		t.Fatalf("expected nil error for code 17, got %v", herr)
	}
	bad, ok := result.(*tg.BadMsgNotification)
	if !ok {
		t.Fatalf("expected *tg.BadMsgNotification, got %T", result)
	}
	if bad.ErrorCode != 17 {
		t.Fatalf("error code: got %d, want 17", bad.ErrorCode)
	}
	if bad.BadMsgID != badMsgID {
		t.Fatalf("bad msg id: got %d, want %d", bad.BadMsgID, badMsgID)
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
	mgr := NewTempKeyManager(2, false, make([]byte, 256), false, nil)
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

func TestTempKeyRotationUsesQuarterLifetimeMargin(t *testing.T) {
	now := time.Now()
	mgr := NewTempKeyManager(2, false, make([]byte, 256), true, nil)
	mgr.tempKey = make([]byte, 256)
	mgr.issuedAt = now.Add(-12 * time.Hour)
	mgr.expiresAt = now.Add(12 * time.Hour)
	if mgr.NeedsRotation() {
		t.Fatal("NeedsRotation = true at half lifetime")
	}
	due := mgr.rotationDueIn()
	if due < 5*time.Hour+59*time.Minute || due > 6*time.Hour+time.Minute {
		t.Fatalf("rotation due in %v, want about 6h", due)
	}

	mgr.issuedAt = now.Add(-19 * time.Hour)
	mgr.expiresAt = now.Add(5 * time.Hour)
	if !mgr.NeedsRotation() {
		t.Fatal("NeedsRotation = false after 75%% of lifetime")
	}
}

func TestGeneratedTempKeyUsesServerTimeForBindExpiry(t *testing.T) {
	mgr := NewTempKeyManager(2, false, make([]byte, 256), true, nil)
	localNow := time.Unix(2_000_000_000, 0)
	const (
		serverTime = int32(1_700_000_000)
		expiresIn  = int32(86_400)
	)
	mgr.installGeneratedKey(&AuthResult{
		AuthKey:    make([]byte, 256),
		ServerTime: serverTime,
	}, expiresIn, localNow)

	mgr.mu.Lock()
	bindExpiresAt := mgr.bindExpiresAt
	issuedAt := mgr.issuedAt
	expiresAt := mgr.expiresAt
	mgr.mu.Unlock()
	if bindExpiresAt != serverTime+expiresIn {
		t.Fatalf("bind expiry = %d, want server time + lifetime (%d)", bindExpiresAt, serverTime+expiresIn)
	}
	if !issuedAt.Equal(localNow) || !expiresAt.Equal(localNow.Add(time.Duration(expiresIn)*time.Second)) {
		t.Fatalf("local rotation window = %v..%v, want %v + %v", issuedAt, expiresAt, localNow, time.Duration(expiresIn)*time.Second)
	}
}

func TestPFSRenewalLoopUsesDeadlineTimer(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mgr := NewTempKeyManager(2, false, make([]byte, 256), true, nil)
	mgr.tempKey = make([]byte, 256)
	mgr.issuedAt = time.Now().Add(-2 * time.Minute)
	mgr.expiresAt = time.Now().Add(40*time.Second + 50*time.Millisecond)
	s.SetPFS(mgr)

	started := time.Now()
	err := s.pfsRenewalLoop(context.Background())
	if err == nil {
		t.Fatal("pfsRenewalLoop returned nil")
	}
	elapsed := time.Since(started)
	if elapsed < 20*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Fatalf("renewal fired after %v, want about 40ms", elapsed)
	}
}

func TestAuthKeyOwnershipIsIsolated(t *testing.T) {
	key := make([]byte, 256)
	key[0] = 1
	s := newSessionWithAuthKey(t)
	s.SetAuthKey(key)
	key[0] = 2
	if got := s.AuthKey()[0]; got != 1 {
		t.Fatalf("session auth key changed through caller slice: got %d", got)
	}

	mgr := NewTempKeyManager(2, false, key, true, nil)
	key[1] = 3
	perm := mgr.PermKey()
	if perm[1] != 0 {
		t.Fatalf("manager permanent key changed through caller slice: got %d", perm[1])
	}
	perm[0] = 9
	if got := mgr.PermKey()[0]; got == 9 {
		t.Fatal("PermKey returned mutable internal key material")
	}
}

func TestDeriveMsgAESKeyIVKnownVectorsDoesNotMutateAuthKey(t *testing.T) {
	var msgKey [16]byte
	for i := range msgKey {
		msgKey[i] = byte(0xf0 + i)
	}
	tests := []struct {
		name    string
		x       int
		wantKey string
		wantIV  string
	}{
		{
			name: "client_to_server",
			x:    0,
			wantKey: "e5f224b6de5d8a70" +
				"8b1c97869df4df3b4a3a26bac26118f90ba06686d599442b",
			wantIV: "8e3ead3b7ddc813131e5cbaa74b131070d0575855a64aa3954d4846ddefea770",
		},
		{
			name: "server_to_client",
			x:    8,
			wantKey: "0e5c6bc61941f368" +
				"ce854ae8380bd009bd7d7dbb5af346a1b9201b70ff5e9dde",
			wantIV: "a3186439cf6b67ea724bad3b6b732354283da80f70df7a9f5dfd75c422c50bd4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authKey := make([]byte, 256)
			for i := range authKey {
				authKey[i] = byte(i)
			}
			before := bytes.Clone(authKey)
			key, iv := deriveMsgAESKeyIV(authKey, msgKey, tt.x)
			if got := hex.EncodeToString(key[:]); got != tt.wantKey {
				t.Fatalf("key = %s, want %s", got, tt.wantKey)
			}
			if got := hex.EncodeToString(iv[:]); got != tt.wantIV {
				t.Fatalf("IV = %s, want %s", got, tt.wantIV)
			}
			if !bytes.Equal(authKey, before) {
				t.Fatal("deriveMsgAESKeyIV mutated the auth key")
			}
		})
	}
}
