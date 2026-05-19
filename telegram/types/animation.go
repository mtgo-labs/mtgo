package types

import (
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/fileid"
	"github.com/mtgo-labs/mtgo/tg"
)

// Animation represents an animated GIF or video animation attached to a message,
// with its dimensions, duration, thumbnails, and file metadata.
//
// Example:
//
//	anim := types.ParseAnimation(rawDoc)
//	fmt.Printf("Animation: %dx%d, duration: %ds, file: %s\n", anim.Width, anim.Height, anim.Duration, anim.FileName)
type Animation struct {
	FileID       string
	FileUniqueID string
	Width        int32
	Height       int32
	Duration     int32
	FileName     string
	MimeType     string
	FileSize     int64
	Date         time.Time
	Thumbs       []*Thumbnail
	Raw          *tg.Document
}

// ParseAnimation converts an MTProto Document into an Animation.
// Extracts dimensions, duration, filename, thumbnails, and file metadata from
// the document's attributes. Returns nil if doc is nil.
//
// Example:
//
//	anim := types.ParseAnimation(doc)
//	if anim != nil {
//	    fmt.Println(anim.FileName, anim.Width, "x", anim.Height)
//	}
func ParseAnimation(doc *tg.Document) *Animation {
	if doc == nil {
		return nil
	}
	a := &Animation{
		FileSize: doc.Size,
		MimeType: doc.MimeType,
		Date:     time.Unix(int64(doc.Date), 0),
		Raw:      doc,
	}
	if encoded, err := fileid.Encode(fileid.FileID{
		Type:          fileid.FileTypeAnimation,
		DCID:          doc.DCID,
		ID:            doc.ID,
		AccessHash:    doc.AccessHash,
		FileReference: doc.FileReference,
	}); err == nil {
		a.FileID = encoded
	}
	a.FileUniqueID = fmt.Sprintf("%d", doc.ID)
	for _, t := range doc.Thumbs {
		if th := ParseThumbnail(t); th != nil {
			a.Thumbs = append(a.Thumbs, th)
		}
	}
	for _, attr := range doc.Attributes {
		switch v := attr.(type) {
		case *tg.DocumentAttributeVideo:
			a.Width = v.W
			a.Height = v.H
			a.Duration = int32(v.Duration)
		case *tg.DocumentAttributeFilename:
			a.FileName = v.FileName
		case *tg.DocumentAttributeImageSize:
			a.Width = v.W
			a.Height = v.H
		}
	}
	return a
}
