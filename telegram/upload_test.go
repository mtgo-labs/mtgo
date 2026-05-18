package telegram

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"github.com/mtgo-labs/mtgo/telegram/params"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

type mockRPCInvoker struct {
	mu                sync.Mutex
	savedParts        map[int32][]byte
	bigParts          map[int32][]byte
	bigFileTotalParts int32
	totalParts        int32
	err               error
	errPart           int32
	fileID            int64
	invokes           atomic.Int32
}

func newMockRPCInvoker() *mockRPCInvoker {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	id := int64(buf[0]) | int64(buf[1])<<8 | int64(buf[2])<<16 | int64(buf[3])<<24
	return &mockRPCInvoker{
		savedParts: make(map[int32][]byte),
		bigParts:   make(map[int32][]byte),
		fileID:     id,
		errPart:    -1,
	}
}

func (m *mockRPCInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	m.invokes.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return nil, m.err
	}

	switch req := input.(type) {
	case *tg.UploadSaveFilePartRequest:
		if m.errPart >= 0 && req.FilePart == m.errPart {
			return nil, fmt.Errorf("mock error on part %d", req.FilePart)
		}
		cp := make([]byte, len(req.Bytes))
		copy(cp, req.Bytes)
		m.savedParts[req.FilePart] = cp
		if req.FilePart+1 > m.totalParts {
			m.totalParts = req.FilePart + 1
		}
		return tg.TLBool(true), nil
	case *tg.UploadSaveBigFilePartRequest:
		if m.errPart >= 0 && req.FilePart == m.errPart {
			return nil, fmt.Errorf("mock error on part %d", req.FilePart)
		}
		cp := make([]byte, len(req.Bytes))
		copy(cp, req.Bytes)
		m.bigParts[req.FilePart] = cp
		m.bigFileTotalParts = req.FileTotalParts
		if req.FilePart+1 > m.totalParts {
			m.totalParts = req.FilePart + 1
		}
		return tg.TLBool(true), nil
	}

	return nil, fmt.Errorf("unexpected request type: %T", input)
}
func (m *mockRPCInvoker) RPCInvokeRaw(_ context.Context, _ tg.TLObject) ([]byte, error) {
	return nil, nil
}


func TestUploadFile_SmallFile(t *testing.T) {
	data := make([]byte, 1024)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	result, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "test.bin", int64(len(data)), nil)
	if err != nil {
		t.Fatalf("UploadFile() error: %v", err)
	}

	inputFile, ok := result.(*tg.InputFile)
	if !ok {
		t.Fatalf("expected *tg.InputFile, got %T", result)
	}
	if inputFile.Parts != 1 {
		t.Errorf("Parts = %d, want 1", inputFile.Parts)
	}
	if inputFile.Name != "test.bin" {
		t.Errorf("Name = %q, want %q", inputFile.Name, "test.bin")
	}

	expectedMD5 := fmt.Sprintf("%x", md5.Sum(data))
	if inputFile.MD5Checksum != expectedMD5 {
		t.Errorf("MD5Checksum = %q, want %q", inputFile.MD5Checksum, expectedMD5)
	}
}

func TestUploadFile_BigFile(t *testing.T) {
	size := int64(15 << 20)
	data := make([]byte, size)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	result, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "big.bin", size, nil)
	if err != nil {
		t.Fatalf("UploadFile() error: %v", err)
	}

	inputFileBig, ok := result.(*tg.InputFileBig)
	if !ok {
		t.Fatalf("expected *tg.InputFileBig, got %T", result)
	}
	if inputFileBig.Name != "big.bin" {
		t.Errorf("Name = %q, want %q", inputFileBig.Name, "big.bin")
	}

	expectedParts := int32(size / uploadPartSize)
	if size%uploadPartSize != 0 {
		expectedParts++
	}
	if inputFileBig.Parts != expectedParts {
		t.Errorf("Parts = %d, want %d", inputFileBig.Parts, expectedParts)
	}
}

