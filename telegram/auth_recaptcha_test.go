package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type recaptchaSolverFunc func(context.Context, string, string, string) (string, error)

func (f recaptchaSolverFunc) SolveRecaptcha(ctx context.Context, packageID, action, key string) (string, error) {
	return f(ctx, packageID, action, key)
}

func TestParseRecaptchaChallenge(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantAction string
		wantKey    string
		wantOK     bool
	}{
		{
			name:       "telegram RPC error",
			err:        tgerr.New(403, "RECAPTCHA_CHECK_signup__site-key"),
			wantAction: "signup",
			wantKey:    "site-key",
			wantOK:     true,
		},
		{
			name: "unrelated error",
			err:  errors.New("PHONE_NUMBER_INVALID"),
		},
		{
			name: "missing key",
			err:  tgerr.New(403, "RECAPTCHA_CHECK_signup__"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, key, ok := parseRecaptchaChallenge(tt.err)
			if action != tt.wantAction || key != tt.wantKey || ok != tt.wantOK {
				t.Fatalf("parseRecaptchaChallenge() = (%q, %q, %v), want (%q, %q, %v)",
					action, key, ok, tt.wantAction, tt.wantKey, tt.wantOK)
			}
		})
	}
}

func TestSendCodeSolvesRecaptcha(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AuthSendCodeTypeID, tgerr.New(403, "RECAPTCHA_CHECK_signup__site-key"))
	mock.setResult(tg.InvokeWithReCaptchaTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash",
		Type:          &tg.AuthSentCodeTypeSms{Length: 5},
	})

	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "value")
	c.updateConfig(func(cfg *Config) {
		cfg.Device.PackageID = "org.telegram.messenger"
		cfg.RecaptchaSolver = recaptchaSolverFunc(func(solveCtx context.Context, packageID, action, key string) (string, error) {
			if solveCtx.Value(contextKey{}) != "value" {
				t.Fatal("solver did not receive the SendCode context")
			}
			if packageID != "org.telegram.messenger" {
				t.Fatalf("package ID = %q, want %q", packageID, "org.telegram.messenger")
			}
			if action != "signup" || key != "site-key" {
				t.Fatalf("challenge = (%q, %q), want (%q, %q)", action, key, "signup", "site-key")
			}
			return " captcha-token ", nil
		})
	})

	result, err := c.SendCode(ctx, "+1234567890")
	if err != nil {
		t.Fatalf("SendCode() error: %v", err)
	}
	if result.PhoneCodeHash != "hash" {
		t.Fatalf("PhoneCodeHash = %q, want %q", result.PhoneCodeHash, "hash")
	}
	if mock.callCount() != 2 {
		t.Fatalf("RPC call count = %d, want 2", mock.callCount())
	}

	wrapped, ok := mock.lastCall().(*tg.InvokeWithReCaptchaRequest)
	if !ok {
		t.Fatalf("last request = %T, want *tg.InvokeWithReCaptchaRequest", mock.lastCall())
	}
	if wrapped.Token != "captcha-token" {
		t.Fatalf("captcha token = %q, want %q", wrapped.Token, "captcha-token")
	}
	if _, ok := wrapped.Query.(*tg.AuthSendCodeRequest); !ok {
		t.Fatalf("wrapped query = %T, want *tg.AuthSendCodeRequest", wrapped.Query)
	}
}

func TestSendCodeReturnsRecaptchaWithoutSolver(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	challengeErr := tgerr.New(403, "RECAPTCHA_CHECK_signup__site-key")
	mock.setError(tg.AuthSendCodeTypeID, challengeErr)

	_, err := c.SendCode(context.Background(), "+1234567890")
	if !errors.Is(err, challengeErr) {
		t.Fatalf("SendCode() error = %v, want original challenge", err)
	}
	if mock.callCount() != 1 {
		t.Fatalf("RPC call count = %d, want 1", mock.callCount())
	}
}

func TestSendCodeReturnsRecaptchaSolverError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AuthSendCodeTypeID, tgerr.New(403, "RECAPTCHA_CHECK_signup__site-key"))
	solverErr := errors.New("solver unavailable")
	c.updateConfig(func(cfg *Config) {
		cfg.RecaptchaSolver = recaptchaSolverFunc(func(context.Context, string, string, string) (string, error) {
			return "", solverErr
		})
	})

	_, err := c.SendCode(context.Background(), "+1234567890")
	if !errors.Is(err, solverErr) {
		t.Fatalf("SendCode() error = %v, want solver error", err)
	}
	if mock.callCount() != 1 {
		t.Fatalf("RPC call count = %d, want 1", mock.callCount())
	}
}
