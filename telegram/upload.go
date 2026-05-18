package telegram

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"hash"
	"io"
	"sync"
	"sync/atomic"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// uploadBufPool reuses upload buffers to avoid allocating and copying a 512 KB
// slice for every upload part. Workers process the data and return the
// underlying buffer to the pool when the RPC call completes.
var uploadBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, uploadPartSize)
		return &b
	},
}

// UploadResult holds the outcome of a successful file upload to Telegram.
// It contains the InputFileClass handle needed to reference the uploaded file
// in subsequent API calls (e.g. sending media messages), along with metadata
// about what was uploaded.
//
// Example:
//
//	result, err := client.UploadFile(ctx, fileReader, "photo.jpg", 102400, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Uploaded %d bytes as %q (big=%v)\n", result.Size, result.Name, result.IsBig)
type UploadResult struct {
	// File is the Telegram input file handle returned by the upload RPC.
	// Pass this to media-sending methods such as SendMedia or attach it to
	// InputMediaUploadedPhoto / InputMediaUploadedDocument constructors.
	File tg.InputFileClass

	// Size is the total size of the uploaded file in bytes, as reported by the caller.
	Size int64

	// Name is the file name string that was reported to Telegram during upload.
	Name string

	// IsBig indicates whether the file was uploaded using the big-file path
	// (UploadSaveBigFilePart). Files at or above bigFileThreshold (10 MB) use this path,
	// which skips MD5 hashing of the content.
	IsBig bool
}

// UploadFile uploads a file to Telegram servers by splitting it into parts and sending them concurrently.
//
// When fileSize is known (> 0), the entire file is read into memory and split into fixed-size parts
// (uploadPartSize, 512 KB). Parts are uploaded in parallel using up to opts.Workers goroutines.
// For files at or above bigFileThreshold (10 MB), the big-file upload path is used which skips MD5 hashing.
//
// When fileSize is 0, a streamed upload is performed. The file is read part-by-part and each part is
// uploaded immediately. This is useful when the total size is unknown beforehand (e.g. piping from an
// encoder). Streamed uploads always use upload.saveBigFilePart with file_total_parts=-1 until the last
// part, as described in https://core.telegram.org/api/files#streamed-uploads.
// Streamed uploads cannot be used for photos (inputMediaUploadedPhoto).
//
// Parameters:
//   - ctx: context for cancellation and timeout. When cancelled, in-flight uploads are abandoned
//     and the first context error encountered is returned.
//   - reader: source of the file content. The caller is responsible for closing it.
//   - fileName: name reported to Telegram for the uploaded file. Must not be empty.
//   - fileSize: total size in bytes; pass 0 for streamed (unknown size) uploads. Must be non-negative
//     and not exceed maxFileSize (2 GB) when > 0.
//   - opts: optional upload settings (worker count, progress callback). May be nil for defaults.
//
// Returns:
//   - *UploadResult: metadata about the uploaded file including its handle.
//     For streamed uploads, Size reflects the actual bytes read from the stream.
//   - error: non-nil if the client is disconnected, the file size is invalid, any part fails,
//     or the context is cancelled.
//
// Errors:
//   - client disconnected (from state.requireConnected).
//   - "upload: file size must be non-negative" — when fileSize < 0.
//   - "upload: file size %d exceeds maximum" — when fileSize > 2 GB.
//   - "upload: streamed upload produced no data" — when fileSize == 0 and the reader yields nothing.
//   - "upload: generate file id" — when the random file ID cannot be generated.
//   - "upload: read part %d" — when reading from the source fails.
//   - "upload: part %d: ..." — when any individual part upload RPC fails.
//
// Example:
//
//	f, _ := os.Open("report.pdf")
//	defer f.Close()
//	info, _ := f.Stat()
//	ctx := context.Background()
//	result, err := client.UploadFile(ctx, f, "report.pdf", info.Size(), nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Uploaded %d bytes\n", result.Size)
func (c *Client) UploadFile(ctx context.Context, reader io.Reader, fileName string, fileSize int64, opts *UploadOptions) (*UploadResult, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	c.Log.Debugf("UploadFile size=%d", fileSize)
	rpc := c.Raw()
	inputFile, actualSize, err := uploadFileRPC(ctx, rpc, reader, fileName, fileSize, opts)
	if err != nil {
		return nil, err
	}

	resultSize := fileSize
	if fileSize == 0 {
		resultSize = actualSize
	}
	_, isBig := inputFile.(*tg.InputFileBig)
	return &UploadResult{
		File:  inputFile,
		Size:  resultSize,
		Name:  fileName,
		IsBig: isBig,
	}, nil
}

