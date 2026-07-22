// Download API
//
// Two download APIs coexist for backwards compatibility:
//
// New API (preferred):
//
//	Download(ctx, input, opts)    — download with any input type (uses any for location)
//	DownloadBytes(ctx, input, opts) — download into []byte
//	StreamMedia(ctx, input, opts)   — stream chunks via channel
//
// Old API (retained for compatibility):
//
//	DownloadFile(ctx, location, fileSize, opts)      — download into []byte
//	DownloadToFile(ctx, location, filePath, size, opts) — download to disk
//	DownloadMedia(ctx, media, opts)                    — download media with progress
//	DownloadMediaToFile(ctx, media, filePath, opts)    — download media to disk
//	StreamFile(ctx, location, fileSize, opts)          — stream chunks via channel
//
// Migration:
//
//	DownloadFile   → DownloadBytes
//	StreamFile     → StreamMedia
//	DownloadMedia  → Download (with progress callback)
//
// Chunk types: StreamChunk supersedes FileChunk. New code should use StreamChunk.
package telegram

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

const maxDownloadRecoveries = 10

// maxFileRefRetries caps the number of automatic file-reference refresh
// attempts during a single download to avoid infinite retry loops.
const maxFileRefRetries = 3

var errParallelDownloadUnsupported = errors.New("download: parallel download unsupported for response")

// FileChunk represents a single chunk of data received during a streamed file download.
// It is delivered over the channel returned by StreamFile. Each chunk contains the raw
// data bytes, cumulative download progress, and an optional terminal error.
//
// Example:
//
//	for chunk := range ch {
//	    if chunk.Err != nil {
//	        log.Fatal(chunk.Err)
//	    }
//	    fmt.Printf("Received %d / %d bytes\n", chunk.Bytes, chunk.Total)
//	    buf.Write(chunk.Data)
//	}
type FileChunk struct {
	// Data holds the raw bytes for this download chunk. Empty when Err is set.
	Data []byte

	// Err is non-nil on the final chunk to signal a terminal download error.
	// When set, Data is empty and the stream is about to close.
	Err error

	// Bytes is the cumulative number of bytes downloaded so far, including this chunk.
	Bytes int64

	// Total is the total expected file size in bytes. May be 0 if the size is unknown.
	Total int64
}

// DownloadFile downloads a file from the given location into memory and returns its contents as a byte slice.
// The fileSize parameter hints at the total expected size for progress reporting.
//
// This method reads the entire file into memory. For large files or when disk storage is preferred,
// use DownloadToFile instead.
//
// Parameters:
//   - ctx: context for cancellation and timeout. When cancelled, the download is aborted immediately.
//   - location: the Telegram file location to download from, typically obtained via GetFileLocation.
//   - fileSize: total expected file size in bytes. Used for progress reporting; pass 0 if unknown.
//   - opts: optional download settings (chunk size, progress callback). May be nil for defaults.
//
// Returns:
//   - []byte: the complete file contents on success.
//   - error: non-nil if the client is disconnected, the download RPC fails, or the context is cancelled.
//
// Errors:
//   - client disconnected (from state.requireConnected).
//   - "download: get file at offset %d: ..." — when an UploadGetFile RPC call fails.
//   - "download: write: ..." — when writing to the internal buffer fails.
//   - "download: CDN redirect not supported in basic mode" — when Telegram returns a CDN redirect.
//   - "download: unexpected result type %T" — when an unrecognized response type is received.
//
// Example:
//
//	ctx := context.Background()
//	location, _, _ := telegram.GetFileLocation(photoMedia, "x")
//	data, err := client.DownloadFile(ctx, location, 512000, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Downloaded %d bytes\n", len(data))
func (c *Client) DownloadFile(ctx context.Context, location tg.InputFileLocationClass, fileSize int64, opts *params.Download) ([]byte, error) {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return nil, err
	}

	c.Log.Debugf("DownloadFile size=%d", fileSize)

	dcID := int32(0)
	if opts != nil {
		dcID = opts.DCID
	}
	rpc, err := c.dcRPC(ctx, int(dcID))
	if err != nil {
		return nil, fmt.Errorf("download: dc rpc: %w", err)
	}

	if shouldParallelDownload(fileSize, (*memoryBuffer)(nil), opts, int(dcID), c.homeDC()) {
		buf := memoryBuffer{data: make([]byte, int(fileSize))}
		rpcs, err := c.dcRPCPool(ctx, int(dcID), downloadWorkers(opts, fileSize, int(dcID), c.homeDC()))
		if err != nil {
			return nil, fmt.Errorf("download: dc rpc pool: %w", err)
		}
		written, err := c.downloadToWriterAt(ctx, rpcs, int(dcID), location, fileSize, &buf, opts)
		if err != nil {
			if errors.Is(err, errParallelDownloadUnsupported) && written == 0 {
				goto serialDownload
			}
			return nil, err
		}
		if written >= fileSize {
			return buf.Bytes(), nil
		}
	}

serialDownload:
	var buf memoryBuffer
	if fileSize > 0 {
		buf.data = make([]byte, 0, int(fileSize))
	}
	_, err = c.downloadToWriter(ctx, rpc, int(dcID), location, fileSize, &buf, opts)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DownloadToFile downloads a file from the given location and writes it directly to disk at filePath.
//
// If the download fails partway through, the partially written file is removed to avoid leaving
// corrupt data on disk. This is safer than DownloadFile for large files since it does not hold
// the entire contents in memory.
//
// Parameters:
//   - ctx: context for cancellation and timeout. When cancelled, the download is aborted and the
//     partial file is removed.
//   - location: the Telegram file location to download from, typically obtained via GetFileLocation.
//   - filePath: destination path on the local filesystem. The file is created (or truncated) and
//     fully written only on success.
//   - fileSize: total expected file size in bytes. Used for progress reporting; pass 0 if unknown.
//   - opts: optional download settings (chunk size, progress callback). May be nil for defaults.
//
// Returns:
//   - error: non-nil if the client is disconnected, the file cannot be created, the download fails,
//     or the context is cancelled.
//
// Errors:
//   - client disconnected (from state.requireConnected).
//   - "download: create file: ..." — when os.Create fails for filePath.
//   - "download: get file at offset %d: ..." — when an UploadGetFile RPC call fails.
//   - "download: write: ..." — when writing to the file fails.
//
// Example:
//
//	ctx := context.Background()
//	location, _, _ := telegram.GetFileLocation(docMedia, "")
//	err := client.DownloadToFile(ctx, location, "/tmp/report.pdf", 2048000, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) DownloadToFile(ctx context.Context, location tg.InputFileLocationClass, filePath string, fileSize int64, opts *params.Download) error {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return err
	}

	c.Log.Debugf("DownloadToFile size=%d", fileSize)

	dcID := int32(0)
	if opts != nil {
		dcID = opts.DCID
	}
	rpc, err := c.dcRPC(ctx, int(dcID))
	if err != nil {
		return fmt.Errorf("download: dc rpc: %w", err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("download: create file: %w", err)
	}
	defer f.Close()

	if shouldParallelDownload(fileSize, f, opts, int(dcID), c.homeDC()) {
		rpcs, err := c.dcRPCPool(ctx, int(dcID), downloadWorkers(opts, fileSize, int(dcID), c.homeDC()))
		if err != nil {
			os.Remove(filePath)
			return fmt.Errorf("download: dc rpc pool: %w", err)
		}
		written, err := c.downloadToWriterAt(ctx, rpcs, int(dcID), location, fileSize, f, opts)
		if err != nil {
			if errors.Is(err, errParallelDownloadUnsupported) && written == 0 {
				if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
					os.Remove(filePath)
					return fmt.Errorf("download: seek file: %w", seekErr)
				}
				if truncateErr := f.Truncate(0); truncateErr != nil {
					os.Remove(filePath)
					return fmt.Errorf("download: truncate file: %w", truncateErr)
				}
				goto serialFileDownload
			}
			os.Remove(filePath)
			return err
		}
		if written >= fileSize {
			return nil
		}
	}

