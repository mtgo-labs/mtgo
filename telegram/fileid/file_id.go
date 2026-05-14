// Package fileid implements encoding and decoding of Telegram file_id strings.
//
// Telegram uses an opaque base64-encoded binary format in file_id values to
// locate files on its CDN. This package decodes that format into a structured
// FileID type exposing the file type, DC ID, access hash, and volume/local
// references. It also supports re-encoding back to the wire format.
//
// The encoding version (currently 4) is embedded in the decoded output and
// respected during encoding. Sub-types 32 and 33 are handled for photo-size
// sources used by thumbnails, dialog photos, and sticker-set thumbnails.
package fileid

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

const (
	fileIDMajor = 4
	fileIDMinor = 30
)

// PhotoSizeSource holds the origin metadata for a photo-size file, describing
// where the thumbnail or photo data comes from (legacy secret, dialog photo,
// sticker set thumbnail, etc.). Only a subset of fields is populated depending
// on the Type value.
type PhotoSizeSource struct {
	Type                 ThumbnailSource
	Secret               int64
	VolumeID             int64
	LocalID              int32
	PhotoID              int64
	ChatID               int64
	ChatAccessHash       int64
	StickerSetID         int64
	StickerSetAccessHash int64
	ThumbnailFileType    FileType
	ThumbnailSize        int32
}

// FileID represents a Telegram bot API file_id, carrying the decoded components
// that uniquely identify a file on Telegram's backend: its type, data-center,
// internal ID, access hash, and optional photo-source metadata.
type FileID struct {
	Type       FileType
	DCID       int32
	ID         int64
	AccessHash int64
	VolumeID   int64
	LocalID    int32
	Source     PhotoSizeSource
}

func rleEncode(data []byte) []byte {
	var buf bytes.Buffer
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			count := byte(1)
			for i+int(count) < len(data) && data[i+int(count)] == 0 && count < 254 {
				count++
			}
			buf.WriteByte(0)
			buf.WriteByte(count)
			i += int(count - 1)
		} else {
			buf.WriteByte(data[i])
		}
	}
	return buf.Bytes()
}

func rleDecode(data []byte) []byte {
	var buf bytes.Buffer
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			if i+1 < len(data) {
				count := data[i+1]
				for j := 0; j < int(count); j++ {
					buf.WriteByte(0)
				}
				i++
			}
		} else {
			buf.WriteByte(data[i])
		}
	}
	return buf.Bytes()
}

func packLE(w *bytes.Buffer, v interface{}) {
	switch x := v.(type) {
	case int8:
		w.WriteByte(byte(x))
	case int32:
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], uint32(x))
		w.Write(b[:])
	case int64:
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(x))
		w.Write(b[:])
	}
}

func unpackLE(r *bytes.Reader, v interface{}) error {
	switch x := v.(type) {
	case *int8:
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		*x = int8(b)
	case *int32:
		var b [4]byte
		if _, err := r.Read(b[:]); err != nil {
			return err
		}
		*x = int32(binary.LittleEndian.Uint32(b[:]))
	case *int64:
		var b [8]byte
		if _, err := r.Read(b[:]); err != nil {
			return err
		}
		*x = int64(binary.LittleEndian.Uint64(b[:]))
	}
	return nil
}

// Encode serializes a FileID into the Bot API file_id string format used by
// Telegram (RLE + raw standard base64, version 4.30). It returns the encoded
// string. The returned error is currently always nil but is reserved for
// future validation.
func Encode(f FileID) (string, error) {
	var buf bytes.Buffer
	packLE(&buf, int32(f.Type))
	packLE(&buf, f.DCID)
	packLE(&buf, f.ID)
	packLE(&buf, f.AccessHash)

	if f.Type.IsPhoto() {
		packLE(&buf, f.VolumeID)
		packLE(&buf, int32(f.Source.Type))
		switch f.Source.Type {
		case ThumbnailSourceLegacy:
			packLE(&buf, f.Source.Secret)
			packLE(&buf, f.Source.LocalID)
		case ThumbnailSourceThumbnail:
			packLE(&buf, int32(f.Source.ThumbnailFileType))
			packLE(&buf, f.Source.ThumbnailSize)
			packLE(&buf, f.Source.LocalID)
		case ThumbnailSourceDialogPhotoSmall, ThumbnailSourceDialogPhotoBig:
			packLE(&buf, f.Source.ChatID)
			packLE(&buf, f.Source.ChatAccessHash)
			packLE(&buf, f.Source.LocalID)
		case ThumbnailSourceStickerSetThumb:
			packLE(&buf, f.Source.StickerSetID)
			packLE(&buf, f.Source.StickerSetAccessHash)
			packLE(&buf, f.Source.LocalID)
		}
	}

	packLE(&buf, int8(fileIDMinor))
	packLE(&buf, int8(fileIDMajor))

	encoded := base64.RawStdEncoding.EncodeToString(rleEncode(buf.Bytes()))
	return encoded, nil
}

