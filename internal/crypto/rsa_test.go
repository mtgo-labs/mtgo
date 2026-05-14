package crypto

import (
	"bytes"
	"testing"
)

func TestRSAEncrypt(t *testing.T) {
	data := make([]byte, 64)
	for i := range data {
		data[i] = byte(i)
	}

	fp := int64(-4344800451088585951)
	enc, err := RSAEncrypt(data, fp)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) != 256 {
		t.Fatalf("expected 256 bytes, got %d", len(enc))
	}
}

func TestRSAEncryptUnknownFingerprint(t *testing.T) {
	_, err := RSAEncrypt(make([]byte, 64), 9999999999)
	if err == nil {
		t.Fatal("expected error for unknown fingerprint")
	}
}

func TestRSAEncryptDataTooLong(t *testing.T) {
	data := make([]byte, 145)
	_, err := RSAEncrypt(data, -4344800451088585951)
	if err == nil {
		t.Fatal("expected error for data exceeding 144 bytes")
	}
}

func TestRSAEncryptDifferentKeys(t *testing.T) {
	data := make([]byte, 64)
	for i := range data {
		data[i] = byte(i)
	}

	enc1, err1 := RSAEncrypt(data, -4344800451088585951)
	enc2, err2 := RSAEncrypt(data, 847625836280919973)

	if err1 != nil || err2 != nil {
		t.Fatalf("encrypt errors: %v, %v", err1, err2)
	}
	if bytes.Equal(enc1, enc2) {
		t.Fatal("different keys should produce different output")
	}
}

func TestRSAEncryptRandomness(t *testing.T) {
	data := make([]byte, 64)
	for i := range data {
		data[i] = byte(i)
	}

	fp := int64(-4344800451088585951)
	enc1, _ := RSAEncrypt(data, fp)
	enc2, _ := RSAEncrypt(data, fp)

	if bytes.Equal(enc1, enc2) {
		t.Fatal("RSAPad should produce different output each time due to random temp_key")
	}
}

func TestRSAEncryptLegacy(t *testing.T) {
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}

	enc, err := RSAEncryptLegacy(data, 847625836280919973)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) != 256 {
		t.Fatalf("expected 256 bytes, got %d", len(enc))
	}
}

func TestRSAEncryptLegacyUnknownFingerprint(t *testing.T) {
	_, err := RSAEncryptLegacy(make([]byte, 100), 9999999999)
	if err == nil {
		t.Fatal("expected error for unknown fingerprint")
	}
}

func TestReverseBytes(t *testing.T) {
	b := []byte{1, 2, 3, 4, 5}
	reverseBytes(b)
	if !bytes.Equal(b, []byte{5, 4, 3, 2, 1}) {
		t.Fatalf("reverseBytes = %v, want [5 4 3 2 1]", b)
	}
}