serialFileDownload:
	_, err = c.downloadToWriter(ctx, rpc, int(dcID), location, fileSize, f, opts)
	if err != nil {
		os.Remove(filePath)
		return err
	}
	return nil
}

// DownloadMedia resolves the file location from the given Media interface and downloads its contents into memory.
//
// This is a convenience method that combines GetFileLocation and DownloadFile. Use it when you already
// have a types.Media value (e.g. from a message) and want to download it without manually resolving
// the file location.
//
// Parameters:
//   - ctx: context for cancellation and timeout.
//   - media: the media object to download (PhotoMedia or DocumentMedia).
//   - thumbSize: thumbnail size to download for photos (e.g. "m", "x"). Pass an empty string to
//     download the full-resolution media. Ignored for documents.
//   - opts: optional download settings (chunk size, progress callback). May be nil for defaults.
//
// Returns:
//   - []byte: the complete file contents on success.
//   - error: non-nil if the client is disconnected, the location cannot be resolved, or the download fails.
//
// Errors:
//   - "download media: ..." — wrapped error from GetFileLocation or DownloadFile.
//
// Example:
//
//	ctx := context.Background()
//	data, err := client.DownloadMedia(ctx, photoMedia, "m", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Downloaded photo thumbnail: %d bytes\n", len(data))
func (c *Client) DownloadMedia(ctx context.Context, media types.Media, thumbSize string, opts *params.Download) ([]byte, error) {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return nil, err
	}

	c.Log.Debug("DownloadMedia")

	location, dcID, err := GetFileLocation(media, thumbSize)
	if err != nil {
		return nil, fmt.Errorf("download media: %w", err)
	}

	if opts == nil {
		opts = &params.Download{DCID: dcID}
	} else if opts.DCID == 0 {
		opts.DCID = dcID
	}

	return c.DownloadFile(ctx, location, getMediaFileSize(media), opts)
}

// DownloadMediaToFile resolves the file location from the given Media interface and downloads it directly to disk.
//
// This is a convenience method that combines GetFileLocation and DownloadToFile. If the download fails,
// the partially written file is removed. Use this for large media files to avoid loading them entirely
// into memory.
//
// Parameters:
//   - ctx: context for cancellation and timeout.
//   - media: the media object to download (PhotoMedia or DocumentMedia).
//   - thumbSize: thumbnail size to download for photos. Pass an empty string for full media.
//   - filePath: destination path on the local filesystem.
//   - fileSize: total expected file size in bytes. Used for progress reporting; pass 0 if unknown.
//   - opts: optional download settings (chunk size, progress callback). May be nil for defaults.
//
// Returns:
//   - error: non-nil if the client is disconnected, the location cannot be resolved, or the download fails.
//
// Errors:
//   - "download media: ..." — wrapped error from GetFileLocation or DownloadToFile.
//
// Example:
//
//	ctx := context.Background()
//	err := client.DownloadMediaToFile(ctx, docMedia, "", "/tmp/video.mp4", 15_000_000, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) DownloadMediaToFile(ctx context.Context, media types.Media, thumbSize string, filePath string, fileSize int64, opts *params.Download) error {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return err
	}

	c.Log.Debug("DownloadMediaToFile")

	location, dcID, err := GetFileLocation(media, thumbSize)
	if err != nil {
		return fmt.Errorf("download media: %w", err)
	}

	if opts == nil {
		opts = &params.Download{DCID: dcID}
	} else if opts.DCID == 0 {
		opts.DCID = dcID
	}

	return c.DownloadToFile(ctx, location, filePath, fileSize, opts)
}

// StreamFile downloads a file in chunks and delivers them over the returned read-only channel.
//
// This method enables streaming processing of large files without loading them entirely into memory.
// Each FileChunk contains the raw data, cumulative byte count, and total size. The channel is closed
// when the download completes or encounters an error. A final FileChunk with a non-nil Err field
// signals the terminal error.
//
// Parameters:
//   - ctx: context for cancellation and timeout. Cancelling the context causes the stream to
//     send a FileChunk with ctx.Err() and then close.
//   - location: the Telegram file location to download from.
//   - fileSize: total expected file size in bytes. Used for progress reporting; pass 0 if unknown.
//   - opts: optional download settings (chunk size, progress callback). May be nil for defaults.
//
// Returns:
//   - <-chan FileChunk: a read-only channel of download chunks. The channel is buffered (capacity 2)
//     and is always closed by the sender. Check Err on each chunk for terminal errors.
//   - error: non-nil only if the client is disconnected before streaming starts.
//
// Example:
//
//	ctx := context.Background()
//	ch, err := client.StreamFile(ctx, location, 5_000_000, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for chunk := range ch {
//	    if chunk.Err != nil {
//	        log.Fatal(chunk.Err)
//	    }
//	    _, _ = os.Stdout.Write(chunk.Data)
//	}
func (c *Client) StreamFile(ctx context.Context, location tg.InputFileLocationClass, fileSize int64, opts *params.Download) (<-chan FileChunk, error) {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return nil, err
	}

	c.Log.Debugf("StreamFile size=%d", fileSize)

	dcID := int32(0)
	if opts != nil {
		dcID = opts.DCID
	}
	rpc, err := c.dcRPC(ctx, int(dcID))
	if err != nil {
		return nil, fmt.Errorf("download: dc rpc: %w", err)
	}

	return c.streamFileRPC(ctx, rpc, int(dcID), location, fileSize, opts)
}

type memoryBuffer struct {
	mu   sync.Mutex
	data []byte
}

func (m *memoryBuffer) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *memoryBuffer) WriteAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset %d", off)
	}
	end := off + int64(len(p))
	if end > int64(len(m.data)) {
		return 0, io.ErrShortWrite
	}
	// Parallel downloads pre-size m.data and assign each worker a disjoint
	// byte range, so the copy itself does not need the serial Write lock.
	copy(m.data[off:end], p)
	return len(p), nil
}

func (m *memoryBuffer) Bytes() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data
}

