package types

import "github.com/mtgo-labs/mtgo/tg"

// ChatInviteLinkJoiner records a single user who joined a chat via an invite
// link, including optional introductory text they provided.
type ChatInviteLinkJoiner struct {
	// UserID is the Telegram user ID of the joiner.
	UserID int64
	// Date is the Unix timestamp when the user joined.
	Date int32
	// About is the optional introductory message the user submitted when
	// requesting to join.
	About string
}

// ParseChatInviteImporter converts an MTProto ChatInviteImporter to a
// ChatInviteLinkJoiner. Returns nil if raw is nil.
func ParseChatInviteImporter(raw *tg.ChatInviteImporter) *ChatInviteLinkJoiner {
	if raw == nil {
		return nil
	}
	j := &ChatInviteLinkJoiner{
		UserID: raw.UserID,
		Date:   raw.Date,
	}
	if raw.About != "" {
		j.About = raw.About
	}
	return j
}
