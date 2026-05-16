package types

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// ChatPhoto represents a chat or user profile photo with small and big variants,
// stripped thumbnail, and data center information.
//
// Example:
//
//	if chat.Photo != nil {
//	    fmt.Printf("Photo DC: %d, animated: %v\n", chat.Photo.DcID, chat.Photo.HasAnimation)
//	}
type ChatPhoto struct {
	SmallFileID        string
	SmallPhotoUniqueID string
	BigFileID          string
	BigPhotoUniqueID   string
	HasAnimation       bool
	IsPersonal         bool
	StrippedThumb      []byte
	DcID               int32
}

func parseUserProfilePhoto(photo tg.UserProfilePhotoClass) *ChatPhoto {
	if photo == nil {
		return nil
	}
	switch p := photo.(type) {
	case *tg.UserProfilePhotoEmpty:
		return nil
	case *tg.UserProfilePhoto:
		cp := &ChatPhoto{
			HasAnimation:  p.HasVideo,
			IsPersonal:    p.Personal,
			StrippedThumb: p.StrippedThumb,
			DcID:          p.DCID,
		}
		cp.SmallPhotoUniqueID = fmt.Sprintf("%d", p.PhotoID)
		cp.BigPhotoUniqueID = fmt.Sprintf("%d", p.PhotoID)
		return cp
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
		cp := &ChatPhoto{
			HasAnimation:  p.HasVideo,
			StrippedThumb: p.StrippedThumb,
			DcID:          p.DCID,
		}
		cp.SmallPhotoUniqueID = fmt.Sprintf("%d", p.PhotoID)
		cp.BigPhotoUniqueID = fmt.Sprintf("%d", p.PhotoID)
		return cp
	}
	return nil
}

// InputChatPhotoPrevious references a previously used chat photo.
type InputChatPhotoPrevious struct {
	PhotoID int64
}

// InputChatPhotoStatic references a static image file for a chat photo.
type InputChatPhotoStatic struct {
	FileID     int64
	AccessHash int64
}

// InputChatPhotoAnimation references an animated file for a chat photo.
type InputChatPhotoAnimation struct {
	FileID     int64
	AccessHash int64
}
