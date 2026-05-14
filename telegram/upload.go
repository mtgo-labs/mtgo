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
// The entire file is read into memory and split into fixed-size parts (uploadPartSize, 512 KB).
// Parts are uploaded in parallel using up to opts.Workers goroutines. For files at or above
// bigFileThreshold (10 MB), the big-file upload path is used which skips MD5 hashing.
//
// Parameters:
//   - ctx: context for cancellation and timeout. When cancelled, in-flight uploads are abandoned
//     and the first context error encountered is returned.
//   - reader: source of the file content. The caller is responsible for closing it.
//   - fileName: name reported to Telegram for the uploaded file. Must not be empty.
//   - fileSize: total size in bytes; must be positive and not exceed maxFileSize (2 GB).
//   - opts: optional upload settings (worker count, progress callback). May be nil for defaults.
//
// Returns:
//   - *UploadResult: metadata about the uploaded file including its handle.
//   - error: non-nil if the client is disconnected, the file size is invalid, any part fails,
//     or the context is cancelled.
//
// Errors:
//   - client disconnected (from state.requireConnected).
//   - "upload: file size must be positive" — when fileSize <= 0.
//   - "upload: file size %d exceeds maximum" — when fileSize > 2 GB.
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
	inputFile, err := uploadFileRPC(ctx, rpc, reader, fileName, fileSize, opts)
	if err != nil {
		c.Log.Warnf("UploadFile failed err=%v", err)
		return nil, err
	}

	_, isBig := inputFile.(*tg.InputFileBig)
	return &UploadResult{
		File:  inputFile,
		Size:  fileSize,
		Name:  fileName,
		IsBig: isBig,
	}, nil
}

func uploadFileRPC(ctx context.Context, rpc *tg.RPCClient, reader io.Reader, fileName string, fileSize int64, opts *UploadOptions) (tg.InputFileClass, error) {
	if fileSize <= 0 {
		return nil, fmt.Errorf("upload: file size must be positive, got %d", fileSize)
	}
	if fileSize > maxFileSize {
		return nil, fmt.Errorf("upload: file size %d exceeds maximum %d", fileSize, maxFileSize)
	}

	workers := 1
	if opts != nil && opts.Workers > 0 {
		workers = opts.Workers
	}
	if workers > 8 {
		workers = 8
	}

	fileID, err := generateFileID()
	if err != nil {
		return nil, fmt.Errorf("upload: generate file id: %w", err)
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

	buf := make([]byte, uploadPartSize)
	partData := make([][]byte, totalParts)

	for i := int32(0); i < totalParts; i++ {
		n, err := io.ReadFull(reader, buf)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return nil, fmt.Errorf("upload: read part %d: %w", i, err)
		}
		if n == 0 {
			break
		}
		chunk := make([]byte, n)
		copy(chunk, buf[:n])
		partData[i] = chunk
		if md5Hash != nil {
			md5Hash.Write(chunk)
		}
	}

	type partResult struct {
		index int32
		err   error
	}

	sem := make(chan struct{}, workers)
	results := make([]partResult, totalParts)
	var wg sync.WaitGroup
	var hasErr atomic.Bool

	for i := int32(0); i < totalParts; i++ {
		if hasErr.Load() {
			break
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(partIdx int32, data []byte) {
			defer wg.Done()
			defer func() { <-sem }()

			if hasErr.Load() {
				return
			}

			select {
			case <-ctx.Done():
				results[partIdx] = partResult{index: partIdx, err: ctx.Err()}
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
				results[partIdx] = partResult{index: partIdx, err: uploadErr}
				hasErr.Store(true)
				return
			}

			results[partIdx] = partResult{index: partIdx}

			if opts != nil && opts.Progress != nil {
				done := min(int64(partIdx+1)*uploadPartSize, fileSize)
				opts.Progress(params.ProgressInfo{
					FileName:      fileName,
					TotalBytes:    fileSize,
					UploadedBytes: done,
					IsUpload:      true,
				})
			}
		}(i, partData[i])
	}

	wg.Wait()

	for _, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("upload: part %d: %w", r.index, r.err)
		}
	}

	if isBig {
		return &tg.InputFileBig{
			ID:    fileID,
			Parts: totalParts,
			Name:  fileName,
		}, nil
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
	}, nil
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
