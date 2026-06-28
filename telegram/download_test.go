package telegram

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/transport"
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
	failOnceAt  int64
}

func newMockDownloadInvoker(data []byte) *mockDownloadInvoker {
	return &mockDownloadInvoker{
		data:       data,
		chunkSize:  downloadChunkSize,
		failOnceAt: -1,
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
	if m.failOnceAt >= 0 && offset == m.failOnceAt {
		m.failOnceAt = -1
		return nil, errors.New("send: session: closed")
	}
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
	written, _, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
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
	written, _, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
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
	written, _, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
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
	_, _, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
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
	_, _, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
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
	_, _, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
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

func TestDownloadToWriterRecoversClosedDCSessionAtOffset(t *testing.T) {
	data := make([]byte, downloadChunkSize*2+512)
	_, _ = rand.Read(data)

	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	c.state.SetDC(2)

	primary := newMockDownloadInvoker(data)
	primary.failOnceAt = downloadChunkSize
	recovered := newMockDownloadInvoker(data)
	c.testInvoker = recovered

	var buf bytes.Buffer
	written, err := c.downloadToWriter(
		context.Background(),
		tg.NewRPCClient(primary),
		0,
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
	if !bytes.Equal(buf.Bytes(), data) {
		t.Fatal("downloaded data does not match original")
	}
}

// failingDialer implements transport.Dialer, always failing — used to verify
// that recoverDownloadRPC evicts a dead cross-DC session even when recreation
// itself cannot succeed (no network in unit tests).
type failingDialer struct{}

func (failingDialer) Dial(string, string, time.Duration) (net.Conn, error) {
	return nil, errors.New("test: no network")
}

func TestIsRecoverableDownloadError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("io: unexpected EOF"), false},
		{"not connected", ErrNotConnected, true},
		{"bare session closed", errors.New("send: session: closed"), true},
		{"wrapped session closed",
			errors.New("download: get file at offset 0: invoke *tg.UploadGetFileRequest(cid=be5335be): retries exhausted (2): invoke *tg.UploadGetFileRequest(cid=be5335be): send: session: closed"),
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRecoverableDownloadError(tt.err); got != tt.want {
				t.Errorf("isRecoverableDownloadError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecoverDownloadRPC_EvictsDeadCrossDCSession(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	c.state.SetDC(2) // home DC 2 → DC 4 is cross-DC

	// Pre-seed a dead cross-DC session.
	c.dcSessions.put(4, &dcSessionEntry{
		rpc: tg.NewRPCClient(&mockDownloadInvoker{err: errors.New("send: session: closed")}),
	})
	if _, ok := c.dcSessions.get(4); !ok {
		t.Fatal("pre-seeded DC session not in cache")
	}

	// Recreation fails fast (no network in unit tests).
	c.testDialer = transport.Dialer(failingDialer{})

	// The exact error the user reported.
	downloadErr := errors.New("download: get file at offset 0: invoke *tg.UploadGetFileRequest(cid=be5335be): retries exhausted (2): invoke *tg.UploadGetFileRequest(cid=be5335be): send: session: closed")

	_, recovered, err := c.recoverDownloadRPC(context.Background(), 4, downloadErr)

	// Recreation fails without network, but the dead session MUST be evicted.
	if _, ok := c.dcSessions.get(4); ok {
		t.Fatal("dead cross-DC session was not evicted from cache")
	}
	if recovered {
		t.Fatal("expected recovery to fail without network, got recovered=true")
	}
	if err == nil {
		t.Fatal("expected reconnect error, got nil")
	}
}

func TestRecoverDownloadWorkerRPCUsesSessionDCForSameDC(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	c.state.SetDC(2)

	st := NewMemoryStorage()
	mainSess, err := session.NewSession(session.DataCenter{ID: 1}, st, "Test", "0.1", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() = %v", err)
	}
	c.mu.Lock()
	c.session = mainSess
	c.testInvoker = newMockDownloadInvoker([]byte("ok"))
	c.mu.Unlock()
	c.testDialer = transport.Dialer(failingDialer{})

	rpc, recovered, err := c.recoverDownloadWorkerRPC(context.Background(), 1, 4, 0, errors.New("send: session: closed"))
	if err != nil {
		t.Fatalf("recoverDownloadWorkerRPC() error = %v", err)
	}
	if !recovered {
		t.Fatal("recoverDownloadWorkerRPC() recovered = false, want true")
	}
	var buf bytes.Buffer
	_, _, err = downloadToFileRPC(context.Background(), rpc, &tg.InputDocumentFileLocation{ID: 1, AccessHash: 2}, 2, &buf, nil)
	if err != nil {
		t.Fatalf("recovered RPC did not use main invoker: %v", err)
	}
	if got := buf.String(); got != "ok" {
		t.Fatalf("recovered RPC downloaded %q, want %q", got, "ok")
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
	_, cdnRedirect, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID: 100, AccessHash: 200,
	}, 1024, &buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cdnRedirect == nil {
		t.Fatal("expected CDN redirect")
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

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"normal", "photo.jpg", "photo.jpg"},
		{"empty", "", ""},
		{"path traversal", "../../etc/passwd", "passwd"},
		{"absolute unix", "/etc/shadow", "shadow"},
		{"absolute windows", "C:\\Windows\\System32\\config", "C:\\Windows\\System32\\config"},
		{"double dot only", "..", ""},
		{"dot only", ".", ""},
		{"slash only", "/", ""},
		{"nested traversal", "foo/bar/../../baz", "baz"},
		{"dotdot in name", "file..name.txt", "filename.txt"},
		{"null bytes", "file\x00name.txt", "file\x00name.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFileName(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeFileName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCDNRedirect_ReturnedCorrectly(t *testing.T) {
	data := make([]byte, downloadChunkSize+500)
	_, _ = rand.Read(data)

	mock := newMockDownloadInvoker(data)
	mock.cdnRedirect = true
	rpc := tg.NewRPCClient(mock)

	var buf bytes.Buffer
	ctx := context.Background()
	written, cdnRedirect, err := downloadToFileRPC(ctx, rpc, &tg.InputDocumentFileLocation{
		ID: 100, AccessHash: 200,
	}, int64(len(data)), &buf, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written != 0 {
		t.Errorf("expected 0 bytes written before CDN, got %d", written)
	}
	if cdnRedirect == nil {
		t.Fatal("expected CDN redirect")
	}
	if cdnRedirect.DCID != 1 {
		t.Errorf("CDN DCID = %d, want 1", cdnRedirect.DCID)
	}
	if !bytes.Equal(cdnRedirect.FileToken, []byte("token")) {
		t.Errorf("CDN file token mismatch")
	}
}

func TestDownloadCDNReuploadNeeded(t *testing.T) {
	reupload := &tg.UploadCDNFileReuploadNeeded{RequestToken: []byte("test-token")}
	if reupload.RequestToken == nil {
		t.Error("request token should not be nil")
	}
}
