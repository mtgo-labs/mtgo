package telegram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type signalingAuthSuccessInvoker struct {
	called chan struct{}
}

type concurrentLogoutLossInvoker struct {
	client *Client
}

type blockingTerminalAuthInvoker struct {
	entered chan struct{}
	release chan struct{}
}

type signOutTestStorage struct {
	*MemoryStorage
	clearStarted chan struct{}
	releaseClear chan struct{}
	failClear    bool
}

func (s *signOutTestStorage) SetAuthKey(key []byte) error {
	if len(key) == 0 {
		if s.clearStarted != nil {
			close(s.clearStarted)
		}
		if s.releaseClear != nil {
			<-s.releaseClear
		}
		if s.failClear {
			return errors.New("clear auth key failed")
		}
	}
	return s.MemoryStorage.SetAuthKey(key)
}

func (i *signalingAuthSuccessInvoker) RPCInvoke(_ context.Context, _ tg.TLObject, _ func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	close(i.called)
	return &tg.AuthAuthorization{User: &tg.User{ID: 99, FirstName: "stale"}}, nil
}

func (*signalingAuthSuccessInvoker) RPCInvokeRaw(context.Context, tg.TLObject) ([]byte, error) {
	return nil, nil
}

func (i *concurrentLogoutLossInvoker) RPCInvoke(_ context.Context, _ tg.TLObject, _ func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	loss, first := i.client.latchMainAuthLoss(tgerr.New(406, "AUTH_KEY_DUPLICATED"))
	_ = i.client.completeMainAuthInvalidation(loss, first)
	return &tg.AuthLoggedOut{}, nil
}

func (*concurrentLogoutLossInvoker) RPCInvokeRaw(context.Context, tg.TLObject) ([]byte, error) {
	return nil, nil
}

func (i *blockingTerminalAuthInvoker) RPCInvoke(_ context.Context, _ tg.TLObject, _ func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	close(i.entered)
	<-i.release
	return nil, tgerr.New(406, "AUTH_KEY_DUPLICATED")
}

func (*blockingTerminalAuthInvoker) RPCInvokeRaw(context.Context, tg.TLObject) ([]byte, error) {
	return nil, nil
}

func TestSendCode_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.SendCode(context.Background(), "+1234567890")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSendCode_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "abc123",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
		NextType:      &tg.AuthCodeTypeCall{},
		Timeout:       60,
	})

	result, err := c.SendCode(context.Background(), "+1234567890")
	if err != nil {
		t.Fatalf("SendCode() error: %v", err)
	}
	if result.PhoneCodeHash != "abc123" {
		t.Errorf("PhoneCodeHash = %q, want %q", result.PhoneCodeHash, "abc123")
	}
	if result.Timeout != 60 {
		t.Errorf("Timeout = %d, want 60", result.Timeout)
	}
	if result.NextType == nil {
		t.Error("NextType should not be nil")
	}
	if result.Type == nil {
		t.Error("Type should not be nil")
	}

	req, ok := mock.lastCall().(*tg.AuthSendCodeRequest)
	if !ok {
		t.Fatalf("expected AuthSendCodeRequest, got %T", mock.lastCall())
	}
	if req.PhoneNumber != "+1234567890" {
		t.Errorf("PhoneNumber = %q, want %q", req.PhoneNumber, "+1234567890")
	}
}

func TestSendCode_AlreadyAuthed(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCodeSuccess{})

	_, err := c.SendCode(context.Background(), "+1234567890")
	if !errors.Is(err, ErrAlreadyAuthed) {
		t.Errorf("expected ErrAlreadyAuthed, got %v", err)
	}
}

func TestSendCode_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AuthSendCodeTypeID, errors.New("rpc error"))

	_, err := c.SendCode(context.Background(), "+1234567890")
	if err == nil {
		t.Fatal("expected error from RPC")
	}
}

func TestSendCode_ZeroTimeout(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
	})

	result, err := c.SendCode(context.Background(), "+1234567890")
	if err != nil {
		t.Fatalf("SendCode() error: %v", err)
	}
	if result.Timeout != 0 {
		t.Errorf("Timeout = %d, want 0 for zero timeout", result.Timeout)
	}
	if result.NextType != nil {
		t.Error("NextType should be nil when not set")
	}
}

