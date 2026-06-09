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
	p := fromHex("C150023E2F70DB7985DED064759CFECF0AF328E69A41DAF4D6F01B538135A6F9" +
		"1F8F8B2A0EC9BA9720CE352EFCF6C5680FFC424BD634864902DE0B4BD6D49F4E" +
		"580230E3AE97D95C8B19442B3C0A10D8F5633FECEDD6926A7F6DAB0DDB7D457F" +
		"9EA81B8465FCD6FFFEED114011DF91C059CAEDAF97625F6C96ECC74725556934" +
		"EF781D866B34F011FCE4D835A090196E9A5F0E4449AF7EB697DDB9076494CA5F" +
		"81104A305B6DD27665722C46B60E5DF680FB16B210607EF217652E60236C255F" +
		"6A28315F4083A96791D7214BF64C1DF4FD0DB1944FB26A2A57031B32EEE64AD1" +
		"5A8BA68885CDE74A5BFC920F6ABF59BA5C75506373E7130F9042DA922179251F")
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
	p := fromHex("C150023E2F70DB7985DED064759CFECF0AF328E69A41DAF4D6F01B538135A6F9" +
		"1F8F8B2A0EC9BA9720CE352EFCF6C5680FFC424BD634864902DE0B4BD6D49F4E" +
		"580230E3AE97D95C8B19442B3C0A10D8F5633FECEDD6926A7F6DAB0DDB7D457F" +
		"9EA81B8465FCD6FFFEED114011DF91C059CAEDAF97625F6C96ECC74725556934" +
		"EF781D866B34F011FCE4D835A090196E9A5F0E4449AF7EB697DDB9076494CA5F" +
		"81104A305B6DD27665722C46B60E5DF680FB16B210607EF217652E60236C255F" +
		"6A28315F4083A96791D7214BF64C1DF4FD0DB1944FB26A2A57031B32EEE64AD1" +
		"5A8BA68885CDE74A5BFC920F6ABF59BA5C75506373E7130F9042DA922179251F")
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

func fromHex(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("invalid hex")
	}
	return n
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
