package telegram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type authKeyClearFailStorage struct {
	*MemoryStorage
}

type authKeySyncFailStorage struct {
	*MemoryStorage
}

type authKeyCloseInspectStorage struct {
	*MemoryStorage
	closed chan int
}

func (s *authKeyCloseInspectStorage) Close() error {
	key, _ := s.AuthKey()
	s.closed <- len(key)
	return s.MemoryStorage.Close()
}

func (s *authKeySyncFailStorage) Sync() error {
	return errors.New("sync failed")
}

func (s *authKeyClearFailStorage) SetAuthKey(key []byte) error {
	if len(key) == 0 {
		return errors.New("clear failed")
	}
	return s.MemoryStorage.SetAuthKey(key)
}

func populatedAuthStorage(t *testing.T) *MemoryStorage {
	t.Helper()
	st := NewMemoryStorage()
	for name, err := range map[string]error{
		"auth key":   st.SetAuthKey(make([]byte, 256)),
		"date":       st.SetDate(123),
		"dc":         st.SetDCID(4),
		"api id":     st.SetAPIID(99),
		"api hash":   st.SetAPIHash("hash"),
		"test mode":  st.SetTestMode(true),
		"user id":    st.SetUserID(42),
		"bot flag":   st.SetIsBot(true),
		"first name": st.SetFirstName("A"),
		"last name":  st.SetLastName("B"),
		"username":   st.SetUsername("user"),
		"state":      st.SetState([]byte("state")),
	} {
		if err != nil {
			t.Fatalf("populate %s: %v", name, err)
		}
	}
	return st
}

func newPFSAuthTestSession(t *testing.T, st *MemoryStorage) *session.Session {
	t.Helper()
	sess, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() = %v", err)
	}
	key, err := st.AuthKey()
	if err != nil {
		t.Fatalf("AuthKey() = %v", err)
	}
	sess.SetPFS(session.NewTempKeyManager(2, false, key, true, nil))
	return sess
}

func TestInvalidateMainAuthClearsCredentialsAndLatches(t *testing.T) {
	client, _ := NewClient(99, "hash", &Config{AutoConnect: true})
	st := populatedAuthStorage(t)
	client.storage = st
	cause := tgerr.New(406, "AUTH_KEY_DUPLICATED")

	err := client.invalidateMainAuth(cause)
	if !errors.Is(err, ErrAuthKeyInvalidated) || !tgerr.Is(err, tgerr.ErrAuthKeyDuplicated) {
		t.Fatalf("invalidateMainAuth() = %v", err)
	}
	if _, ok := errors.AsType[*AuthKeyInvalidatedError](err); !ok {
		t.Fatalf("invalidateMainAuth() type = %T, want *AuthKeyInvalidatedError", err)
	}
	if key, _ := st.AuthKey(); len(key) != 0 {
		t.Fatalf("auth key length = %d, want 0", len(key))
	}
	if date, _ := st.Date(); date != 0 {
		t.Fatalf("date = %d, want 0", date)
	}
	if userID, _ := st.UserID(); userID != 0 {
		t.Fatalf("user id = %d, want 0", userID)
	}
	if isBot, _ := st.IsBot(); isBot {
		t.Fatal("bot flag was not cleared")
	}
	if first, _ := st.FirstName(); first != "" {
		t.Fatalf("first name = %q, want empty", first)
	}
	if last, _ := st.LastName(); last != "" {
		t.Fatalf("last name = %q, want empty", last)
	}
	if username, _ := st.Username(); username != "" {
		t.Fatalf("username = %q, want empty", username)
	}
	if state, _ := st.State(); len(state) != 0 {
		t.Fatalf("state length = %d, want 0", len(state))
	}
	if dc, _ := st.DCID(); dc != 4 {
		t.Fatalf("dc = %d, want 4", dc)
	}
	if apiID, _ := st.APIID(); apiID != 99 {
		t.Fatalf("api id = %d, want 99", apiID)
	}
	if testMode, _ := st.TestMode(); !testMode {
		t.Fatal("test mode was cleared")
	}
	if err := client.ensureConnected(); !errors.Is(err, ErrAuthKeyInvalidated) {
		t.Fatalf("ensureConnected() = %v, want ErrAuthKeyInvalidated", err)
	}
	if err := client.prepareExplicitAuthRecovery(); err != nil {
		t.Fatalf("prepareExplicitAuthRecovery() = %v", err)
	}
}

