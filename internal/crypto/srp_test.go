package crypto

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"
)

func TestComputePasswordHash(t *testing.T) {
	salt1 := []byte("salt1")
	salt2 := []byte("salt2")
	password := "testpassword"

	hash := ComputePasswordHash(password, salt1, salt2)
	if len(hash) != 32 {
		t.Fatalf("expected 32-byte hash, got %d", len(hash))
	}

	hash2 := ComputePasswordHash(password, salt1, salt2)
	if !bytes.Equal(hash, hash2) {
		t.Fatal("hash should be deterministic")
	}

	expected, err := hex.DecodeString("92d7a0f414b43eae3d70a41b838c737ab665cc63c325a9faed3b3fe77f45aa25")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(hash, expected) {
		t.Fatalf("hash mismatch: got %x, want %x", hash, expected)
	}
}

func TestComputePasswordHashDifferentInputs(t *testing.T) {
	salt1 := []byte("salt1")
	salt2 := []byte("salt2")

	h1 := ComputePasswordHash("password1", salt1, salt2)
	h2 := ComputePasswordHash("password2", salt1, salt2)
	if bytes.Equal(h1, h2) {
		t.Fatal("different passwords should produce different hashes")
	}

	h3 := ComputePasswordHash("password1", []byte("other"), salt2)
	if bytes.Equal(h1, h3) {
		t.Fatal("different salt1 should produce different hashes")
	}
}

func TestXorBytes(t *testing.T) {
	a := []byte{0xff, 0x00, 0x0f}
	b := []byte{0x0f, 0x00, 0xf0}
	expected := []byte{0xf0, 0x00, 0xff}
	result := xorBytes(a, b)
	if !bytes.Equal(result, expected) {
		t.Fatalf("xor mismatch: got %x, want %x", result, expected)
	}
}

func TestComputeSRP(t *testing.T) {
	// GetDHPrime is a verified 2048-bit safe prime; the SRP prime has the
	// same structure. The previous inline fixture was not actually prime.
	p := GetDHPrime()
	g := big.NewInt(3)
	salt1 := make([]byte, 32)
	salt2 := make([]byte, 32)
	for i := range salt1 {
		salt1[i] = byte(i + 1)
		salt2[i] = byte(i + 33)
	}

	password := "testpassword123"

	gB := new(big.Int).Exp(g, big.NewInt(12345), p)
	srpB := pad256Big(gB)
	srpID := int64(42)

	result, err := ComputeSRP(salt1, salt2, g, p, srpB, srpID, password)
	if err != nil {
		t.Fatalf("ComputeSRP failed: %v", err)
	}
	if result.SrpID != srpID {
		t.Fatalf("SrpID mismatch: got %d, want %d", result.SrpID, srpID)
	}
	if len(result.A) != 256 {
		t.Fatalf("A should be 256 bytes, got %d", len(result.A))
	}
	if len(result.M1) != 32 {
		t.Fatalf("M1 should be 32 bytes, got %d", len(result.M1))
	}
}

func TestComputeSRPDeterministic(t *testing.T) {
	p := GetDHPrime()
	g := big.NewInt(3)
	salt1 := make([]byte, 32)
	salt2 := make([]byte, 32)

	srpB := pad256Big(new(big.Int).Exp(g, big.NewInt(12345), p))
	result1, err := ComputeSRP(salt1, salt2, g, p, srpB, 1, "password")
	if err != nil {
		t.Fatal(err)
	}
	result2, err := ComputeSRP(salt1, salt2, g, p, srpB, 1, "password")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(result1.A, result2.A) {
		t.Fatal("A should differ between calls (random ephemeral key)")
	}
}

// canonicalSRPPrime is a verified 2048-bit safe prime (the same structure the
// Telegram SRP spec requires). Reused from the DH-exchange prime constant.
var canonicalSRPPrime = GetDHPrime()

func TestComputeSRPRejectsInvalidPrime(t *testing.T) {
	g := big.NewInt(3)
	salt1 := make([]byte, 32)
	salt2 := make([]byte, 32)
	srpB := pad256Big(new(big.Int).Exp(g, big.NewInt(12345), canonicalSRPPrime))

	// A small composite is neither 2048-bit nor prime: must fail closed.
	badP := big.NewInt(15)
	if _, err := ComputeSRP(salt1, salt2, g, badP, srpB, 1, "password"); err == nil {
		t.Fatal("ComputeSRP accepted a small composite prime; expected error")
	}

	// A 2048-bit composite (canonical prime + 2) must also be rejected.
	tampered := new(big.Int).Add(canonicalSRPPrime, big.NewInt(2))
	if _, err := ComputeSRP(salt1, salt2, g, tampered, srpB, 1, "password"); err == nil {
		t.Fatal("ComputeSRP accepted a non-prime 2048-bit value; expected error")
	}
}

func TestComputeSRPRejectsInvalidGenerator(t *testing.T) {
	salt1 := make([]byte, 32)
	salt2 := make([]byte, 32)
	srpB := pad256Big(new(big.Int).Exp(big.NewInt(3), big.NewInt(12345), canonicalSRPPrime))

	for _, badG := range []int64{0, 1, 8, 100} {
		if _, err := ComputeSRP(salt1, salt2, big.NewInt(badG), canonicalSRPPrime, srpB, 1, "password"); err == nil {
			t.Fatalf("ComputeSRP accepted generator g=%d; expected error (must be in [2,7])", badG)
		}
	}
}

func TestValidateSRPPrime(t *testing.T) {
	if !ValidateSRPPrime(canonicalSRPPrime) {
		t.Fatal("ValidateSRPPrime rejected the canonical Telegram SRP prime")
	}
	if ValidateSRPPrime(big.NewInt(15)) {
		t.Fatal("ValidateSRPPrime accepted a small composite")
	}
}

func pad256Big(n *big.Int) []byte {
	b := n.Bytes()
	if len(b) < 256 {
		padded := make([]byte, 256)
		copy(padded[256-len(b):], b)
		return padded
	}
	return b
}
