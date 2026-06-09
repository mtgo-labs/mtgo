package telegram

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestCDNDecryptChunk(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 32)
	_, _ = rand.Read(key)
	_, _ = rand.Read(iv)

	originalData := make([]byte, 1024)
	_, _ = rand.Read(originalData)

	encrypted, err := crypto.CTREncrypt(originalData, key, iv[:16])
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := cdnDecryptChunk(encrypted, key, iv, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, originalData) {
		t.Error("CDN decryption failed: data mismatch")
	}
}

func TestCDNDecryptChunkWithOffset(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 32)
	_, _ = rand.Read(key)
	_, _ = rand.Read(iv)

	originalData := make([]byte, downloadChunkSize*2)
	_, _ = rand.Read(originalData)

	encrypted, err := crypto.CTREncrypt(originalData, key, iv[:16])
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := cdnDecryptChunk(encrypted[downloadChunkSize:], key, iv, int64(downloadChunkSize))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, originalData[downloadChunkSize:]) {
		t.Error("CDN decryption with offset failed: data mismatch")
	}
}

func TestCDNVerifyHash(t *testing.T) {
	data := make([]byte, 1024)
	_, _ = rand.Read(data)

	hash := sha256.Sum256(data)
	fileHash := &tg.FileHash{
		Offset: 0,
		Limit:  1024,
		Hash:   hash[:],
	}

	if !cdnVerifyHash(data, fileHash, 0) {
		t.Error("hash verification should succeed for correct data")
	}

	wrongData := make([]byte, 1024)
	_, _ = rand.Read(wrongData)
	if cdnVerifyHash(wrongData, fileHash, 0) {
		t.Error("hash verification should fail for wrong data")
	}
}

func TestCDNDecryptChunk_EmptyData(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 32)
	result, err := cdnDecryptChunk(nil, key, iv, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil for empty data, got %d bytes", len(result))
	}
}

func TestCDNDecryptWithCTRIVDerivation(t *testing.T) {
	key := make([]byte, 32)
	baseIV := make([]byte, 16)
	_, _ = rand.Read(key)
	_, _ = rand.Read(baseIV)

	fullIV := make([]byte, 32)
	copy(fullIV, baseIV)
	copy(fullIV[16:], baseIV)

	originalData := make([]byte, 256)
	_, _ = rand.Read(originalData)

	block, _ := aes.NewCipher(key)
	ctrIV := make([]byte, 16)
	copy(ctrIV, baseIV)

	keystream := make([]byte, 16)
	block.Encrypt(keystream, ctrIV)

	encrypted := make([]byte, len(originalData))
	pos := 0
	for i := range originalData {
		encrypted[i] = originalData[i] ^ keystream[pos]
		pos++
		if pos >= 16 {
			pos = 0
			incrementIV(ctrIV)
			block.Encrypt(keystream, ctrIV)
		}
	}

	decrypted, err := cdnDecryptChunk(encrypted, key, fullIV, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, originalData) {
		t.Error("CTR IV derivation mismatch")
	}
}
