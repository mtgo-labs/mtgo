package tdesktop

import (
	"bytes"
	"crypto/aes"
	"crypto/sha1" // #nosec G505
	"encoding/binary"
	"fmt"
	"io/fs"
	"testing"
)

func TestCreateLocalKeyNoPasscode(t *testing.T) {
	salt := make([]byte, saltSize)
	for i := range salt {
		salt[i] = byte(i)
	}
	key := createLocalKey(nil, salt)
	if key == [256]byte{} {
		t.Fatal("key should not be zero")
	}
}

func TestCreateLocalKeyWithPasscode(t *testing.T) {
	salt := make([]byte, saltSize)
	for i := range salt {
		salt[i] = byte(i)
	}
	key := createLocalKey([]byte("mypassword"), salt)
	key2 := createLocalKey([]byte("wrongpassword"), salt)
	if key == key2 {
		t.Fatal("different passcodes should produce different keys")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	var localKey [256]byte
	for i := range localKey {
		localKey[i] = byte(i)
	}

	plaintext := make([]byte, 64)
	for i := range plaintext {
		plaintext[i] = byte(i * 3)
	}

	// Encrypt: compute msg_key, then IGE encrypt.
	hash := sha1.Sum(plaintext)
	var msgKey [16]byte
	copy(msgKey[:], hash[:])

	aesKey, aesIV := deriveAESKeyIV(localKey, msgKey, 8)
	encrypted, err := igeEncrypt(aesKey, aesIV, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Prepend msg_key to form the full encrypted blob.
	full := make([]byte, 0, 16+len(encrypted))
	full = append(full, msgKey[:]...)
	full = append(full, encrypted...)

	// Decrypt using decryptLocal.
	decrypted, err := decryptLocal(full, localKey)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("round-trip mismatch:\n  got:  %x\n  want: %x", decrypted, plaintext)
	}
}

func TestDecryptLocalWrongKey(t *testing.T) {
	var localKey [256]byte
	for i := range localKey {
		localKey[i] = byte(i)
	}
	plaintext := make([]byte, 32)
	for i := range plaintext {
		plaintext[i] = byte(i)
	}

	hash := sha1.Sum(plaintext)
	var msgKey [16]byte
	copy(msgKey[:], hash[:])

	aesKey, aesIV := deriveAESKeyIV(localKey, msgKey, 8)
	encrypted, err := igeEncrypt(aesKey, aesIV, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	full := append(msgKey[:], encrypted...)

	// Try with wrong key.
	var wrongKey [256]byte
	for i := range wrongKey {
		wrongKey[i] = byte(255 - i)
	}
	_, err = decryptLocal(full, wrongKey)
	if err == nil {
		t.Fatal("should fail with wrong key")
	}
}

func TestComputeFileHash(t *testing.T) {
	data := []byte("hello")
	h := computeFileHash(data, 1)
	if h == [16]byte{} {
		t.Fatal("hash should not be zero")
	}
	// Same input should produce same hash.
	h2 := computeFileHash(data, 1)
	if h != h2 {
		t.Fatal("hashes should match")
	}
}

func TestFileKeyDeterministic(t *testing.T) {
	k1 := fileKey("data")
	k2 := fileKey("data")
	if k1 != k2 {
		t.Fatal("file keys should be deterministic")
	}
	if len(k1) != 16 {
		t.Fatalf("file key length = %d, want 16", len(k1))
	}
	// Different inputs should produce different keys.
	k3 := fileKey("data#2")
	if k1 == k3 {
		t.Fatal("different inputs should produce different keys")
	}
}

func TestReadArray(t *testing.T) {
	// Build a tdesktop file with one array.
	buf := new(bytes.Buffer)
	data := []byte("payload")
	binary.Write(buf, binary.BigEndian, uint32(len(data)))
	buf.Write(data)

	tdf := &tdesktopFile{data: buf.Bytes()}

	arr, err := tdf.readArray()
	if err != nil {
		t.Fatalf("readArray: %v", err)
	}
	if string(arr) != "payload" {
		t.Fatalf("got %q, want %q", arr, "payload")
	}
}

func TestReadArrayNull(t *testing.T) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(0xFFFFFFFF))
	tdf := &tdesktopFile{data: buf.Bytes()}

	arr, err := tdf.readArray()
	if err != nil {
		t.Fatalf("readArray: %v", err)
	}
	if arr != nil {
		t.Fatalf("expected nil for 0xFFFFFFFF, got %x", arr)
	}
}