// Decode parses a Bot API file_id string produced by Telegram and returns the
// corresponding FileID. It returns an error if the string cannot be base64-
// decoded, the data is too short, or any required field cannot be read.
func Decode(s string) (FileID, error) {
	decoded, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil {
		return FileID{}, fmt.Errorf("fileid: base64 decode: %w", err)
	}

	data := rleDecode(decoded)
	if len(data) < 2 {
		return FileID{}, fmt.Errorf("fileid: data too short")
	}

	major := data[len(data)-1]
	if major >= 4 {
		data = data[:len(data)-2]
	} else {
		data = data[:len(data)-1]
	}

	r := bytes.NewReader(data)

	var fileType int32
	if err := unpackLE(r, &fileType); err != nil {
		return FileID{}, fmt.Errorf("fileid: read type: %w", err)
	}
	f := FileID{Type: FileType(fileType)}

	if err := unpackLE(r, &f.DCID); err != nil {
		return FileID{}, fmt.Errorf("fileid: read dc_id: %w", err)
	}
	if err := unpackLE(r, &f.ID); err != nil {
		return FileID{}, fmt.Errorf("fileid: read id: %w", err)
	}
	if err := unpackLE(r, &f.AccessHash); err != nil {
		return FileID{}, fmt.Errorf("fileid: read access_hash: %w", err)
	}

	if f.Type.IsPhoto() {
		if err := unpackLE(r, &f.VolumeID); err != nil {
			return FileID{}, fmt.Errorf("fileid: read volume_id: %w", err)
		}
		if major >= 4 {
			var srcType int32
			if err := unpackLE(r, &srcType); err != nil {
				return FileID{}, fmt.Errorf("fileid: read source type: %w", err)
			}
			f.Source.Type = ThumbnailSource(srcType)

			switch f.Source.Type {
			case ThumbnailSourceLegacy:
				if err := unpackLE(r, &f.Source.Secret); err != nil {
					return FileID{}, fmt.Errorf("fileid: read secret: %w", err)
				}
				if err := unpackLE(r, &f.Source.LocalID); err != nil {
					return FileID{}, fmt.Errorf("fileid: read local_id: %w", err)
				}
			case ThumbnailSourceThumbnail:
				var thumbFileType int32
				if err := unpackLE(r, &thumbFileType); err != nil {
					return FileID{}, fmt.Errorf("fileid: read thumb file type: %w", err)
				}
				f.Source.ThumbnailFileType = FileType(thumbFileType)
				if err := unpackLE(r, &f.Source.ThumbnailSize); err != nil {
					return FileID{}, fmt.Errorf("fileid: read thumb size: %w", err)
				}
				if err := unpackLE(r, &f.Source.LocalID); err != nil {
					return FileID{}, fmt.Errorf("fileid: read local_id: %w", err)
				}
			case ThumbnailSourceDialogPhotoSmall, ThumbnailSourceDialogPhotoBig:
				if err := unpackLE(r, &f.Source.ChatID); err != nil {
					return FileID{}, fmt.Errorf("fileid: read chat_id: %w", err)
				}
				if err := unpackLE(r, &f.Source.ChatAccessHash); err != nil {
					return FileID{}, fmt.Errorf("fileid: read chat_access_hash: %w", err)
				}
				if err := unpackLE(r, &f.Source.LocalID); err != nil {
					return FileID{}, fmt.Errorf("fileid: read local_id: %w", err)
				}
			case ThumbnailSourceStickerSetThumb:
				if err := unpackLE(r, &f.Source.StickerSetID); err != nil {
					return FileID{}, fmt.Errorf("fileid: read sticker_set_id: %w", err)
				}
				if err := unpackLE(r, &f.Source.StickerSetAccessHash); err != nil {
					return FileID{}, fmt.Errorf("fileid: read sticker_set_access_hash: %w", err)
				}
				if err := unpackLE(r, &f.Source.LocalID); err != nil {
					return FileID{}, fmt.Errorf("fileid: read local_id: %w", err)
				}
			}
		}
	}

	return f, nil
}