// validateDownloadChunkSize ensures size satisfies Telegram's upload.getFile
// limit constraint: must be a multiple of 4096 and in [4096, 1 MiB]. If size
// is zero or negative, downloadChunkSize is returned.
func validateDownloadChunkSize(size int32) int32 {
	const (
		minDownloadChunk = 4096
		maxDownloadChunk = 1 << 20
	)
	if size <= 0 {
		return int32(downloadChunkSize)
	}
	if size < minDownloadChunk {
		return minDownloadChunk
	}
	if size > maxDownloadChunk {
		return maxDownloadChunk
	}
	return size - size%minDownloadChunk
}

func chunkSizeForDownload(opts *params.Download) int32 {
	if opts != nil {
		return validateDownloadChunkSize(opts.ChunkSize)
	}
	return int32(downloadChunkSize)
}

func downloadToFileRPC(ctx context.Context, rpc *tg.RPCClient, location tg.InputFileLocationClass, fileSize int64, writer io.Writer, opts *params.Download) (int64, *tg.UploadFileCDNRedirect, error) {
	chunkSize := chunkSizeForDownload(opts)

	var totalWritten int64
	offset := int64(0)
	refRetries := 0

	for {
		select {
		case <-ctx.Done():
			return totalWritten, nil, ctx.Err()
		default:
		}

		req := &tg.UploadGetFileRequest{
			Location: location,
			Offset:   offset,
			Limit:    chunkSize,
		}

		result, err := rpc.UploadGetFile(ctx, req)
		if err != nil {
			// G11: automatic file reference refresh on FILE_REFERENCE_EXPIRED.
			if opts != nil && opts.FileRefresher != nil &&
				tgerr.Is(err, tgerr.ErrFileReferenceExpired) && refRetries < maxFileRefRetries {
				if newLoc, refErr := tryRefreshLocationFileRef(ctx, location, opts.FileRefresher); refErr == nil {
					location = newLoc
					refRetries++
					continue
				}
			}
			return totalWritten, nil, fmt.Errorf("download: get file at offset %d: %w", offset, err)
		}

		switch file := result.(type) {
		case *tg.UploadFile:
			if len(file.Bytes) == 0 {
				return totalWritten, nil, nil
			}

			n, err := writer.Write(file.Bytes)
			if err != nil {
				return totalWritten, nil, fmt.Errorf("download: write: %w", err)
			}
			totalWritten += int64(n)

			// G10: verify SHA-256 hashes for this chunk if enabled.
			if opts != nil && opts.VerifyHashes {
				if hashErr := verifyDownloadChunkHash(ctx, rpc, location, offset, file.Bytes); hashErr != nil {
					return totalWritten, nil, hashErr
				}
			}

			offset += int64(n)

			if n < int(chunkSize) {
				return totalWritten, nil, nil
			}

			if opts != nil && opts.Progress != nil {
				opts.Progress(params.ProgressInfo{
					TotalBytes:      fileSize,
					DownloadedBytes: totalWritten,
					IsUpload:        false,
				})
			}

			if fileSize > 0 && totalWritten >= fileSize {
				return totalWritten, nil, nil
			}

		case *tg.UploadFileCDNRedirect:
			return totalWritten, file, nil

		default:
			return totalWritten, nil, fmt.Errorf("download: unexpected result type %T", result)
		}
	}
}

func (c *Client) downloadToWriter(ctx context.Context, rpc *tg.RPCClient, dcID int, location tg.InputFileLocationClass, fileSize int64, writer io.Writer, opts *params.Download) (int64, error) {
	written, cdnRedirect, err := downloadToFileRPC(ctx, rpc, location, fileSize, writer, opts)
	if cdnRedirect != nil {
		cdnWritten, cdnErr := c.downloadCDNToWriter(ctx, cdnRedirect, written, fileSize, writer, opts)
		if cdnErr != nil {
			return written + cdnWritten, cdnErr
		}
		return written + cdnWritten, nil
	}
	if err != nil {
		migratedRPC, ok, migrateErr := c.fileMigrationRPC(ctx, dcID, written, err)
		if migrateErr != nil {
			return written, migrateErr
		}
		if !ok {
			recoveredRPC, recovered, recoverErr := c.recoverDownloadRPC(ctx, dcID, err)
			if recoverErr != nil {
				return written, recoverErr
			}
			if !recovered {
				return written, err
			}
			recoveredWritten, e := c.downloadToWriterFromOffset(ctx, recoveredRPC, dcID, location, fileSize, writer, opts, written)
			return recoveredWritten, e
		}
		w, _, e := downloadToFileRPC(ctx, migratedRPC, location, fileSize, writer, opts)
		return w, e
	}
	return written, nil
}

func shouldParallelDownload(fileSize int64, writer io.Writer, opts *params.Download, dcID int, homeDC int) bool {
	if fileSize <= 0 || downloadWorkers(opts, fileSize, dcID, homeDC) <= 1 {
		return false
	}
	_, ok := writer.(io.WriterAt)
	return ok
}

func downloadWorkers(opts *params.Download, fileSize int64, dcID int, homeDC int) int {
	// Cross-DC downloads use a single worker to avoid DC session auth race
	// conditions (AUTH_KEY_UNREGISTERED) when multiple sessions are created
	// before auth export/import completes.
	if dcID > 0 && homeDC > 0 && dcID != homeDC {
		return 1
	}
	if opts != nil && opts.Workers == 1 {
		return 1
	}
	if opts != nil && opts.Workers > 1 {
		return clampTransferWorkers(opts.Workers)
	}
	if fileSize <= int64(downloadChunkSize) {
		return 1
	}
	parts := int((fileSize + int64(downloadChunkSize) - 1) / int64(downloadChunkSize))
	return min(defaultTransferWorkers, parts)
}

