package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
	"github.com/mtgo-labs/mtgo/internal/storage"
)

func TestConnStateTransitions(t *testing.T) {
	cs := newConnStateManager()

	if cs.State() != ConnStateDisconnected {
		t.Fatalf("initial state = %v, want disconnected", cs.State())
	}

	if err := cs.SetConnecting(2); err != nil {
		t.Fatalf("SetConnecting from disconnected: %v", err)
	}
	if cs.State() != ConnStateConnecting {
		t.Fatalf("state after SetConnecting = %v, want connecting", cs.State())
	}
	if cs.Health().CurrentDC != 2 {
		t.Fatalf("CurrentDC = %d, want 2", cs.Health().CurrentDC)
	}

	if err := cs.SetConnecting(3); err == nil {
		t.Fatal("SetConnecting from connecting should fail")
	}

	cs.SetConnected()
	if cs.State() != ConnStateConnected {
		t.Fatalf("state after SetConnected = %v, want connected", cs.State())
	}
	h := cs.Health()
	if h.ConnectedSince.IsZero() {
		t.Fatal("ConnectedSince should be set")
	}

	if err := cs.SetConnecting(2); err == nil {
		t.Fatal("SetConnecting from connected should fail")
	}

	cs.SetReconnecting(errors.New("test"))
	if cs.State() != ConnStateReconnecting {
		t.Fatalf("state after SetReconnecting = %v, want reconnecting", cs.State())
	}
	if cs.Health().ReconnectCount != 1 {
		t.Fatalf("ReconnectCount = %d, want 1", cs.Health().ReconnectCount)
	}
	if cs.Health().LastError == nil {
		t.Fatal("LastError should be set")
	}

	if err := cs.SetConnecting(2); err != nil {
		t.Fatalf("SetConnecting from reconnecting: %v", err)
	}

	cs.SetClosed()
	if cs.State() != ConnStateClosed {
		t.Fatalf("state after SetClosed = %v, want closed", cs.State())
	}

	if err := cs.SetConnecting(2); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("SetConnecting from closed: got %v, want ErrClientClosed", err)
	}
	cs.SetReconnecting(nil)
	if cs.State() != ConnStateClosed {
		t.Fatal("SetReconnecting on closed should be no-op")
	}
}

func TestConnStateRecordTimestamps(t *testing.T) {
	cs := newConnStateManager()

	before := time.Now()
	cs.RecordPing()
	cs.RecordPong()
	cs.RecordRead()
	cs.RecordWrite()
	after := time.Now()

	h := cs.Health()
	if h.LastPingTime.Before(before) || h.LastPingTime.After(after) {
		t.Fatalf("LastPingTime = %v, want between %v and %v", h.LastPingTime, before, after)
	}
	if h.LastPongTime.Before(before) || h.LastPongTime.After(after) {
		t.Fatalf("LastPongTime = %v, want between %v and %v", h.LastPongTime, before, after)
	}
	if h.LastReadTime.Before(before) || h.LastReadTime.After(after) {
		t.Fatalf("LastReadTime = %v, want between %v and %v", h.LastReadTime, before, after)
	}
	if h.LastWriteTime.Before(before) || h.LastWriteTime.After(after) {
		t.Fatalf("LastWriteTime = %v, want between %v and %v", h.LastWriteTime, before, after)
	}
}

func TestConnStateRequireConnected(t *testing.T) {
	cs := newConnStateManager()

	if err := cs.RequireConnected(); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("disconnected: got %v, want ErrNotConnected", err)
	}

	cs.SetConnecting(2)
	if err := cs.RequireConnected(); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("connecting: got %v, want ErrNotConnected", err)
	}

	cs.SetConnected()
	if err := cs.RequireConnected(); err != nil {
		t.Fatalf("connected: got %v, want nil", err)
	}

	cs.SetReconnecting(errors.New("x"))
	if err := cs.RequireConnected(); !errors.Is(err, ErrReconnectFailed) {
		t.Fatalf("reconnecting: got %v, want ErrReconnectFailed", err)
	}

	cs.SetClosed()
	if err := cs.RequireConnected(); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("closed: got %v, want ErrClientClosed", err)
	}
}