func TestParseMTPAuthorization(t *testing.T) {
	// Build a minimal MTPAuthorization with wide IDs tag and one key.
	buf := new(bytes.Buffer)

	// dbiMtpAuth marker
	binary.Write(buf, binary.BigEndian, uint32(dbiMtpAuth))
	// mainLength (skip)
	binary.Write(buf, binary.BigEndian, uint32(0))

	// legacyUserID + legacyMainDC that form wideIDsTag (0xFFFFFFFFFFFFFFFF)
	binary.Write(buf, binary.BigEndian, uint32(0xFFFFFFFF))
	binary.Write(buf, binary.BigEndian, uint32(0xFFFFFFFF))

	// userID (wide)
	binary.Write(buf, binary.BigEndian, uint64(12345678))
	// mainDC
	binary.Write(buf, binary.BigEndian, uint32(2))

	// key count = 1
	binary.Write(buf, binary.BigEndian, uint32(1))
	// dc id
	binary.Write(buf, binary.BigEndian, uint32(2))
	// auth key (256 bytes)
	var key [256]byte
	for i := range key {
		key[i] = byte(i)
	}
	buf.Write(key[:])

	auth, err := parseMTPAuthorization(buf.Bytes())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if auth.UserID != 12345678 {
		t.Fatalf("UserID = %d, want 12345678", auth.UserID)
	}
	if auth.MainDC != 2 {
		t.Fatalf("MainDC = %d, want 2", auth.MainDC)
	}
	if len(auth.Keys) != 1 {
		t.Fatalf("Keys count = %d, want 1", len(auth.Keys))
	}
	if auth.Keys[2] != key {
		t.Fatal("auth key mismatch")
	}
}

func TestParseMTPAuthorizationLegacy(t *testing.T) {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, uint32(dbiMtpAuth))
	binary.Write(buf, binary.BigEndian, uint32(0))

	// legacy format: userID and mainDC directly
	binary.Write(buf, binary.BigEndian, uint32(42))
	binary.Write(buf, binary.BigEndian, uint32(4))

	// key count = 0
	binary.Write(buf, binary.BigEndian, uint32(0))

	auth, err := parseMTPAuthorization(buf.Bytes())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if auth.UserID != 42 {
		t.Fatalf("UserID = %d, want 42", auth.UserID)
	}
	if auth.MainDC != 4 {
		t.Fatalf("MainDC = %d, want 4", auth.MainDC)
	}
}

func TestOpenFileNotFound(t *testing.T) {
	memFS := &memoryFS{files: map[string][]byte{}}
	_, err := openFile(memFS, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFromReaderWrongMagic(t *testing.T) {
	data := []byte("XXXX")
	data = append(data, 0, 0, 0, 0)          // version
	data = append(data, make([]byte, 16)...) // fake checksum
	_, err := fromReader(bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for wrong magic")
	}
}

func TestErrNoAccounts(t *testing.T) {
	if ErrNoAccounts.Error() == "" {
		t.Fatal("ErrNoAccounts should have a message")
	}
}

func TestErrKeyDecrypt(t *testing.T) {
	if ErrKeyDecrypt.Error() == "" {
		t.Fatal("ErrKeyDecrypt should have a message")
	}
}

// ---------- IGE encrypt helper (for tests only) ----------

func igeEncrypt(key [32]byte, iv [32]byte, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	bs := aes.BlockSize
	ciphertext := make([]byte, len(data))

	xPrev := iv[:bs]
	yPrev := iv[bs:]

	for i := 0; i < len(data); i += bs {
		ptBlock := data[i : i+bs]
		var temp []byte
		for j := 0; j < bs; j++ {
			temp = append(temp, ptBlock[j]^xPrev[j])
		}
		enc := make([]byte, bs)
		block.Encrypt(enc, temp)
		for j := 0; j < bs; j++ {
			ciphertext[i+j] = enc[j] ^ yPrev[j]
		}
		xPrev = ciphertext[i : i+bs]
		yPrev = ptBlock
	}

	return ciphertext, nil
}

// ---------- minimal in-memory fs for tests ----------

type memoryFS struct {
	files map[string][]byte
}

func (m *memoryFS) Open(name string) (fs.File, error) {
	data, ok := m.files[name]
	if !ok {
		return nil, fmt.Errorf("file does not exist: %s", name)
	}
	return &memoryFile{data: data}, nil
}

type memoryFile struct {
	data   []byte
	offset int
}

func (m *memoryFile) Read(p []byte) (int, error) {
	if m.offset >= len(m.data) {
		return 0, fs.ErrNotExist // EOF
	}
	n := copy(p, m.data[m.offset:])
	m.offset += n
	return n, nil
}

func (m *memoryFile) Stat() (fs.FileInfo, error) { return nil, fmt.Errorf("not implemented") }
func (m *memoryFile) Close() error               { return nil }