func TestSignIn_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.SignIn(context.Background(), "+1234567890", "hash", "12345")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSignIn_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AuthSignInTypeID, &tg.AuthAuthorization{
		User: &tg.User{
			ID:        42,
			FirstName: "John",
			LastName:  "Doe",
		},
	})

	user, err := c.SignIn(context.Background(), "+1234567890", "hash123", "12345")
	if err != nil {
		t.Fatalf("SignIn() error: %v", err)
	}
	if user.ID != 42 {
		t.Errorf("user ID = %d, want 42", user.ID)
	}
	if user.FirstName != "John" {
		t.Errorf("FirstName = %q, want %q", user.FirstName, "John")
	}

	req, ok := mock.lastCall().(*tg.AuthSignInRequest)
	if !ok {
		t.Fatalf("expected AuthSignInRequest, got %T", mock.lastCall())
	}
	if req.PhoneNumber != "+1234567890" {
		t.Errorf("PhoneNumber = %q, want %q", req.PhoneNumber, "+1234567890")
	}
	if req.PhoneCodeHash != "hash123" {
		t.Errorf("PhoneCodeHash = %q, want %q", req.PhoneCodeHash, "hash123")
	}
	if req.PhoneCode != "12345" {
		t.Errorf("PhoneCode = %q, want %q", req.PhoneCode, "12345")
	}
}

