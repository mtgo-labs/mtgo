package telegram

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

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

	dcID := int32(0)
	if opts != nil {
		dcID = opts.DCID
	}
	rpc, err := c.dcRPC(ctx, int(dcID))
	if err != nil {
		return nil, fmt.Errorf("download: dc rpc: %w", err)
	}

	var buf memoryBuffer
	if fileSize > 0 {
		buf.data = make([]byte, 0, fileSize)
	}
	_, err = downloadToFileRPC(ctx, rpc, location, fileSize, &buf, opts)
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
	if err := c.ensureConnected(); err != nil {
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

	_, err = downloadToFileRPC(ctx, rpc, location, fileSize, f, opts)
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
	if err := c.ensureConnected(); err != nil {
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
	if err := c.ensureConnected(); err != nil {
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
	if err := c.ensureConnected(); err != nil {
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

func sendOrCancel[T any](ctx context.Context, ch chan<- T, v T) {
	select {
	case ch <- v:
	case <-ctx.Done():
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
		defer func() {
			if r := recover(); r != nil {
				sendOrCancel(ctx, ch, FileChunk{Err: fmt.Errorf("download: stream panic: %v", r)})
			}
		}()
		offset := int64(0)
		var totalWritten int64

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

func cdnDecryptChunk(data, key, iv []byte, offset int64) []byte {
	if len(data) == 0 {
		return nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return data
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

const defaultDownloadDir = "downloads"

type downloadInput struct {
	media    types.Media
	fileName string
	fileSize int64
	mimeType string
}

func resolveDownloadInput(input interface{}) (*downloadInput, error) {
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
	fileName := info.fileName

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

func (c *Client) downloadToPath(ctx context.Context, input interface{}, filePath string, progress params.ProgressFunc) (string, error) {
	if err := c.ensureConnected(); err != nil {
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

func (c *Client) downloadBytes(ctx context.Context, input interface{}, progress params.ProgressFunc) ([]byte, error) {
	if err := c.ensureConnected(); err != nil {
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
func (c *Client) Download(ctx context.Context, input interface{}, fileName string, progress params.ProgressFunc) (string, error) {
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
func (c *Client) DownloadBytes(ctx context.Context, input interface{}, progress params.ProgressFunc) ([]byte, error) {
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
func (c *Client) StreamMedia(ctx context.Context, input interface{}, opts *params.Download) (<-chan StreamChunk, error) {
	if err := c.ensureConnected(); err != nil {
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

	rpc, err := c.dcRPC(ctx, int(opts.DCID))
	if err != nil {
		return nil, fmt.Errorf("stream media: dc rpc: %w", err)
	}

	chunkSize := int32(downloadChunkSize)
	if opts != nil && opts.ChunkSize > 0 {
		chunkSize = opts.ChunkSize
	}

	ch := make(chan StreamChunk, 2)

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				sendOrCancel(ctx, ch, StreamChunk{Err: fmt.Errorf("stream media: panic: %v", r)})
			}
		}()
		offset := int64(0)

		for {
			select {
			case <-ctx.Done():
				sendOrCancel(ctx, ch, StreamChunk{Err: ctx.Err()})
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
				sendOrCancel(ctx, ch, StreamChunk{Err: fmt.Errorf("stream: get file at offset %d: %w", offset, err)})
				return
			}

			file, ok := result.(*tg.UploadFile)
			if !ok {
				sendOrCancel(ctx, ch, StreamChunk{Err: fmt.Errorf("stream: unexpected result type %T", result)})
				return
			}

			if len(file.Bytes) == 0 {
				return
			}

			sendOrCancel(ctx, ch, StreamChunk{Data: file.Bytes})
			offset += int64(len(file.Bytes))

			if opts != nil && opts.Progress != nil {
				opts.Progress(params.ProgressInfo{
					TotalBytes:      info.fileSize,
					DownloadedBytes: offset,
					IsUpload:        false,
				})
			}

			if info.fileSize > 0 && offset >= info.fileSize {
				return
			}
		}
	}()

	return ch, nil
}