func uploadFileRPC(ctx context.Context, rpc *tg.RPCClient, reader io.Reader, fileName string, fileSize int64, opts *UploadOptions) (tg.InputFileClass, int64, error) {
	if fileSize < 0 {
		return nil, 0, fmt.Errorf("upload: file size must be non-negative, got %d", fileSize)
	}
	if fileSize > maxFileSize {
		return nil, 0, fmt.Errorf("upload: file size %d exceeds maximum %d", fileSize, maxFileSize)
	}
	if fileSize == 0 {
		return uploadFileStreamRPC(ctx, rpc, reader, fileName, opts)
	}
	return uploadFileKnownRPC(ctx, rpc, reader, fileName, fileSize, opts)
}

func uploadFileStreamRPC(ctx context.Context, rpc *tg.RPCClient, reader io.Reader, fileName string, opts *UploadOptions) (tg.InputFileClass, int64, error) {
	workers := 1
	if opts != nil && opts.Workers > 0 {
		workers = opts.Workers
	}
	if workers > 8 {
		workers = 8
	}

	fileID, err := generateFileID()
	if err != nil {
		return nil, 0, fmt.Errorf("upload: generate file id: %w", err)
	}

	type streamPart struct {
		idx       int32
		data      []byte
		totalSize int64
		isLast    bool
		bufPtr    *[]byte // pool buffer to return after RPC (nil for empty/EOF)
	}
	type partResult struct {
		idx int32
		err error
	}

	jobs := make(chan streamPart)
	var results []partResult
	var resultsMu sync.Mutex
	var wg sync.WaitGroup
	var hasErr atomic.Bool

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					resultsMu.Lock()
					results = append(results, partResult{err: fmt.Errorf("upload worker panic: %v", r)})
					resultsMu.Unlock()
					hasErr.Store(true)
				}
			}()
			for job := range jobs {
				if hasErr.Load() {
					resultsMu.Lock()
					results = append(results, partResult{idx: job.idx})
					resultsMu.Unlock()
					continue
				}

				select {
				case <-ctx.Done():
					resultsMu.Lock()
					results = append(results, partResult{idx: job.idx, err: ctx.Err()})
					resultsMu.Unlock()
					hasErr.Store(true)
					continue
				default:
				}

				totalParts := int32(-1)
				if job.isLast {
					totalParts = int32((job.totalSize + int64(uploadPartSize) - 1) / int64(uploadPartSize))
				}

				_, uploadErr := rpc.UploadSaveBigFilePart(ctx, &tg.UploadSaveBigFilePartRequest{
					FileID:         fileID,
					FilePart:       job.idx,
					FileTotalParts: totalParts,
					Bytes:          job.data,
				})

				if job.bufPtr != nil {
					uploadBufPool.Put(job.bufPtr)
				}

				if uploadErr != nil {
					resultsMu.Lock()
					results = append(results, partResult{idx: job.idx, err: uploadErr})
					resultsMu.Unlock()
					hasErr.Store(true)
					continue
				}

				if opts != nil && opts.Progress != nil {
					opts.Progress(params.ProgressInfo{
						FileName:      fileName,
						TotalBytes:    0,
						UploadedBytes: job.totalSize,
						IsUpload:      true,
					})
				}
			}
		}()
	}

	var totalStreamSize int64
	partIdx := int32(0)

	for !hasErr.Load() {
		poolBuf := uploadBufPool.Get().(*[]byte)

		n, readErr := io.ReadFull(reader, *poolBuf)

		if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
			uploadBufPool.Put(poolBuf)
			close(jobs)
			wg.Wait()
			return nil, 0, fmt.Errorf("upload: read part %d: %w", partIdx, readErr)
		}

		isEOF := readErr == io.EOF || readErr == io.ErrUnexpectedEOF

		if n > 0 {
			totalStreamSize += int64(n)

			jobs <- streamPart{
				idx:       partIdx,
				data:      (*poolBuf)[:n],
				totalSize: totalStreamSize,
				isLast:    isEOF,
				bufPtr:    poolBuf,
			}
			partIdx++
		} else {
			if poolBuf != nil {
				uploadBufPool.Put(poolBuf)
			}
			if isEOF {
				jobs <- streamPart{
					idx:       partIdx,
					data:      []byte{},
					totalSize: totalStreamSize,
					isLast:    true,
				}
				partIdx++
			}
		}

		if isEOF {
			break
		}
	}

	close(jobs)
	wg.Wait()

	if totalStreamSize == 0 {
		return nil, 0, ErrUploadNoData
	}

	for _, r := range results {
		if r.err != nil {
			return nil, 0, fmt.Errorf("upload: part %d: %w", r.idx, r.err)
		}
	}

	return &tg.InputFileBig{
		ID:    fileID,
		Parts: partIdx,
		Name:  fileName,
	}, totalStreamSize, nil
}

