package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"math/big"

	"golang.org/x/crypto/pbkdf2"
)

func sha256sum(data ...[]byte) []byte {
	h := sha256.New()
	for _, d := range data {
		h.Write(d)
	}
	return h.Sum(nil)
}

func xorBytes(a, b []byte) []byte {
	n := min(len(a), len(b))
	out := make([]byte, n)
	for i := range n {
		out[i] = a[i] ^ b[i]
	}
	return out
}

func pad256(n *big.Int) []byte {
	b := n.Bytes()
	if len(b) > 256 {
		b = b[len(b)-256:]
	}
	if len(b) < 256 {
		padded := make([]byte, 256)
		copy(padded[256-len(b):], b)
		return padded
	}
	return b
}

// ComputePasswordHash derives the 2FA password hash using the MTProto SRP
// algorithm: SHA-256(salt2 + PBKDF2-HMAC-SHA512(SHA-256(salt2 + SHA-256(salt1 +
// password + salt1) + salt2), salt1, 100000) + salt2). Returns a 32-byte hash.
//
// This function uses PBKDF2 with 100,000 iterations and is expensive. Callers
// that may invoke it multiple times with the same password and salts (e.g.
// retry scenarios) should cache the result rather than recomputing it.
//
// See https://core.telegram.org/api/srp#checking-the-password-with-srp.
func ComputePasswordHash(password string, salt1, salt2 []byte) []byte {
	pw := []byte(password)
	hash1 := sha256sum(salt1, pw, salt1)
	hash2 := sha256sum(salt2, hash1, salt2)
	hash3 := pbkdf2.Key(hash2, salt1, 100000, 64, sha512.New)
	return sha256sum(salt2, hash3, salt2)
}

// SRPResult holds the output of the Secure Remote Password computation for
// two-factor authentication.
type SRPResult struct {
	SrpID int64
	A     []byte
	M1    []byte
}

// ValidateSRPPrime checks that p is the expected safe 2048-bit SRP prime: both
// p and (p-1)/2 must be prime. Uses 20 rounds of Miller-Rabin, matching the
// hardening applied to the DH-exchange prime by ValidateDHPrime.
//
// Telegram publishes a fixed SRP prime; a server returning anything else is
// either broken or malicious. Rejecting it prevents small-subgroup and
// fake-prime attacks on 2FA. See https://core.telegram.org/api/srp.
func ValidateSRPPrime(p *big.Int) bool {
	if p.BitLen() != 2048 {
		return false
	}
	if !p.ProbablyPrime(20) {
		return false
	}
	// Check that (p-1)/2 is also prime (safe prime / Sophie Germain).
	pMinus1Over2 := new(big.Int).Sub(p, big.NewInt(1))
	pMinus1Over2.Rsh(pMinus1Over2, 1)
	return pMinus1Over2.ProbablyPrime(20)
}

// ComputeSRP performs the client side of the Secure Remote Password protocol
// for two-factor authentication. Parameters salt1 and salt2 are server-provided
// salts, g is the SRP generator, p is the SRP prime, srpB is the server's
// public ephemeral value, srpID identifies the SRP session, and password is the
// user's 2FA password.
//
// p and g are validated before any computation: p must be the expected safe
// 2048-bit prime (see ValidateSRPPrime) and g must be in [2,7]. This is
// defense-in-depth — the same validation the DH-exchange path applies via
// ValidateDHPrime/ValidateGA — and fails closed on attacker-controlled input.
//
// Returns an SRPResult containing the session ID, the client's public ephemeral
// A, and the proof M1. Returns an error if the parameters are invalid or if
// secure random generation fails.
//
// See https://core.telegram.org/api/srp.
func ComputeSRP(salt1, salt2 []byte, g *big.Int, p *big.Int, srpB []byte, srpID int64, password string) (*SRPResult, error) {
	// Reject attacker-controlled p/g before touching them. A non-safe or
	// non-2048-bit p enables small-subgroup / fake-prime attacks that can
	// recover the password or forge the SRP proof; g outside [2,7] is never
	// sent by Telegram and would weaken the exchange (e.g. g=1 collapses g^x).
	if !ValidateSRPPrime(p) {
		return nil, fmt.Errorf("srp: invalid prime")
	}
	if g.Cmp(big.NewInt(2)) < 0 || g.Cmp(big.NewInt(7)) > 0 {
		return nil, fmt.Errorf("srp: invalid generator")
	}

	xBytes := ComputePasswordHash(password, salt1, salt2)
	x := new(big.Int).SetBytes(xBytes)

	gBytes := pad256(g)
	pBytes := p.Bytes()
	k := new(big.Int).SetBytes(sha256sum(pBytes, gBytes))

	gX := new(big.Int).Exp(g, x, p)
	kgX := new(big.Int).Mul(k, gX)
	kgX.Mod(kgX, p)

	B := new(big.Int).SetBytes(srpB)

	// Reject a server B outside (1, p-1): such values (0, 1, p, p-1, multiples of
	// p) force the shared secret to a value the server controls or predicts,
	// breaking the SRP zero-knowledge property. See https://core.telegram.org/api/srp.
	pMinus1 := new(big.Int).Sub(p, big.NewInt(1))
	if B.Cmp(big.NewInt(1)) <= 0 || B.Cmp(pMinus1) >= 0 {
		return nil, fmt.Errorf("srp: server B out of range")
	}

	var a *big.Int
	var A *big.Int
	var ABytes []byte
	var u *big.Int

	for {
		aBytes := make([]byte, 256)
		if _, err := rand.Read(aBytes); err != nil {
			return nil, err
		}
		a = new(big.Int).SetBytes(aBytes)
		A = new(big.Int).Exp(g, a, p)
		ABytes = pad256(A)

		uBytes := sha256sum(ABytes, pad256(B))
		u = new(big.Int).SetBytes(uBytes)
		if u.Sign() > 0 {
			break
		}
	}

	gB := new(big.Int).Sub(B, kgX)
	gB.Mod(gB, p)

	ux := new(big.Int).Mul(u, x)
	ax := new(big.Int).Add(a, ux)
	S := new(big.Int).Exp(gB, ax, p)
	SBytes := pad256(S)

	K := sha256sum(SBytes)

	M1 := sha256sum(
		xorBytes(sha256sum(pBytes), sha256sum(gBytes)),
		sha256sum(salt1),
		sha256sum(salt2),
		ABytes,
		srpB,
		K,
	)

	return &SRPResult{
		SrpID: srpID,
		A:     ABytes,
		M1:    M1,
	}, nil
}
