package types

import (
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/fileid"
	"github.com/mtgo-labs/mtgo/tg"
)

// Sticker represents a Telegram sticker with its dimensions, emoji, set name,
// type (regular, mask, custom emoji), and optional premium animation.
//
// Example:
//
//	s := types.ParseSticker(rawDoc)
//	fmt.Printf("Sticker: %s (%dx%d, set: %s)\n", s.Emoji, s.Width, s.Height, s.SetName)
type Sticker struct {
	FileID           string
	FileUniqueID     string
	Type             StickerType
	Width            int32
	Height           int32
	IsAnimated       bool
	IsVideo          bool
	FileName         string
	MimeType         string
	FileSize         int64
	Date             time.Time
	Emoji            string
	SetName          string
	PremiumAnimation *DocumentMedia
	MaskPosition     *MaskPosition
	CustomEmojiID    string
	NeedsRepainting  bool
	Thumbs           []*Thumbnail
	Raw              *tg.Document
}

func maskPoint(n int32) MaskPointType {
	switch n {
	case 0:
		return MaskPointForehead
	case 1:
		return MaskPointEyes
	case 2:
		return MaskPointMouth
	case 3:
		return MaskPointChin
	default:
		return ""
	}
}

// MaskPosition describes where a mask sticker is placed on a user's face photo,
// including shifts and scale.
type MaskPosition struct {
	Point  MaskPointType
	XShift float64
	YShift float64
	Scale  float64
}

// ParseSticker converts a TL Document into a Sticker, extracting dimensions,
// emoji, set name, mask position, and premium animation from document attributes.
// Returns nil if doc is nil.
//
// Example:
//
//	sticker := types.ParseSticker(doc)
//	if sticker != nil {
//	    fmt.Printf("%s from set %s\n", sticker.Emoji, sticker.SetName)
//	}
func ParseSticker(doc *tg.Document) *Sticker {
	if doc == nil {
		return nil
	}
	s := &Sticker{
		FileSize: doc.Size,
		MimeType: doc.MimeType,
		Raw:      doc,
	}
	if doc.Date != 0 {
		s.Date = time.Unix(int64(doc.Date), 0)
	}
	if encoded, err := fileid.Encode(fileid.FileID{
		Type:          fileid.FileTypeSticker,
		DCID:          doc.DCID,
		ID:            doc.ID,
		AccessHash:    doc.AccessHash,
		FileReference: doc.FileReference,
	}); err == nil {
		s.FileID = encoded
	}
	s.FileUniqueID = fmt.Sprintf("%d", doc.ID)
	for _, t := range doc.Thumbs {
		if th := ParseThumbnail(t); th != nil {
			s.Thumbs = append(s.Thumbs, th)
		}
	}
	for _, vt := range doc.VideoThumbs {
		if vs, ok := vt.(*tg.VideoSize); ok {
			s.PremiumAnimation = &DocumentMedia{
				FileSize:    int64(vs.Size),
				RawDocument: doc,
			}
			break
		}
	}
	for _, attr := range doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeSticker:
			s.Emoji = a.Alt
			s.Type = StickerTypeRegular
			if a.Mask {
				s.Type = StickerTypeMask
			}
			if ss, ok := a.Stickerset.(*tg.InputStickerSetShortName); ok {
				s.SetName = ss.ShortName
			}
			if a.MaskCoords != nil {
				s.MaskPosition = &MaskPosition{
					Point:  maskPoint(a.MaskCoords.N),
					XShift: a.MaskCoords.X,
					YShift: a.MaskCoords.Y,
					Scale:  a.MaskCoords.Zoom,
				}
			}
		case *tg.DocumentAttributeCustomEmoji:
			s.Emoji = a.Alt
			s.Type = StickerTypeCustomEmoji
			s.CustomEmojiID = fmt.Sprintf("%d", doc.ID)
			s.NeedsRepainting = a.TextColor
			if ss, ok := a.Stickerset.(*tg.InputStickerSetShortName); ok {
				s.SetName = ss.ShortName
			}
		case *tg.DocumentAttributeImageSize:
			s.Width = a.W
			s.Height = a.H
		case *tg.DocumentAttributeFilename:
			s.FileName = a.FileName
		}
	}
	if doc.MimeType == "application/x-tgsticker" {
		s.IsAnimated = true
	}
	if doc.MimeType == "video/webm" {
		s.IsVideo = true
	}
	return s
}
