package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// ChatEvent represents a single entry in a chat's admin log, capturing what
// changed (title, photo, permissions, members, etc.) and who performed the action.
//
// Example:
//
//	events, _ := client.GetChatEventLog(ctx, chatID, "", 100)
//	for _, e := range events {
//	    fmt.Printf("[%s] %s by %s\n", e.Action, e.Date, e.User.FirstName)
//	}
type ChatEvent struct {
	ID                         int64
	Date                       time.Time
	Action                     ChatEventAction
	User                       *User
	OldDescription             string
	NewDescription             string
	OldHistoryTTL              int32
	NewHistoryTTL              int32
	OldLinkedChat              *Chat
	NewLinkedChat              *Chat
	OldPhoto                   *Photo
	NewPhoto                   *Photo
	OldTitle                   string
	NewTitle                   string
	OldUsername                string
	NewUsername                string
	OldChatPermissions         *ChatPermissions
	NewChatPermissions         *ChatPermissions
	DeletedMessage             *Message
	OldMessage                 *Message
	NewMessage                 *Message
	InvitedMember              *ChatMember
	OldAdministratorPrivileges *ChatMember
	NewAdministratorPrivileges *ChatMember
	OldMemberPermissions       *ChatMember
	NewMemberPermissions       *ChatMember
	StoppedPoll                *Message
	InvitesEnabled             bool
	HistoryHidden              bool
	SignaturesEnabled          bool
	OldSlowMode                int32
	NewSlowMode                int32
	PinnedMessage              *Message
	UnpinnedMessage            *Message
	OldInviteLink              *ChatInviteLink
	NewInviteLink              *ChatInviteLink
	RevokedInviteLink          *ChatInviteLink
	DeletedInviteLink          *ChatInviteLink
	CreatedForumTopic          *ForumTopic
	OldForumTopic              *ForumTopic
	NewForumTopic              *ForumTopic
	DeletedForumTopic          *ForumTopic
}

// ParseChatEvent converts a TL ChannelAdminLogEvent into a ChatEvent, resolving
// the acting user and dispatching the action-specific parser. Returns nil if raw is nil.
//
// Example:
//
//	event := types.ParseChatEvent(rawLogEvent, users, peerMap)
//	fmt.Printf("Event %d: %s at %s\n", event.ID, event.Action, event.Date)
func ParseChatEvent(raw *tg.ChannelAdminLogEvent, users map[int64]tg.UserClass, chats *PeerMap) *ChatEvent {
	if raw == nil {
		return nil
	}
	e := &ChatEvent{
		ID:   raw.ID,
		Date: time.Unix(int64(raw.Date), 0),
		User: getUser(users, raw.UserID),
	}
	parseChatEventAction(raw.Action, e, users, chats)
	return e
}