func TestUploadFile_MultipleParts(t *testing.T) {
	size := int64(uploadPartSize*2 + 100)
	data := make([]byte, size)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	result, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "multi.bin", size, nil)
	if err != nil {
		t.Fatalf("UploadFile() error: %v", err)
	}

	inputFile, ok := result.(*tg.InputFile)
	if !ok {
		t.Fatalf("expected *tg.InputFile for < 10MiB, got %T", result)
	}
	if inputFile.Parts != 3 {
		t.Errorf("Parts = %d, want 3", inputFile.Parts)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.savedParts) != 3 {
		t.Errorf("saved %d parts, want 3", len(mock.savedParts))
	}

	var reassembled []byte
	for i := int32(0); i < 3; i++ {
		part, ok := mock.savedParts[i]
		if !ok {
			t.Errorf("missing part %d", i)
			continue
		}
		reassembled = append(reassembled, part...)
	}
	if !bytes.Equal(reassembled, data) {
		t.Errorf("reassembled data does not match original (len=%d vs %d)", len(reassembled), len(data))
	}
}

func TestUploadFile_EmptyFile(t *testing.T) {
	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	_, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(nil), "empty.txt", 0, nil)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestUploadFile_TooLarge(t *testing.T) {
	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	_, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(make([]byte, 100)), "big.txt", maxFileSize+1, nil)
	if err == nil {
		t.Fatal("expected error for file exceeding max size")
	}
}

func TestUploadFile_ProgressCallback(t *testing.T) {
	data := make([]byte, uploadPartSize+100)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	var lastProgress float64
	var callCount int32
	progress := func(info params.ProgressInfo) {
		p := info.Progress()
		if p < lastProgress {
			t.Errorf("progress went backwards: %f -> %f", lastProgress, p)
		}
		lastProgress = p
		callCount++
	}

	ctx := context.Background()
	_, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "progress.bin", int64(len(data)), &UploadOptions{Progress: progress})
	if err != nil {
		t.Fatalf("UploadFile() error: %v", err)
	}
	if callCount == 0 {
		t.Error("progress callback was never called")
	}
}

func TestUploadFile_PartRetry(t *testing.T) {
	data := make([]byte, uploadPartSize*3)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	mock.errPart = 1
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	_, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "retry.bin", int64(len(data)), &UploadOptions{Workers: 1})
	if err == nil {
		t.Fatal("expected error when part 1 fails")
	}
}

func TestUploadFile_ContextCancelled(t *testing.T) {
	data := make([]byte, uploadPartSize*3)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "cancel.bin", int64(len(data)), nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestUploadFile_ConcurrentWorkers(t *testing.T) {
	size := int64(uploadPartSize * 5)
	data := make([]byte, size)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	result, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "concurrent.bin", size, &UploadOptions{Workers: 3})
	if err != nil {
		t.Fatalf("UploadFile() error: %v", err)
	}

	inputFile, ok := result.(*tg.InputFile)
	if !ok {
		t.Fatalf("expected *tg.InputFile, got %T", result)
	}
	if inputFile.Parts != 5 {
		t.Errorf("Parts = %d, want 5", inputFile.Parts)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	var reassembled []byte
	for i := int32(0); i < 5; i++ {
		part, ok := mock.savedParts[i]
		if !ok {
			t.Errorf("missing part %d", i)
			continue
		}
		reassembled = append(reassembled, part...)
	}
	if !bytes.Equal(reassembled, data) {
		t.Error("reassembled data does not match original")
	}
}

func TestUploadFile_StreamSmallFile(t *testing.T) {
	data := make([]byte, 1024)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	result, actualSize, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "stream.bin", 0, nil)
	if err != nil {
		t.Fatalf("streamed upload error: %v", err)
	}

	big, ok := result.(*tg.InputFileBig)
	if !ok {
		t.Fatalf("expected *tg.InputFileBig for streamed upload, got %T", result)
	}
	if big.Parts != 1 {
		t.Errorf("Parts = %d, want 1", big.Parts)
	}
	if actualSize != int64(len(data)) {
		t.Errorf("actualSize = %d, want %d", actualSize, len(data))
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.bigParts) != 1 {
		t.Fatalf("saved %d big parts, want 1", len(mock.bigParts))
	}
	if !bytes.Equal(mock.bigParts[0], data) {
		t.Error("part data does not match original")
	}
	if mock.bigFileTotalParts != 1 {
		t.Errorf("file_total_parts = %d, want 1", mock.bigFileTotalParts)
	}
}

