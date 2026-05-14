package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// ChatInviteLink represents an exported invite link for a chat, including its
// usage statistics and creator information.
type ChatInviteLink struct {
	// InviteLink is the full invite URL (e.g. "https://t.me/+AbCdEf...").
	InviteLink string
	// Creator is the user who created this link.
	Creator *User
	// CreatesJoinRequest is true when users joining via this link must be
	// approved by an admin.
	CreatesJoinRequest bool
	// IsPrimary is true when this is the primary (non-revocable) invite link for
	// the chat.
	IsPrimary bool
	// IsRevoked is true when the link has been revoked and can no longer be used.
	IsRevoked bool
	// Name is the optional display name for the link.
	Name string
	// MemberCount is the number of users who have joined via this link.
	MemberCount int
	// MemberLimit is the maximum number of users allowed to join via this link,
	// or 0 for unlimited.
	MemberLimit int
	// CreatedAt is when the link was created.
	CreatedAt time.Time
	// ExpiresAt is when the link expires, or zero if it never expires.
	ExpiresAt time.Time
}

// ParseChatInviteLink converts an MTProto exported chat invite to a
// ChatInviteLink. The users map is used to resolve the creator of the link.
// Returns nil if raw is nil.
func ParseChatInviteLink(raw *tg.ChatInviteExported, users map[int64]tg.UserClass) *ChatInviteLink {
	if raw == nil {
		return nil
	}
	link := &ChatInviteLink{
		InviteLink:         raw.Link,
		CreatesJoinRequest: raw.RequestNeeded,
		IsPrimary:          raw.Permanent,
		IsRevoked:          raw.Revoked,
	}
	if raw.Title != "" {
		link.Name = raw.Title
	}
	if raw.UsageLimit != 0 {
		link.MemberLimit = int(raw.UsageLimit)
	}
	if raw.Usage != 0 {
		link.MemberCount = int(raw.Usage)
	}
	if raw.Date != 0 {
		link.CreatedAt = time.Unix(int64(raw.Date), 0)
	}
	if raw.ExpireDate != 0 {
		link.ExpiresAt = time.Unix(int64(raw.ExpireDate), 0)
	}
	if users != nil {
		if u, ok := users[raw.AdminID]; ok {
			link.Creator = ParseUser(u)
		}
	}
	return link
}