func (c *Client) downloadToWriterAt(ctx context.Context, rpcs []*tg.RPCClient, dcID int, location tg.InputFileLocationClass, fileSize int64, writer io.WriterAt, opts *params.Download) (int64, error) {
	chunkSize := chunkSizeForDownload(opts)

	workers := downloadWorkers(opts, fileSize, dcID, c.homeDC())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type job struct {
		offset int64
		limit  int32
	}
	type result struct {
		offset int64
		n      int
		err    error
	}

	jobs := make(chan job, workers)
	results := make(chan result, workers)
	var done atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerIdx int) {
			defer wg.Done()
			currentRPC := rpcs[workerIdx%len(rpcs)]
			recoveries := 0
			for j := range jobs {
				var file *tg.UploadFile
				for {
					select {
					case <-ctx.Done():
						results <- result{offset: j.offset, err: ctx.Err()}
						return
					default:
					}

					res, err := currentRPC.UploadGetFile(ctx, &tg.UploadGetFileRequest{
						Location: location,
						Offset:   j.offset,
						Limit:    j.limit,
					})
					if err != nil {
						recoveredRPC, recovered, recoverErr := c.recoverDownloadWorkerRPC(ctx, dcID, len(rpcs), workerIdx, err)
						if recoverErr != nil {
							results <- result{offset: j.offset, err: recoverErr}
							return
						}
						if recovered && recoveries < maxDownloadRecoveries {
							recoveries++
							currentRPC = recoveredRPC
							continue
						}
						results <- result{offset: j.offset, err: fmt.Errorf("download: get file at offset %d: %w", j.offset, err)}
						return
					}
					recoveries = 0

					var ok bool
					file, ok = res.(*tg.UploadFile)
					if !ok {
						if _, ok := res.(*tg.UploadFileCDNRedirect); ok {
							results <- result{offset: j.offset, err: errParallelDownloadUnsupported}
							return
						}
						results <- result{offset: j.offset, err: fmt.Errorf("download: unexpected result type %T", res)}
						return
					}
					break
				}

				if len(file.Bytes) == 0 {
					results <- result{offset: j.offset}
					continue
				}
				n, err := writer.WriteAt(file.Bytes, j.offset)
				if err != nil {
					results <- result{offset: j.offset, n: n, err: fmt.Errorf("download: write at offset %d: %w", j.offset, err)}
					return
				}
				written := done.Add(int64(n))
				if opts != nil && opts.Progress != nil {
					opts.Progress(params.ProgressInfo{
						TotalBytes:      fileSize,
						DownloadedBytes: written,
						IsUpload:        false,
					})
				}
				results <- result{offset: j.offset, n: n}
			}
		}(i)
	}

	go func() {
		defer close(jobs)
		for offset := int64(0); offset < fileSize; offset += int64(chunkSize) {
			// Always request the full chunk size. Telegram requires the limit
			// to be a multiple of 4096; clamping it to the remaining file size
			// would produce a non-aligned limit and trigger LIMIT_INVALID.
			// The server returns fewer bytes for the final chunk (short read),
			// and the expected-byte check below already accounts for that.
			select {
			case jobs <- job{offset: offset, limit: chunkSize}:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	expectedParts := int((fileSize + int64(chunkSize) - 1) / int64(chunkSize))
	var firstErr error
	for i := 0; i < expectedParts; i++ {
		select {
		case <-ctx.Done():
			if firstErr != nil {
				return done.Load(), firstErr
			}
			return done.Load(), ctx.Err()
		case r, ok := <-results:
			if !ok {
				if firstErr != nil {
					return done.Load(), firstErr
				}
				if done.Load() >= fileSize {
					return done.Load(), nil
				}
				return done.Load(), io.ErrUnexpectedEOF
			}
			if r.err != nil {
				if firstErr == nil {
					firstErr = r.err
					cancel()
				}
				continue
			}
			expected := int(chunkSize)
			if remaining := fileSize - r.offset; remaining < int64(expected) {
				expected = int(remaining)
			}
			if r.n != expected {
				if firstErr == nil {
					firstErr = fmt.Errorf("download: short read at offset %d: got %d, want %d", r.offset, r.n, expected)
					cancel()
				}
			}
		}
	}
	if firstErr != nil {
		return done.Load(), firstErr
	}
	return done.Load(), nil
}

func (c *Client) downloadToWriterFromOffset(ctx context.Context, rpc *tg.RPCClient, dcID int, location tg.InputFileLocationClass, fileSize int64, writer io.Writer, opts *params.Download, offset int64) (int64, error) {
	chunkSize := chunkSizeForDownload(opts)

	totalWritten := offset
	recoveries := 0
	refRetries := 0
	for {
		select {
		case <-ctx.Done():
			return totalWritten, ctx.Err()
		default:
		}

		result, err := rpc.UploadGetFile(ctx, &tg.UploadGetFileRequest{
			Location: location,
			Offset:   totalWritten,
			Limit:    chunkSize,
		})
		if err != nil {
			// G11: automatic file reference refresh on FILE_REFERENCE_EXPIRED.
			if opts != nil && opts.FileRefresher != nil &&
				tgerr.Is(err, tgerr.ErrFileReferenceExpired) && refRetries < maxFileRefRetries {
				if newLoc, refErr := tryRefreshLocationFileRef(ctx, location, opts.FileRefresher); refErr == nil {
					location = newLoc
					refRetries++
					continue
				}
			}
			recoveredRPC, recovered, recoverErr := c.recoverDownloadRPC(ctx, dcID, err)
			if recoverErr != nil {
				return totalWritten, recoverErr
			}
			if !recovered || recoveries >= maxDownloadRecoveries {
				return totalWritten, fmt.Errorf("download: get file at offset %d: %w", totalWritten, err)
			}
			recoveries++
			rpc = recoveredRPC
			continue
		}
		recoveries = 0

		file, ok := result.(*tg.UploadFile)
		if !ok {
			if cdnRedirect, ok := result.(*tg.UploadFileCDNRedirect); ok {
				cdnWritten, cdnErr := c.downloadCDNToWriter(ctx, cdnRedirect, totalWritten, fileSize, writer, opts)
				return totalWritten + cdnWritten, cdnErr
			}
			return totalWritten, fmt.Errorf("download: unexpected result type %T", result)
		}

		if len(file.Bytes) == 0 {
			return totalWritten, nil
		}

		n, err := writer.Write(file.Bytes)
		if err != nil {
			return totalWritten, fmt.Errorf("download: write: %w", err)
		}
		chunkOffset := totalWritten
		totalWritten += int64(n)

		// G10: verify SHA-256 hashes for this chunk if enabled.
		if opts != nil && opts.VerifyHashes {
			if hashErr := verifyDownloadChunkHash(ctx, rpc, location, chunkOffset, file.Bytes); hashErr != nil {
				return totalWritten, hashErr
			}
		}

		if opts != nil && opts.Progress != nil {
			opts.Progress(params.ProgressInfo{
				TotalBytes:      fileSize,
				DownloadedBytes: totalWritten,
				IsUpload:        false,
			})
		}

		if n < int(chunkSize) || fileSize > 0 && totalWritten >= fileSize {
			return totalWritten, nil
		}
	}
}

func (c *Client) fileMigrationRPC(ctx context.Context, dcID int, written int64, err error) (*tg.RPCClient, bool, error) {
	if err == nil || written != 0 {
		return nil, false, nil
	}
	rpcErr, ok := tgerr.AsType(err, tgerr.ErrFileMigrate)
	if !ok || rpcErr.Argument <= 0 || rpcErr.Argument == dcID {
		return nil, false, nil
	}

	migratedRPC, dcErr := c.dcRPC(ctx, rpcErr.Argument)
	if dcErr != nil {
		return nil, false, fmt.Errorf("download: migrate to dc %d: %w", rpcErr.Argument, dcErr)
	}
	return migratedRPC, true, nil
}

func (c *Client) recoverDownloadRPC(ctx context.Context, dcID int, err error) (*tg.RPCClient, bool, error) {
	if err == nil || !isRecoverableDownloadError(err) {
		return nil, false, nil
	}
	if dcID > 0 {
		c.dcSessions.remove(dcID)
	}
	rpc, dcErr := c.dcRPC(ctx, dcID)
	if dcErr != nil {
		return nil, false, fmt.Errorf("download: reconnect dc %d: %w", dcID, dcErr)
	}
	return rpc, true, nil
}

func (c *Client) recoverDownloadWorkerRPC(ctx context.Context, dcID int, poolSize int, workerIdx int, err error) (*tg.RPCClient, bool, error) {
	if err == nil || !isRecoverableDownloadError(err) {
		return nil, false, nil
	}
	// Same-DC (or unknown DC): the main session recovers independently.
	// Return it directly instead of creating/replacing side sessions.
	homeDC := c.homeDC()
	if dcID <= 0 || dcID == homeDC || homeDC == 0 {
		return c.Raw(), true, nil
	}
	rpc, dcErr := c.replaceDCRPCPoolEntry(ctx, dcID, poolSize, workerIdx)
	if dcErr != nil {
		return nil, false, fmt.Errorf("download: reconnect dc %d: %w", dcID, dcErr)
	}
	return rpc, true, nil
}

func isRecoverableDownloadError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNotConnected) || errors.Is(err, ErrReconnectFailed) || isTransferSessionDeadErr(err) {
		return true
	}
	if strings.Contains(err.Error(), "session: closed") {
		return true
	}
	return false
}