func TestAsyncAuthCleanupKeepsSuccessfulResultStable(t *testing.T) {
	client, _ := NewClient(99, "hash", nil)
	st := populatedAuthStorage(t)
	client.storage = st
	client.autoConnectMu.Lock()
	err := client.invalidateMainAuth(tgerr.New(406, "AUTH_KEY_DUPLICATED"))
	initial, ok := errors.AsType[*AuthKeyInvalidatedError](err)
	if !ok {
		client.autoConnectMu.Unlock()
		t.Fatalf("invalidateMainAuth() type = %T", err)
	}
	if initial.Cleanup != nil {
		client.autoConnectMu.Unlock()
		t.Fatalf("asynchronous cleanup reported a failure before completion: %v", initial.Cleanup)
	}
	client.authLossMu.Lock()
	loss := client.authLoss
	client.authLossMu.Unlock()
	client.autoConnectMu.Unlock()
	select {
	case <-loss.done:
	case <-time.After(time.Second):
		t.Fatal("asynchronous auth cleanup did not finish")
	}
	final, ok := errors.AsType[*AuthKeyInvalidatedError](client.authLossError())
	if !ok || final != initial || final.Cleanup != nil {
		t.Fatalf("final auth loss = %#v, want stable successful result %#v", final, initial)
	}
	if key, _ := st.AuthKey(); len(key) != 0 {
		t.Fatalf("auth key length = %d, want 0", len(key))
	}
}

func TestReconnectRefreshInvalidatesTerminalAuthError(t *testing.T) {
	client, _ := NewClient(99, "hash", nil)
	st := populatedAuthStorage(t)
	client.storage = st
	sess, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	client.session = sess
	client.state.SetConnecting(2)
	client.state.SetConnected()

	if client.handleReconnectRefreshResult(sess, client.authGeneration.Load(), tgerr.New(406, "AUTH_KEY_DUPLICATED")) {
		t.Fatal("terminal refresh error allowed reconnect hooks to run")
	}
	if !errors.Is(client.authLossError(), ErrAuthKeyInvalidated) {
		t.Fatalf("auth loss = %v, want ErrAuthKeyInvalidated", client.authLossError())
	}
	if key, _ := st.AuthKey(); len(key) != 0 {
		t.Fatalf("auth key length = %d, want 0", len(key))
	}
}

func TestStaleReconnectRefreshCannotInvalidateFreshGeneration(t *testing.T) {
	client, _ := NewClient(99, "hash", nil)
	st := populatedAuthStorage(t)
	client.storage = st
	oldSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	client.session = oldSession
	client.state.SetConnecting(2)
	client.state.SetConnected()
	oldGeneration := client.authGeneration.Load()

	if err := client.advanceAuthGeneration(); err != nil {
		t.Fatal(err)
	}
	freshKey := bytes.Repeat([]byte{0x3c}, 256)
	if err := st.SetAuthKey(freshKey); err != nil {
		t.Fatal(err)
	}
	newSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	client.mu.Lock()
	client.session = newSession
	client.mu.Unlock()

	if client.handleReconnectRefreshResult(oldSession, oldGeneration, tgerr.New(406, "AUTH_KEY_DUPLICATED")) {
		t.Fatal("stale terminal refresh allowed reconnect hooks to run")
	}
	if err := client.authLossError(); err != nil {
		t.Fatalf("stale refresh latched auth loss: %v", err)
	}
	key, err := st.AuthKey()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(key, freshKey) {
		t.Fatal("stale refresh cleared the fresh replacement key")
	}
}