func parseChatEventAction(action tg.ChannelAdminLogEventActionClass, e *ChatEvent, users map[int64]tg.UserClass, chats *PeerMap) {
	if action == nil {
		return
	}
	switch a := action.(type) {
	case *tg.ChannelAdminLogEventActionChangeTitle:
		e.Action = ChatEventActionTitleChanged
		e.OldTitle = a.PrevValue
		e.NewTitle = a.NewValue
	case *tg.ChannelAdminLogEventActionChangeAbout:
		e.Action = ChatEventActionDescriptionChanged
		e.OldDescription = a.PrevValue
		e.NewDescription = a.NewValue
	case *tg.ChannelAdminLogEventActionChangeUsername:
		e.Action = ChatEventActionUsernameChanged
		e.OldUsername = a.PrevValue
		e.NewUsername = a.NewValue
	case *tg.ChannelAdminLogEventActionChangePhoto:
		e.Action = ChatEventActionPhotoChanged
		if p, ok := a.PrevPhoto.(*tg.Photo); ok {
			e.OldPhoto = parsePhoto(p)
		}
		if p, ok := a.NewPhoto.(*tg.Photo); ok {
			e.NewPhoto = parsePhoto(p)
		}
	case *tg.ChannelAdminLogEventActionChangeLinkedChat:
		e.Action = ChatEventActionLinkedChatChanged
		if chats != nil {
			e.OldLinkedChat = ParseChatFromPeer(&tg.PeerChannel{ChannelID: a.PrevValue}, chats)
			e.NewLinkedChat = ParseChatFromPeer(&tg.PeerChannel{ChannelID: a.NewValue}, chats)
		}
	case *tg.ChannelAdminLogEventActionChangeHistoryTTL:
		e.Action = ChatEventActionHistoryTTLChanged
		e.OldHistoryTTL = a.PrevValue
		e.NewHistoryTTL = a.NewValue
	case *tg.ChannelAdminLogEventActionDefaultBannedRights:
		e.Action = ChatEventActionChatPermissionsChanged
		e.OldChatPermissions = ParseChatPermissions(a.PrevBannedRights)
		e.NewChatPermissions = ParseChatPermissions(a.NewBannedRights)
	case *tg.ChannelAdminLogEventActionToggleInvites:
		e.Action = ChatEventActionInvitesEnabled
		e.InvitesEnabled = a.NewValue
	case *tg.ChannelAdminLogEventActionToggleSignatures:
		e.Action = ChatEventActionSignaturesEnabled
		e.SignaturesEnabled = a.NewValue
	case *tg.ChannelAdminLogEventActionTogglePreHistoryHidden:
		e.Action = ChatEventActionHistoryHidden
		e.HistoryHidden = a.NewValue
	case *tg.ChannelAdminLogEventActionToggleSlowMode:
		e.Action = ChatEventActionSlowModeChanged
		e.OldSlowMode = a.PrevValue
		e.NewSlowMode = a.NewValue
	case *tg.ChannelAdminLogEventActionUpdatePinned:
		if msg, ok := a.Message.(*tg.Message); ok {
			e.PinnedMessage = parseRegularMessage(msg, chats)
			e.Action = ChatEventActionMessagePinned
		} else {
			e.Action = ChatEventActionMessageUnpinned
			e.UnpinnedMessage = parseMessageClass(a.Message, chats)
		}
	case *tg.ChannelAdminLogEventActionEditMessage:
		e.Action = ChatEventActionMessageEdited
		e.OldMessage = parseMessageClass(a.PrevMessage, chats)
		e.NewMessage = parseMessageClass(a.NewMessage, chats)
	case *tg.ChannelAdminLogEventActionDeleteMessage:
		e.Action = ChatEventActionMessageDeleted
		e.DeletedMessage = parseMessageClass(a.Message, chats)
	case *tg.ChannelAdminLogEventActionParticipantInvite:
		e.Action = ChatEventActionMemberInvited
		e.InvitedMember = ParseChannelParticipant(a.Participant, users)
	case *tg.ChannelAdminLogEventActionParticipantToggleAdmin:
		e.Action = ChatEventActionAdministratorPrivilegesChanged
		e.OldAdministratorPrivileges = ParseChannelParticipant(a.PrevParticipant, users)
		e.NewAdministratorPrivileges = ParseChannelParticipant(a.NewParticipant, users)
	case *tg.ChannelAdminLogEventActionParticipantToggleBan:
		e.Action = ChatEventActionMemberPermissionsChanged
		e.OldMemberPermissions = ParseChannelParticipant(a.PrevParticipant, users)
		e.NewMemberPermissions = ParseChannelParticipant(a.NewParticipant, users)
	case *tg.ChannelAdminLogEventActionParticipantJoin:
		e.Action = ChatEventActionMemberJoined
	case *tg.ChannelAdminLogEventActionParticipantLeave:
		e.Action = ChatEventActionMemberLeft
	case *tg.ChannelAdminLogEventActionStopPoll:
		e.Action = ChatEventActionPollStopped
		e.StoppedPoll = parseMessageClass(a.Message, chats)
	case *tg.ChannelAdminLogEventActionExportedInviteEdit:
		e.Action = ChatEventActionInviteLinkEdited
		e.OldInviteLink = parseExportedInvite(a.PrevInvite, users)
		e.NewInviteLink = parseExportedInvite(a.NewInvite, users)
	case *tg.ChannelAdminLogEventActionExportedInviteRevoke:
		e.Action = ChatEventActionInviteLinkRevoked
		e.RevokedInviteLink = parseExportedInvite(a.Invite, users)
	case *tg.ChannelAdminLogEventActionExportedInviteDelete:
		e.Action = ChatEventActionInviteLinkDeleted
		e.DeletedInviteLink = parseExportedInvite(a.Invite, users)
	case *tg.ChannelAdminLogEventActionCreateTopic:
		e.Action = ChatEventActionCreatedForumTopic
		e.CreatedForumTopic = ParseForumTopic(a.Topic)
	case *tg.ChannelAdminLogEventActionEditTopic:
		e.Action = ChatEventActionEditedForumTopic
		e.OldForumTopic = ParseForumTopic(a.PrevTopic)
		e.NewForumTopic = ParseForumTopic(a.NewTopic)
	case *tg.ChannelAdminLogEventActionDeleteTopic:
		e.Action = ChatEventActionDeletedForumTopic
		e.DeletedForumTopic = ParseForumTopic(a.Topic)
	}
}

func parseExportedInvite(raw tg.ExportedChatInviteClass, users map[int64]tg.UserClass) *ChatInviteLink {
	if raw == nil {
		return nil
	}
	if inv, ok := raw.(*tg.ChatInviteExported); ok {
		return ParseChatInviteLink(inv, users)
	}
	return nil
}

func parseMessageClass(raw tg.MessageClass, chats *PeerMap) *Message {
	if raw == nil {
		return nil
	}
	if msg, ok := raw.(*tg.Message); ok {
		return parseRegularMessage(msg, chats)
	}
	if svc, ok := raw.(*tg.MessageService); ok {
		return parseServiceMessage(svc, chats)
	}
	return nil
}