func TestConnStateCanReconnect(t *testing.T) {
	cs := newConnStateManager()
	if !cs.CanReconnect() {
		t.Fatal("disconnected should be reconnectable")
	}

	cs.SetReconnecting(errors.New("x"))
	if !cs.CanReconnect() {
		t.Fatal("reconnecting should be reconnectable")
	}

	cs.SetConnected()
	if cs.CanReconnect() {
		t.Fatal("connected should not be reconnectable")
	}

	cs.SetClosed()
	if cs.CanReconnect() {
		t.Fatal("closed should not be reconnectable")
	}
}

func TestConnStateResetReconnectCount(t *testing.T) {
	cs := newConnStateManager()
	cs.SetReconnecting(errors.New("a"))
	cs.SetReconnecting(errors.New("b"))
	if cs.Health().ReconnectCount != 2 {
		t.Fatalf("ReconnectCount = %d, want 2", cs.Health().ReconnectCount)
	}

	cs.ResetReconnectCount()
	if cs.Health().ReconnectCount != 0 {
		t.Fatalf("after reset: ReconnectCount = %d, want 0", cs.Health().ReconnectCount)
	}
}

func TestConnStateBackwardCompat(t *testing.T) {
	cs := newConnectionState()

	if cs.isConnected() {
		t.Fatal("new state should not be connected")
	}
	cs.setConnected(true)
	if !cs.isConnected() {
		t.Fatal("should be connected after setConnected(true)")
	}
	cs.setConnected(false)
	if cs.isConnected() {
		t.Fatal("should not be connected after setConnected(false)")
	}
}

func TestBackoffConfig(t *testing.T) {
	cfg := defaultBackoffConfig

	if d := cfg.delay(0); d != cfg.BaseDelay {
		t.Fatalf("delay(0) = %v, want %v", d, cfg.BaseDelay)
	}
	if d := cfg.delay(1); d != 2*time.Second {
		t.Fatalf("delay(1) = %v, want 2s", d)
	}
	if d := cfg.delay(10); d != cfg.MaxDelay {
		t.Fatalf("delay(10) = %v, want MaxDelay %v", d, cfg.MaxDelay)
	}
}

func TestReconnectManagerPreventDuplicates(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	client.state.SetConnecting(2)
	client.state.SetConnected()

	rm := newReconnectManager(client, backoffConfig{
		BaseDelay:   100 * time.Hour,
		MaxDelay:    200 * time.Hour,
		MaxAttempts: 0,
	})

	rm.Start(context.Background())
	if !rm.IsRunning() {
		t.Fatal("should be running after Start")
	}

	rm.Start(context.Background())
	if rm.Attempts() != 0 {
		t.Fatal("duplicate Start should be no-op")
	}

	rm.Stop()
	if rm.IsRunning() {
		t.Fatal("should not be running after Stop")
	}
}

func TestReconnectManagerMaxAttempts(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	client.state.SetConnecting(2)
	client.state.SetConnected()

	cfg := backoffConfig{
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond,
		MaxAttempts: 2,
	}
	rm := newReconnectManager(client, cfg)
	client.state.SetReconnecting(errors.New("test"))

	rm.Start(context.Background())

	time.Sleep(500 * time.Millisecond)

	if rm.IsRunning() {
		t.Fatal("should have stopped after max attempts")
	}
	if rm.Attempts() != 3 {
		t.Fatalf("Attempts = %d, want 3 (1 + 2 retries)", rm.Attempts())
	}
	h := client.Health()
	if h.State != ConnStateDisconnected {
		t.Fatalf("state = %v, want disconnected", h.State)
	}
}

