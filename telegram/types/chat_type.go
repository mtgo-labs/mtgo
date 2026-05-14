package types

// ChatType enumerates the kinds of Telegram chats. It is used to classify
// a Chat without inspecting the underlying TL type.
//
// Example:
//
//	switch chat.Type {
//	case types.ChatTypePrivate:
//	    fmt.Println("Private chat")
//	case types.ChatTypeChannel:
//	    fmt.Println("Channel")
//	}
type ChatType string

const (
	// ChatTypePrivate is a one-to-one conversation with a regular user.
	ChatTypePrivate ChatType = "private"
	// ChatTypeBot is a one-to-one conversation with a bot.
	ChatTypeBot ChatType = "bot"
	// ChatTypeGroup is a basic (non-supergroup) group chat.
	ChatTypeGroup ChatType = "group"
	// ChatTypeSupergroup is a supergroup (large group with admin tools).
	ChatTypeSupergroup ChatType = "supergroup"
	// ChatTypeChannel is a broadcast channel.
	ChatTypeChannel ChatType = "channel"
	// ChatTypeForum is a supergroup with forum (topics) enabled.
	ChatTypeForum ChatType = "forum"
	// ChatTypeDirect is a direct message conversation (alias for private in some
	// contexts).
	ChatTypeDirect ChatType = "direct"
)

// String returns the string representation of the ChatType.
func (c ChatType) String() string { return string(c) }
