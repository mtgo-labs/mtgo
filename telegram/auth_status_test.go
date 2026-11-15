package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tgerr"
)

func TestAccountStatusErrorsAreAuthLost(t *testing.T) {
	for _, msg := range []string{"USER_DEACTIVATED", "USER_DEACTIVATED_BAN"} {
		if !isAuthLostError(tgerr.New(401, msg)) {
			t.Fatalf("%s not classified as terminal auth loss", msg)
		}
	}
	// Peer-scoped deactivation is not account loss: it means a *different*
	// user was deleted and must surface as a normal RPC error.
	if isAuthLostError(tgerr.New(400, "INPUT_USER_DEACTIVATED")) {
		t.Fatal("INPUT_USER_DEACTIVATED misclassified as account auth loss")
	}
	if isAuthLostError(tgerr.New(420, "FLOOD_WAIT_5")) {
		t.Fatal("flood wait misclassified as auth loss")
	}
}

func TestAccountStatusErrorsInvalidateMainAuth(t *testing.T) {
	c, _ := NewClient(1, "test", nil)
	for _, msg := range []string{"USER_DEACTIVATED", "USER_DEACTIVATED_BAN"} {
		if !c.shouldInvalidateMainAuthFrom(nil, tgerr.New(401, msg)) {
			t.Fatalf("%s does not trigger main-auth invalidation", msg)
		}
	}
}

func TestAuthKeyInvalidatedErrorAccountTerminated(t *testing.T) {
	terminated := &AuthKeyInvalidatedError{Cause: tgerr.New(401, "USER_DEACTIVATED_BAN")}
	if !terminated.AccountTerminated() {
		t.Fatal("banned account not reported as terminated")
	}
	revoked := &AuthKeyInvalidatedError{Cause: tgerr.New(401, "SESSION_REVOKED")}
	if revoked.AccountTerminated() {
		t.Fatal("session revocation misreported as account termination")
	}
	var none *AuthKeyInvalidatedError
	if none.AccountTerminated() {
		t.Fatal("nil error reported as terminated")
	}
}