func TestUploadFile_StreamMultipleParts(t *testing.T) {
	size := int64(uploadPartSize*2 + 100)
	data := make([]byte, size)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	result, actualSize, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "stream_multi.bin", 0, nil)
	if err != nil {
		t.Fatalf("streamed upload error: %v", err)
	}

	big, ok := result.(*tg.InputFileBig)
	if !ok {
		t.Fatalf("expected *tg.InputFileBig, got %T", result)
	}
	if big.Parts != 3 {
		t.Errorf("Parts = %d, want 3", big.Parts)
	}
	if actualSize != size {
		t.Errorf("actualSize = %d, want %d", actualSize, size)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	var reassembled []byte
	for i := int32(0); i < 3; i++ {
		part, ok := mock.bigParts[i]
		if !ok {
			t.Errorf("missing part %d", i)
			continue
		}
		reassembled = append(reassembled, part...)
	}
	if !bytes.Equal(reassembled, data) {
		t.Errorf("reassembled data does not match original (len=%d vs %d)", len(reassembled), len(data))
	}
	if mock.bigFileTotalParts != 3 {
		t.Errorf("file_total_parts = %d, want 3", mock.bigFileTotalParts)
	}
}

func TestUploadFile_StreamExactPartBoundary(t *testing.T) {
	size := int64(uploadPartSize * 3)
	data := make([]byte, size)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	result, actualSize, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "stream_exact.bin", 0, nil)
	if err != nil {
		t.Fatalf("streamed upload error: %v", err)
	}

	big := result.(*tg.InputFileBig)
	if big.Parts != 4 {
		t.Errorf("Parts = %d, want 4 (3 data + 1 empty terminator)", big.Parts)
	}
	if actualSize != size {
		t.Errorf("actualSize = %d, want %d", actualSize, size)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	if len(mock.bigParts) != 4 {
		t.Fatalf("saved %d big parts, want 4", len(mock.bigParts))
	}
	lastPart := mock.bigParts[3]
	if len(lastPart) != 0 {
		t.Errorf("last (terminator) part has %d bytes, want 0", len(lastPart))
	}
	if mock.bigFileTotalParts != 3 {
		t.Errorf("file_total_parts = %d, want 3", mock.bigFileTotalParts)
	}
}

func TestUploadFile_StreamEmptyReader(t *testing.T) {
	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	_, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(nil), "empty.bin", 0, nil)
	if err == nil {
		t.Fatal("expected error for empty streamed reader")
	}
}

func TestUploadFile_StreamNegativeSize(t *testing.T) {
	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	_, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader([]byte{1}), "neg.bin", -1, nil)
	if err == nil {
		t.Fatal("expected error for negative file size")
	}
}

func TestUploadFile_StreamProgressCallback(t *testing.T) {
	data := make([]byte, uploadPartSize+100)
	_, _ = rand.Read(data)

	mock := newMockRPCInvoker()
	rpc := tg.NewRPCClient(mock)

	var lastUploaded int64
	var callCount int32
	progress := func(info params.ProgressInfo) {
		if info.UploadedBytes < lastUploaded {
			t.Errorf("uploaded bytes went backwards: %d -> %d", lastUploaded, info.UploadedBytes)
		}
		lastUploaded = info.UploadedBytes
		callCount++
	}

	ctx := context.Background()
	_, _, err := uploadFileRPC(ctx, rpc, bytes.NewReader(data), "stream_progress.bin", 0, &UploadOptions{Progress: progress})
	if err != nil {
		t.Fatalf("streamed upload error: %v", err)
	}
	if callCount == 0 {
		t.Error("progress callback was never called")
	}
	if lastUploaded != int64(len(data)) {
		t.Errorf("final uploaded = %d, want %d", lastUploaded, len(data))
	}
}