func TestHealthCheckerStartStop(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	hc := newHealthChecker(client, healthCheckConfig{
		PingInterval: 10 * time.Millisecond,
		PongTimeout:  5 * time.Millisecond,
	})

	hc.Start(context.Background())
	if !hc.IsRunning() {
		t.Fatal("should be running")
	}

	hc.Start(context.Background())

	hc.Stop()
	if hc.IsRunning() {
		t.Fatal("should not be running after Stop")
	}

	hc.Stop()
}

func TestHealthCheckerStopsOnDisconnected(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	hc := newHealthChecker(client, healthCheckConfig{
		PingInterval: 10 * time.Millisecond,
		PongTimeout:  5 * time.Millisecond,
	})

	hc.Start(context.Background())
	time.Sleep(50 * time.Millisecond)

	hc.Stop()
}

func TestMigrationErrorParsing(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	client.state.SetConnecting(2)
	client.state.SetConnected()
	client.storage = NewMemoryStorage()

	rpcErr := &tgerr.Error{Code: 303, Type: "PHONE_MIGRATE", Argument: 4}
	_, err := client.handleMigrationError(rpcErr, nil)
	if err == nil {
		t.Fatal("should return error for nil query with non-idempotent migration path")
	}

	rpcErrBad := &tgerr.Error{Code: 303, Type: "PHONE_MIGRATE", Argument: 0}
	_, err = client.handleMigrationError(rpcErrBad, nil)
	if err == nil {
		t.Fatal("should return error for DC 0")
	}
	var migErr *MigrationError
	if !errors.As(err, &migErr) {
		t.Fatalf("got %T, want *MigrationError", err)
	}
	if migErr.TargetDC != 0 {
		t.Fatalf("TargetDC = %d, want 0", migErr.TargetDC)
	}
}

func TestMigrationErrorNon303(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	client.state.SetConnecting(2)
	client.state.SetConnected()
	client.storage = NewMemoryStorage()

	rpcErr := &tgerr.Error{Code: 400, Type: "BAD_REQUEST", Argument: 5}
	_, err := client.handleMigrationError(rpcErr, nil)
	if err != rpcErr {
		t.Fatalf("non-303 with valid argument: got %v, want original error", err)
	}
}

func TestMigrationUnknownType(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	client.state.SetConnecting(2)
	client.state.SetConnected()
	client.storage = NewMemoryStorage()

	rpcErr := &tgerr.Error{Code: 303, Type: "GARBAGE_MIGRATE", Argument: 5}
	_, err := client.handleMigrationError(rpcErr, nil)
	var migErr *MigrationError
	if !errors.As(err, &migErr) {
		t.Fatalf("got %T, want *MigrationError", err)
	}
	if migErr.TargetDC != 5 {
		t.Fatalf("TargetDC = %d, want 5", migErr.TargetDC)
	}
}

func TestMigrationUnsafeNonIdempotent(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	client.state.SetConnecting(2)
	client.state.SetConnected()
	client.storage = NewMemoryStorage()

	query := &tg.MessagesSendMessageRequest{}
	rpcErr := &tgerr.Error{Code: 303, Type: "PHONE_MIGRATE", Argument: 4}
	_, err := client.handleMigrationError(rpcErr, query)
	var unsafeErr *UnsafeMigrationError
	if !errors.As(err, &unsafeErr) {
		t.Fatalf("got %T, want *UnsafeMigrationError", err)
	}
	if unsafeErr.TargetDC != 4 {
		t.Fatalf("TargetDC = %d, want 4", unsafeErr.TargetDC)
	}
}

func TestIsIdempotent(t *testing.T) {
	if isIdempotent(nil) {
		t.Fatal("nil should not be idempotent")
	}
	if isIdempotent(&tg.MessagesSendMessageRequest{}) {
		t.Fatal("MessagesSendMessage should not be idempotent")
	}
	if !isIdempotent(&tg.AuthExportAuthorizationRequest{}) {
		t.Fatal("AuthExportAuthorization should be idempotent")
	}
}

