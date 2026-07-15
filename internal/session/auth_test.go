package session

import (
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/internal/crypto"
)

func TestAuthCreatePQRequest(t *testing.T) {
	t.Skip("full DH handshake requires mock server")
}

// TestKeyVerificationError_WrapsSentinel verifies the typed error satisfies
// errors.Is(err, ErrKeyVerificationFailed) so callers can detect MITM.
func TestKeyVerificationError_WrapsSentinel(t *testing.T) {
	kve := &KeyVerificationError{
		Reason:   "fingerprint_mismatch",
		Observed: 42,
		Expected: []int64{1, 2, 3},
	}
	if !errors.Is(kve, ErrKeyVerificationFailed) {
		t.Fatal("errors.Is(KeyVerificationError, ErrKeyVerificationFailed) = false")
	}
	if !errors.Is(kve, crypto.ErrKeyVerificationFailed) {
		t.Fatal("errors.Is for cross-package sentinel failed")
	}
	if kve.Error() == "" {
		t.Fatal("Error() is empty")
	}
}

// TestFindKeyFingerprint_WithKeySet verifies the RSAKeySet-based lookup path.
func TestFindKeyFingerprint_WithKeySet(t *testing.T) {
	ks := crypto.NewRSAKeySet()
	a := &Auth{keySet: ks}

	// A bundled fingerprint should be found.
	trustedFPs := ks.TrustedFingerprints()
	if len(trustedFPs) == 0 {
		t.Fatal("no trusted fingerprints")
	}
	fp, ok := a.findKeyFingerprint(trustedFPs)
	if !ok {
		t.Fatal("findKeyFingerprint failed for a trusted fingerprint")
	}
	if fp != trustedFPs[0] {
		t.Errorf("got fp %d, want %d", fp, trustedFPs[0])
	}

	// An untrusted fingerprint should not be found.
	_, ok = a.findKeyFingerprint([]int64{99999999})
	if ok {
		t.Fatal("findKeyFingerprint accepted an untrusted fingerprint")
	}
}

// TestFindKeyFingerprint_NilKeySetFallback verifies backward compat: when
// no RSAKeySet is set, the static crypto.ServerPublicKeys map is used.
func TestFindKeyFingerprint_NilKeySetFallback(t *testing.T) {
	a := &Auth{} // keySet is nil → backward compat path
	// Pick any bundled fingerprint.
	var bundledFP int64
	for _, fp := range crypto.ServerKeyFingerprints() {
		bundledFP = fp
		break
	}
	fp, ok := a.findKeyFingerprint([]int64{bundledFP})
	if !ok {
		t.Fatal("nil keySet: findKeyFingerprint failed for bundled fingerprint")
	}
	if fp != bundledFP {
		t.Errorf("got fp %d, want %d", fp, bundledFP)
	}
}
