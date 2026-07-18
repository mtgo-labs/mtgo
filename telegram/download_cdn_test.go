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

// TestCDNAddToIVLE confirms the O(1) counter advance matches repeated
// incrementIV calls (the O(N) path it replaces) across several magnitudes,
// including values that carry across byte boundaries.
func TestCDNAddToIVLE(t *testing.T) {
	cases := []int64{1, 2, 255, 256, 65536, 1 << 20, (1 << 20) + 257}
	for _, n := range cases {
		base := make([]byte, 16)
		_, _ = rand.Read(base)

		want := make([]byte, 16)
		copy(want, base)
		for i := int64(0); i < n; i++ {
			incrementIV(want)
		}

		got := make([]byte, 16)
		copy(got, base)
		addToIVLE(got, n)

		if !bytes.Equal(got, want) {
			t.Errorf("addToIVLE(_, %d) = %x, want %x (base %x)", n, got, want, base)
		}
	}
}

// TestCDNHashCheckerSpanning verifies that a hash straddling a chunk boundary
// is verified correctly using bytes from both chunks (the regression fixed in
// M1: the old code hashed the wrong window and falsely rejected valid data).
func TestCDNHashCheckerSpanning(t *testing.T) {
	full := make([]byte, 1024)
	_, _ = rand.Read(full)

	const chunkSize = 256
	// A hash that starts in chunk 1 (offset 240) and ends in chunk 2 (offset 272).
	spanning := full[240:272]
	h := sha256.Sum256(spanning)
	hashes := []*tg.FileHash{{Offset: 240, Limit: 32, Hash: h[:]}}

	c := &cdnHashChecker{hashes: hashes}
	if err := c.feed(full[0:chunkSize], 0); err != nil {
		t.Fatalf("feed chunk 0: %v", err)
	}
	if err := c.feed(full[chunkSize:2*chunkSize], chunkSize); err != nil {
		t.Fatalf("feed chunk 1 (hash completes here): %v", err)
	}
	if c.idx != len(hashes) {
		t.Errorf("after feed: idx = %d, want %d", c.idx, len(hashes))
	}

	// A tampered spanning region must be rejected.
	tampered := append([]byte(nil), full...)
	tampered[260] ^= 0xff
	bad := &cdnHashChecker{hashes: hashes}
	if err := bad.feed(tampered[0:chunkSize], 0); err != nil {
		t.Fatalf("feed chunk 0 (tampered): unexpected error before hash completes: %v", err)
	}
	if err := bad.feed(tampered[chunkSize:2*chunkSize], chunkSize); err == nil {
		t.Fatal("tampered spanning hash should fail verification")
	}
}

func TestCDNHashCheckerCoverage(t *testing.T) {
	hashA := sha256.Sum256([]byte("a"))
	hashB := sha256.Sum256([]byte("b"))
	checker := &cdnHashChecker{hashes: []*tg.FileHash{
		{Offset: 1, Limit: 1, Hash: hashB[:]},
		{Offset: 0, Limit: 1, Hash: hashA[:]},
	}}

	end, err := checker.ensureCoverage(t.Context(), nil, nil, 0)
	if err != nil {
		t.Fatalf("ensure coverage: %v", err)
	}
	if end != 2 {
		t.Fatalf("coverage end = %d, want 2", end)
	}
}

func TestCDNHashCheckerRejectsInvalidCoverage(t *testing.T) {
	validHash := sha256.Sum256([]byte("a"))
	tests := []struct {
		name   string
		hashes []*tg.FileHash
	}{
		{name: "nil descriptor", hashes: []*tg.FileHash{nil}},
		{name: "wrong hash size", hashes: []*tg.FileHash{{Offset: 0, Limit: 1, Hash: []byte{1}}}},
		{name: "overlap", hashes: []*tg.FileHash{
			{Offset: 0, Limit: 2, Hash: validHash[:]},
			{Offset: 1, Limit: 1, Hash: validHash[:]},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &cdnHashChecker{hashes: tt.hashes}
			if _, err := checker.ensureCoverage(t.Context(), nil, nil, 0); err == nil {
				t.Fatal("expected invalid coverage error")
			}
		})
	}
}

func TestCDNHashCheckerFinishRejectsPartialHash(t *testing.T) {
	data := []byte("authenticated")
	hash := sha256.Sum256(data)
	checker := &cdnHashChecker{hashes: []*tg.FileHash{{Offset: 0, Limit: int32(len(data)), Hash: hash[:]}}}

	if err := checker.feed(data[:4], 0); err != nil {
		t.Fatalf("feed: %v", err)
	}
	if err := checker.finish(); err == nil {
		t.Fatal("expected incomplete authenticated range error")
	}
}
