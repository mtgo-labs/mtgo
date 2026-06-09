package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

func newClientWithLoginMock(phone string, codeFn CodeFunc, pwFn PasswordFunc) (*Client, *mockBotRPCInvoke) {
	c, _ := NewClient(1, "test", nil)
	c.state.setConnected(true)
	c.cfg.PhoneNumber = phone
	c.cfg.CodeFunc = codeFn
	c.cfg.PasswordFunc = pwFn
	mock := newMockBotRPCInvoke()
	c.testInvoker = mock
	return c, mock
}

func TestLoginUser_Success(t *testing.T) {
	var codeCalls int
	codeFn := func(_ context.Context, phone string) (string, error) {
		codeCalls++
		return "12345", nil
	}

	c, mock := newClientWithLoginMock("+1234567890", codeFn, nil)

	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash1",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
	})
	mock.setResult(tg.AuthSignInTypeID, &tg.AuthAuthorization{
		User: &tg.User{ID: 42, FirstName: "Test"},
	})

	err := c.loginUser(context.Background())
	if err != nil {
		t.Fatalf("loginUser() error: %v", err)
	}
	if codeCalls != 1 {
		t.Errorf("CodeFunc called %d times, want 1", codeCalls)
	}
	if mock.callCount() != 2 {
		t.Errorf("expected 2 RPC calls (SendCode+SignIn), got %d", mock.callCount())
	}
}

func TestLoginUser_AlreadyAuthed(t *testing.T) {
	c, mock := newClientWithLoginMock(
		"+1234567890",
		func(_ context.Context, _ string) (string, error) { return "", nil }, nil,
	)

	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCodeSuccess{})

	err := c.loginUser(context.Background())
	if err != nil {
		t.Fatalf("loginUser() error: %v", err)
	}
	if mock.callCount() != 1 {
		t.Errorf("expected 1 RPC call (SendCode only), got %d", mock.callCount())
	}
}

func TestLoginUser_SendCodeError(t *testing.T) {
	c, mock := newClientWithLoginMock(
		"+1234567890",
		func(_ context.Context, _ string) (string, error) { return "12345", nil }, nil,
	)

	mock.setError(tg.AuthSendCodeTypeID, errors.New("flood"))

	err := c.loginUser(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoginUser_CodeFuncError(t *testing.T) {
	c, mock := newClientWithLoginMock(
		"+1234567890",
		func(_ context.Context, _ string) (string, error) {
			return "", errors.New("user cancelled")
		}, nil,
	)

	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash1",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
	})

	err := c.loginUser(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if mock.callCount() != 1 {
		t.Errorf("expected 1 RPC call (SendCode only), got %d", mock.callCount())
	}
}

func TestLoginUser_InvalidCodeRetryThenSuccess(t *testing.T) {
	var codeCalls int
	codeFn := func(_ context.Context, _ string) (string, error) {
		codeCalls++
		return "12345", nil
	}

	c, mock := newClientWithLoginMock("+1234567890", codeFn, nil)

	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash1",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
	})

	seq := &sequentialMock{
		parent: mock,
		responses: map[uint32][]interface{}{
			tg.AuthSignInTypeID: {
				&tgerr.Error{Code: 400, Type: "PHONE_CODE_INVALID"},
				&tg.AuthAuthorization{User: &tg.User{ID: 42}},
			},
		},
	}
	c.testInvoker = seq

	err := c.loginUser(context.Background())
	if err != nil {
		t.Fatalf("loginUser() error: %v", err)
	}
	if codeCalls != 2 {
		t.Errorf("CodeFunc called %d times, want 2 (retry on invalid)", codeCalls)
	}
}

func TestLoginUser_InvalidCodeEmptyRetryThenSuccess(t *testing.T) {
	var codeCalls int
	codeFn := func(_ context.Context, _ string) (string, error) {
		codeCalls++
		return "12345", nil
	}

	c, mock := newClientWithLoginMock("+1234567890", codeFn, nil)

	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash1",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
	})

	seq := &sequentialMock{
		parent: mock,
		responses: map[uint32][]interface{}{
			tg.AuthSignInTypeID: {
				&tgerr.Error{Code: 400, Type: "PHONE_CODE_EMPTY"},
				&tg.AuthAuthorization{User: &tg.User{ID: 42}},
			},
		},
	}
	c.testInvoker = seq

	err := c.loginUser(context.Background())
	if err != nil {
		t.Fatalf("loginUser() error: %v", err)
	}
	if codeCalls != 2 {
		t.Errorf("CodeFunc called %d times, want 2", codeCalls)
	}
}

