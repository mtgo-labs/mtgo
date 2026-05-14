package telegram

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

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
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.Log.Debugf("DownloadFile size=%d", fileSize)

	rpc := c.Raw()
	var buf memoryBuffer
	_, err := downloadToFileRPC(ctx, rpc, location, fileSize, &buf, opts)
	if err != nil {
		c.Log.Warnf("DownloadFile failed err=%v", err)
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
	if err := c.ensureConnected(); err != nil {
		return err
	}

	c.Log.Debugf("DownloadToFile size=%d", fileSize)

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("download: create file: %w", err)
	}
	defer f.Close()

	rpc := c.Raw()
	_, err = downloadToFileRPC(ctx, rpc, location, fileSize, f, opts)
	if err != nil {
		c.Log.Warnf("DownloadToFile failed err=%v", err)
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
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.Log.Debug("DownloadMedia")

	location, _, err := GetFileLocation(media, thumbSize)
	if err != nil {
		c.Log.Warnf("DownloadMedia failed err=%v", err)
		return nil, fmt.Errorf("download media: %w", err)
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
	if err := c.ensureConnected(); err != nil {
		return err
	}

	c.Log.Debug("DownloadMediaToFile")

	location, _, err := GetFileLocation(media, thumbSize)
	if err != nil {
		return fmt.Errorf("download media: %w", err)
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
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.Log.Debugf("StreamFile size=%d", fileSize)

	rpc := c.Raw()
	return streamFileRPC(ctx, rpc, location, fileSize, opts)
}

type memoryBuffer struct {
	data []byte
}

func (m *memoryBuffer) Write(p []byte) (int, error) {
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *memoryBuffer) Bytes() []byte {
	return m.data
}

func downloadToFileRPC(ctx context.Context, rpc *tg.RPCClient, location tg.InputFileLocationClass, fileSize int64, writer io.Writer, opts *params.Download) (int64, error) {
	chunkSize := int32(downloadChunkSize)
	if opts != nil && opts.ChunkSize > 0 {
		chunkSize = opts.ChunkSize
	}

	var totalWritten int64
	offset := int64(0)

	for {
		select {
		case <-ctx.Done():
			return totalWritten, ctx.Err()
		default:
		}

		req := &tg.UploadGetFileRequest{
			Location: location,
			Offset:   offset,
			Limit:    chunkSize,
		}

		result, err := rpc.UploadGetFile(ctx, req)
		if err != nil {
			return totalWritten, fmt.Errorf("download: get file at offset %d: %w", offset, err)
		}

		switch file := result.(type) {
		case *tg.UploadFile:
			if len(file.Bytes) == 0 {
				return totalWritten, nil
			}

			n, err := writer.Write(file.Bytes)
			if err != nil {
				return totalWritten, fmt.Errorf("download: write: %w", err)
			}
			totalWritten += int64(n)
			offset += int64(n)

			if opts != nil && opts.Progress != nil {
				opts.Progress(params.ProgressInfo{
					TotalBytes:      fileSize,
					DownloadedBytes: totalWritten,
					IsUpload:        false,
				})
			}

			if fileSize > 0 && totalWritten >= fileSize {
				return totalWritten, nil
			}

		case *tg.UploadFileCDNRedirect:
			return totalWritten, fmt.Errorf("download: CDN redirect not supported in basic mode (dc=%d)", file.DCID)

		default:
			return totalWritten, fmt.Errorf("download: unexpected result type %T", result)
		}
	}
}

func streamFileRPC(ctx context.Context, rpc *tg.RPCClient, location tg.InputFileLocationClass, fileSize int64, opts *params.Download) (<-chan FileChunk, error) {
	chunkSize := int32(downloadChunkSize)
	if opts != nil && opts.ChunkSize > 0 {
		chunkSize = opts.ChunkSize
	}

	ch := make(chan FileChunk, 2)

	go func() {
		defer close(ch)
		offset := int64(0)
		var totalWritten int64

		for {
			select {
			case <-ctx.Done():
				ch <- FileChunk{Err: ctx.Err()}
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
				ch <- FileChunk{Err: fmt.Errorf("download: get file at offset %d: %w", offset, err)}
				return
			}

			file, ok := result.(*tg.UploadFile)
			if !ok {
				ch <- FileChunk{Err: fmt.Errorf("download: unexpected result type %T", result)}
				return
			}

			if len(file.Bytes) == 0 {
				return
			}

			totalWritten += int64(len(file.Bytes))

			ch <- FileChunk{
				Data:  file.Bytes,
				Bytes: totalWritten,
				Total: fileSize,
			}

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

func cdnDecryptChunk(data, key, iv []byte, offset int64) []byte {
	if len(data) == 0 {
		return nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}

	chunkIV := make([]byte, 16)
	copy(chunkIV, iv[:16])

	chunkIndex := offset / 16
	for i := int64(0); i < chunkIndex; i++ {
		incrementIV(chunkIV)
	}

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
	return out
}

func cdnVerifyHash(data []byte, hash *tg.FileHash, baseOffset int64) bool {
	if hash == nil {
		return true
	}

	effectiveOffset := hash.Offset
	if baseOffset > 0 {
		effectiveOffset = hash.Offset - baseOffset
	}
	if effectiveOffset < 0 {
		effectiveOffset = 0
	}

	end := effectiveOffset + int64(hash.Limit)
	if end > int64(len(data)) {
		return false
	}

	chunk := data[effectiveOffset:end]
	computed := sha256.Sum256(chunk)
	return bytes.Equal(computed[:], hash.Hash)
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