func TestBotAuthUserMigrationReturnsToCaller(t *testing.T) {
	rpcErr := &tgerr.Error{Code: 303, Type: "USER_MIGRATE", Argument: 4}
	if !shouldReturnMigrationToCaller(&tg.AuthImportBotAuthorizationRequest{}, rpcErr) {
		t.Fatal("bot auth USER_MIGRATE should return to connectTransport")
	}
	if shouldReturnMigrationToCaller(&tg.MessagesSendMessageRequest{}, rpcErr) {
		t.Fatal("non-bot auth USER_MIGRATE should use generic migration handling")
	}
	if shouldReturnMigrationToCaller(&tg.AuthImportBotAuthorizationRequest{}, &tgerr.Error{Code: 303, Type: "PHONE_MIGRATE", Argument: 4}) {
		t.Fatal("bot auth should only handle USER_MIGRATE locally")
	}
}

func TestSentinelErrors(t *testing.T) {
	errs := []error{ErrNotConnected, ErrAlreadyConnected, ErrPeerNotFound, ErrClientClosed, ErrReconnectFailed, ErrHealthTimeout, ErrMigrationFailed, ErrMigrationUnsafe, ErrMigrationUnknown}
	for _, e := range errs {
		if e == nil {
			t.Fatal("sentinel error should not be nil")
		}
		if e.Error() == "" {
			t.Fatal("sentinel error should have a message")
		}
	}
}

func TestReconnectErrorUnwrap(t *testing.T) {
	inner := errors.New("inner")
	re := &ReconnectError{Attempts: 3, Err: inner}
	if !errors.Is(re, inner) {
		t.Fatal("ReconnectError should unwrap to inner")
	}
	if re.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
}

func TestMigrationErrorUnwrap(t *testing.T) {
	inner := errors.New("inner")
	me := &MigrationError{TargetDC: 4, Err: inner}
	if !errors.Is(me, inner) {
		t.Fatal("MigrationError should unwrap to inner")
	}
	if me.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
}

func TestUnsafeMigrationError(t *testing.T) {
	e := &UnsafeMigrationError{TargetDC: 5, Method: "messages.sendMessage"}
	if e.Error() == "" {
		t.Fatal("Error() should not be empty")
	}
}

func TestHealthStatusFromClient(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	h := client.Health()
	if h.State != ConnStateDisconnected {
		t.Fatalf("initial Health().State = %v, want disconnected", h.State)
	}
}

func TestCloseStopsEverything(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	client.state.SetConnecting(2)
	client.state.SetConnected()
	client.storage = NewMemoryStorage()

	client.Close()

	if !client.state.IsClosed() {
		t.Fatal("state should be closed after Close()")
	}
	if client.healthCheck != nil && client.healthCheck.IsRunning() {
		t.Fatal("health checker should be stopped")
	}
	if client.reconnectMgr != nil && client.reconnectMgr.IsRunning() {
		t.Fatal("reconnect manager should be stopped")
	}
}

func TestTriggerReconnectDisabled(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{
		InMemory:         true,
		ReconnectEnabled: false,
	})
	client.state.SetConnecting(2)
	client.state.SetConnected()
	client.storage = NewMemoryStorage()

	client.triggerReconnect(errors.New("test"))

	h := client.Health()
	if h.State != ConnStateDisconnected {
		t.Fatalf("state = %v, want disconnected when reconnect disabled", h.State)
	}
}

func TestTriggerReconnectClosed(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	client.state.SetClosed()

	client.triggerReconnect(errors.New("test"))

	if !client.state.IsClosed() {
		t.Fatal("should remain closed")
	}
}

func TestConfigReconnectDefaults(t *testing.T) {
	cfg := DefaultConfig
	if !cfg.ReconnectEnabled {
		t.Error("ReconnectEnabled should be true by default")
	}
	if cfg.ReconnectBaseDelay != 1*time.Second {
		t.Errorf("ReconnectBaseDelay = %v, want 1s", cfg.ReconnectBaseDelay)
	}
	if cfg.ReconnectMaxDelay != 60*time.Second {
		t.Errorf("ReconnectMaxDelay = %v, want 60s", cfg.ReconnectMaxDelay)
	}
	if !cfg.HealthEnabled {
		t.Error("HealthEnabled should be true by default")
	}
	if cfg.HealthPingInterval != 60*time.Second {
		t.Errorf("HealthPingInterval = %v, want 60s", cfg.HealthPingInterval)
	}
	if cfg.HealthPongTimeout != 30*time.Second {
		t.Errorf("HealthPongTimeout = %v, want 30s", cfg.HealthPongTimeout)
	}
}