func (c *Client) streamFileRPC(ctx context.Context, rpc *tg.RPCClient, dcID int, location tg.InputFileLocationClass, fileSize int64, opts *params.Download) (<-chan FileChunk, error) {
	return streamFileRPCWithMigration(ctx, rpc, location, fileSize, opts, func(written int64, err error) (*tg.RPCClient, bool, error) {
		if migratedRPC, ok, migrateErr := c.fileMigrationRPC(ctx, dcID, written, err); migrateErr != nil || ok {
			return migratedRPC, ok, migrateErr
		}
		return c.recoverDownloadRPC(ctx, dcID, err)
	})
}

func sendOrCancel[T any](ctx context.Context, ch chan<- T, v T) {
	select {
	case ch <- v:
	case <-ctx.Done():
	}
}

func streamFileRPC(ctx context.Context, rpc *tg.RPCClient, location tg.InputFileLocationClass, fileSize int64, opts *params.Download) (<-chan FileChunk, error) {
	return streamFileRPCWithMigration(ctx, rpc, location, fileSize, opts, nil)
}

func streamFileRPCWithMigration(ctx context.Context, rpc *tg.RPCClient, location tg.InputFileLocationClass, fileSize int64, opts *params.Download, migrate func(written int64, err error) (*tg.RPCClient, bool, error)) (<-chan FileChunk, error) {
	chunkSize := chunkSizeForDownload(opts)

	ch := make(chan FileChunk, 2)

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				sendOrCancel(ctx, ch, FileChunk{Err: fmt.Errorf("download: stream panic: %v", r)})
			}
		}()
		offset := int64(0)
		var totalWritten int64
		refRetries := 0

		for {
			select {
			case <-ctx.Done():
				sendOrCancel(ctx, ch, FileChunk{Err: ctx.Err()})
				return
			default:
			}

			req := &tg.UploadGetFileRequest{
				Location: location,
				Offset:   offset,
				Limit:    chunkSize,
			}

			result, err := rpc.UploadGetFile(ctx, req)
			if err != nil {
				// G11: automatic file reference refresh on FILE_REFERENCE_EXPIRED.
				if opts != nil && opts.FileRefresher != nil &&
					tgerr.Is(err, tgerr.ErrFileReferenceExpired) && refRetries < maxFileRefRetries {
					if newLoc, refErr := tryRefreshLocationFileRef(ctx, location, opts.FileRefresher); refErr == nil {
						location = newLoc
						refRetries++
						continue
					}
				}
				if migrate != nil {
					migratedRPC, ok, migrateErr := migrate(totalWritten, err)
					if migrateErr != nil {
						sendOrCancel(ctx, ch, FileChunk{Err: migrateErr})
						return
					}
					if ok {
						rpc = migratedRPC
						continue
					}
				}
				sendOrCancel(ctx, ch, FileChunk{Err: fmt.Errorf("download: get file at offset %d: %w", offset, err)})
				return
			}

			file, ok := result.(*tg.UploadFile)
			if !ok {
				sendOrCancel(ctx, ch, FileChunk{Err: fmt.Errorf("download: unexpected result type %T", result)})
				return
			}

			if len(file.Bytes) == 0 {
				return
			}

			// G10: verify SHA-256 hashes for this chunk if enabled.
			if opts != nil && opts.VerifyHashes {
				if hashErr := verifyDownloadChunkHash(ctx, rpc, location, offset, file.Bytes); hashErr != nil {
					sendOrCancel(ctx, ch, FileChunk{Err: hashErr})
					return
				}
			}

			totalWritten += int64(len(file.Bytes))

			sendOrCancel(ctx, ch, FileChunk{
				Data:  file.Bytes,
				Bytes: totalWritten,
				Total: fileSize,
			})

			offset += int64(len(file.Bytes))

			if opts != nil && opts.Progress != nil {
				opts.Progress(params.ProgressInfo{
					TotalBytes:      fileSize,
					DownloadedBytes: totalWritten,
					IsUpload:        false,
				})
			}

			if fileSize > 0 && totalWritten >= fileSize {
				return
			}
		}
	}()

	return ch, nil
}

func getMediaFileSize(media types.Media) int64 {
	switch m := media.(type) {
	case *types.DocumentMedia:
		return m.FileSize
	default:
		return 0
	}
}

func (c *Client) downloadCDNToWriter(ctx context.Context, redirect *tg.UploadFileCDNRedirect, startOffset, fileSize int64, writer io.Writer, opts *params.Download) (int64, error) {
	cdnRPC, err := c.dcRPC(ctx, int(redirect.DCID))
	if err != nil {
		return 0, fmt.Errorf("cdn: connect to dc %d: %w", redirect.DCID, err)
	}

	chunkSize := chunkSizeForDownload(opts)

	key := redirect.EncryptionKey
	iv := make([]byte, len(redirect.EncryptionIv))
	copy(iv, redirect.EncryptionIv)

	hasher := &cdnHashChecker{hashes: redirect.FileHashes}
	var totalWritten int64
	offset := startOffset
	reuploadAttempts := 0

	for {
		select {
		case <-ctx.Done():
			return totalWritten, ctx.Err()
		default:
		}

		coverageEnd, err := hasher.ensureCoverage(ctx, cdnRPC, redirect.FileToken, offset)
		if err != nil {
			return totalWritten, err
		}
		limit := int64(chunkSize)
		if available := coverageEnd - offset; available < limit {
			limit = available
		}
		if limit <= 0 {
			return totalWritten, fmt.Errorf("cdn: no authenticated range at offset %d", offset)
		}

		req := &tg.UploadGetCDNFileRequest{
			FileToken: redirect.FileToken,
			Offset:    offset,
			Limit:     int32(limit),
		}

		result, err := cdnRPC.UploadGetCDNFile(ctx, req)
		if err != nil {
			return totalWritten, fmt.Errorf("cdn: get file at offset %d: %w", offset, err)
		}

		cdnFile, ok := result.(*tg.UploadCDNFile)
		if !ok {
			if reupload, ok := result.(*tg.UploadCDNFileReuploadNeeded); ok {
				reuploadAttempts++
				if reuploadAttempts > 3 {
					return totalWritten, fmt.Errorf("cdn: too many reupload attempts at offset %d", offset)
				}
				_, reuploadErr := cdnRPC.UploadReuploadCDNFile(ctx, &tg.UploadReuploadCDNFileRequest{
					FileToken:    redirect.FileToken,
					RequestToken: reupload.RequestToken,
				})
				if reuploadErr != nil {
					return totalWritten, fmt.Errorf("cdn: reupload needed but failed: %w", reuploadErr)
				}
				continue
			}
			return totalWritten, fmt.Errorf("cdn: unexpected result type %T", result)
		}
		reuploadAttempts = 0
		if len(cdnFile.Bytes) == 0 {
			return totalWritten, fmt.Errorf("cdn: empty response inside authenticated range at offset %d", offset)
		}
		if len(cdnFile.Bytes) > int(limit) {
			return totalWritten, fmt.Errorf("cdn: response at offset %d exceeds requested authenticated range", offset)
		}

		decrypted, err := cdnDecryptChunk(cdnFile.Bytes, key, iv, offset)
		if err != nil {
			return totalWritten, err
		}

		// Verify CDN file hashes. hash.Offset is absolute into the file; a hash
		// may span a chunk boundary, so the checker buffers bytes of the
		// in-progress hash across consecutive chunks. A mismatch means the CDN
		// served corrupted/tampered data and the download must abort.
		if err := hasher.feed(decrypted, offset); err != nil {
			return totalWritten, err
		}

		n, err := writer.Write(decrypted)
		if err != nil {
			return totalWritten, fmt.Errorf("cdn: write: %w", err)
		}
		if n != len(decrypted) {
			return totalWritten, fmt.Errorf("cdn: write: %w", io.ErrShortWrite)
		}
		totalWritten += int64(n)
		offset += int64(len(decrypted))

		if len(cdnFile.Bytes) < int(limit) {
			if err := hasher.finish(); err != nil {
				return totalWritten, err
			}
			return totalWritten, nil
		}

		if opts != nil && opts.Progress != nil {
			opts.Progress(params.ProgressInfo{
				TotalBytes:      fileSize,
				DownloadedBytes: startOffset + totalWritten,
				IsUpload:        false,
			})
		}

		if fileSize > 0 && offset >= fileSize {
			if err := hasher.finish(); err != nil {
				return totalWritten, err
			}
			return totalWritten, nil
		}
	}
}