func TestSignInDoesNotRestoreIdentityAfterTerminalAuthCleanup(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	st := populatedAuthStorage(t)
	c.storage = st
	invoker := &signalingAuthSuccessInvoker{called: make(chan struct{})}
	c.testInvoker = invoker

	// Hold a classification reader so SignIn completes its RPC but cannot
	// publish the successful authorization until cleanup has latched the loss.
	c.authDecisionMu.RLock()
	done := make(chan error, 1)
	go func() {
		_, err := c.SignIn(context.Background(), "+1234567890", "hash", "12345")
		done <- err
	}()
	select {
	case <-invoker.called:
	case <-time.After(time.Second):
		c.authDecisionMu.RUnlock()
		t.Fatal("SignIn did not reach the auth RPC")
	}

	cleanupErr := c.invalidateMainAuth(tgerr.New(406, "AUTH_KEY_DUPLICATED"))
	if !errors.Is(cleanupErr, ErrAuthKeyInvalidated) {
		c.authDecisionMu.RUnlock()
		t.Fatalf("invalidateMainAuth() = %v, want ErrAuthKeyInvalidated", cleanupErr)
	}
	// Explicit recovery clears the loss object. The credential generation must
	// still reject the pre-loss authorization result once it can commit.
	if err := c.prepareExplicitAuthRecovery(); err != nil {
		c.authDecisionMu.RUnlock()
		t.Fatalf("prepareExplicitAuthRecovery() = %v", err)
	}
	c.authDecisionMu.RUnlock()

	select {
	case err := <-done:
		if !errors.Is(err, ErrNotConnected) {
			t.Fatalf("SignIn() = %v, want ErrNotConnected after credential generation changed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SignIn did not return after terminal auth cleanup")
	}
	if me := c.Me(); me != nil {
		t.Fatalf("Me() = %#v, want nil after terminal auth cleanup", me)
	}
	if userID, _ := st.UserID(); userID != 0 {
		t.Fatalf("stored user ID = %d, want 0", userID)
	}
	if firstName, _ := st.FirstName(); firstName != "" {
		t.Fatalf("stored first name = %q, want empty", firstName)
	}
}

func TestAuthAttemptGenerationFollowsPrimaryMigrationRetry(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{DC: 2, RetryRPCOnReconnect: false})
	st := NewMemoryStorage()
	if err := st.SetDCID(2); err != nil {
		t.Fatal(err)
	}
	if err := st.SetAuthKey(bytes.Repeat([]byte{0x11}, 256)); err != nil {
		t.Fatal(err)
	}
	client.storage = st
	oldSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	client.session = oldSession
	client.state.SetConnecting(2)
	client.state.SetConnected()

	ctx, attempt := client.withAuthAttempt(context.Background())
	err = client.retrySessionErr(ctx, func(*session.Session) error {
		return tgerr.New(303, "PHONE_MIGRATE_4")
	})
	if !tgerr.Is(err, tgerr.ErrPhoneMigrate) {
		t.Fatalf("initial auth attempt = %v, want PHONE_MIGRATE", err)
	}

	migrateDone := make(chan error, 1)
	go func() {
		migrateDone <- client.switchPrimaryDC(ctx, 4, st, func(time.Duration) error {
			if err := client.advanceAuthGeneration(); err != nil {
				return err
			}
			if err := st.SetAuthKey(bytes.Repeat([]byte{0x44}, 256)); err != nil {
				return err
			}
			targetSession, err := session.NewSession(session.DataCenter{ID: 4}, st, "test", "1", "en", "en")
			if err != nil {
				return err
			}
			client.mu.Lock()
			client.session = targetSession
			client.mu.Unlock()
			client.state.SetConnecting(4)
			client.state.SetConnected()
			return nil
		})
	}()
	select {
	case err := <-migrateDone:
		if err != nil {
			t.Fatalf("switchPrimaryDC() = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("primary migration deadlocked after the 303 auth attempt")
	}

	if err := client.retrySessionErr(ctx, func(sess *session.Session) error {
		if sess == nil || sess.DC().ID != 4 {
			return fmt.Errorf("retry session DC = %v, want 4", sess)
		}
		return nil
	}); err != nil {
		t.Fatalf("retried auth attempt = %v", err)
	}
	user := &types.User{ID: 77, FirstName: "migrated"}
	if err := client.commitAuthorizedUser(user, attempt.generation.Load()); err != nil {
		t.Fatalf("commitAuthorizedUser() = %v", err)
	}
	if got, _ := st.UserID(); got != user.ID {
		t.Fatalf("stored user ID = %d, want %d", got, user.ID)
	}
}

func TestLateTerminalResultFromRetiredGenerationDoesNotInvalidateFreshKey(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	st := populatedAuthStorage(t)
	oldSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	c.storage = st
	c.session = oldSession
	c.state.setConnected(true)
	c.mainAuthKeyOrigin.Store(authKeyOriginLoaded)

	enteredA := make(chan struct{})
	enteredB := make(chan struct{})
	releaseA := make(chan struct{})
	releaseB := make(chan struct{})
	doneA := make(chan error, 1)
	doneB := make(chan error, 1)
	cause := tgerr.New(406, "AUTH_KEY_DUPLICATED")
	go func() {
		doneA <- c.retrySessionErr(context.Background(), func(*session.Session) error {
			close(enteredA)
			<-releaseA
			return cause
		})
	}()
	go func() {
		doneB <- c.retrySessionErr(context.Background(), func(*session.Session) error {
			close(enteredB)
			<-releaseB
			return cause
		})
	}()
	for _, entered := range []<-chan struct{}{enteredA, enteredB} {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("old-session RPC did not start")
		}
	}

	close(releaseA)
	select {
	case err := <-doneA:
		if !errors.Is(err, ErrAuthKeyInvalidated) {
			t.Fatalf("first terminal RPC = %v, want ErrAuthKeyInvalidated", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first terminal RPC did not finish cleanup")
	}
	if err := c.prepareExplicitAuthRecovery(); err != nil {
		t.Fatalf("prepareExplicitAuthRecovery() = %v", err)
	}

	freshKey := bytes.Repeat([]byte{0x7a}, 256)
	if err := st.SetAuthKey(freshKey); err != nil {
		t.Fatal(err)
	}
	newSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	c.session = newSession
	c.mu.Unlock()
	c.mainAuthKeyOrigin.Store(authKeyOriginFresh)
	c.state.SetConnected()

	close(releaseB)
	select {
	case err := <-doneB:
		if !tgerr.Is(err, tgerr.ErrAuthKeyDuplicated) {
			t.Fatalf("late terminal RPC = %v, want original AUTH_KEY_DUPLICATED", err)
		}
	case <-time.After(time.Second):
		t.Fatal("late terminal RPC did not return")
	}
	if err := c.authLossError(); err != nil {
		t.Fatalf("late terminal RPC relatched auth loss: %v", err)
	}
	storedKey, err := st.AuthKey()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(storedKey, freshKey) {
		t.Fatal("late terminal RPC cleared the fresh replacement key")
	}
}

func TestSupersededCompleteConnectDoesNotInvalidateReplacement(t *testing.T) {
	c, _ := NewClient(1, "h", &Config{BotToken: "1:test", ReconnectEnabled: false})
	st := NewMemoryStorage()
	if err := st.SetAuthKey(bytes.Repeat([]byte{0x11}, 256)); err != nil {
		t.Fatal(err)
	}
	oldSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	c.storage = st
	c.session = oldSession
	c.state.SetConnecting(2)
	c.state.SetConnected()
	oldReady := &connectionReadiness{sess: oldSession, done: make(chan struct{}), requiresAuth: true}
	c.ready = oldReady
	invoker := &blockingTerminalAuthInvoker{entered: make(chan struct{}), release: make(chan struct{})}
	c.testInvoker = invoker

	done := make(chan error, 1)
	go func() { done <- c.completeConnect(time.Second, oldSession, oldReady) }()
	select {
	case <-invoker.entered:
	case <-time.After(time.Second):
		t.Fatal("startup bot authentication did not begin")
	}

	if err := c.advanceAuthGeneration(); err != nil {
		t.Fatal(err)
	}
	freshKey := bytes.Repeat([]byte{0x22}, 256)
	if err := st.SetAuthKey(freshKey); err != nil {
		t.Fatal(err)
	}
	newSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	c.session = newSession
	c.mu.Unlock()
	c.beginMainReadiness(newSession, true)
	close(invoker.release)

	select {
	case err := <-done:
		if !errors.Is(err, errConnectionReplaced) {
			t.Fatalf("completeConnect() = %v, want errConnectionReplaced", err)
		}
	case <-time.After(time.Second):
		t.Fatal("superseded completeConnect did not return")
	}
	if c.Session() != newSession {
		t.Fatal("superseded completeConnect detached the replacement session")
	}
	if err := c.authLossError(); err != nil {
		t.Fatalf("superseded completeConnect relatched auth loss: %v", err)
	}
	key, err := st.AuthKey()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(key, freshKey) {
		t.Fatal("superseded completeConnect cleared the replacement key")
	}
}

func TestSignIn_2FARequired(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AuthSignInTypeID, &tgerr.Error{
		Code:    401,
		Message: "SESSION_PASSWORD_NEEDED",
		Type:    "SESSION_PASSWORD_NEEDED",
	})

	_, err := c.SignIn(context.Background(), "+1234567890", "hash", "12345")
	if !errors.Is(err, Err2FARequired) {
		t.Errorf("expected Err2FARequired, got %v", err)
	}
}

func TestSignIn_SignUpRequired(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AuthSignInTypeID, &tg.AuthAuthorizationSignUpRequired{})

	_, err := c.SignIn(context.Background(), "+1234567890", "hash", "12345")
	if !errors.Is(err, ErrSignUpRequired) {
		t.Errorf("expected ErrSignUpRequired, got %v", err)
	}
}

func TestSignUp_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.SignUp(context.Background(), "+1234567890", "hash", "John")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSignUp_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AuthSignUpTypeID, &tg.AuthAuthorization{
		User: &tg.User{
			ID:        99,
			FirstName: "Jane",
			LastName:  "Smith",
		},
	})

	user, err := c.SignUp(context.Background(), "+1234567890", "hash", "Jane", "Smith")
	if err != nil {
		t.Fatalf("SignUp() error: %v", err)
	}
	if user.ID != 99 {
		t.Errorf("user ID = %d, want 99", user.ID)
	}
	if user.FirstName != "Jane" {
		t.Errorf("FirstName = %q, want %q", user.FirstName, "Jane")
	}
	if user.LastName != "Smith" {
		t.Errorf("LastName = %q, want %q", user.LastName, "Smith")
	}

	req, ok := mock.lastCall().(*tg.AuthSignUpRequest)
	if !ok {
		t.Fatalf("expected AuthSignUpRequest, got %T", mock.lastCall())
	}
	if req.FirstName != "Jane" {
		t.Errorf("FirstName = %q, want %q", req.FirstName, "Jane")
	}
	if req.LastName != "Smith" {
		t.Errorf("LastName = %q, want %q", req.LastName, "Smith")
	}
}

