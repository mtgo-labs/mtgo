package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// ChatMemberUpdated represents a change in a member's status or permissions
// within a chat (e.g., promoted, demoted, kicked, or joined).
//
// Example:
//
//	update := types.ParseChatMemberUpdated(rawUpdate, users, peerMap)
//	fmt.Printf("%s: %s -> %s\n", update.FromUser.FirstName, update.OldChatMember.Status, update.NewChatMember.Status)
type ChatMemberUpdated struct {
	Chat           *Chat
	FromUser       *User
	Date           time.Time
	OldChatMember  *ChatMember
	NewChatMember  *ChatMember
	InviteLink     *ChatInviteLink
	ViaJoinRequest bool
}

// ParseChatMemberUpdated converts a TL UpdateChatParticipant or UpdateChannelParticipant
// into a ChatMemberUpdated. Returns nil if raw is nil.
//
// Example:
//
//	update := types.ParseChatMemberUpdated(rawUpdate, users, peerMap)
//	fmt.Printf("Member changed in chat %s\n", update.Chat.Title)
func ParseChatMemberUpdated(
	raw any,
	users map[int64]tg.UserClass,
	chats *PeerMap,
) *ChatMemberUpdated {
	if raw == nil {
		return nil
	}
	m := &ChatMemberUpdated{}
	switch v := raw.(type) {
	case *tg.UpdateChatParticipant:
		m.Chat = ParseChatFromPeer(&tg.PeerChat{ChatID: v.ChatID}, chats)
		m.FromUser = getUser(users, v.ActorID)
		m.Date = time.Unix(int64(v.Date), 0)
		m.OldChatMember = ParseChatParticipant(v.PrevParticipant, users)
		m.NewChatMember = ParseChatParticipant(v.NewParticipant, users)
		m.InviteLink = parseExportedInvite(v.Invite, users)
		if inv, ok := v.Invite.(*tg.ChatInviteExported); ok && inv.RequestNeeded {
			m.ViaJoinRequest = true
		}
	case *tg.UpdateChannelParticipant:
		m.Chat = ParseChatFromPeer(&tg.PeerChannel{ChannelID: v.ChannelID}, chats)
		m.FromUser = getUser(users, v.ActorID)
		m.Date = time.Unix(int64(v.Date), 0)
		m.OldChatMember = ParseChannelParticipant(v.PrevParticipant, users)
		m.NewChatMember = ParseChannelParticipant(v.NewParticipant, users)
		m.InviteLink = parseExportedInvite(v.Invite, users)
		if inv, ok := v.Invite.(*tg.ChatInviteExported); ok && inv.RequestNeeded {
			m.ViaJoinRequest = true
		}
	default:
		return nil
	}
	return m
}
