package telegram

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type mockDownloadInvoker struct {
	data        []byte
	chunkSize   int32
	offsets     atomic.Int32
	err         error
	cdnRedirect bool
}

func newMockDownloadInvoker(data []byte) *mockDownloadInvoker {
	return &mockDownloadInvoker{
		data:      data,
		chunkSize: downloadChunkSize,
	}
}

func (m *mockDownloadInvoker) RPCInvoke(ctx context.Context, input tg.TLObject, decode func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	if m.err != nil {
		return nil, m.err
	}

	req, ok := input.(*tg.UploadGetFileRequest)
	if !ok {
		return nil, fmt.Errorf("unexpected request type: %T", input)
	}

	if m.cdnRedirect {
		return &tg.UploadFileCDNRedirect{
			DCID:          1,
			FileToken:     []byte("token"),
			EncryptionKey: make([]byte, 32),
			EncryptionIv:  make([]byte, 32),
			FileHashes:    nil,
		}, nil
	}

	offset := req.Offset
	limit := req.Limit

	start := offset
	if start >= int64(len(m.data)) {
		return &tg.UploadFile{
			Type:  &tg.StorageFileUnknown{},
			Mtime: 0,
			Bytes: nil,
		}, nil
	}

	end := start + int64(limit)
	if end > int64(len(m.data)) {
		end = int64(len(m.data))
	}

	chunk := make([]byte, end-start)
	copy(chunk, m.data[start:end])
	m.offsets.Add(1)

	return &tg.UploadFile{
		Type:  &tg.StorageFileJPEG{},
		Mtime: 12345,
		Bytes: chunk,
	}, nil
}
func (m *mockDownloadInvoker) RPCInvokeRaw(_ context.Context, _ tg.TLObject) ([]byte, error) {
	return nil, nil
}

func TestDownloadFile_ToBuffer(t *testing.T) {
	data := make([]byte, downloadChunkSize*2+500)
	_, _ = rand.Read(data)

	mock := newMockDownloadInvoker(data)
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	ctx := context.Background()
	written, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID:         100,
		AccessHash: 200,
	}, int64(len(data)), &buf, nil)
	if err != nil {
		t.Fatalf("DownloadFile() error: %v", err)
	}
	if written != int64(len(data)) {
		t.Errorf("written = %d, want %d", written, int64(len(data)))
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Errorf("downloaded data does not match original")
	}
}

func TestDownloadFile_SmallFile(t *testing.T) {
	data := make([]byte, 1024)
	_, _ = rand.Read(data)

	mock := newMockDownloadInvoker(data)
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	ctx := context.Background()
	written, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID:         100,
		AccessHash: 200,
	}, int64(len(data)), &buf, nil)
	if err != nil {
		t.Fatalf("DownloadFile() error: %v", err)
	}
	if written != 1024 {
		t.Errorf("written = %d, want 1024", written)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Error("data mismatch")
	}
}

func TestDownloadFile_EmptyFile(t *testing.T) {
	mock := newMockDownloadInvoker(nil)
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	ctx := context.Background()
	written, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID: 100, AccessHash: 200,
	}, 0, &buf, nil)
	if err != nil {
		t.Fatalf("DownloadFile() error: %v", err)
	}
	if written != 0 {
		t.Errorf("written = %d, want 0", written)
	}
}

func TestDownloadFile_ProgressCallback(t *testing.T) {
	data := make([]byte, downloadChunkSize*2)
	_, _ = rand.Read(data)

	mock := newMockDownloadInvoker(data)
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

	var buf bytes.Buffer
	ctx := context.Background()
	_, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID: 100, AccessHash: 200,
	}, int64(len(data)), &buf, &params.Download{Progress: progress})
	if err != nil {
		t.Fatalf("DownloadFile() error: %v", err)
	}
	if callCount == 0 {
		t.Error("progress callback was never called")
	}
}

func TestDownloadFile_ContextCancelled(t *testing.T) {
	data := make([]byte, downloadChunkSize*3)
	_, _ = rand.Read(data)

	mock := newMockDownloadInvoker(data)
	rpc := tg.NewRPCClient(mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	_, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID: 100, AccessHash: 200,
	}, int64(len(data)), &buf, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestDownloadFile_RPCError(t *testing.T) {
	mock := &mockDownloadInvoker{err: errors.New("rpc error")}
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	ctx := context.Background()
	_, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID: 100, AccessHash: 200,
	}, 1024, &buf, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDownloadToWriterRetriesFileMigrate(t *testing.T) {
	data := []byte("migrated file")
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	c.state.SetDC(2)

	primary := &mockDownloadInvoker{err: tgerr.New(303, "FILE_MIGRATE_4")}
	migrated := newMockDownloadInvoker(data)
	c.dcSessions.put(4, &dcSessionEntry{rpc: tg.NewRPCClient(migrated)})

	var buf bytes.Buffer
	written, err := c.downloadToWriter(
		context.Background(),
		tg.NewRPCClient(primary),
		2,
		&tg.InputDocumentFileLocation{ID: 100, AccessHash: 200},
		int64(len(data)),
		&buf,
		nil,
	)
	if err != nil {
		t.Fatalf("downloadToWriter() error: %v", err)
	}
	if written != int64(len(data)) {
		t.Fatalf("written = %d, want %d", written, len(data))
	}
	if got := buf.String(); got != string(data) {
		t.Fatalf("downloaded %q, want %q", got, data)
	}
}

func TestDownloadFile_CDNRedirect(t *testing.T) {
	data := make([]byte, 1024)
	_, _ = rand.Read(data)

	mock := newMockDownloadInvoker(data)
	mock.cdnRedirect = true
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	ctx := context.Background()
	_, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID: 100, AccessHash: 200,
	}, 1024, &buf, nil)
	if err == nil {
		t.Fatal("expected error for CDN redirect without CDN handler")
	}
}