func TestSignUp_NoLastName(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AuthSignUpTypeID, &tg.AuthAuthorization{
		User: &tg.User{ID: 1, FirstName: "Test"},
	})

	user, err := c.SignUp(context.Background(), "+1234567890", "hash", "Test")
	if err != nil {
		t.Fatalf("SignUp() error: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}

	req, ok := mock.lastCall().(*tg.AuthSignUpRequest)
	if !ok {
		t.Fatalf("expected AuthSignUpRequest, got %T", mock.lastCall())
	}
	if req.LastName != "" {
		t.Errorf("LastName = %q, want empty", req.LastName)
	}
}

func TestSignUp_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AuthSignUpTypeID, errors.New("rpc error"))

	_, err := c.SignUp(context.Background(), "+1234567890", "hash", "Test")
	if err == nil {
		t.Fatal("expected error from RPC")
	}
}

func TestSignOut_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.SignOut(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSignOut_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.cfg.SessionString = "revoked-session"
	c.storage = populatedAuthStorage(t)
	mock.setResult(tg.AuthLogOutTypeID, &tg.AuthLoggedOut{})

	ok, err := c.SignOut(context.Background())
	if err != nil {
		t.Fatalf("SignOut() error: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}
	if !c.sessionStringInvalidated.Load() {
		t.Fatal("successful SignOut did not invalidate the configured session string")
	}

	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}
	_, reqOk := mock.lastCall().(*tg.AuthLogOutRequest)
	if !reqOk {
		t.Fatalf("expected AuthLogOutRequest, got %T", mock.lastCall())
	}
}

func TestSignOutBlocksAutoConnectUntilCredentialsAreCleared(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.cfg.AutoConnect = true
	c.cfg.SessionString = "revoked-session"
	st := &signOutTestStorage{
		MemoryStorage: populatedAuthStorage(t),
		clearStarted:  make(chan struct{}),
		releaseClear:  make(chan struct{}),
	}
	c.storage = st
	mock.setResult(tg.AuthLogOutTypeID, &tg.AuthLoggedOut{})

	signOutDone := make(chan error, 1)
	go func() {
		_, err := c.SignOut(context.Background())
		signOutDone <- err
	}()
	select {
	case <-st.clearStarted:
	case <-time.After(time.Second):
		t.Fatal("SignOut did not reach credential cleanup")
	}

	// Storage still contains the revoked key while SetAuthKey is blocked. An
	// AutoConnect attempt must fail at the explicit-logout gate instead of
	// waiting for lifecycle ownership and then reusing that key.
	ensureDone := make(chan error, 1)
	go func() { ensureDone <- c.ensureConnectedContext(context.Background()) }()
	select {
	case err := <-ensureDone:
		if !errors.Is(err, ErrAuthKeyInvalidated) {
			close(st.releaseClear)
			<-signOutDone
			t.Fatalf("ensureConnectedContext() = %v, want ErrAuthKeyInvalidated", err)
		}
	case <-time.After(time.Second):
		close(st.releaseClear)
		<-signOutDone
		t.Fatal("AutoConnect waited on logout while the revoked key remained persisted")
	}

	close(st.releaseClear)
	select {
	case err := <-signOutDone:
		if err != nil {
			t.Fatalf("SignOut() = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SignOut did not finish after credential cleanup was released")
	}
	if key, _ := st.AuthKey(); len(key) != 0 {
		t.Fatalf("auth key length = %d, want 0", len(key))
	}
	if c.Storage() != nil {
		t.Fatal("SignOut retained storage after credential cleanup")
	}
}

func TestSignOutCleanupFailureRemainsFailClosed(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.cfg.AutoConnect = true
	c.cfg.SessionString = "revoked-session"
	st := &signOutTestStorage{MemoryStorage: populatedAuthStorage(t), failClear: true}
	c.storage = st
	mock.setResult(tg.AuthLogOutTypeID, &tg.AuthLoggedOut{})

	ok, err := c.SignOut(context.Background())
	if err == nil || ok {
		t.Fatalf("SignOut() = (%v, %v), want cleanup error", ok, err)
	}
	if !c.explicitLogout.Load() || !c.sessionStringInvalidated.Load() {
		t.Fatal("credential cleanup failure reopened automatic session reuse")
	}
	if key, _ := st.AuthKey(); len(key) == 0 {
		t.Fatal("failing storage unexpectedly cleared the auth key")
	}
	if err := c.ensureConnectedContext(context.Background()); !errors.Is(err, ErrAuthKeyInvalidated) {
		t.Fatalf("ensureConnectedContext() = %v, want ErrAuthKeyInvalidated", err)
	}
}

func TestSignOutCompletesConcurrentAuthLossBeforeClosingStorage(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	st := populatedAuthStorage(t)
	c.storage = st
	c.testInvoker = &concurrentLogoutLossInvoker{client: c}

	ok, err := c.SignOut(context.Background())
	if err != nil || !ok {
		t.Fatalf("SignOut() = (%v, %v), want successful durable logout", ok, err)
	}
	c.authLossMu.Lock()
	loss := c.authLoss
	c.authLossMu.Unlock()
	if loss == nil {
		t.Fatal("concurrent terminal auth loss was not retained")
	}
	select {
	case <-loss.done:
	case <-time.After(time.Second):
		t.Fatal("concurrent terminal auth cleanup did not complete")
	}
	if loss.result.Cleanup != nil {
		t.Fatalf("terminal auth cleanup ran after storage close: %v", loss.result.Cleanup)
	}
	if key, _ := st.AuthKey(); len(key) != 0 {
		t.Fatalf("auth key length = %d, want 0", len(key))
	}
}

func TestSignOutContextDisablesReconnectRetry(t *testing.T) {
	c, _ := NewClient(1, "h", &Config{RetryRPCOnReconnect: true})
	ctx := context.WithValue(context.Background(), signOutContextKey{}, true)
	if c.retryRPCOnReconnect(ctx) {
		t.Fatal("auth.logOut context allowed reconnect retry")
	}
}

func TestSignOut_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AuthLogOutTypeID, errors.New("rpc error"))

	ok, err := c.SignOut(context.Background())
	if err == nil {
		t.Fatal("expected error from RPC")
	}
	if ok {
		t.Error("expected false on error")
	}
}