func cdnDecryptChunk(data, key, iv []byte, offset int64) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cdn: invalid encryption key (len %d): %w", len(key), err)
	}
	if len(iv) < 16 {
		return nil, fmt.Errorf("cdn: encryption iv too short: %d", len(iv))
	}

	chunkIV := make([]byte, 16)
	copy(chunkIV, iv[:16])

	// Derive the CTR counter for the first 16-byte block of this chunk. The IV
	// is a 128-bit little-endian counter (incrementIV adds 1 to it per block),
	// so advancing offset/16 blocks is a single O(1) add rather than an O(N)
	// loop — this keeps large CDN downloads linear rather than quadratic.
	addToIVLE(chunkIV, offset/16)

	out := make([]byte, len(data))
	keystream := make([]byte, 16)
	pos := 0
	for i := range data {
		if pos == 0 {
			block.Encrypt(keystream, chunkIV)
		}
		out[i] = data[i] ^ keystream[pos]
		pos++
		if pos >= 16 {
			pos = 0
			incrementIV(chunkIV)
		}
	}
	return out, nil
}

// addToIVLE adds n to the 128-bit little-endian integer represented by the
// 16-byte iv, matching incrementIV's carry direction (byte 15 is the least
// significant). Equivalent to calling incrementIV n times but in O(1).
func addToIVLE(iv []byte, n int64) {
	if n <= 0 {
		return
	}
	carry := uint64(n)
	for i := 15; i >= 0 && carry != 0; i-- {
		sum := uint64(iv[i]) + (carry & 0xff)
		iv[i] = byte(sum)
		carry = (sum >> 8) + (carry >> 8)
	}
}

// cdnHashChecker verifies CDN file hashes in stream order. A hash may span a
// chunk boundary, so the checker buffers bytes of the in-progress hash across
// consecutive chunks. Each hash is verified exactly once it becomes complete.
type cdnHashChecker struct {
	hashes []*tg.FileHash
	idx    int
	buf    []byte
}

func (c *cdnHashChecker) ensureCoverage(ctx context.Context, rpc *tg.RPCClient, fileToken []byte, offset int64) (int64, error) {
	for attempt := 0; attempt < 3; attempt++ {
		for i := c.idx; i < len(c.hashes); i++ {
			h := c.hashes[i]
			if h == nil || h.Offset < 0 || h.Limit <= 0 || len(h.Hash) != sha256.Size {
				return 0, fmt.Errorf("cdn: invalid hash descriptor at index %d", i)
			}
		}
		sort.Slice(c.hashes[c.idx:], func(i, j int) bool {
			return c.hashes[c.idx+i].Offset < c.hashes[c.idx+j].Offset
		})
		end, err := c.coverageEnd(offset)
		if err != nil {
			return 0, err
		}
		if end > offset {
			return end, nil
		}

		result, err := rpc.Invoke(ctx, &tg.UploadGetCDNFileHashesRequest{
			FileToken: fileToken,
			Offset:    offset,
		}, decodeFileHashVector)
		if err != nil {
			return 0, fmt.Errorf("cdn: get hashes at offset %d: %w", offset, err)
		}
		fhr, ok := result.(*fileHashResult)
		if !ok || len(fhr.items) == 0 {
			return 0, fmt.Errorf("cdn: server returned no hashes at offset %d", offset)
		}
		c.hashes = append(c.hashes, fhr.items...)
		sort.Slice(c.hashes[c.idx:], func(i, j int) bool {
			return c.hashes[c.idx+i].Offset < c.hashes[c.idx+j].Offset
		})
	}
	return 0, fmt.Errorf("cdn: no authenticated range at offset %d", offset)
}

func (c *cdnHashChecker) coverageEnd(offset int64) (int64, error) {
	for c.idx < len(c.hashes) {
		h := c.hashes[c.idx]
		if h == nil || h.Offset < 0 || h.Limit <= 0 || len(h.Hash) != sha256.Size {
			return 0, fmt.Errorf("cdn: invalid hash descriptor at index %d", c.idx)
		}
		if h.Offset+int64(h.Limit) > offset {
			break
		}
		c.idx++
	}

	end := offset
	for i := c.idx; i < len(c.hashes); i++ {
		h := c.hashes[i]
		if h == nil || h.Offset < 0 || h.Limit <= 0 || len(h.Hash) != sha256.Size {
			return 0, fmt.Errorf("cdn: invalid hash descriptor at index %d", i)
		}
		hashEnd := h.Offset + int64(h.Limit)
		if i == c.idx && len(c.buf) != 0 {
			if h.Offset >= offset || hashEnd <= offset {
				return 0, fmt.Errorf("cdn: hash range does not cover buffered offset %d", offset)
			}
			end = hashEnd
			continue
		}
		if h.Offset > end {
			break
		}
		if h.Offset < end {
			return 0, fmt.Errorf("cdn: overlapping hash range at offset %d", h.Offset)
		}
		end = hashEnd
	}
	return end, nil
}