func TestConfigReconnectOverrides(t *testing.T) {
	c, _ := NewClient(111, "hash", &Config{
		InMemory:             true,
		ReconnectEnabled:     true,
		ReconnectBaseDelay:   2 * time.Second,
		ReconnectMaxDelay:    120 * time.Second,
		ReconnectMaxAttempts: 5,
		HealthEnabled:        true,
		HealthPingInterval:   30 * time.Second,
		HealthPongTimeout:    10 * time.Second,
	})
	cfg := c.Config()
	if cfg.ReconnectBaseDelay != 2*time.Second {
		t.Errorf("ReconnectBaseDelay = %v", cfg.ReconnectBaseDelay)
	}
	if cfg.ReconnectMaxDelay != 120*time.Second {
		t.Errorf("ReconnectMaxDelay = %v", cfg.ReconnectMaxDelay)
	}
	if cfg.ReconnectMaxAttempts != 5 {
		t.Errorf("ReconnectMaxAttempts = %d", cfg.ReconnectMaxAttempts)
	}
	if cfg.HealthPingInterval != 30*time.Second {
		t.Errorf("HealthPingInterval = %v", cfg.HealthPingInterval)
	}
	if cfg.HealthPongTimeout != 10*time.Second {
		t.Errorf("HealthPongTimeout = %v", cfg.HealthPongTimeout)
	}
}

func TestMemoryStorageImplementsDCAuthStore(t *testing.T) {
	var _ storage.DCAuthStore = NewMemoryStorage()
}

func TestMemoryStorageDCAuthRoundTrip(t *testing.T) {
	ms := NewMemoryStorage()

	entry := storage.DCAuthEntry{
		DCID:    4,
		AuthKey: []byte("test-auth-key-32-bytes-xxxxxxxx"),
	}
	if err := ms.SaveDCAuth(entry); err != nil {
		t.Fatalf("SaveDCAuth: %v", err)
	}

	loaded, err := ms.LoadDCAuth(4)
	if err != nil {
		t.Fatalf("LoadDCAuth: %v", err)
	}
	if loaded.DCID != 4 {
		t.Fatalf("DCID = %d, want 4", loaded.DCID)
	}
	if string(loaded.AuthKey) != string(entry.AuthKey) {
		t.Fatalf("AuthKey mismatch")
	}

	_, err = ms.LoadDCAuth(99)
	if err == nil {
		t.Fatal("loading non-existent DC should fail")
	}
}

func TestInvokeMigrationOn303Error(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	client.state.SetConnecting(2)
	client.state.SetConnected()
	client.storage = NewMemoryStorage()

	migrating := &tgerr.Error{Code: 303, Type: "PHONE_MIGRATE", Argument: 4}
	_, err := client.handleMigrationError(migrating, &tg.MessagesSendMessageRequest{})
	var unsafeErr *UnsafeMigrationError
	if !errors.As(err, &unsafeErr) {
		t.Fatalf("got %T, want *UnsafeMigrationError", err)
	}

	result, err := client.handleMigrationError(migrating, &tg.AuthExportAuthorizationRequest{})
	_ = result
	_ = err
}

func TestConnStateString(t *testing.T) {
	tests := []struct {
		state ConnState
		want  string
	}{
		{ConnStateDisconnected, "disconnected"},
		{ConnStateConnecting, "connecting"},
		{ConnStateConnected, "connected"},
		{ConnStateReconnecting, "reconnecting"},
		{ConnStateClosed, "closed"},
		{ConnState(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("ConnState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