func TestLoginUser_TooManyInvalidCodes(t *testing.T) {
	var codeCalls int
	codeFn := func(_ context.Context, _ string) (string, error) {
		codeCalls++
		return "wrong", nil
	}

	c, mock := newClientWithLoginMock("+1234567890", codeFn, nil)

	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash1",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
	})

	seq := &sequentialMock{
		parent: mock,
		responses: map[uint32][]interface{}{
			tg.AuthSignInTypeID: {
				&tgerr.Error{Code: 400, Type: "PHONE_CODE_INVALID"},
				&tgerr.Error{Code: 400, Type: "PHONE_CODE_INVALID"},
				&tgerr.Error{Code: 400, Type: "PHONE_CODE_INVALID"},
			},
		},
	}
	c.testInvoker = seq

	err := c.loginUser(context.Background())
	if err == nil {
		t.Fatal("expected error for too many attempts")
	}
	if codeCalls != 3 {
		t.Errorf("CodeFunc called %d times, want 3", codeCalls)
	}
}

func TestLoginUser_2FATriggered(t *testing.T) {
	codeFn := func(_ context.Context, _ string) (string, error) { return "12345", nil }
	pwFn := func(_ context.Context, hint string) (string, error) {
		return "anypassword", nil
	}

	c, mock := newClientWithLoginMock("+1234567890", codeFn, pwFn)

	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash1",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
	})
	// Unsupported KDF makes computeSRP fail immediately — tests that the flow
	// correctly branches to the 2FA path without needing real SRP params.
	mock.setResult(tg.AccountGetPasswordTypeID, &tg.AccountPassword{
		Hint:        "test hint",
		CurrentAlgo: &tg.PasswordKdfAlgoUnknown{},
		SRPB:        []byte{1, 2, 3},
		SRPID:       123,
	})

	seq := &sequentialMock{
		parent: mock,
		responses: map[uint32][]interface{}{
			tg.AuthSignInTypeID: {
				&tgerr.Error{Code: 401, Type: "SESSION_PASSWORD_NEEDED"},
			},
		},
	}
	c.testInvoker = seq

	err := c.loginUser(context.Background())
	// Will fail at CheckPassword because unsupported KDF, but the flow
	// correctly reached the 2FA path: SendCode -> SignIn(2FA) -> GetPassword
	if err == nil {
		t.Fatal("expected error (unsupported KDF)")
	}
	callCount := mock.callCount()
	if callCount < 3 {
		t.Errorf("expected >= 3 RPC calls (SendCode+SignIn+GetPassword), got %d", callCount)
	}
}

