package telegram

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestVerifyDownloadChunkHash_CorrectData(t *testing.T) {
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i)
	}

	hash := sha256.Sum256(data[:4096])

	mock := &hashVerifyInvoker{
		data:      data,
		chunkSize: 4096,
		hashes: []*tg.FileHash{
			{Offset: 0, Limit: 4096, Hash: hash[:]},
		},
		supportHash: true,
	}
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	opts := &params.Download{VerifyHashes: true}
	written, _, err := downloadToFileRPC(context.Background(), rpc, &tg.InputDocumentFileLocation{
		ID: 1, AccessHash: 2,
	}, int64(len(data)), &buf, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written != int64(len(data)) {
		t.Errorf("written = %d, want %d", written, int64(len(data)))
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Error("data mismatch")
	}
}

func TestVerifyDownloadChunkHash_TamperedData(t *testing.T) {
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i)
	}

	// Hash of the original data — serve tampered data so verification fails.
	hash := sha256.Sum256(data[:4096])
	tampered := make([]byte, 4096)
	copy(tampered, data)
	tampered[100] ^= 0xff

	mock := &hashVerifyInvoker{
		data:      tampered,
		chunkSize: 4096,
		hashes: []*tg.FileHash{
			{Offset: 0, Limit: 4096, Hash: hash[:]},
		},
		supportHash: true,
	}
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	opts := &params.Download{VerifyHashes: true}
	_, _, err := downloadToFileRPC(context.Background(), rpc, &tg.InputDocumentFileLocation{
		ID: 1, AccessHash: 2,
	}, int64(len(data)), &buf, opts)
	if err == nil {
		t.Fatal("expected hash verification error")
	}
}

func TestVerifyDownloadChunkHash_NotSupported(t *testing.T) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}

	mock := &hashVerifyInvoker{
		data:        data,
		chunkSize:   4096,
		supportHash: false, // server returns error for getFileHashes
	}
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	opts := &params.Download{VerifyHashes: true}
	written, _, err := downloadToFileRPC(context.Background(), rpc, &tg.InputDocumentFileLocation{
		ID: 1, AccessHash: 2,
	}, int64(len(data)), &buf, opts)
	if err != nil {
		t.Fatalf("verification should be skipped when unsupported: %v", err)
	}
	if written != int64(len(data)) {
		t.Errorf("written = %d, want %d", written, int64(len(data)))
	}
}

func TestVerifyDownloadChunkHash_EmptyHashes(t *testing.T) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}

	mock := &hashVerifyInvoker{
		data:        data,
		chunkSize:   4096,
		hashes:      nil, // server returns empty vector
		supportHash: true,
	}
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	opts := &params.Download{VerifyHashes: true}
	written, _, err := downloadToFileRPC(context.Background(), rpc, &tg.InputDocumentFileLocation{
		ID: 1, AccessHash: 2,
	}, int64(len(data)), &buf, opts)
	if err != nil {
		t.Fatalf("empty hashes should be skipped: %v", err)
	}
	if written != int64(len(data)) {
		t.Errorf("written = %d, want %d", written, int64(len(data)))
	}
}

func TestVerifyDownloadChunkHash_DisabledByDefault(t *testing.T) {
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i)
	}

	hash := sha256.Sum256(data[:4096])

	mock := &hashVerifyInvoker{
		data:      data,
		chunkSize: 4096,
		hashes: []*tg.FileHash{
			{Offset: 0, Limit: 4096, Hash: hash[:]},
		},
		supportHash: true,
	}
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	// VerifyHashes not set — should not call getFileHashes at all.
	_, _, err := downloadToFileRPC(context.Background(), rpc, &tg.InputDocumentFileLocation{
		ID: 1, AccessHash: 2,
	}, int64(len(data)), &buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.hashCallCount > 0 {
		t.Error("getFileHashes should not be called when VerifyHashes is false")
	}
}

