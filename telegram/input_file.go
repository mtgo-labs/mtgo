package telegram

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mtgo-labs/mtgo/telegram/fileid"
	"github.com/mtgo-labs/mtgo/telegram/types"
	tg "github.com/mtgo-labs/mtgo/tg"
)

// InputFile represents a file input for uploading or referencing media on Telegram.
// Construct one using the builder functions: FileID, FromIDs, URL, Path, Reader, or FromBytes.
//
// InputFile auto-detects the media type (photo, video, audio, document) from the file
// extension or MIME type. It handles upload, URL resolution, and file ID decoding internally.
//
// Example (upload from file path):
//
//	media := telegram.Path("/tmp/photo.jpg")
//	client.SendPhoto(chatID, media)
//
// Example (from existing file ID):
//
//	media := telegram.FileID("AQADBAAT0vJpWA...")
//	client.SendPhoto(chatID, media)
type InputFile = types.InputFile

// FileID creates an InputFile that references an already-uploaded file by its Telegram
// file ID string. This is the fastest option since no upload is required.
//
// Example:
//
//	media := telegram.FileID("AQADBAAT0vJpWA...")
//	client.SendPhoto(chatID, media)
var FileID = types.FileID

// FromIDs creates an InputFile from a file's numeric ID, access hash, and file reference.
// Use this when you have already decoded a file ID or obtained these fields from an API response.
//
// Example:
//
//	media := telegram.FromIDs(123456, 78901234, []byte{0x01, 0x02}, fileid.FileTypePhoto)
//	client.SendPhoto(chatID, media)
var FromIDs = types.FromIDs

// URL creates an InputFile that downloads media from an HTTP or HTTPS URL. The server
// fetches the file directly, so no local upload is needed.
//
// Example:
//
//	media := telegram.URL("https://example.com/image.png")
//	client.SendPhoto(chatID, media)
var URL = types.URL

// Path creates an InputFile that uploads a local file by its filesystem path. The file
// is opened, read, and uploaded to Telegram.
//
// Example:
//
//	media := telegram.Path("/tmp/document.pdf")
//	client.SendDocument(chatID, media)
var Path = types.Path

// Reader creates an InputFile from an io.Reader. Use this for in-memory data such as
// generated images or downloaded content. The fileName is used for MIME type detection.
//
// Example:
//
//	buf := bytes.NewReader(imageData)
//	media := telegram.Reader(buf, "photo.png")
//	client.SendPhoto(chatID, media)
var Reader = types.Reader

// FromBytes creates an InputFile from a raw byte slice. This is a convenience wrapper
// around Reader for cases where the entire file content is already in memory.
//
// Example:
//
//	media := telegram.FromBytes([]byte(data), "audio.mp3")
//	client.SendAudio(chatID, media)
var FromBytes = types.FromBytes

type mediaKind = types.MediaKind

const (
	mediaKindAuto      = types.MediaKindAuto
	mediaKindPhoto     = types.MediaKindPhoto
	mediaKindDocument  = types.MediaKindDocument
	mediaKindAudio     = types.MediaKindAudio
	mediaKindVideo     = types.MediaKindVideo
	mediaKindAnimation = types.MediaKindAnimation
	mediaKindVoice     = types.MediaKindVoice
	mediaKindVideoNote = types.MediaKindVideoNote
	mediaKindSticker   = types.MediaKindSticker
)

func resolveFile(ctx context.Context, f *InputFile, c *Client, kind mediaKind) (tg.InputMediaClass, error) {
	if f.GetFileID() != "" {
		return resolveFromFileID(f, kind)
	}
	if f.GetID() != 0 {
		return resolveFromIDs(f, kind)
	}
	if f.GetURL() != "" {
		return resolveFromURL(f, kind)
	}
	if f.GetPath() != "" {
		if err := openFile(f); err != nil {
			return nil, err
		}
	}
	if f.GetReader() != nil {
		return resolveUpload(ctx, f, c, kind)
	}
	return nil, ErrInputFileEmpty
}

func resolveFromIDs(f *InputFile, kind mediaKind) (tg.InputMediaClass, error) {
	if isPhotoKind(kind, f.GetFileName()) || f.GetFileType() == fileid.FileTypePhoto {
		return &tg.InputMediaPhoto{
			ID: &tg.InputPhoto{
				ID:            f.ID,
				AccessHash:    f.AccessHash,
				FileReference: f.FileRef,
			},
		}, nil
	}
	return &tg.InputMediaDocument{
		ID: &tg.InputDocument{
			ID:            f.ID,
			AccessHash:    f.AccessHash,
			FileReference: f.FileRef,
		},
	}, nil
}

