package session

import (
	"errors"
	"math/big"
	"testing"

	"github.com/mtgo-labs/mtgo/internal/crypto"
)

func TestFactorPQ(t *testing.T) {
	p, q, err := factorPQ(big.NewInt(323).Bytes())
	if err != nil {
		t.Fatalf("factorPQ: %v", err)
	}
	if p != 17 || q != 19 {
		t.Fatalf("factorPQ = (%d, %d), want (17, 19)", p, q)
	}
}

func TestFactorPQRejectsMalformedValues(t *testing.T) {
	tests := [][]byte{
		nil,
		{1},
		big.NewInt(17).Bytes(),
		big.NewInt(49).Bytes(),
		{0x80, 0, 0, 0, 0, 0, 0, 0},
		make([]byte, 9),
	}
	for _, pq := range tests {
		if _, _, err := factorPQ(pq); err == nil {
			t.Fatalf("factorPQ(%x) unexpectedly succeeded", pq)
		}
	}
}

func TestValidateDHGenResponse(t *testing.T) {
	var nonce, serverNonce [16]byte
	nonce[0] = 1
	serverNonce[0] = 2
	newNonce := make([]byte, 32)
	authKey := make([]byte, 256)
	copy(newNonce, []byte("new nonce"))
	copy(authKey, []byte("auth key"))

	for hashNumber := byte(1); hashNumber <= 3; hashNumber++ {
		var hash [16]byte
		copy(hash[:], computeNewNonceHash(newNonce, authKey, hashNumber))
		if err := validateDHGenResponse(nonce, serverNonce, nonce, serverNonce, hash, newNonce, authKey, hashNumber); err != nil {
			t.Fatalf("hash %d: %v", hashNumber, err)
		}

		badNonce := nonce
		badNonce[0]++
		if err := validateDHGenResponse(badNonce, serverNonce, nonce, serverNonce, hash, newNonce, authKey, hashNumber); !errors.Is(err, ErrNonceMismatch) {
			t.Fatalf("hash %d nonce error = %v", hashNumber, err)
		}

		hash[0]++
		if err := validateDHGenResponse(nonce, serverNonce, nonce, serverNonce, hash, newNonce, authKey, hashNumber); !errors.Is(err, ErrNewNonceHashMismatch) {
			t.Fatalf("hash %d mismatch error = %v", hashNumber, err)
		}
	}
}

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