func uploadFileKnownRPC(ctx context.Context, rpc *tg.RPCClient, reader io.Reader, fileName string, fileSize int64, opts *UploadOptions) (tg.InputFileClass, int64, error) {
	workers := 1
	if opts != nil && opts.Workers > 0 {
		workers = opts.Workers
	}
	if workers > 8 {
		workers = 8
	}

	fileID, err := generateFileID()
	if err != nil {
		return nil, 0, fmt.Errorf("upload: generate file id: %w", err)
	}

	totalParts := int32(fileSize / uploadPartSize)
	if fileSize%uploadPartSize != 0 {
		totalParts++
	}

	isBig := fileSize >= bigFileThreshold
	var md5Hash hash.Hash
	if !isBig {
		md5Hash = md5.New()
	}

	type partResult struct {
		index int32
		err   error
	}

	sem := make(chan struct{}, workers)
	results := make(chan partResult, totalParts)
	var wg sync.WaitGroup
	var hasErr atomic.Bool

	for i := int32(0); i < totalParts; i++ {
		if hasErr.Load() {
			break
		}

		bufPtr := uploadBufPool.Get().(*[]byte)
		n, readErr := io.ReadFull(reader, *bufPtr)
		if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
			close(sem)
			wg.Wait()
			return nil, 0, fmt.Errorf("upload: read part %d: %w", i, readErr)
		}
		if n == 0 {
			uploadBufPool.Put(bufPtr)
			break
		}

		data := (*bufPtr)[:n]
		if md5Hash != nil {
			md5Hash.Write(data)
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(partIdx int32, data []byte, bufPtr *[]byte) {
			defer wg.Done()
			defer func() { <-sem }()
			defer uploadBufPool.Put(bufPtr)
			defer func() {
				if r := recover(); r != nil {
					results <- partResult{index: partIdx, err: fmt.Errorf("upload part panic: %v", r)}
					hasErr.Store(true)
				}
			}()

			if hasErr.Load() {
				return
			}

			select {
			case <-ctx.Done():
				results <- partResult{index: partIdx, err: ctx.Err()}
				hasErr.Store(true)
				return
			default:
			}

			var uploadErr error
			if isBig {
				_, uploadErr = rpc.UploadSaveBigFilePart(ctx, &tg.UploadSaveBigFilePartRequest{
					FileID:         fileID,
					FilePart:       partIdx,
					FileTotalParts: totalParts,
					Bytes:          data,
				})
			} else {
				_, uploadErr = rpc.UploadSaveFilePart(ctx, &tg.UploadSaveFilePartRequest{
					FileID:   fileID,
					FilePart: partIdx,
					Bytes:    data,
				})
			}

			if uploadErr != nil {
				results <- partResult{index: partIdx, err: uploadErr}
				hasErr.Store(true)
				return
			}

			results <- partResult{index: partIdx}

			if opts != nil && opts.Progress != nil {
				done := min(int64(partIdx+1)*uploadPartSize, fileSize)
				opts.Progress(params.ProgressInfo{
					FileName:      fileName,
					TotalBytes:    fileSize,
					UploadedBytes: done,
					IsUpload:      true,
				})
			}
		}(i, data, bufPtr)
	}

	wg.Wait()
	close(results)

	var firstErr error
	for r := range results {
		if r.err != nil {
			firstErr = fmt.Errorf("upload: part %d: %w", r.index, r.err)
		}
	}
	if firstErr != nil {
		return nil, 0, firstErr
	}

	if hasErr.Load() {
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
			return nil, 0, fmt.Errorf("upload: parts failed")
		}
	}

	if isBig {
		return &tg.InputFileBig{
			ID:    fileID,
			Parts: totalParts,
			Name:  fileName,
		}, fileSize, nil
	}

	var md5Str string
	if md5Hash != nil {
		md5Str = fmt.Sprintf("%x", md5Hash.Sum(nil))
	}

	return &tg.InputFile{
		ID:          fileID,
		Parts:       totalParts,
		Name:        fileName,
		MD5Checksum: md5Str,
	}, fileSize, nil
}

