package types

import "github.com/mtgo-labs/mtgo/tg"

// BotCommand represents a single bot command with its description, shown in
// the bot's command menu or autocomplete list.
type BotCommand struct {
	// Command is the command string without the leading slash (e.g. "start").
	Command string
	// Description is the human-readable explanation shown alongside the command.
	Description string
}

// ParseBotCommand converts a TL BotCommand into a BotCommand.
// Returns nil if raw is nil.
func ParseBotCommand(raw *tg.BotCommand) *BotCommand {
	if raw == nil {
		return nil
	}
	return &BotCommand{
		Command:     raw.Command,
		Description: raw.Description,
	}
}

// BotCommandScopeType identifies the scope where bot commands apply,
// such as all private chats, a specific chat, or a specific user within a chat.
type BotCommandScopeType string

const (
	// BotCommandScopeDefault applies commands to all users in all chats.
	BotCommandScopeDefault BotCommandScopeType = "default"
	// BotCommandScopeAllPrivate applies commands to all private (1-on-1) chats.
	BotCommandScopeAllPrivate BotCommandScopeType = "all_private_chats"
	// BotCommandScopeAllGroups applies commands to all group and supergroup chats.
	BotCommandScopeAllGroups BotCommandScopeType = "all_group_chats"
	// BotCommandScopeAllChatAdmins applies commands to all chat administrators.
	BotCommandScopeAllChatAdmins BotCommandScopeType = "all_chat_administrators"
	// BotCommandScopeChat applies commands to a specific chat identified by ChatID.
	BotCommandScopeChat BotCommandScopeType = "chat"
	// BotCommandScopeChatAdmins applies commands to admins of a specific chat identified by ChatID.
	BotCommandScopeChatAdmins BotCommandScopeType = "chat_administrators"
	// BotCommandScopeChatMember applies commands to a specific member (UserID) in a specific chat (ChatID).
	BotCommandScopeChatMember BotCommandScopeType = "chat_member"
)

// BotCommandScope describes the scope where a set of bot commands is active.
// The Type field determines which of the other fields are populated.
type BotCommandScope struct {
	// Type identifies the scope level (default, all private, specific chat, etc.).
	Type BotCommandScopeType
	// ChatID is the target chat ID for chat, chat_admins, and chat_member scopes.
	ChatID int64
	// UserID is the target user ID within the chat for chat_member scope.
	UserID int64
}

// ParseBotCommandScope converts a TL BotCommandScopeClass into a BotCommandScope.
// Returns nil if raw is nil.
func ParseBotCommandScope(raw tg.BotCommandScopeClass) *BotCommandScope {
	if raw == nil {
		return nil
	}
	s := &BotCommandScope{}
	switch v := raw.(type) {
	case *tg.BotCommandScopeDefault:
		s.Type = BotCommandScopeDefault
	case *tg.BotCommandScopeUsers:
		s.Type = BotCommandScopeAllPrivate
	case *tg.BotCommandScopeChats:
		s.Type = BotCommandScopeAllGroups
	case *tg.BotCommandScopeChatAdmins:
		s.Type = BotCommandScopeAllChatAdmins
	case *tg.BotCommandScopePeer:
		s.Type = BotCommandScopeChat
		s.ChatID = inputPeerID(v.Peer)
	case *tg.BotCommandScopePeerAdmins:
		s.Type = BotCommandScopeChatAdmins
		s.ChatID = inputPeerID(v.Peer)
	case *tg.BotCommandScopePeerUser:
		s.Type = BotCommandScopeChatMember
		s.ChatID = inputPeerID(v.Peer)
		s.UserID = inputUserID(v.UserID)
	}
	return s
}

// inputPeerID extracts the numeric ID from an InputPeerClass.
func inputPeerID(peer tg.InputPeerClass) int64 {
	if peer == nil {
		return 0
	}
	switch p := peer.(type) {
	case *tg.InputPeerChat:
		return p.ChatID
	case *tg.InputPeerUser:
		return p.UserID
	case *tg.InputPeerChannel:
		return p.ChannelID
	}
	return 0
}

// inputUserID extracts the numeric ID from an InputUserClass.
func inputUserID(user tg.InputUserClass) int64 {
	if user == nil {
		return 0
	}
	switch u := user.(type) {
	case *tg.InputUser:
		return u.UserID
	case *tg.InputUserFromMessage:
		return u.UserID
	}
	return 0
}