func (c *cdnHashChecker) finish() error {
	if len(c.buf) != 0 {
		return fmt.Errorf("cdn: incomplete authenticated range (%d buffered bytes)", len(c.buf))
	}
	return nil
}

// feed offers the decrypted bytes covering absolute range [base, base+len(data))
// and verifies every hash that completes within this range. Returns an error on
// any mismatch.
func (c *cdnHashChecker) feed(data []byte, base int64) error {
	chunkEnd := base + int64(len(data))
	for c.idx < len(c.hashes) {
		h := c.hashes[c.idx]
		if h == nil {
			c.idx++
			continue
		}
		if h.Offset >= chunkEnd {
			return nil // hash starts in a later chunk
		}
		hashEnd := h.Offset + int64(h.Limit)

		// Fast path: hash fully contained in this chunk with nothing carried
		// over from a previous chunk (the normal case when hashes align to chunk
		// boundaries).
		if len(c.buf) == 0 && h.Offset >= base && hashEnd <= chunkEnd {
			if !cdnVerifyHash(data, h, base) {
				return fmt.Errorf("cdn: hash verification failed at offset %d", h.Offset)
			}
			c.idx++
			continue
		}

		// Spanning hash: append this chunk's contribution and verify once the
		// full range has been accumulated.
		start := h.Offset
		if start < base {
			start = base
		}
		end := hashEnd
		if end > chunkEnd {
			end = chunkEnd
		}
		c.buf = append(c.buf, data[start-base:end-base]...)

		if hashEnd <= chunkEnd {
			computed := sha256.Sum256(c.buf)
			if !bytes.Equal(computed[:], h.Hash) {
				return fmt.Errorf("cdn: hash verification failed at offset %d", h.Offset)
			}
			c.buf = c.buf[:0]
			c.idx++
		} else {
			return nil // extends past this chunk; wait for more
		}
	}
	return nil
}

func cdnVerifyHash(data []byte, hash *tg.FileHash, baseOffset int64) bool {
	if hash == nil {
		return true
	}
	// The hash must be fully contained within [baseOffset, baseOffset+len(data)).
	// A hash that starts before or ends after this chunk cannot be verified from
	// this chunk alone and is handled by cdnHashChecker's spanning path.
	if hash.Offset < baseOffset {
		return false
	}
	start := hash.Offset - baseOffset
	end := start + int64(hash.Limit)
	if end > int64(len(data)) {
		return false
	}
	chunk := data[start:end]
	computed := sha256.Sum256(chunk)
	return bytes.Equal(computed[:], hash.Hash)
}

// fileHashResult wraps the []*FileHash vector returned by upload.getFileHashes.
// The generated UploadGetFileHashes method uses ReadTLObject which returns an
// empty Vector for bare vectors, so we provide a custom decoder.
type fileHashResult struct {
	items []*tg.FileHash
}

func (*fileHashResult) ConstructorID() uint32        { return tg.VectorTypeID }
func (*fileHashResult) Encode(_ *bytes.Buffer) error { return nil }

// decodeFileHashVector decodes a bare TL vector<fileHash> response from
// upload.getFileHashes.
func decodeFileHashVector(r *tg.Reader) (tg.TLObject, error) {
	hdr, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if hdr != tg.VectorTypeID {
		return nil, fmt.Errorf("download: expected fileHash vector, got constructor 0x%x", hdr)
	}
	count, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	if err := tg.CheckVectorCount(count); err != nil {
		return nil, err
	}
	hashes := make([]*tg.FileHash, count)
	for i := range hashes {
		h, err := tg.DecodeFileHash(r)
		if err != nil {
			return nil, err
		}
		hashes[i] = h
	}
	return &fileHashResult{items: hashes}, nil
}

// verifyDownloadChunkHash calls upload.getFileHashes for the given offset and
// verifies each returned hash against the chunk data. Hashes that extend
// beyond the chunk boundary are skipped (they will be verified in a later
// chunk or not at all if this is the last chunk). Returns nil if the server
// returns no hashes or an error (verification is silently skipped), or an
// error if any hash mismatches.
func verifyDownloadChunkHash(ctx context.Context, rpc *tg.RPCClient, location tg.InputFileLocationClass, offset int64, data []byte) error {
	result, err := rpc.Invoke(ctx, &tg.UploadGetFileHashesRequest{
		Location: location,
		Offset:   offset,
	}, decodeFileHashVector)
	if err != nil {
		return nil // server may not support — skip verification
	}
	fhr, ok := result.(*fileHashResult)
	if !ok || len(fhr.items) == 0 {
		return nil
	}
	for _, h := range fhr.items {
		if h == nil || h.Offset < offset {
			continue
		}
		start := h.Offset - offset
		end := start + int64(h.Limit)
		if end > int64(len(data)) {
			continue // extends beyond this chunk
		}
		chunk := data[start:end]
		computed := sha256.Sum256(chunk)
		if !bytes.Equal(computed[:], h.Hash) {
			return fmt.Errorf("download: hash verification failed at offset %d", h.Offset)
		}
	}
	return nil
}

// locationDocID extracts the document or photo ID from an InputFileLocation,
// used as the key for file reference refresh lookups.
func locationDocID(location tg.InputFileLocationClass) int64 {
	switch loc := location.(type) {
	case *tg.InputDocumentFileLocation:
		return loc.ID
	case *tg.InputPhotoFileLocation:
		return loc.ID
	default:
		return 0
	}
}

// updateLocationFileRef returns a copy of location with its FileReference
// replaced by fileRef. Returns an error for unsupported location types.
func updateLocationFileRef(location tg.InputFileLocationClass, fileRef []byte) (tg.InputFileLocationClass, error) {
	switch loc := location.(type) {
	case *tg.InputDocumentFileLocation:
		return &tg.InputDocumentFileLocation{
			ID:            loc.ID,
			AccessHash:    loc.AccessHash,
			FileReference: fileRef,
			ThumbSize:     loc.ThumbSize,
		}, nil
	case *tg.InputPhotoFileLocation:
		return &tg.InputPhotoFileLocation{
			ID:            loc.ID,
			AccessHash:    loc.AccessHash,
			FileReference: fileRef,
			ThumbSize:     loc.ThumbSize,
		}, nil
	default:
		return nil, fmt.Errorf("download: unsupported location type %T for file reference refresh", location)
	}
}

// tryRefreshLocationFileRef attempts to refresh the file reference for the
// given location using the provided refresher. Returns the updated location
// on success, or an error if refresh fails.
func tryRefreshLocationFileRef(ctx context.Context, location tg.InputFileLocationClass, refresher params.FileRefresher) (tg.InputFileLocationClass, error) {
	docID := locationDocID(location)
	if docID == 0 {
		return nil, fmt.Errorf("download: cannot extract document ID from location %T", location)
	}
	newRef, err := refresher.RefreshFileReference(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("download: refresh file reference: %w", err)
	}
	if len(newRef) == 0 {
		return nil, fmt.Errorf("download: refreshed file reference is empty")
	}
	return updateLocationFileRef(location, newRef)
}

func incrementIV(iv []byte) {
	for i := 15; i >= 0; i-- {
		iv[i]++
		if iv[i] != 0 {
			break
		}
	}
}