func generateFileID() (int64, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return 0, err
	}
	id := int64(buf[0]) | int64(buf[1])<<8 | int64(buf[2])<<16 | int64(buf[3])<<24 |
		int64(buf[4])<<32 | int64(buf[5])<<40 | int64(buf[6])<<48 | int64(buf[7])<<56
	if id == 0 {
		id = 1
	}
	return id, nil
}

// SendPhoto uploads a photo and sends it to the specified chat.
//
// This is a convenience method that combines UploadFile and SendMedia into a single call.
// The file is uploaded as a photo and sent with an optional caption.
//
// Parameters:
//   - ctx: context for cancellation and timeout.
//   - chatID: target chat identifier (user ID, group ID, or channel ID).
//   - reader: photo file content. The caller is responsible for closing it.
//   - fileName: name of the photo file reported to Telegram.
//   - fileSize: size in bytes of the photo.
//   - caption: optional caption text for the photo. May be empty.
//   - opts: optional upload settings. May be nil for defaults.
//
// Returns:
//   - *types.Message: the sent message object on success.
//   - error: non-nil if the client is disconnected, the upload fails, or sending fails.
//
// Errors:
//   - "send photo: upload: ..." — wrapped upload failure.
//
// Example:
//
//	f, _ := os.Open("photo.jpg")
//	defer f.Close()
//	info, _ := f.Stat()
//	ctx := context.Background()
//	msg, err := client.SendPhoto(ctx, 12345678, f, "photo.jpg", info.Size(), "Sunset at the beach", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Photo sent, message ID: %d\n", msg.ID)
func (c *Client) SendPhoto(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendPhoto) (*types.Message, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	opt := params.GetOptDef(&params.SendPhoto{}, opts...)
	if opt.FileName != "" && file.GetFileName() == "" {
		file.SetFileName(opt.FileName)
	}
	media, err := resolveFile(file, c, mediaKindPhoto)
	if err != nil {
		return nil, fmt.Errorf("send_photo: %w", err)
	}
	return c.SendMedia(ctx, chatID, media, caption, opt.ToSendMsg())
}

func (c *Client) SendDocument(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*types.Message, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	opt := params.GetOptDef(&params.SendDocument{}, opts...)
	if opt.FileName != "" && file.GetFileName() == "" {
		file.SetFileName(opt.FileName)
	}
	media, err := resolveFile(file, c, mediaKindDocument)
	if err != nil {
		return nil, fmt.Errorf("send_document: %w", err)
	}
	if uploaded, ok := media.(*tg.InputMediaUploadedDocument); ok {
		mimeType := opt.MimeType
		if mimeType == "" {
			mimeType = GuessMIMEType(file.GetFileName())
		}
		attrs := buildDocumentAttributes(file.GetFileName(), mimeType, 0, 0)
		media = buildInputMediaUploadedDocument(uploaded.File, nil, mimeType, attrs)
	}
	return c.SendMedia(ctx, chatID, media, caption, opt.ToSendMsg())
}

func (c *Client) SendVideo(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVideo) (*types.Message, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	opt := params.GetOptDef(&params.SendVideo{}, opts...)
	if opt.FileName != "" && file.GetFileName() == "" {
		file.SetFileName(opt.FileName)
	}
	media, err := resolveFile(file, c, mediaKindVideo)
	if err != nil {
		return nil, fmt.Errorf("send_video: %w", err)
	}
	if uploaded, ok := media.(*tg.InputMediaUploadedDocument); ok {
		mimeType := GuessMIMEType(file.GetFileName())
		attrs := buildVideoAttributes(file.GetFileName(), opt.Duration, opt.Width, opt.Height)
		media = buildInputMediaUploadedDocument(uploaded.File, nil, mimeType, attrs)
	}
	return c.SendMedia(ctx, chatID, media, caption, opt.ToSendMsg())
}

