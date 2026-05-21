package crypto

import (
	"bytes"
	"testing"
)

func TestXOR(t *testing.T) {
	a := []byte{0x01, 0x02, 0x03, 0x04}
	b := []byte{0xFF, 0x00, 0xFF, 0x00}
	want := []byte{0xFE, 0x02, 0xFC, 0x04}
	got := make([]byte, len(a))
	xorInPlace(got, a, b)
	if !bytes.Equal(got, want) {
		t.Fatalf("xor: expected %x, got %x", want, got)
	}
}

func TestXOREmpty(t *testing.T) {
	got := make([]byte, 0)
	xorInPlace(got, nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected empty, got %x", got)
	}
}

func TestIGE256EncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	for i := range iv {
		iv[i] = byte(255 - i)
	}

	data := []byte("Hello, MTProto world!!")
	data = append(data, make([]byte, 10)...)

	encrypted, err := IGEEncrypt(data, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(encrypted, data) {
		t.Fatal("encrypted should differ from plaintext")
	}
	if len(encrypted) != len(data) {
		t.Fatalf("encrypted length %d != data length %d", len(encrypted), len(data))
	}

	decrypted, err := IGEDecrypt(encrypted, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, data) {
		t.Fatalf("round-trip failed:\n  want %x\n  got  %x", data, decrypted)
	}
}

func TestIGEEncryptEmpty(t *testing.T) {
	encrypted, err := IGEEncrypt([]byte{}, make([]byte, 32), make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	if len(encrypted) != 0 {
		t.Fatalf("expected empty, got %d bytes", len(encrypted))
	}
}

func TestIGEDecryptEmpty(t *testing.T) {
	decrypted, err := IGEDecrypt([]byte{}, make([]byte, 32), make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	if len(decrypted) != 0 {
		t.Fatalf("expected empty, got %d bytes", len(decrypted))
	}
}

func TestIGE256KnownVector(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 32)
	data := make([]byte, 16)

	encrypted, err := IGEEncrypt(data, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := IGEDecrypt(encrypted, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, data) {
		t.Fatal("known vector round-trip failed")
	}

	encrypted2, err := IGEEncrypt(data, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encrypted, encrypted2) {
		t.Fatal("encryption should be deterministic")
	}
}

func TestIGEInvalidKeyLength(t *testing.T) {
	_, err := IGEEncrypt(make([]byte, 16), make([]byte, 15), make([]byte, 32))
	if err == nil {
		t.Fatal("expected error for invalid key size")
	}
}

func TestIGEInvalidDataLength(t *testing.T) {
	_, err := IGEEncrypt(make([]byte, 17), make([]byte, 32), make([]byte, 32))
	if err == nil {
		t.Fatal("expected error for non-aligned data")
	}
}

func TestCTREncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 16)
	for i := range key {
		key[i] = byte(i)
	}
	for i := range iv {
		iv[i] = byte(i * 3)
	}
	data := []byte("CTR mode test data for MTProto obfuscated transport")

	encrypted, err := CTREncrypt(data, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(encrypted, data) {
		t.Fatal("encrypted should differ from plaintext")
	}
	if len(encrypted) != len(data) {
		t.Fatalf("CTR should preserve length: %d != %d", len(encrypted), len(data))
	}

	decrypted, err := CTRDecrypt(encrypted, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, data) {
		t.Fatalf("CTR round-trip failed:\n  want %x\n  got  %x", data, decrypted)
	}
}

func TestCTREmpty(t *testing.T) {
	got, err := CTREncrypt([]byte{}, make([]byte, 32), make([]byte, 16))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d bytes", len(got))
	}
}

func TestCTRDeterministic(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 16)
	data := []byte("deterministic test")

	e1, err := CTREncrypt(data, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	e2, err := CTREncrypt(data, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e1, e2) {
		t.Fatal("CTR encrypt should be deterministic with same key/iv")
	}
}

func TestCTRCipherStateful(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 16)
	for i := range key {
		key[i] = byte(i)
	}
	data := []byte("12345678901234567890123456789012")

	allEncrypted, err := CTREncrypt(data, key, iv)
	if err != nil {
		t.Fatal(err)
	}

	enc, err := NewCTRCipher(key, iv)
	if err != nil {
		t.Fatal(err)
	}
	part1 := enc.Process(data[:16])
	part2 := enc.Process(data[16:])

	concat := append(part1, part2...)
	if !bytes.Equal(concat, allEncrypted) {
		t.Fatal("stateful CTRCipher should match single CTREncrypt call")
	}
}

func TestCTRCipherRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 16)
	for i := range key {
		key[i] = byte(i)
	}

	enc, err := NewCTRCipher(key, iv)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("stateful cipher test data!")
	encrypted := enc.Process(data)

	dec, err := NewCTRCipher(key, iv)
	if err != nil {
		t.Fatal(err)
	}
	decrypted := dec.Process(encrypted)

	if !bytes.Equal(decrypted, data) {
		t.Fatalf("CTRCipher round-trip failed:\n  want %x\n  got  %x", data, decrypted)
	}
}