func TestVerifyDownloadChunkHash_MultiChunk(t *testing.T) {
	// 2 full chunks + partial.
	data := make([]byte, downloadChunkSize*2+500)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Build hashes per chunk.
	var hashes []*tg.FileHash
	for off := int64(0); off < int64(len(data)); off += int64(downloadChunkSize) {
		end := off + int64(downloadChunkSize)
		if end > int64(len(data)) {
			end = int64(len(data))
		}
		h := sha256.Sum256(data[off:end])
		hashes = append(hashes, &tg.FileHash{
			Offset: off,
			Limit:  int32(end - off),
			Hash:   h[:],
		})
	}

	mock := &hashVerifyInvoker{
		data:          data,
		chunkSize:     downloadChunkSize,
		hashes:        hashes,
		supportHash:   true,
		perOffsetHash: true, // return only hashes matching the requested offset
	}
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	opts := &params.Download{VerifyHashes: true, ChunkSize: downloadChunkSize}
	written, _, err := downloadToFileRPC(context.Background(), rpc, &tg.InputDocumentFileLocation{
		ID: 1, AccessHash: 2,
	}, int64(len(data)), &buf, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written != int64(len(data)) {
		t.Errorf("written = %d, want %d", written, int64(len(data)))
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Error("data mismatch")
	}
}

func TestVerifyDownloadChunkHash_MultiChunkTampered(t *testing.T) {
	data := make([]byte, downloadChunkSize*2)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Hashes are correct for the original data.
	var hashes []*tg.FileHash
	for off := int64(0); off < int64(len(data)); off += int64(downloadChunkSize) {
		h := sha256.Sum256(data[off : off+int64(downloadChunkSize)])
		hashes = append(hashes, &tg.FileHash{
			Offset: off,
			Limit:  downloadChunkSize,
			Hash:   h[:],
		})
	}

	// Tamper the second chunk.
	tampered := make([]byte, len(data))
	copy(tampered, data)
	tampered[downloadChunkSize+50] ^= 0xff

	mock := &hashVerifyInvoker{
		data:          tampered,
		chunkSize:     downloadChunkSize,
		hashes:        hashes,
		supportHash:   true,
		perOffsetHash: true,
	}
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	opts := &params.Download{VerifyHashes: true, ChunkSize: downloadChunkSize}
	_, _, err := downloadToFileRPC(context.Background(), rpc, &tg.InputDocumentFileLocation{
		ID: 1, AccessHash: 2,
	}, int64(len(data)), &buf, opts)
	if err == nil {
		t.Fatal("expected hash verification error on second chunk")
	}
}

// hashVerifyInvoker is a mock that serves file chunks and file hashes.
type hashVerifyInvoker struct {
	data          []byte
	chunkSize     int32
	hashes        []*tg.FileHash
	supportHash   bool
	hashCallCount int
	perOffsetHash bool // return only hashes matching the requested offset
}

func (m *hashVerifyInvoker) RPCInvoke(_ context.Context, input tg.TLObject, _ func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	switch req := input.(type) {
	case *tg.UploadGetFileRequest:
		start := req.Offset
		if start >= int64(len(m.data)) {
			return &tg.UploadFile{Bytes: nil}, nil
		}
		end := start + int64(m.chunkSize)
		if end > int64(len(m.data)) {
			end = int64(len(m.data))
		}
		return &tg.UploadFile{Bytes: m.data[start:end]}, nil

	case *tg.UploadGetFileHashesRequest:
		m.hashCallCount++
		if !m.supportHash {
			return nil, errors.New("RPCError FILE_HASH_INVALID")
		}
		var result []*tg.FileHash
		for _, h := range m.hashes {
			if m.perOffsetHash {
				if h.Offset == req.Offset {
					result = append(result, h)
				}
			} else {
				result = append(result, h)
			}
		}
		return &fileHashResult{items: result}, nil

	default:
		return nil, fmt.Errorf("unexpected request type: %T", input)
	}
}

func (m *hashVerifyInvoker) RPCInvokeRaw(_ context.Context, _ tg.TLObject) ([]byte, error) {
	return nil, nil
}
