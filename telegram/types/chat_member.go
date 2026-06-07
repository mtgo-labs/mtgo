package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// ChatMember represents a participant in a chat or channel with their role,
// join date, permissions, and user information.
//
// Example:
//
//	member, _ := client.GetChatMember(ctx, chatID, userID)
//	fmt.Printf("%s joined at %s (status: %s)\n", member.User.FirstName, member.JoinedDate, member.Status)
type ChatMember struct {
	Status                ChatMemberStatus
	Tag                   string
	User                  *User
	Chat                  *Chat
	JoinedDate            time.Time
	CustomTitle           string
	UntilDate             time.Time
	InvitedBy             *User
	PromotedBy            *User
	RestrictedBy          *User
	IsMember              bool
	CanBeEdited           bool
	Permissions           *ChatPermissions
	Privileges            *ChatAdminRights
	SubscriptionUntilDate time.Time
}

// ParseChatParticipant converts a TL ChatParticipantClass (basic group member)
// into a ChatMember. Returns nil if raw is nil.
//
// Example:
//
//	member := types.ParseChatParticipant(rawParticipant, users)
//	fmt.Printf("%s is %s\n", member.User.FirstName, member.Status)
func ParseChatParticipant(raw tg.ChatParticipantClass, users map[int64]tg.UserClass) *ChatMember {
	if raw == nil {
		return nil
	}
	m := &ChatMember{}
	switch p := raw.(type) {
	case *tg.ChatParticipantCreator:
		m.User = getUser(users, p.UserID)
		m.Status = ChatMemberStatusOwner
		if p.Rank != "" {
			m.CustomTitle = p.Rank
		}
	case *tg.ChatParticipantAdmin:
		m.User = getUser(users, p.UserID)
		m.Status = ChatMemberStatusAdministrator
		m.InvitedBy = getUser(users, p.InviterID)
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
		if p.Rank != "" {
			m.CustomTitle = p.Rank
		}
	case *tg.ChatParticipant:
		m.User = getUser(users, p.UserID)
		m.Status = ChatMemberStatusMember
		m.InvitedBy = getUser(users, p.InviterID)
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
		if p.Rank != "" {
			m.CustomTitle = p.Rank
		}
	}
	return m
}

// ParseChannelParticipant converts a TL ChannelParticipantClass (supergroup/channel member)
// into a ChatMember with admin privileges or ban status. Returns nil if raw is nil.
//
// Example:
//
//	member := types.ParseChannelParticipant(rawParticipant, users)
//	if member.Status == types.ChatMemberStatusBanned {
//	    fmt.Printf("User %s is banned\n", member.User.FirstName)
//	}
func ParseChannelParticipant(raw tg.ChannelParticipantClass, users map[int64]tg.UserClass) *ChatMember {
	if raw == nil {
		return nil
	}
	if p, ok := raw.(*tg.ChannelsChannelParticipant); ok {
		return ParseChannelParticipant(p.Participant, mergeUserClasses(users, p.Users))
	}
	m := &ChatMember{}
	switch p := raw.(type) {
	case *tg.ChannelParticipantCreator:
		m.User = getUser(users, p.UserID)
		m.Status = ChatMemberStatusOwner
		m.Privileges = ParseChatAdminRights(p.AdminRights)
		if p.Rank != "" {
			m.CustomTitle = p.Rank
		}
	case *tg.ChannelParticipantAdmin:
		m.User = getUser(users, p.UserID)
		m.Status = ChatMemberStatusAdministrator
		m.CanBeEdited = p.CanEdit
		m.InvitedBy = getUser(users, p.InviterID)
		if p.PromotedBy != 0 {
			m.PromotedBy = getUser(users, p.PromotedBy)
		}
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
		m.Privileges = ParseChatAdminRights(p.AdminRights)
		if p.Rank != "" {
			m.CustomTitle = p.Rank
		}
	case *tg.ChannelParticipant:
		m.User = getUser(users, p.UserID)
		m.Status = ChatMemberStatusMember
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
		if p.SubscriptionUntilDate != 0 {
			m.SubscriptionUntilDate = time.Unix(int64(p.SubscriptionUntilDate), 0)
		}
		if p.Rank != "" {
			m.CustomTitle = p.Rank
		}
	case *tg.ChannelParticipantSelf:
		m.User = getUser(users, p.UserID)
		m.Status = ChatMemberStatusMember
		m.InvitedBy = getUser(users, p.InviterID)
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
	case *tg.ChannelParticipantBanned:
		userID := getPeerUserID(p.Peer)
		m.User = getUser(users, userID)
		m.RestrictedBy = getUser(users, p.KickedBy)
		m.Status = ChatMemberStatusBanned
		m.IsMember = !p.Left
		m.Permissions = ParseChatPermissions(p.BannedRights)
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
		if p.Rank != "" {
			m.CustomTitle = p.Rank
		}
	case *tg.ChannelParticipantLeft:
		peerID := getPeerUserID(p.Peer)
		m.User = getUser(users, peerID)
		m.Status = ChatMemberStatusLeft
	}
	return m
}

func getUser(users map[int64]tg.UserClass, id int64) *User {
	if users == nil {
		return nil
	}
	if u, ok := users[id]; ok {
		return ParseUser(u)
	}
	return nil
}

func mergeUserClasses(base map[int64]tg.UserClass, users []tg.UserClass) map[int64]tg.UserClass {
	if len(users) == 0 {
		return base
	}
	merged := make(map[int64]tg.UserClass, len(base)+len(users))
	for id, user := range base {
		merged[id] = user
	}
	for _, user := range users {
		if parsed := ParseUser(user); parsed != nil {
			merged[parsed.ID] = user
		}
	}
	return merged
}

func getPeerUserID(peer tg.PeerClass) int64 {
	if peer == nil {
		return 0
	}
	if p, ok := peer.(*tg.PeerUser); ok {
		return p.UserID
	}
	return 0
}
