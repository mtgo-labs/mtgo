package crypto

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"math/big"
	"testing"
)

func makeTestKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, SecretChatKeyLen)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

func TestGenerateDHSecret(t *testing.T) {
	secret, err := GenerateDHSecret(CurrentDHPrime, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if secret == nil {
		t.Fatal("secret is nil")
	}
	if secret.Sign() <= 0 {
		t.Fatal("secret must be positive")
	}
	primeMinus1 := new(big.Int).Sub(CurrentDHPrime, big.NewInt(1))
	if secret.Cmp(primeMinus1) >= 0 {
		t.Fatal("secret must be less than dhPrime-1")
	}
	if secret.Cmp(big.NewInt(1)) <= 0 {
		t.Fatal("secret must be greater than 1")
	}
}

func TestGenerateDHSecretUniqueness(t *testing.T) {
	s1, err := GenerateDHSecret(CurrentDHPrime, 2048)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := GenerateDHSecret(CurrentDHPrime, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Cmp(s2) == 0 {
		t.Fatal("two generated secrets should not be equal")
	}
}

func TestComputeGA(t *testing.T) {
	a, _ := GenerateDHSecret(CurrentDHPrime, 2048)
	ga := ComputeGA(2, a, CurrentDHPrime)
	if ga == nil {
		t.Fatal("ga is nil")
	}
	if ga.Sign() <= 0 {
		t.Fatal("ga must be positive")
	}
	if ga.Cmp(CurrentDHPrime) >= 0 {
		t.Fatal("ga must be less than dhPrime")
	}
}

func TestComputeGADeterministic(t *testing.T) {
	a := big.NewInt(12345)
	ga1 := ComputeGA(2, a, CurrentDHPrime)
	ga2 := ComputeGA(2, a, CurrentDHPrime)
	if ga1.Cmp(ga2) != 0 {
		t.Fatal("ComputeGA should be deterministic")
	}
}

func TestComputeGAKnownValue(t *testing.T) {
	smallPrime := big.NewInt(23)
	a := big.NewInt(6)
	ga := ComputeGA(5, a, smallPrime)
	want := big.NewInt(8)
	if ga.Cmp(want) != 0 {
		t.Fatalf("ComputeGA(5, 6, 23): got %d, want %d", ga, want)
	}
}

func TestDHKeyExchange(t *testing.T) {
	a, err := GenerateDHSecret(CurrentDHPrime, 2048)
	if err != nil {
		t.Fatal(err)
	}
	b, err := GenerateDHSecret(CurrentDHPrime, 2048)
	if err != nil {
		t.Fatal(err)
	}

	ga := ComputeGA(2, a, CurrentDHPrime)
	gb := ComputeGA(2, b, CurrentDHPrime)

	keyAB := ComputeSharedKey(ga, b, CurrentDHPrime)
	keyBA := ComputeSharedKey(gb, a, CurrentDHPrime)

	if len(keyAB) != SecretChatKeyLen {
		t.Fatalf("keyAB length: got %d, want %d", len(keyAB), SecretChatKeyLen)
	}
	if len(keyBA) != SecretChatKeyLen {
		t.Fatalf("keyBA length: got %d, want %d", len(keyBA), SecretChatKeyLen)
	}
	if !bytes.Equal(keyAB, keyBA) {
		t.Fatal("shared keys must match: Alice and Bob should derive the same key")
	}
}

func TestDHKeyExchangeSmallPrime(t *testing.T) {
	p := big.NewInt(23)
	g := int32(5)
	a := big.NewInt(6)
	b := big.NewInt(15)

	ga := ComputeGA(g, a, p)
	gb := ComputeGA(g, b, p)

	keyAB := ComputeSharedKey(ga, b, p)
	keyBA := ComputeSharedKey(gb, a, p)

	if !bytes.Equal(keyAB, keyBA) {
		t.Fatalf("small prime DH exchange mismatch:\n  AB: %x\n  BA: %x", keyAB, keyBA)
	}
}

func TestKeyFingerprint(t *testing.T) {
	key := make([]byte, 256)
	for i := range key {
		key[i] = byte(i)
	}
	fp1 := KeyFingerprint(key)
	fp2 := KeyFingerprint(key)
	if fp1 != fp2 {
		t.Fatal("KeyFingerprint must be deterministic")
	}

	differentKey := make([]byte, 256)
	copy(differentKey, key)
	differentKey[0] ^= 0xFF
	fp3 := KeyFingerprint(differentKey)
	if fp1 == fp3 {
		t.Fatal("different keys should produce different fingerprints (with high probability)")
	}
}

func TestKeyFingerprintManual(t *testing.T) {
	key := []byte("test key for fingerprint")
	h := sha1.Sum(key)
	want := int64(binary.LittleEndian.Uint64(h[12:20]))
	got := KeyFingerprint(key)
	if got != want {
		t.Fatalf("KeyFingerprint: got %d, want %d", got, want)
	}
}

func TestKeyVisualization(t *testing.T) {
	key := make([]byte, 256)
	for i := range key {
		key[i] = byte(i)
	}
	emojis := KeyVisualization(key)
	if len(emojis) != 4 {
		t.Fatalf("KeyVisualization: got %d emojis, want 4", len(emojis))
	}
	for i, e := range emojis {
		if len(e) == 0 {
			t.Fatalf("emoji[%d] is empty", i)
		}
	}
}

func TestKeyVisualizationDeterministic(t *testing.T) {
	key := make([]byte, 256)
	rand.Read(key)
	v1 := KeyVisualization(key)
	v2 := KeyVisualization(key)
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("KeyVisualization not deterministic at index %d: %q != %q", i, v1[i], v2[i])
		}
	}
}