func TestGetPasswordHint_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.GetPasswordHint(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestGetPasswordHint_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountGetPasswordTypeID, &tg.AccountPassword{
		Hint: "my secret hint",
	})

	hint, err := c.GetPasswordHint(context.Background())
	if err != nil {
		t.Fatalf("GetPasswordHint() error: %v", err)
	}
	if hint != "my secret hint" {
		t.Errorf("hint = %q, want %q", hint, "my secret hint")
	}
}

func TestGetPasswordHint_EmptyHint(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountGetPasswordTypeID, &tg.AccountPassword{
		Hint: "",
	})

	hint, err := c.GetPasswordHint(context.Background())
	if err != nil {
		t.Fatalf("GetPasswordHint() error: %v", err)
	}
	if hint != "" {
		t.Errorf("hint = %q, want empty", hint)
	}
}

func TestGetPasswordHint_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AccountGetPasswordTypeID, errors.New("rpc error"))

	_, err := c.GetPasswordHint(context.Background())
	if err == nil {
		t.Fatal("expected error from RPC")
	}
}

func TestCheckPassword_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.CheckPassword(context.Background(), "mypassword")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestCheckPassword_UnsupportedKDF(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountGetPasswordTypeID, &tg.AccountPassword{
		CurrentAlgo: &tg.PasswordKdfAlgoUnknown{},
		SRPB:        []byte{1, 2, 3},
		SRPID:       123,
	})

	_, err := c.CheckPassword(context.Background(), "mypassword")
	if err == nil {
		t.Fatal("expected error for unsupported KDF")
	}
}