func TestStaleTerminalSessionExitCannotReconnectFreshGeneration(t *testing.T) {
	client, _ := NewClient(99, "hash", &Config{ReconnectEnabled: true})
	st := populatedAuthStorage(t)
	client.storage = st
	oldSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	client.session = oldSession
	client.state.SetConnecting(2)
	client.state.SetConnected()
	oldGeneration := client.authGeneration.Load()

	// Model the exact post-detach pause: classification is blocked while an
	// explicit recovery installs a fresh key/session in a new generation.
	client.mu.Lock()
	client.session = nil
	client.mu.Unlock()
	client.authDecisionMu.Lock()
	exitDone := make(chan struct{})
	go func() {
		client.handleDetachedMainSessionExit(
			oldSession,
			oldGeneration,
			fmt.Errorf("session exited [readLoop]: %w", tgerr.New(406, "AUTH_KEY_DUPLICATED")),
		)
		close(exitDone)
	}()

	if err := client.advanceAuthGeneration(); err != nil {
		client.authDecisionMu.Unlock()
		t.Fatal(err)
	}
	freshKey := bytes.Repeat([]byte{0x5a}, 256)
	if err := st.SetAuthKey(freshKey); err != nil {
		client.authDecisionMu.Unlock()
		t.Fatal(err)
	}
	newSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		client.authDecisionMu.Unlock()
		t.Fatal(err)
	}
	client.mu.Lock()
	client.session = newSession
	client.mu.Unlock()
	client.authDecisionMu.Unlock()

	select {
	case <-exitDone:
	case <-time.After(time.Second):
		t.Fatal("stale session-exit classification did not finish")
	}
	if client.Session() != newSession {
		t.Fatal("stale terminal exit detached the fresh replacement session")
	}
	if !client.state.IsConnected() {
		t.Fatal("stale terminal exit changed the fresh connection state")
	}
	if client.reconnectMgr.IsRunning() {
		t.Fatal("stale terminal exit started reconnect for the fresh generation")
	}
	if err := client.authLossError(); err != nil {
		t.Fatalf("stale terminal exit latched auth loss: %v", err)
	}
	key, err := st.AuthKey()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(key, freshKey) {
		t.Fatal("stale terminal exit cleared the fresh replacement key")
	}
}

func TestDetachedSessionExitCannotReconnectPublishedReplacement(t *testing.T) {
	tests := []struct {
		name    string
		old     func(*testing.T, *MemoryStorage) *session.Session
		exitErr error
	}{
		{
			name: "temporary PFS rejection",
			old:  newPFSAuthTestSession,
			exitErr: fmt.Errorf("session exited [readLoop]: %w", &transport.TransportError{
				Code: transport.ErrCodeAuthKeyNotFound,
			}),
		},
		{
			name: "transient transport exit",
			old: func(t *testing.T, st *MemoryStorage) *session.Session {
				t.Helper()
				sess, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
				if err != nil {
					t.Fatal(err)
				}
				return sess
			},
			exitErr: errors.New("session exited [readLoop]: connection reset"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := NewClient(99, "hash", &Config{ReconnectEnabled: true})
			st := populatedAuthStorage(t)
			client.storage = st
			oldSession := tt.old(t, st)
			client.session = oldSession
			client.state.SetConnecting(2)
			client.state.SetConnected()
			generation := client.authGeneration.Load()

			client.mu.Lock()
			client.session = nil
			client.mu.Unlock()
			client.authDecisionMu.Lock()
			exitDone := make(chan struct{})
			go func() {
				client.handleDetachedMainSessionExit(oldSession, generation, tt.exitErr)
				close(exitDone)
			}()

			replacement, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
			if err != nil {
				client.authDecisionMu.Unlock()
				t.Fatal(err)
			}
			client.mu.Lock()
			client.session = replacement
			client.mu.Unlock()
			client.authDecisionMu.Unlock()

			select {
			case <-exitDone:
			case <-time.After(time.Second):
				t.Fatal("detached session-exit classification did not finish")
			}
			if client.Session() != replacement {
				t.Fatal("detached session exit replaced the published session")
			}
			if !client.state.IsConnected() {
				t.Fatal("detached session exit changed replacement connection state")
			}
			if client.reconnectMgr.IsRunning() {
				t.Fatal("detached session exit started a stale reconnect")
			}
		})
	}
}

