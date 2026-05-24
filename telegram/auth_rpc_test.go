package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

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
	mock.setResult(tg.AuthLogOutTypeID, &tg.AuthLoggedOut{})

	ok, err := c.SignOut(context.Background())
	if err != nil {
		t.Fatalf("SignOut() error: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}

	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}
	_, reqOk := mock.lastCall().(*tg.AuthLogOutRequest)
	if !reqOk {
		t.Fatalf("expected AuthLogOutRequest, got %T", mock.lastCall())
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
