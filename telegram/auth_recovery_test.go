package telegram

import (
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

func TestRequestPasswordRecovery(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AuthRequestPasswordRecoveryTypeID, &tg.AuthPasswordRecovery{EmailPattern: "sm***@example.com"})

	pattern, err := c.RequestPasswordRecovery(t.Context())
	if err != nil {
		t.Fatalf("RequestPasswordRecovery: %v", err)
	}
	if pattern != "sm***@example.com" {
		t.Fatalf("pattern = %q", pattern)
	}
}

func TestQRLoginTwoFAMappedToErr2FARequired(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AuthImportLoginTokenTypeID, tgerr.New(401, "SESSION_PASSWORD_NEEDED"))

	_, err := c.CheckQRCodeLoginToken(t.Context(), []byte("token"))
	if !errors.Is(err, Err2FARequired) {
		t.Fatalf("err = %v, want Err2FARequired", err)
	}
}