func TestRecoverPassword_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.RecoverPassword(context.Background(), "recoverycode")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestRecoverPassword_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AuthRecoverPasswordTypeID, &tg.AuthAuthorization{
		User: &tg.User{
			ID:        55,
			FirstName: "Recovered",
		},
	})

	user, err := c.RecoverPassword(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("RecoverPassword() error: %v", err)
	}
	if user.ID != 55 {
		t.Errorf("user ID = %d, want 55", user.ID)
	}

	req, ok := mock.lastCall().(*tg.AuthRecoverPasswordRequest)
	if !ok {
		t.Fatalf("expected AuthRecoverPasswordRequest, got %T", mock.lastCall())
	}
	if req.Code != "abc123" {
		t.Errorf("Code = %q, want %q", req.Code, "abc123")
	}
}

func TestRecoverPassword_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AuthRecoverPasswordTypeID, errors.New("rpc error"))

	_, err := c.RecoverPassword(context.Background(), "code")
	if err == nil {
		t.Fatal("expected error from RPC")
	}
}

func TestComputeSRP_UnsupportedKDF(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountGetPasswordTypeID, &tg.AccountPassword{
		CurrentAlgo: &tg.PasswordKdfAlgoUnknown{},
		SRPB:        []byte{1, 2, 3},
		SRPID:       123,
	})

	_, err := c.computeSRP(context.Background(), "testpass")
	if err == nil {
		t.Fatal("expected error for unsupported KDF")
	}
}