func TestInvalidateMainAuthClearFailureRemainsFailClosed(t *testing.T) {
	base := populatedAuthStorage(t)
	st := &authKeyClearFailStorage{MemoryStorage: base}
	client, _ := NewClient(99, "hash", &Config{AutoConnect: true})
	client.storage = st

	err := client.invalidateMainAuth(tgerr.New(401, "SESSION_REVOKED"))
	loss, ok := errors.AsType[*AuthKeyInvalidatedError](err)
	if !ok || loss.Cleanup == nil {
		t.Fatalf("invalidateMainAuth() = %v, want cleanup failure", err)
	}
	if key, _ := st.AuthKey(); len(key) == 0 {
		t.Fatal("failing storage unexpectedly cleared auth key")
	}
	if err := client.prepareExplicitAuthRecovery(); !errors.Is(err, ErrAuthKeyInvalidated) {
		t.Fatalf("prepareExplicitAuthRecovery() = %v, want latched auth loss", err)
	}
	if err := client.ensureConnected(); !errors.Is(err, ErrAuthKeyInvalidated) {
		t.Fatalf("ensureConnected() = %v, want latched auth loss", err)
	}
}

func TestInvalidateMainAuthSyncFailureRemainsFailClosed(t *testing.T) {
	st := &authKeySyncFailStorage{MemoryStorage: populatedAuthStorage(t)}
	client, _ := NewClient(99, "hash", &Config{AutoConnect: true})
	client.storage = st

	err := client.invalidateMainAuth(tgerr.New(401, "SESSION_REVOKED"))
	loss, ok := errors.AsType[*AuthKeyInvalidatedError](err)
	if !ok || loss.Cleanup == nil {
		t.Fatalf("invalidateMainAuth() = %v, want durable sync failure", err)
	}
	if err := client.prepareExplicitAuthRecovery(); !errors.Is(err, ErrAuthKeyInvalidated) {
		t.Fatalf("prepareExplicitAuthRecovery() = %v, want latched auth loss", err)
	}
}

func TestExplicitAuthRecoveryAfterDisconnect(t *testing.T) {
	client, _ := NewClient(99, "hash", &Config{AutoConnect: true})
	client.storage = populatedAuthStorage(t)
	if err := client.invalidateMainAuth(tgerr.New(401, "SESSION_REVOKED")); err == nil {
		t.Fatal("invalidateMainAuth() unexpectedly succeeded without an error")
	}
	if err := client.Disconnect(); err != nil {
		t.Fatalf("Disconnect() = %v", err)
	}
	if client.Storage() != nil {
		t.Fatal("Disconnect() retained storage")
	}
	if err := client.prepareExplicitAuthRecovery(); err != nil {
		t.Fatalf("prepareExplicitAuthRecovery() = %v", err)
	}
}

func TestAuthLossClassificationPreservesFreshUnregisteredKey(t *testing.T) {
	client, _ := NewClient(1, "hash", nil)
	st := NewMemoryStorage()
	_ = st.SetAuthKey(make([]byte, 256))
	client.storage = st
	unregistered := tgerr.New(401, "AUTH_KEY_UNREGISTERED")
	client.mainAuthKeyOrigin.Store(authKeyOriginFresh)
	if client.shouldInvalidateMainAuth(unregistered) {
		t.Fatal("fresh unauthenticated key classified as terminal auth loss")
	}
	client.mainAuthKeyOrigin.Store(authKeyOriginLoaded)
	if !client.shouldInvalidateMainAuth(unregistered) {
		t.Fatal("loaded auth-only key was not classified as terminal auth loss")
	}
	client.mainAuthKeyOrigin.Store(authKeyOriginFresh)
	_ = st.SetUserID(42)
	if !client.shouldInvalidateMainAuth(unregistered) {
		t.Fatal("authorized unregistered key was not classified as terminal auth loss")
	}
	if client.shouldInvalidateMainAuth(fmt.Errorf("%w: %v", errTemporaryAuthKeyRejected, unregistered)) {
		t.Fatal("temporary PFS key rejection classified as permanent auth loss")
	}
}

