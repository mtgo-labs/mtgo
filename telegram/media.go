package telegram

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// GetFileLocation resolves a file location from the given Media for downloading.
// It returns an InputFileLocationClass describing where to fetch the file, the DC ID
// (currently always 0 for types.Media), or an error if the media type is unsupported or nil.
//
// This function is the primary entry point for resolving download coordinates from the
// high-level types.Media interface. Use it when you already hold a PhotoMedia or
// DocumentMedia and need the low-level InputFileLocationClass required by download methods.
//
// Parameters:
//   - media: the media object to resolve (PhotoMedia or DocumentMedia).
//   - thumbSize: requested thumbnail size for photos (e.g. "m", "x"). Ignored for documents.
//     If empty, the largest available photo size is selected.
//
// Returns:
//   - tg.InputFileLocationClass: the resolved file location suitable for download RPC calls.
//   - int32: the datacenter ID that holds the file (always 0 for types.Media values).
//   - error: non-nil if media is nil, the media type is unsupported, or internal data is missing.
//
// Errors:
//   - "media: nil media" — when media is nil.
//   - "media: unsupported type %T" — when the media is neither *PhotoMedia nor *DocumentMedia.
//   - "media: photo has no photo data" — when PhotoMedia.Photo is nil.
//   - "media: no photo sizes available" — when the photo has zero sizes.
//   - "media: cannot parse document file_id" — when DocumentMedia.FileID cannot be parsed.
//
// Example:
//
//	location, dc, err := telegram.GetFileLocation(photoMedia, "m")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("File on DC %d\n", dc)
//	data, _ := client.DownloadFile(ctx, location, 0, nil)
func GetFileLocation(media types.Media, thumbSize string) (tg.InputFileLocationClass, int32, error) {
	if media == nil {
		return nil, 0, fmt.Errorf("media: nil media")
	}

	switch m := media.(type) {
	case *types.PhotoMedia:
		return getPhotoFileLocation(m, thumbSize)
	case *types.DocumentMedia:
		return getDocumentFileLocation(m, thumbSize)
	default:
		return nil, 0, fmt.Errorf("media: unsupported type %T", media)
	}
}

func getPhotoFileLocation(m *types.PhotoMedia, thumbSize string) (tg.InputFileLocationClass, int32, error) {
	if m.Photo == nil {
		return nil, 0, fmt.Errorf("media: photo has no photo data")
	}

	size := getPhotoSize(m.Photo.Sizes, thumbSize)
	if size == nil {
		return nil, 0, fmt.Errorf("media: no photo sizes available")
	}

	return &tg.InputPhotoFileLocation{
		ID:            m.Photo.ID,
		AccessHash:    m.Photo.AccessHash,
		FileReference: nil,
		ThumbSize:     size.Type,
	}, 0, nil
}

func getDocumentFileLocation(m *types.DocumentMedia, thumbSize string) (tg.InputFileLocationClass, int32, error) {
	var id, accessHash int64
	var fileRef []byte

	if m.RawDocument != nil {
		id = m.RawDocument.ID
		accessHash = m.RawDocument.AccessHash
		fileRef = m.RawDocument.FileReference
	} else {
		_, err := fmt.Sscanf(m.FileID, "%d_%d", &id, &accessHash)
		if err != nil {
			return nil, 0, fmt.Errorf("media: cannot parse document file_id %q: %w", m.FileID, err)
		}
	}

	return &tg.InputDocumentFileLocation{
		ID:            id,
		AccessHash:    accessHash,
		FileReference: fileRef,
		ThumbSize:     thumbSize,
	}, 0, nil
}

func getPhotoSize(sizes []types.PhotoSize, thumbSize string) *types.PhotoSize {
	if len(sizes) == 0 {
		return nil
	}

	if thumbSize != "" {
		for i := range sizes {
			if sizes[i].Type == thumbSize {
				return &sizes[i]
			}
		}
	}

	largest := &sizes[0]
	for i := 1; i < len(sizes); i++ {
		if sizes[i].Size > largest.Size {
			largest = &sizes[i]
		}
	}
	return largest
}

func getFileLocationFromTL(media tg.MessageMediaClass, thumbSize string) (tg.InputFileLocationClass, int32, error) {
	switch m := media.(type) {
	case *tg.MessageMediaPhoto:
		if m.Photo == nil {
			return nil, 0, fmt.Errorf("media: photo is nil")
		}
		photo, ok := m.Photo.(*tg.Photo)
		if !ok {
			return nil, 0, fmt.Errorf("media: unexpected photo type %T", m.Photo)
		}

		chosen := pickTLPhotoSize(photo.Sizes, thumbSize)

		return &tg.InputPhotoFileLocation{
			ID:            photo.ID,
			AccessHash:    photo.AccessHash,
			FileReference: photo.FileReference,
			ThumbSize:     chosen,
		}, photo.DCID, nil

	case *tg.MessageMediaDocument:
		if m.Document == nil {
			return nil, 0, fmt.Errorf("media: document is nil")
		}
		doc, ok := m.Document.(*tg.Document)
		if !ok {
			return nil, 0, fmt.Errorf("media: unexpected document type %T", m.Document)
		}

		return &tg.InputDocumentFileLocation{
			ID:            doc.ID,
			AccessHash:    doc.AccessHash,
			FileReference: doc.FileReference,
			ThumbSize:     thumbSize,
		}, doc.DCID, nil

	default:
		return nil, 0, fmt.Errorf("media: unsupported TL media type %T", media)
	}
}

func pickTLPhotoSize(sizes []tg.PhotoSizeClass, thumbSize string) string {
	if len(sizes) == 0 {
		return ""
	}

	var best string
	var bestSize int32

	for _, s := range sizes {
		switch sz := s.(type) {
		case *tg.PhotoSize:
			if thumbSize != "" && sz.Type == thumbSize {
				return sz.Type
			}
			if sz.Size > bestSize {
				bestSize = sz.Size
				best = sz.Type
			}
		case *tg.PhotoSizeProgressive:
			if thumbSize != "" && sz.Type == thumbSize {
				return sz.Type
			}
			if len(sz.Sizes) > 0 {
				last := sz.Sizes[len(sz.Sizes)-1]
				if last > bestSize {
					bestSize = last
					best = sz.Type
				}
			}
		}
	}

	return best
}