func isFileReferenceError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "FILEREF_") ||
		strings.Contains(msg, "FILE_REFERENCE_")
}

const defaultDownloadDir = "downloads"

type downloadInput struct {
	media    types.Media
	fileName string
	fileSize int64
	mimeType string
}

func resolveDownloadInput(input any) (*downloadInput, error) {
	switch v := input.(type) {
	case *types.Message:
		if v.Media == nil {
			return nil, ErrNoDownloadableMedia
		}
		return resolveMediaInput(v.Media)
	case types.Media:
		return resolveMediaInput(v)
	default:
		return nil, fmt.Errorf("unsupported input type %T; expected *types.Message or types.Media", input)
	}
}

func resolveMediaInput(media types.Media) (*downloadInput, error) {
	switch m := media.(type) {
	case *types.PhotoMedia:
		return &downloadInput{
			media:    m,
			fileName: "",
			fileSize: 0,
			mimeType: "image/jpeg",
		}, nil
	case *types.DocumentMedia:
		return &downloadInput{
			media:    m,
			fileName: m.FileName,
			fileSize: m.FileSize,
			mimeType: m.MimeType,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported media type %T", m)
	}
}

func guessExtension(mimeType string) string {
	switch mimeType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "audio/ogg", "audio/opus":
		return ".ogg"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "application/zip":
		return ".zip"
	case "application/pdf":
		return ".pdf"
	case "application/x-tgsticker":
		return ".tgs"
	case "image/gif":
		return ".gif"
	default:
		if strings.HasPrefix(mimeType, "video/") {
			return ".mp4"
		}
		if strings.HasPrefix(mimeType, "audio/") {
			return ".mp3"
		}
		return ".bin"
	}
}

func buildDownloadPath(dir string, info *downloadInput) (string, error) {
	fileName := sanitizeFileName(info.fileName)

	if fileName == "" {
		ext := guessExtension(info.mimeType)
		ts := time.Now().Format("2006-01-02_15-04-05")
		fileName = fmt.Sprintf("%s_%s%s", strings.ToLower(info.media.MediaType().String()), ts, ext)
	}

	if dir == "" {
		dir = defaultDownloadDir
	}

	if strings.HasSuffix(dir, "/") || filepath.Base(dir) == dir && !strings.Contains(dir, ".") {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("download: create directory: %w", err)
		}
		return filepath.Join(dir, fileName), nil
	}

	parentDir := filepath.Dir(dir)
	if parentDir != "." && parentDir != "/" {
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			return "", fmt.Errorf("download: create directory: %w", err)
		}
	}
	return dir, nil
}

func sanitizeFileName(name string) string {
	if name == "" {
		return ""
	}
	name = filepath.Base(name)
	cleaned := strings.ReplaceAll(name, "..", "")
	if cleaned == "" || cleaned == "." || cleaned == "/" {
		return ""
	}
	return cleaned
}

func (c *Client) downloadToPath(ctx context.Context, input any, filePath string, progress params.ProgressFunc) (string, error) {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return "", err
	}

	info, err := resolveDownloadInput(input)
	if err != nil {
		return "", err
	}

	if filePath == "" {
		filePath, err = buildDownloadPath(defaultDownloadDir, info)
		if err != nil {
			return "", err
		}
	} else if strings.HasSuffix(filePath, "/") {
		filePath, err = buildDownloadPath(filePath, info)
		if err != nil {
			return "", err
		}
	} else {
		dir := filepath.Dir(filePath)
		if dir != "." && dir != "/" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return "", fmt.Errorf("download: create directory: %w", err)
			}
		}
	}

	opts := &params.Download{
		Progress: progress,
	}

	err = c.DownloadMediaToFile(ctx, info.media, "", filePath, info.fileSize, opts)
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(filePath)
	if err != nil {
		return filePath, nil
	}
	return abs, nil
}

func (c *Client) downloadBytes(ctx context.Context, input any, progress params.ProgressFunc) ([]byte, error) {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return nil, err
	}

	info, err := resolveDownloadInput(input)
	if err != nil {
		return nil, err
	}

	opts := &params.Download{
		Progress: progress,
	}

	return c.DownloadMedia(ctx, info.media, "", opts)
}

// Download downloads media from a message or media object to a file and returns
// the absolute file path.
//
// The input parameter accepts either a *types.Message (extracts media automatically)
// or a types.Media value (PhotoMedia, DocumentMedia, etc.).
//
// If fileName is empty, files are saved in the "downloads" directory with an
// auto-generated name based on the media type and timestamp. Paths ending with "/"
// are treated as directories. Non-existent directories are created automatically.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: *types.Message or types.Media to download
//   - fileName: custom file path or directory (empty for auto-generated)
//   - progress: optional progress callback
//
// Returns the absolute path of the downloaded file.
func (c *Client) Download(ctx context.Context, input any, fileName string, progress params.ProgressFunc) (string, error) {
	return c.downloadToPath(ctx, input, fileName, progress)
}

// DownloadBytes downloads media from a message or media object into memory and
// returns the raw bytes. This is the in-memory equivalent of [Client.Download].
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: *types.Message or types.Media to download
//   - progress: optional progress callback
//
// Returns the raw file contents as a byte slice.
func (c *Client) DownloadBytes(ctx context.Context, input any, progress params.ProgressFunc) ([]byte, error) {
	return c.downloadBytes(ctx, input, progress)
}

// StreamChunk is a single chunk of data yielded by [Client.StreamMedia].
type StreamChunk struct {
	Data []byte
	Err  error
}

// StreamMedia downloads media from a message or media object chunk by chunk,
// returning a channel that yields [StreamChunk] values. Each chunk contains up
// to 1 MiB of data (configurable via ChunkSize in opts).
//
// The channel is always closed by the sender. A chunk with a non-nil Err field
// signals the terminal error. The caller should drain the channel until it closes.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - input: *types.Message or types.Media to stream
//   - opts: optional download settings (chunk size). May be nil for defaults.
func (c *Client) StreamMedia(ctx context.Context, input any, opts *params.Download) (<-chan StreamChunk, error) {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return nil, err
	}

	info, err := resolveDownloadInput(input)
	if err != nil {
		return nil, err
	}

	location, dcID, err := GetFileLocation(info.media, "")
	if err != nil {
		return nil, fmt.Errorf("stream media: %w", err)
	}

	if opts == nil {
		opts = &params.Download{DCID: dcID}
	} else if opts.DCID == 0 {
		opts.DCID = dcID
	}

	fileCh, err := c.StreamFile(ctx, location, info.fileSize, opts)
	if err != nil {
		return nil, fmt.Errorf("stream media: %w", err)
	}

	ch := make(chan StreamChunk, 2)
	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				sendOrCancel(ctx, ch, StreamChunk{Err: fmt.Errorf("stream relay panic: %v", r)})
			}
		}()
		for chunk := range fileCh {
			if chunk.Err != nil {
				sendOrCancel(ctx, ch, StreamChunk{Err: chunk.Err})
				return
			}
			sendOrCancel(ctx, ch, StreamChunk{Data: chunk.Data})
		}
	}()

	return ch, nil
}
