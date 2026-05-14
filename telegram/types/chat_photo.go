package types

import "github.com/mtgo-labs/mtgo/tg"

// ChatPhoto represents a chat or user profile photo with small and big variants,
// an optional stripped thumbnail, and the DC (data center) where the photo is
// stored.
type ChatPhoto struct {
	// SmallFileID is the file reference for the 160×160 px variant.
	SmallFileID string
	// SmallPhotoUniqueID is a stable unique identifier for deduplication of the
	// small variant.
	SmallPhotoUniqueID string
	// BigFileID is the file reference for the 640×640 px variant.
	BigFileID string
	// BigPhotoUniqueID is a stable unique identifier for deduplication of the big
	// variant.
	BigPhotoUniqueID string
	// StrippedThumb is a JPEG thumbnail stripped of metadata, used for inline
	// previews without downloading the full photo.
	StrippedThumb []byte
	// HasVideo is true when the profile photo is an animated video.
	HasVideo bool
	// DcID is the data center ID where the photo file is stored.
	DcID int32
}

func parseUserProfilePhoto(photo tg.UserProfilePhotoClass) *ChatPhoto {
	if photo == nil {
		return nil
	}
	switch p := photo.(type) {
	case *tg.UserProfilePhotoEmpty:
		return nil
	case *tg.UserProfilePhoto:
		return &ChatPhoto{
			StrippedThumb: p.StrippedThumb,
			HasVideo:      p.HasVideo,
			DcID:          p.DCID,
		}
	}
	return nil
}

func parseChatPhoto(photo tg.ChatPhotoClass) *ChatPhoto {
	if photo == nil {
		return nil
	}
	switch p := photo.(type) {
	case *tg.ChatPhotoEmpty:
		return nil
	case *tg.ChatPhoto:
		return &ChatPhoto{
			StrippedThumb: p.StrippedThumb,
			HasVideo:      p.HasVideo,
			DcID:          p.DCID,
		}
	}
	return nil
}