func (c *Client) SendAudio(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAudio) (*types.Message, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	opt := params.GetOptDef(&params.SendAudio{}, opts...)
	if opt.FileName != "" && file.GetFileName() == "" {
		file.SetFileName(opt.FileName)
	}
	media, err := resolveFile(file, c, mediaKindAudio)
	if err != nil {
		return nil, fmt.Errorf("send_audio: %w", err)
	}
	if uploaded, ok := media.(*tg.InputMediaUploadedDocument); ok {
		attrs := buildAudioAttributes(file.GetFileName(), opt.Duration, opt.Performer, opt.Title)
		media = buildInputMediaUploadedDocument(uploaded.File, nil, GuessMIMEType(file.GetFileName()), attrs)
	}
	return c.SendMedia(ctx, chatID, media, caption, opt.ToSendMsg())
}

func (c *Client) SendAnimation(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAnimation) (*types.Message, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	opt := params.GetOptDef(&params.SendAnimation{}, opts...)
	if opt.FileName != "" && file.GetFileName() == "" {
		file.SetFileName(opt.FileName)
	}
	media, err := resolveFile(file, c, mediaKindAnimation)
	if err != nil {
		return nil, fmt.Errorf("send_animation: %w", err)
	}
	return c.SendMedia(ctx, chatID, media, caption, opt.ToSendMsg())
}

func (c *Client) SendVoice(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVoice) (*types.Message, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	opt := params.GetOptDef(&params.SendVoice{}, opts...)
	if opt.FileName != "" && file.GetFileName() == "" {
		file.SetFileName(opt.FileName)
	}
	media, err := resolveFile(file, c, mediaKindVoice)
	if err != nil {
		return nil, fmt.Errorf("send_voice: %w", err)
	}
	return c.SendMedia(ctx, chatID, media, caption, opt.ToSendMsg())
}

func (c *Client) SendVideoNote(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendVideoNote) (*types.Message, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	opt := params.GetOptDef(&params.SendVideoNote{}, opts...)
	if opt.FileName != "" && file.GetFileName() == "" {
		file.SetFileName(opt.FileName)
	}
	media, err := resolveFile(file, c, mediaKindVideoNote)
	if err != nil {
		return nil, fmt.Errorf("send_video_note: %w", err)
	}
	return c.SendMedia(ctx, chatID, media, "", opt.ToSendMsg())
}

func (c *Client) SendSticker(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendSticker) (*types.Message, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	opt := params.GetOptDef(&params.SendSticker{}, opts...)
	if opt.FileName != "" && file.GetFileName() == "" {
		file.SetFileName(opt.FileName)
	}
	media, err := resolveFile(file, c, mediaKindSticker)
	if err != nil {
		return nil, fmt.Errorf("send_sticker: %w", err)
	}
	return c.SendMedia(ctx, chatID, media, "", opt.ToSendMsg())
}

func buildInputMediaUploadedPhoto(file tg.InputFileClass) tg.InputMediaClass {
	return &tg.InputMediaUploadedPhoto{
		File: file,
	}
}

func buildInputMediaUploadedDocument(file tg.InputFileClass, thumb tg.InputFileClass, mimeType string, attrs []tg.DocumentAttributeClass) tg.InputMediaClass {
	return &tg.InputMediaUploadedDocument{
		File:       file,
		Thumb:      thumb,
		MimeType:   mimeType,
		Attributes: attrs,
	}
}

func buildDocumentAttributes(fileName, mimeType string, width, height int) []tg.DocumentAttributeClass {
	attrs := []tg.DocumentAttributeClass{
		&tg.DocumentAttributeFilename{FileName: fileName},
	}

	switch {
	case isVideoMIME(mimeType):
		attrs = append(attrs, &tg.DocumentAttributeVideo{
			W:                 int32(width),
			H:                 int32(height),
			SupportsStreaming: true,
		})
	case isAudioMIME(mimeType):
		attrs = append(attrs, &tg.DocumentAttributeAudio{})
	}

	return attrs
}

func isVideoMIME(mime string) bool {
	switch mime {
	case "video/mp4", "video/webm", "video/quicktime", "video/x-msvideo",
		"video/x-matroska", "video/3gpp", "video/3gpp2":
		return true
	}
	return false
}

func isAudioMIME(mime string) bool {
	switch mime {
	case "audio/mpeg", "audio/ogg", "audio/mp4", "audio/x-m4a",
		"audio/flac", "audio/wav", "audio/opus", "audio/aac":
		return true
	}
	return false
}