func resolveFromFileID(f *InputFile, kind mediaKind) (tg.InputMediaClass, error) {
	decoded, err := fileid.Decode(f.GetFileID())
	if err != nil {
		return nil, fmt.Errorf("input_file: decode file_id: %w", err)
	}
	f.SetID(decoded.ID)
	f.SetAccessHash(decoded.AccessHash)
	f.SetFileType(decoded.Type)
	return resolveFromIDs(f, kind)
}

func resolveFromURL(f *InputFile, kind mediaKind) (tg.InputMediaClass, error) {
	u := f.GetURL()
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return nil, fmt.Errorf("input_file: invalid URL: %s", u)
	}
	if isPhotoKind(kind, f.GetFileName()) {
		return &tg.InputMediaPhotoExternal{URL: u}, nil
	}
	return &tg.InputMediaDocumentExternal{URL: u}, nil
}

func openFile(f *InputFile) error {
	file, err := os.Open(f.GetPath())
	if err != nil {
		return fmt.Errorf("input_file: open %s: %w", f.GetPath(), err)
	}
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("input_file: stat %s: %w", f.GetPath(), err)
	}
	if f.GetFileName() == "" {
		f.SetFileName(stat.Name())
	}
	f.SetReader(file)
	f.SetFileSize(stat.Size())
	return nil
}

func resolveUpload(ctx context.Context, f *InputFile, c *Client, kind mediaKind) (tg.InputMediaClass, error) {
	if kind == mediaKindAuto {
		kind = guessKind(f.GetFileName())
	}

	result, err := c.UploadFile(ctx, f.GetReader(), f.GetFileName(), f.GetFileSize(), nil)
	if err != nil {
		return nil, fmt.Errorf("input_file: upload: %w", err)
	}

	mimeType := GuessMIMEType(f.GetFileName())

	switch kind {
	case mediaKindPhoto:
		return buildInputMediaUploadedPhoto(result.File), nil
	case mediaKindAudio:
		attrs := buildAudioAttributes(f.GetFileName(), 0, "", "")
		return buildInputMediaUploadedDocument(result.File, nil, mimeType, attrs), nil
	case mediaKindVideo:
		attrs := buildVideoAttributes(f.GetFileName(), 0, 0, 0)
		return buildInputMediaUploadedDocument(result.File, nil, mimeType, attrs), nil
	case mediaKindVoice:
		attrs := []tg.DocumentAttributeClass{&tg.DocumentAttributeAudio{Voice: true}}
		return buildInputMediaUploadedDocument(result.File, nil, mimeType, attrs), nil
	case mediaKindVideoNote:
		attrs := []tg.DocumentAttributeClass{&tg.DocumentAttributeVideo{RoundMessage: true}}
		return buildInputMediaUploadedDocument(result.File, nil, mimeType, attrs), nil
	case mediaKindSticker:
		attrs := []tg.DocumentAttributeClass{&tg.DocumentAttributeSticker{}}
		return buildInputMediaUploadedDocument(result.File, nil, mimeType, attrs), nil
	case mediaKindAnimation:
		attrs := []tg.DocumentAttributeClass{&tg.DocumentAttributeAnimated{}}
		return buildInputMediaUploadedDocument(result.File, nil, mimeType, attrs), nil
	default:
		attrs := buildDocumentAttributes(f.GetFileName(), mimeType, 0, 0)
		return buildInputMediaUploadedDocument(result.File, nil, mimeType, attrs), nil
	}
}

func isPhotoKind(kind mediaKind, name string) bool {
	if kind == mediaKindPhoto {
		return true
	}
	if kind != mediaKindAuto {
		return false
	}
	ext := strings.ToLower(name)
	return strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg") ||
		strings.HasSuffix(ext, ".png") || strings.HasSuffix(ext, ".webp")
}

func guessKind(name string) mediaKind {
	mime := strings.ToLower(GuessMIMEType(name))
	switch {
	case strings.HasPrefix(mime, "image/"):
		return mediaKindPhoto
	case strings.HasPrefix(mime, "video/"):
		return mediaKindVideo
	case strings.HasPrefix(mime, "audio/"):
		return mediaKindAudio
	default:
		return mediaKindDocument
	}
}

func buildAudioAttributes(fileName string, duration int32, performer, title string) []tg.DocumentAttributeClass {
	attrs := []tg.DocumentAttributeClass{
		&tg.DocumentAttributeAudio{
			Duration:  duration,
			Performer: performer,
			Title:     title,
		},
	}
	attrs = append(attrs, &tg.DocumentAttributeFilename{FileName: fileName})
	return attrs
}

func buildVideoAttributes(fileName string, duration float64, width, height int32) []tg.DocumentAttributeClass {
	attrs := []tg.DocumentAttributeClass{
		&tg.DocumentAttributeVideo{
			Duration:          duration,
			W:                 width,
			H:                 height,
			SupportsStreaming: true,
		},
	}
	attrs = append(attrs, &tg.DocumentAttributeFilename{FileName: fileName})
	return attrs
}