func TestAuthLossClassification(t *testing.T) {
	client, _ := NewClient(1, "hash", nil)
	client.storage = populatedAuthStorage(t)
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "duplicated", err: tgerr.New(406, "AUTH_KEY_DUPLICATED"), want: true},
		{name: "revoked", err: tgerr.New(401, "SESSION_REVOKED"), want: true},
		{name: "expired", err: tgerr.New(401, "SESSION_EXPIRED"), want: true},
		{name: "invalid", err: tgerr.New(400, "AUTH_KEY_INVALID"), want: true},
		{name: "transport unknown key", err: &transport.TransportError{Code: transport.ErrCodeAuthKeyNotFound}, want: true},
		{name: "temporary unknown key", err: fmt.Errorf("%w: -404", errTemporaryAuthKeyRejected), want: false},
		{name: "PFS key rotation", err: session.ErrBindRequiresKeyRotation, want: true},
		{name: "password needed", err: tgerr.New(401, "SESSION_PASSWORD_NEEDED"), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := client.shouldInvalidateMainAuth(test.err); got != test.want {
				t.Fatalf("shouldInvalidateMainAuth(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

func TestSessionStringReimportsAfterNormalStorageReplacement(t *testing.T) {
	source := populatedAuthStorage(t)
	encoded, err := source.ExportSessionString()
	if err != nil {
		t.Fatalf("ExportSessionString() = %v", err)
	}
	client, _ := NewClient(99, "hash", &Config{SessionString: encoded})

	for i := range 2 {
		st := NewMemoryStorage()
		if err := client.importSessionString(st); err != nil {
			t.Fatalf("importSessionString() attempt %d = %v", i+1, err)
		}
		key, _ := st.AuthKey()
		want, _ := source.AuthKey()
		if !bytes.Equal(key, want) {
			t.Fatalf("attempt %d imported key length = %d, want %d", i+1, len(key), len(want))
		}
	}
}

func TestInvalidatedSessionStringIsNotReimported(t *testing.T) {
	source := populatedAuthStorage(t)
	encoded, err := source.ExportSessionString()
	if err != nil {
		t.Fatalf("ExportSessionString() = %v", err)
	}
	client, _ := NewClient(99, "hash", &Config{SessionString: encoded})
	active := NewMemoryStorage()
	if err := client.importSessionString(active); err != nil {
		t.Fatalf("importSessionString() = %v", err)
	}
	client.storage = active
	_ = client.invalidateMainAuth(tgerr.New(406, "AUTH_KEY_DUPLICATED"))

	replacement := NewMemoryStorage()
	if err := client.importSessionString(replacement); err != nil {
		t.Fatalf("importSessionString() after invalidation = %v", err)
	}
	if key, _ := replacement.AuthKey(); len(key) != 0 {
		t.Fatalf("invalidated session string reimported %d-byte key", len(key))
	}
}

func TestRetrySessionErrInvalidatesDuplicatedKey(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{RetryRPCOnReconnect: true})
	st := populatedAuthStorage(t)
	client.storage = st
	client.session = newPFSAuthTestSession(t, st)
	client.state.SetConnecting(2)
	client.state.SetConnected()
	calls := 0
	err := client.retrySessionErr(context.Background(), func(*session.Session) error {
		calls++
		return tgerr.New(406, "AUTH_KEY_DUPLICATED")
	})
	if !errors.Is(err, ErrAuthKeyInvalidated) {
		t.Fatalf("retrySessionErr() = %v, want ErrAuthKeyInvalidated", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if key, _ := st.AuthKey(); len(key) != 0 {
		t.Fatalf("auth key length = %d, want 0", len(key))
	}
}

func TestDropRPCInvalidatesDuplicatedMainKey(t *testing.T) {
	client, server := newTestClient(1, "hash", Config{
		NoUpdates:           true,
		ReconnectEnabled:    false,
		RetryRPCOnReconnect: false,
	})
	defer server.Close()
	if err := client.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	server.rpcError.Store(&tg.RPCError{ErrorCode: 406, ErrorMessage: "AUTH_KEY_DUPLICATED"})

	err := client.DropRPC(context.Background(), 12345)
	if !errors.Is(err, ErrAuthKeyInvalidated) {
		t.Fatalf("DropRPC() = %v, want ErrAuthKeyInvalidated", err)
	}
	st := client.Storage()
	if st == nil {
		t.Fatal("storage was released before rejected-key verification")
	}
	if key, keyErr := st.AuthKey(); keyErr != nil || len(key) != 0 {
		t.Fatalf("stored auth key after DropRPC = %d bytes, err=%v", len(key), keyErr)
	}
}

func TestPFSFreshUnregisteredRemainsAvailableForLogin(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{ReconnectEnabled: false})
	st := NewMemoryStorage()
	if err := st.SetAuthKey(make([]byte, 256)); err != nil {
		t.Fatal(err)
	}
	client.storage = st
	client.mainAuthKeyOrigin.Store(authKeyOriginFresh)
	sess := newPFSAuthTestSession(t, st)
	client.session = sess
	client.state.SetConnecting(2)
	client.state.SetConnected()

	err := client.retrySessionErr(context.Background(), func(source *session.Session) error {
		if source != sess {
			t.Fatalf("source session = %p, want %p", source, sess)
		}
		return tgerr.New(401, "AUTH_KEY_UNREGISTERED")
	})
	if !tgerr.Is(err, tgerr.ErrAuthKeyUnregistered) {
		t.Fatalf("retrySessionErr() = %v, want AUTH_KEY_UNREGISTERED", err)
	}
	if client.authLossError() != nil {
		t.Fatalf("fresh key latched auth loss: %v", client.authLossError())
	}
	if client.Session() != sess {
		t.Fatal("fresh unauthenticated PFS session was detached")
	}
	if key, _ := st.AuthKey(); len(key) != 256 {
		t.Fatalf("auth key length = %d, want 256", len(key))
	}
}

func TestPFSRejectionUsesProducingSession(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{ReconnectEnabled: false})
	st := populatedAuthStorage(t)
	client.storage = st
	old := newPFSAuthTestSession(t, st)
	replacement, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	client.session = old
	client.state.SetConnecting(2)
	client.state.SetConnected()

	err = client.retrySessionErr(context.Background(), func(source *session.Session) error {
		if source != old {
			t.Fatalf("source session = %p, want old %p", source, old)
		}
		client.mu.Lock()
		client.session = replacement
		client.mu.Unlock()
		return &transport.TransportError{Code: transport.ErrCodeAuthKeyNotFound}
	})
	if !errors.Is(err, errTemporaryAuthKeyRejected) {
		t.Fatalf("retrySessionErr() = %v, want temporary PFS rejection", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if client.Session() == replacement {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if client.Session() != replacement {
		t.Fatal("stale PFS error detached the replacement session")
	}
	if client.authLossError() != nil {
		t.Fatalf("stale PFS error invalidated permanent key: %v", client.authLossError())
	}
	if key, _ := st.AuthKey(); len(key) != 256 {
		t.Fatalf("auth key length = %d, want 256", len(key))
	}
}

func TestDisconnectWaitsForLateDuplicatedKeyCleanup(t *testing.T) {
	st := &authKeyCloseInspectStorage{
		MemoryStorage: populatedAuthStorage(t),
		closed:        make(chan int, 1),
	}
	client, _ := NewClient(1, "hash", &Config{RetryRPCOnReconnect: false})
	client.storage = st
	sess, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	client.session = sess
	client.state.SetConnecting(2)
	client.state.SetConnected()

	invokeEntered := make(chan struct{})
	invokeRelease := make(chan struct{})
	retryDone := make(chan error, 1)
	go func() {
		retryDone <- client.retrySessionErr(context.Background(), func(*session.Session) error {
			close(invokeEntered)
			<-invokeRelease
			return tgerr.New(406, "AUTH_KEY_DUPLICATED")
		})
	}()
	<-invokeEntered
	disconnectDone := make(chan error, 1)
	go func() { disconnectDone <- client.Disconnect() }()
	deadline := time.Now().Add(time.Second)
	for client.state.IsConnected() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(invokeRelease)

	select {
	case keyLen := <-st.closed:
		if keyLen != 0 {
			t.Fatalf("storage closed with %d-byte rejected key", keyLen)
		}
	case <-time.After(time.Second):
		t.Fatal("Disconnect did not close storage")
	}
	if err := <-disconnectDone; err != nil {
		t.Fatalf("Disconnect() = %v", err)
	}
	if err := <-retryDone; !errors.Is(err, ErrAuthKeyInvalidated) {
		t.Fatalf("retrySessionErr() = %v, want ErrAuthKeyInvalidated", err)
	}
}

func TestCleanupClassifierClearsRecordedTerminalSessionCause(t *testing.T) {
	st := populatedAuthStorage(t)
	client, _ := NewClient(1, "hash", &Config{ReconnectEnabled: false})
	client.storage = st
	client.mainAuthKeyOrigin.Store(authKeyOriginLoaded)
	sess, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	client.session = sess
	client.state.SetConnecting(2)
	client.state.SetConnected()

	loss, accepted := client.latchSessionAuthLossFromError(
		sess,
		client.authGeneration.Load(),
		fmt.Errorf("session exited [readLoop]: %w", tgerr.New(406, "AUTH_KEY_DUPLICATED")),
	)
	if !accepted {
		t.Fatal("terminal shutdown cause was not accepted")
	}
	client.autoConnectMu.Lock()
	client.finishMainAuthInvalidation(loss)
	client.autoConnectMu.Unlock()
	if key, _ := st.AuthKey(); len(key) != 0 {
		t.Fatalf("auth key length = %d, want 0", len(key))
	}
}

func TestDeferredPFSRefreshRejectionDoesNotDeadlockDisconnect(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{ReconnectEnabled: false})
	st := populatedAuthStorage(t)
	client.storage = st
	sess := newPFSAuthTestSession(t, st)
	client.session = sess
	client.state.SetConnecting(2)
	client.state.SetConnected()

	client.authDecisionMu.RLock()
	disconnectDone := make(chan error, 1)
	go func() { disconnectDone <- client.Disconnect() }()
	deadline := time.Now().Add(time.Second)
	for client.state.IsConnected() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if client.state.IsConnected() {
		client.authDecisionMu.RUnlock()
		t.Fatal("Disconnect did not reach the auth-decision barrier")
	}
	client.authDecisionMu.RUnlock()

	rejectDone := make(chan error, 1)
	go func() {
		rejectDone <- client.rejectActivePFSKey(sess, tgerr.New(401, "AUTH_KEY_PERM_EMPTY"))
	}()
	select {
	case err := <-disconnectDone:
		if err != nil {
			t.Fatalf("Disconnect() = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Disconnect deadlocked with deferred PFS rejection")
	}
	select {
	case err := <-rejectDone:
		if !errors.Is(err, errTemporaryAuthKeyRejected) {
			t.Fatalf("rejectActivePFSKey() = %v, want temporary rejection", err)
		}
	case <-time.After(time.Second):
		t.Fatal("deferred PFS rejection deadlocked after Disconnect")
	}
}