func TestGetActiveSessions_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.GetActiveSessions(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestGetActiveSessions_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountGetAuthorizationsTypeID, &tg.AccountAuthorizations{
		AuthorizationTTLDays: 180,
		Authorizations: []tg.AuthorizationClass{
			&tg.Authorization{
				Hash:        100,
				DeviceModel: "iPhone",
				Platform:    "iOS",
				Ip:          "1.2.3.4",
				Country:     "US",
			},
			&tg.Authorization{
				Hash:        200,
				DeviceModel: "Android",
				Platform:    "Android",
				Ip:          "5.6.7.8",
				Country:     "GB",
			},
		},
	})

	result, err := c.GetActiveSessions(context.Background())
	if err != nil {
		t.Fatalf("GetActiveSessions() error: %v", err)
	}
	if result.AuthorizationTTLDays != 180 {
		t.Errorf("AuthorizationTTLDays = %d, want 180", result.AuthorizationTTLDays)
	}
	if len(result.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(result.Sessions))
	}
	if result.Sessions[0].DeviceModel != "iPhone" {
		t.Errorf("DeviceModel = %q, want %q", result.Sessions[0].DeviceModel, "iPhone")
	}
	if result.Sessions[0].IP != "1.2.3.4" {
		t.Errorf("IP = %q, want %q", result.Sessions[0].IP, "1.2.3.4")
	}
	if result.Sessions[1].Country != "GB" {
		t.Errorf("Country = %q, want %q", result.Sessions[1].Country, "GB")
	}
}

func TestGetActiveSessions_EmptySessions(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountGetAuthorizationsTypeID, &tg.AccountAuthorizations{
		AuthorizationTTLDays: 30,
		Authorizations:       []tg.AuthorizationClass{},
	})

	result, err := c.GetActiveSessions(context.Background())
	if err != nil {
		t.Fatalf("GetActiveSessions() error: %v", err)
	}
	if len(result.Sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(result.Sessions))
	}
}

func TestGetActiveSessions_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AccountGetAuthorizationsTypeID, errors.New("rpc error"))

	_, err := c.GetActiveSessions(context.Background())
	if err == nil {
		t.Fatal("expected error from RPC")
	}
}

func TestResetSession_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.ResetSession(context.Background(), 12345)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestResetSession_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountResetAuthorizationTypeID, &tg.BoolTrue{})

	err := c.ResetSession(context.Background(), 12345)
	if err != nil {
		t.Fatalf("ResetSession() error: %v", err)
	}

	req, ok := mock.lastCall().(*tg.AccountResetAuthorizationRequest)
	if !ok {
		t.Fatalf("expected AccountResetAuthorizationRequest, got %T", mock.lastCall())
	}
	if req.Hash != 12345 {
		t.Errorf("Hash = %d, want 12345", req.Hash)
	}
}

func TestResetSession_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AccountResetAuthorizationTypeID, errors.New("rpc error"))

	err := c.ResetSession(context.Background(), 12345)
	if err == nil {
		t.Fatal("expected error from RPC")
	}
}
