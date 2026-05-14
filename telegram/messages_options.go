package telegram

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// SearchMessagesOption configures optional parameters for a targeted message search
// within a specific chat. Use it to filter by date range, sender, message type, or
// to paginate through large result sets.
type SearchMessagesOption struct {
	// Limit is the maximum number of messages to return. Defaults to 100 when zero or negative.
	Limit int
	// OffsetID is the message ID from which to start searching; use 0 to start from the newest message.
	OffsetID int32
	// MinDate is the Unix timestamp of the earliest message to include in results.
	MinDate int32
	// MaxDate is the Unix timestamp of the latest message to include in results.
	MaxDate int32
	// FromID filters results to messages sent by this peer (user, channel, or chat).
	FromID tg.InputPeerClass
	// Filter restricts results to a specific message type (photos, videos, documents, etc.).
	// Defaults to InputMessagesFilterEmpty (no filter) when nil.
	Filter tg.MessagesFilterClass
	// TopMsgID limits the search to messages within a specific forum topic (for forum-enabled channels).
	TopMsgID *int32
}

// SearchGlobalOption configures optional parameters for a global message search across
// all of the user's chats. Use it to scope results to specific chat types, date ranges,
// or folder IDs, and to paginate through large result sets.
type SearchGlobalOption struct {
	// Limit is the maximum number of messages to return. Defaults to 100 when zero or negative.
	Limit int
	// OffsetRate is a server-side pagination token from a previous search result; use 0 for the first page.
	OffsetRate int32
	// OffsetID is the message ID from which to continue searching; use 0 to start from the beginning.
	OffsetID int32
	// OffsetPeer is the peer from which to continue searching; use nil (InputPeerEmpty) for the first page.
	OffsetPeer tg.InputPeerClass
	// MinDate is the Unix timestamp of the earliest message to include in results.
	MinDate int32
	// MaxDate is the Unix timestamp of the latest message to include in results.
	MaxDate int32
	// BroadcastsOnly restricts results to messages from channels (broadcasts) only.
	BroadcastsOnly bool
	// GroupsOnly restricts results to messages from group chats only.
	GroupsOnly bool
	// FolderID restricts results to chats within the specified chat folder.
	FolderID *int32
	// Filter restricts results to a specific message type (photos, videos, documents, etc.).
	// Defaults to InputMessagesFilterEmpty (no filter) when nil.
	Filter tg.MessagesFilterClass
}

// SendDiceOption configures the animated emoji used in a dice message. When Emoticon
// is empty, the default game die (🎲) is used. Other supported values include
// 🎯 (dart), 🏀 (basketball), ⚽ (football), 🎰 (slot machine), and 🎳 (bowling).
type SendDiceOption struct {
	// Emoticon is the Unicode emoji representing the animated dice type.
	// Leave empty to use the default 🎲.
	Emoticon string
}