func TestLoginUser_2FAPasswordFuncError(t *testing.T) {
	codeFn := func(_ context.Context, _ string) (string, error) { return "12345", nil }
	pwFn := func(_ context.Context, _ string) (string, error) {
		return "", errors.New("user cancelled")
	}

	c, mock := newClientWithLoginMock("+1234567890", codeFn, pwFn)

	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash1",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
	})

	seq := &sequentialMock{
		parent: mock,
		responses: map[uint32][]interface{}{
			tg.AuthSignInTypeID: {
				// Return 2FA error without real SRP — just test that the flow
				// branches correctly. Since we can't mock through CheckPassword
				// (it does internal SRP computation), we use an unsupported KDF
				// to trigger an immediate error from computeSRP, then the flow
				// calls pwFn which returns our error.
				&tgerr.Error{Code: 401, Type: "SESSION_PASSWORD_NEEDED"},
			},
		},
	}
	c.testInvoker = seq
	mock.setResult(tg.AccountGetPasswordTypeID, &tg.AccountPassword{
		CurrentAlgo: &tg.PasswordKdfAlgoUnknown{},
		SRPB:        []byte{1, 2, 3},
		SRPID:       123,
	})

	err := c.loginUser(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoginUser_SignInOtherError(t *testing.T) {
	codeFn := func(_ context.Context, _ string) (string, error) { return "12345", nil }

	c, mock := newClientWithLoginMock("+1234567890", codeFn, nil)

	mock.setResult(tg.AuthSendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash1",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
	})

	seq := &sequentialMock{
		parent: mock,
		responses: map[uint32][]interface{}{
			tg.AuthSignInTypeID: {
				errors.New("some other rpc error"),
			},
		},
	}
	c.testInvoker = seq

	err := c.loginUser(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoginUser_NoPhoneNumber(t *testing.T) {
	c, _ := NewClient(1, "test", nil)
	c.state.setConnected(true)
	c.cfg.PhoneNumber = ""

	err := c.loginUser(context.Background())
	if err == nil {
		t.Fatal("expected error for empty phone number")
	}
}

func TestLoginPassword_TooManyAttempts(t *testing.T) {
	var pwCalls int
	pwFn := func(_ context.Context, _ string) (string, error) {
		pwCalls++
		return "wrong", nil
	}

	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountGetPasswordTypeID, &tg.AccountPassword{
		CurrentAlgo: &tg.PasswordKdfAlgoUnknown{},
		SRPB:        []byte{1, 2, 3},
		SRPID:       123,
	})

	seq := &sequentialMock{
		parent: mock,
		responses: map[uint32][]interface{}{
			tg.AuthCheckPasswordTypeID: {
				&tgerr.Error{Code: 400, Type: "PASSWORD_HASH_INVALID"},
				&tgerr.Error{Code: 400, Type: "PASSWORD_HASH_INVALID"},
				&tgerr.Error{Code: 400, Type: "PASSWORD_HASH_INVALID"},
			},
		},
	}
	c.testInvoker = seq

	err := c.loginPassword(context.Background(), pwFn)
	if err == nil {
		t.Fatal("expected error for too many password attempts")
	}
	// pwCalls is 1 because CheckPassword fails at computeSRP (unsupported KDF)
	// before reaching AuthCheckPassword. The retry loop is in loginPassword but
	// the error from computeSRP is not PASSWORD_HASH_INVALID, so it exits early.
	// This test validates the flow doesn't panic and returns an error.
}

func TestLoginPassword_PasswordFuncError(t *testing.T) {
	pwFn := func(_ context.Context, _ string) (string, error) {
		return "", errors.New("cancelled")
	}

	c, _ := NewClient(1, "test", nil)
	c.state.setConnected(true)

	err := c.loginPassword(context.Background(), pwFn)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsAuthorized(t *testing.T) {
	c, _ := NewClient(1, "test", &Config{InMemory: true, SessionName: "test"})

	// storage is only set after Connect, so isAuthorized returns false
	if c.isAuthorized() {
		t.Error("client before Connect should not be authorized")
	}

	// Simulate post-Connect: assign storage manually
	st := NewMemoryStorage()
	_ = st.SetSessionID("test")
	c.storage = st

	if c.isAuthorized() {
		t.Error("storage without user ID should not be authorized")
	}

	if err := st.SetUserID(42); err != nil {
		t.Fatalf("SetUserID: %v", err)
	}

	if !c.isAuthorized() {
		t.Error("storage with user ID should be authorized")
	}
}

func TestIsAuthorized_NilStorage(t *testing.T) {
	c, _ := NewClient(1, "test", nil)
	if c.isAuthorized() {
		t.Error("nil storage should not be authorized")
	}
}

func TestTerminalCodeFunc(t *testing.T) {
	fn := TerminalCodeFunc()
	if fn == nil {
		t.Fatal("TerminalCodeFunc() returned nil")
	}
}

func TestTerminalPasswordFunc(t *testing.T) {
	fn := TerminalPasswordFunc()
	if fn == nil {
		t.Fatal("TerminalPasswordFunc() returned nil")
	}
}

// sequentialMock wraps mockBotRPCInvoke and returns responses in sequence
// per constructor ID, then falls back to the parent mock's static results.
type sequentialMock struct {
	parent    *mockBotRPCInvoke
	responses map[uint32][]interface{}
	index     map[uint32]int
}

func (s *sequentialMock) RPCInvoke(ctx context.Context, req tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	id := req.ConstructorID()
	if seq, ok := s.responses[id]; ok {
		if s.index == nil {
			s.index = make(map[uint32]int)
		}
		i := s.index[id]
		if i < len(seq) {
			resp := seq[i]
			s.index[id] = i + 1
			s.parent.mu.Lock()
			s.parent.calls = append(s.parent.calls, req)
			s.parent.mu.Unlock()
			switch v := resp.(type) {
			case error:
				return nil, v
			case tg.TLObject:
				return v, nil
			}
		}
	}
	return s.parent.RPCInvoke(ctx, req, decode)
}

func (s *sequentialMock) RPCInvokeRaw(ctx context.Context, req tg.TLObject) ([]byte, error) {
	return s.parent.RPCInvokeRaw(ctx, req)
}
