package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// ChatMember represents a member of a chat with their role, join metadata, and
// the user who invited, promoted, or restricted them.
//
// Example:
//
//	members, _ := chat.GetMembers(10, 0)
//	for _, m := range members {
//	    fmt.Printf("%s — %s (rank: %s)\n", m.User.String(), m.Status, m.Rank)
//	}
type ChatMember struct {
	// User is the Telegram user who is a member of the chat.
	User *User
	// Status is the member's role in the chat (owner, administrator, member,
	// restricted, left, or banned).
	Status ChatMemberStatus
	// JoinedDate is when the user joined or was promoted. Zero value when not
	// available.
	JoinedDate time.Time
	// InvitedBy is the user who invited this member, or nil if not applicable.
	InvitedBy *User
	// PromotedBy is the user who promoted this member to admin, or nil if not
	// applicable.
	PromotedBy *User
	// RestrictedBy is the user who applied restrictions to this member, or nil
	// if not applicable.
	RestrictedBy *User
	// Permissions describes the restrictions applied to this member when their
	// status is restricted or banned.
	Permissions *ChatPermissions
	// Rank is the custom admin title displayed in the chat for admins and owners,
	// or empty for regular members.
	Rank string
}

// ParseChatParticipant converts an MTProto chat participant to a ChatMember.
// The users map is used to resolve user references. Returns nil if raw is nil.
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
			m.Rank = p.Rank
		}
	case *tg.ChatParticipantAdmin:
		m.User = getUser(users, p.UserID)
		m.Status = ChatMemberStatusAdministrator
		m.InvitedBy = getUser(users, p.InviterID)
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
		if p.Rank != "" {
			m.Rank = p.Rank
		}
	case *tg.ChatParticipant:
		m.User = getUser(users, p.UserID)
		m.Status = ChatMemberStatusMember
		m.InvitedBy = getUser(users, p.InviterID)
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
		if p.Rank != "" {
			m.Rank = p.Rank
		}
	}
	return m
}

// ParseChannelParticipant converts an MTProto channel participant to a
// ChatMember. The users map is used to resolve user references. Returns nil if
// raw is nil.
func ParseChannelParticipant(raw tg.ChannelParticipantClass, users map[int64]tg.UserClass) *ChatMember {
	if raw == nil {
		return nil
	}
	m := &ChatMember{}
	switch p := raw.(type) {
	case *tg.ChannelParticipantCreator:
		m.User = getUserFromPeer(users, p.UserID)
		m.Status = ChatMemberStatusOwner
		if p.Rank != "" {
			m.Rank = p.Rank
		}
	case *tg.ChannelParticipantAdmin:
		m.User = getUserFromPeer(users, p.UserID)
		m.Status = ChatMemberStatusAdministrator
		if p.PromotedBy != 0 {
			m.PromotedBy = getUserFromPeer(users, p.PromotedBy)
		}
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
		if p.Rank != "" {
			m.Rank = p.Rank
		}
	case *tg.ChannelParticipant:
		m.User = getUserFromPeer(users, p.UserID)
		m.Status = ChatMemberStatusMember
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
		if p.Rank != "" {
			m.Rank = p.Rank
		}
	case *tg.ChannelParticipantSelf:
		m.User = getUserFromPeer(users, p.UserID)
		m.Status = ChatMemberStatusMember
		m.InvitedBy = getUserFromPeer(users, p.InviterID)
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
	case *tg.ChannelParticipantBanned:
		userID := getPeerUserID(p.Peer)
		m.User = getUserFromPeer(users, userID)
		m.RestrictedBy = getUserFromPeer(users, p.KickedBy)
		m.Status = ChatMemberStatusBanned
		m.Permissions = ParseChatPermissions(p.BannedRights)
		if p.Date != 0 {
			m.JoinedDate = time.Unix(int64(p.Date), 0)
		}
	case *tg.ChannelParticipantLeft:
		peerID := getPeerUserID(p.Peer)
		m.User = getUserFromPeer(users, peerID)
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

func getUserFromPeer(users map[int64]tg.UserClass, id int64) *User {
	return getUser(users, id)
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