func TestValidateGA(t *testing.T) {
	tests := []struct {
		name string
		ga   *big.Int
		want bool
	}{
		{"zero", big.NewInt(0), false},
		{"one", big.NewInt(1), false},
		{"negative", big.NewInt(-5), false},
		{"two", big.NewInt(2), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateGA(tt.ga, CurrentDHPrime); got != tt.want {
				t.Fatalf("ValidateGA(%s): got %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestValidateGAPrimeMinusOne(t *testing.T) {
	primeMinus1 := new(big.Int).Sub(CurrentDHPrime, big.NewInt(1))
	if ValidateGA(primeMinus1, CurrentDHPrime) {
		t.Fatal("ValidateGA should reject ga == p-1")
	}
}

func TestValidateGAValid(t *testing.T) {
	lowerBound := new(big.Int).Sub(
		CurrentDHPrime,
		new(big.Int).Lsh(big.NewInt(1), uint(SecretChatMinGA)),
	)
	validGA := new(big.Int).Sub(CurrentDHPrime, big.NewInt(100))
	if !ValidateGA(validGA, CurrentDHPrime) {
		t.Fatal("ValidateGA should accept ga near p (within lower bound)")
	}
	if validGA.Cmp(lowerBound) < 0 {
		t.Fatal("test ga should be above lower bound")
	}
}

func TestValidateGABelowLowerBound(t *testing.T) {
	lowerBound := new(big.Int).Sub(
		CurrentDHPrime,
		new(big.Int).Lsh(big.NewInt(1), uint(SecretChatMinGA)),
	)
	midRange := new(big.Int).Rsh(CurrentDHPrime, 1)
	if ValidateGA(midRange, CurrentDHPrime) {
		t.Fatal("ValidateGA should reject ga in mid range (below lower bound)")
	}
	_ = lowerBound
}

func TestSecretEncryptDecryptRoundTripOutgoing(t *testing.T) {
	key := makeTestKey(t)
	plaintext := []byte("hello world!!")

	encrypted, err := SecretEncrypt(plaintext, key, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(encrypted) <= len(plaintext) {
		t.Fatal("encrypted should be larger than plaintext due to msgKey + padding")
	}
	if len(encrypted)%16 != 0 {
		t.Fatalf("encrypted must be 16-aligned, got length %d", len(encrypted))
	}

	decrypted, err := SecretDecrypt(encrypted, key, true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestSecretEncryptDecryptRoundTripIncoming(t *testing.T) {
	key := makeTestKey(t)
	plaintext := []byte("incoming msg!")

	encrypted, err := SecretEncrypt(plaintext, key, false)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := SecretDecrypt(encrypted, key, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestSecretEncryptEmptyPlaintext(t *testing.T) {
	key := makeTestKey(t)
	_, err := SecretEncrypt([]byte{}, key, true)
	if err == nil {
		t.Fatal("expected error for empty plaintext")
	}
}

func TestSecretDecryptTooShort(t *testing.T) {
	key := makeTestKey(t)
	tests := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"15 bytes", make([]byte, 15)},
		{"16 bytes", make([]byte, 16)},
		{"31 bytes", make([]byte, 31)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SecretDecrypt(tt.data, key, true)
			if err == nil {
				t.Fatal("expected error for too-short ciphertext")
			}
		})
	}
}

func TestSecretDecryptTampered(t *testing.T) {
	key := makeTestKey(t)
	plaintext := []byte("tamper test!!")

	encrypted, err := SecretEncrypt(plaintext, key, true)
	if err != nil {
		t.Fatal(err)
	}

	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[0] ^= 0xFF

	_, err = SecretDecrypt(tampered, key, true)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestSecretDecryptWrongDirection(t *testing.T) {
	key := makeTestKey(t)
	plaintext := []byte("direction test")

	encrypted, err := SecretEncrypt(plaintext, key, true)
	if err != nil {
		t.Fatal(err)
	}

	_, err = SecretDecrypt(encrypted, key, false)
	if err == nil {
		t.Fatal("expected error when decrypting outgoing message as incoming")
	}
}

func TestSecretEncryptLargePlaintext(t *testing.T) {
	key := makeTestKey(t)
	plaintext := make([]byte, 4096)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	encrypted, err := SecretEncrypt(plaintext, key, true)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := SecretDecrypt(encrypted, key, true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("large plaintext round-trip failed")
	}
}

func TestSecretDecryptPaddingValidation(t *testing.T) {
	key := makeTestKey(t)
	_, err := SecretDecrypt(make([]byte, 32), key, true)
	if err == nil {
		t.Fatal("expected error for random 32-byte ciphertext")
	}
}

func TestSecretEncryptMultipleSizes(t *testing.T) {
	key := makeTestKey(t)
	sizes := []int{1, 12, 28, 44, 60, 100, 256, 1024}
	for _, size := range sizes {
		t.Run("", func(t *testing.T) {
			plaintext := make([]byte, size)
			for i := range plaintext {
				plaintext[i] = byte(i)
			}
			encrypted, err := SecretEncrypt(plaintext, key, true)
			if err != nil {
				t.Fatalf("encrypt size %d: %v", size, err)
			}
			decrypted, err := SecretDecrypt(encrypted, key, true)
			if err != nil {
				t.Fatalf("decrypt size %d: %v", size, err)
			}
			if !bytes.Equal(decrypted, plaintext) {
				t.Fatalf("round-trip size %d failed", size)
			}
		})
	}
}

func TestEncryptDecryptFileRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 32)
	rand.Read(key)
	rand.Read(iv)

	data := []byte("file encryption test data")

	encrypted, err := EncryptFile(data, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if len(encrypted)%16 != 0 {
		t.Fatal("encrypted file must be aligned to 16 bytes")
	}
	if len(encrypted) <= len(data) {
		t.Fatal("encrypted should be >= plaintext due to padding")
	}

	decrypted, err := DecryptFile(encrypted, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(decrypted, data) {
		t.Fatalf("file round-trip failed: decrypted doesn't start with original data")
	}
	if len(decrypted) < len(data) {
		t.Fatalf("decrypted too short: %d < %d", len(decrypted), len(data))
	}
}

func TestEncryptDecryptFileExactBlocks(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 32)
	rand.Read(key)
	rand.Read(iv)

	data := make([]byte, 32)
	copy(data, "exactly two blocks!!")

	encrypted, err := EncryptFile(data, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if len(encrypted) != 48 {
		t.Fatalf("32 bytes padded to 48, got %d", len(encrypted))
	}

	decrypted, err := DecryptFile(encrypted, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(decrypted, data) {
		t.Fatalf("exact block round-trip failed: got %x", decrypted[:32])
	}
}

func TestDecryptFileNotAligned(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 32)
	_, err := DecryptFile([]byte{1, 2, 3, 4, 5}, key, iv)
	if err == nil {
		t.Fatal("expected error for non-aligned ciphertext")
	}
}

func TestEncryptDecryptFileEmpty(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 32)

	encrypted, err := EncryptFile([]byte{}, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := DecryptFile(encrypted, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	if len(decrypted) != 16 {
		t.Fatalf("empty file should pad to one block (16 bytes), got %d", len(decrypted))
	}
}

func TestFileKeyFingerprint(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 32)
	rand.Read(key)
	rand.Read(iv)

	fp1 := FileKeyFingerprint(key, iv)
	fp2 := FileKeyFingerprint(key, iv)
	if fp1 != fp2 {
		t.Fatal("FileKeyFingerprint must be deterministic")
	}
}

func TestFileKeyFingerprintManual(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	iv := []byte("abcdefghijklmnopqrstuvwxyz123456")
	h := md5.Sum(append(key, iv...))
	fp := uint32(h[0])<<24 | uint32(h[1])<<16 | uint32(h[2])<<8 | uint32(h[3])
	fp ^= uint32(h[4])<<24 | uint32(h[5])<<16 | uint32(h[6])<<8 | uint32(h[7])
	want := int32(fp)
	got := FileKeyFingerprint(key, iv)
	if got != want {
		t.Fatalf("FileKeyFingerprint: got %d, want %d", got, want)
	}
}

func TestFileKeyFingerprintDifferent(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	iv := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)
	rand.Read(iv)

	fp1 := FileKeyFingerprint(key1, iv)
	fp2 := FileKeyFingerprint(key2, iv)
	if fp1 == fp2 {
		t.Fatal("different keys should produce different fingerprints (with high probability)")
	}
}

func TestGenerateFileKeyIV(t *testing.T) {
	key, iv, err := GenerateFileKeyIV()
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 32 {
		t.Fatalf("key length: got %d, want 32", len(key))
	}
	if len(iv) != 32 {
		t.Fatalf("iv length: got %d, want 32", len(iv))
	}
}

func TestGenerateFileKeyIVUniqueness(t *testing.T) {
	k1, iv1, err := GenerateFileKeyIV()
	if err != nil {
		t.Fatal(err)
	}
	k2, iv2, err := GenerateFileKeyIV()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(k1, k2) {
		t.Fatal("two generated keys should not be equal")
	}
	if bytes.Equal(iv1, iv2) {
		t.Fatal("two generated IVs should not be equal")
	}
}

func TestPadFile(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"1 byte", make([]byte, 1)},
		{"15 bytes", make([]byte, 15)},
		{"16 bytes", make([]byte, 16)},
		{"17 bytes", make([]byte, 17)},
		{"31 bytes", make([]byte, 31)},
		{"32 bytes", make([]byte, 32)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			padded := padFile(tt.data)
			if len(padded)%16 != 0 {
				t.Fatalf("padded length %d not aligned to 16", len(padded))
			}
			if len(padded) < len(tt.data) {
				t.Fatalf("padded length %d < original %d", len(padded), len(tt.data))
			}
			if len(tt.data) > 0 && !bytes.Equal(padded[:len(tt.data)], tt.data) {
				t.Fatal("padded prefix must match original data")
			}
		})
	}
}

func TestPadFileExactBlockAddsBlock(t *testing.T) {
	data := make([]byte, 16)
	padded := padFile(data)
	if len(padded) != 32 {
		t.Fatalf("exact-block data should add one block: got %d, want 32", len(padded))
	}
}

func TestPadFilePreservesData(t *testing.T) {
	data := []byte("hello world test")
	padded := padFile(data)
	if !bytes.Equal(padded[:len(data)], data) {
		t.Fatal("padFile must preserve original data")
	}
}

func TestSecretKDF(t *testing.T) {
	key := make([]byte, 256)
	for i := range key {
		key[i] = byte(i)
	}
	msgKey := make([]byte, 16)
	for i := range msgKey {
		msgKey[i] = byte(i + 100)
	}

	aesKey1, aesIV1 := secretKDF(key, msgKey, 0)
	aesKey2, aesIV2 := secretKDF(key, msgKey, 0)

	if !bytes.Equal(aesKey1[:], aesKey2[:]) {
		t.Fatal("secretKDF must be deterministic for same inputs")
	}
	if !bytes.Equal(aesIV1[:], aesIV2[:]) {
		t.Fatal("secretKDF must be deterministic for same inputs")
	}

	aesKey3, aesIV3 := secretKDF(key, msgKey, 8)
	if bytes.Equal(aesKey1[:], aesKey3[:]) {
		t.Fatal("different x values should produce different keys")
	}
	if bytes.Equal(aesIV1[:], aesIV3[:]) {
		t.Fatal("different x values should produce different IVs")
	}
}

func TestComputeSharedKeyPadding(t *testing.T) {
	padded := ComputeSharedKey(big.NewInt(1), big.NewInt(12345), CurrentDHPrime)
	if len(padded) != SecretChatKeyLen {
		t.Fatalf("padded key length: got %d, want %d", len(padded), SecretChatKeyLen)
	}
	if padded[SecretChatKeyLen-1] != 1 {
		t.Fatalf("last byte should be 1 (1^x mod p = 1), got %d", padded[SecretChatKeyLen-1])
	}
	for i, b := range padded[:SecretChatKeyLen-1] {
		if b != 0 {
			t.Fatalf("byte %d should be 0 (zero-padded), got %d", i, b)
		}
	}
}

func TestComputeSharedKeyLength256(t *testing.T) {
	a, _ := GenerateDHSecret(CurrentDHPrime, 2048)
	ga := ComputeGA(2, a, CurrentDHPrime)
	key := ComputeSharedKey(ga, a, CurrentDHPrime)
	if len(key) != SecretChatKeyLen {
		t.Fatalf("shared key length: got %d, want %d", len(key), SecretChatKeyLen)
	}
}
