package types

import "github.com/mtgo-labs/mtgo/tg"

// ForumTopic represents a forum topic in a Telegram supergroup.
type ForumTopic struct {
	// ID is the unique identifier of the forum topic.
	ID int32
	// Date when the topic was created (Unix timestamp).
	Date int32
	// Title is the name of the forum topic.
	Title string
	// IconColor is the color of the topic icon in RGB format.
	IconColor int32
	// IconEmojiID is the custom emoji sticker used as the topic icon.
	IconEmojiID int64
	// TopMessage is the ID of the last message in the topic.
	TopMessage int32
	// ReadInboxMaxID is the ID up to which all messages have been read incoming.
	ReadInboxMaxID int32
	// ReadOutboxMaxID is the ID up to which all messages have been read outgoing.
	ReadOutboxMaxID int32
	// UnreadCount is the number of unread messages in the topic.
	UnreadCount int32
	// UnreadMentionsCount is the number of unread mention messages.
	UnreadMentionsCount int32
	// UnreadReactionsCount is the number of unread reaction messages.
	UnreadReactionsCount int32
	// Closed indicates whether the topic is closed for new messages.
	Closed bool
	// Pinned indicates whether the topic is pinned in the forum.
	Pinned bool
	// Short indicates whether the topic info is abbreviated.
	Short bool
	// Hidden indicates whether the topic is hidden from the forum list.
	Hidden bool
	// My indicates whether the current user created this topic.
	My bool
}

// ForumTopicCreated represents the initial creation data of a forum topic.
type ForumTopicCreated struct {
	// Title is the name of the created forum topic.
	Title string
	// IconColor is the color of the topic icon in RGB format.
	IconColor int32
	// IconEmojiID is the custom emoji sticker used as the topic icon.
	IconEmojiID int64
}

// ForumTopicEdited represents changes made to an existing forum topic.
type ForumTopicEdited struct {
	// Title is the updated name of the forum topic.
	Title string
	// IconEmojiID is the updated custom emoji sticker for the topic icon.
	IconEmojiID int64
	// Closed indicates whether the topic was closed.
	Closed bool
	// Hidden indicates whether the topic was hidden.
	Hidden bool
}

// ParseForumTopic converts a TL ForumTopicClass into a ForumTopic.
// Returns nil if the raw value is not a *tg.ForumTopic.
func ParseForumTopic(raw tg.ForumTopicClass) *ForumTopic {
	if raw == nil {
		return nil
	}
	v, ok := raw.(*tg.ForumTopic)
	if !ok {
		return nil
	}
	out := &ForumTopic{
		ID:                   v.ID,
		Date:                 v.Date,
		Title:                v.Title,
		IconColor:            v.IconColor,
		TopMessage:           v.TopMessage,
		ReadInboxMaxID:       v.ReadInboxMaxID,
		ReadOutboxMaxID:      v.ReadOutboxMaxID,
		UnreadCount:          v.UnreadCount,
		UnreadMentionsCount:  v.UnreadMentionsCount,
		UnreadReactionsCount: v.UnreadReactionsCount,
		Closed:               v.Closed,
		Pinned:               v.Pinned,
		Short:                v.Short,
		Hidden:               v.Hidden,
		My:                   v.My,
	}
	if v.IconEmojiID != 0 {
		out.IconEmojiID = v.IconEmojiID
	}
	return out
}
