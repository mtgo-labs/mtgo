package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// ChatInviteLink represents a chat invite link with its metadata, usage limits,
// creator, and expiration.
//
// Example:
//
//	link := types.ParseChatInviteLink(rawLink, users)
//	fmt.Printf("Invite: %s (uses: %d/%d, creator: %s)\n", link.InviteLink, link.MemberCount, link.MemberLimit, link.Creator.FirstName)
type ChatInviteLink struct {
	InviteLink              string
	Date                    time.Time
	IsPrimary               bool
	IsRevoked               bool
	Creator                 *User
	Name                    string
	CreatesJoinRequest      bool
	StartDate               time.Time
	ExpireDate              time.Time
	MemberLimit             int
	MemberCount             int
	PendingJoinRequestCount int
}

// ParseChatInviteLink converts a TL ChatInviteExported into a ChatInviteLink.
// Returns nil if raw is nil.
//
// Example:
//
//	link := types.ParseChatInviteLink(rawInvite, users)
//	if link != nil && link.IsRevoked {
//	    fmt.Println("Invite link has been revoked")
//	}
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
	if raw.Requested != 0 {
		link.PendingJoinRequestCount = int(raw.Requested)
	}
	if raw.Date != 0 {
		link.Date = time.Unix(int64(raw.Date), 0)
	}
	if raw.StartDate != 0 {
		link.StartDate = time.Unix(int64(raw.StartDate), 0)
	}
	if raw.ExpireDate != 0 {
		link.ExpireDate = time.Unix(int64(raw.ExpireDate), 0)
	}
	if users != nil {
		if u, ok := users[raw.AdminID]; ok {
			link.Creator = ParseUser(u)
		}
	}
	return link
}