func TestClientDownloadFile_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.DownloadFile(context.Background(), &tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, 1024, nil)
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestClientDownloadToFile_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.DownloadToFile(context.Background(), &tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, "/tmp/test.bin", 1024, nil)
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestStreamFile_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.StreamFile(context.Background(), &tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, 1024, nil)
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestStreamFile_StreamsChunks(t *testing.T) {
	data := make([]byte, downloadChunkSize+500)
	_, _ = rand.Read(data)

	mock := newMockDownloadInvoker(data)
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	ch, err := streamFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID: 100, AccessHash: 200,
	}, int64(len(data)), nil)
	if err != nil {
		t.Fatalf("StreamFile() error: %v", err)
	}

	var reassembled []byte
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		reassembled = append(reassembled, chunk.Data...)
	}

	if !bytes.Equal(reassembled, data) {
		t.Errorf("streamed data does not match original (len=%d vs %d)", len(reassembled), len(data))
	}
}

func TestStreamFileRetriesFileMigrate(t *testing.T) {
	data := []byte("migrated stream")
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	c.state.SetDC(2)
	c.testInvoker = &mockDownloadInvoker{err: tgerr.New(303, "FILE_MIGRATE_4")}
	c.dcSessions.put(4, &dcSessionEntry{rpc: tg.NewRPCClient(newMockDownloadInvoker(data))})

	ch, err := c.StreamFile(context.Background(), &tg.InputDocumentFileLocation{
		ID: 100, AccessHash: 200,
	}, int64(len(data)), &params.Download{DCID: 2})
	if err != nil {
		t.Fatalf("StreamFile() error: %v", err)
	}

	var buf bytes.Buffer
	for chunk := range ch {
		if chunk.Err != nil {
			t.Fatalf("stream error: %v", chunk.Err)
		}
		buf.Write(chunk.Data)
	}
	if got := buf.String(); got != string(data) {
		t.Fatalf("streamed %q, want %q", got, data)
	}
}

func TestStreamFile_EmptyFile(t *testing.T) {
	mock := newMockDownloadInvoker(nil)
	rpc := tg.NewRPCClient(mock)

	ctx := context.Background()
	ch, err := streamFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID: 100, AccessHash: 200,
	}, 0, nil)
	if err != nil {
		t.Fatalf("StreamFile() error: %v", err)
	}

	chunkCount := 0
	for range ch {
		chunkCount++
	}
	if chunkCount != 0 {
		t.Errorf("expected 0 chunks for empty file, got %d", chunkCount)
	}
}

func TestDownloadMedia_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.DownloadMedia(context.Background(), &types.PhotoMedia{
		Photo: &types.Photo{ID: 1, AccessHash: 2, Sizes: []types.PhotoSize{{Type: "x"}}},
	}, "x", nil)
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestDownloadMediaToFile_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.DownloadMediaToFile(context.Background(), &types.DocumentMedia{
		FileID:   "100_200",
		FileName: "test.pdf",
	}, "", "/tmp/test.pdf", 1024, nil)
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestDownloadMedia_NilMedia(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	_, err := c.DownloadMedia(context.Background(), nil, "", nil)
	if err == nil {
		t.Fatal("expected error for nil media")
	}
}

func TestDownloadMedia_UnsupportedMedia(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	_, err := c.DownloadMedia(context.Background(), &types.ContactMedia{FirstName: "test"}, "", nil)
	if err == nil {
		t.Fatal("expected error for contact media")
	}
}

func TestIsFileReferenceError(t *testing.T) {
	tests := []struct {
		err  string
		want bool
	}{
		{"FILEREF_UPGRADE_NEEDED", true},
		{"FILE_REFERENCE_EXPIRED", true},
		{"FILE_REFERENCE_", true},
		{"some other error", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.err, func(t *testing.T) {
			got := isFileReferenceError(errors.New(tt.err))
			if got != tt.want {
				t.Errorf("isFileReferenceError(%q) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
