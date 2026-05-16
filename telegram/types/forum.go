package types

import (
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// ForumTopic represents a forum topic with its ID, title, icon, unread counts,
// and state flags (pinned, closed, hidden, etc.).
//
// Example:
//
//	topic := types.ParseForumTopic(rawTopic)
//	fmt.Printf("Topic: %s (ID: %d, pinned: %v)\n", topic.Title, topic.ID, topic.IsPinned)
type ForumTopic struct {
	ID                   int32
	Title                string
	Date                 time.Time
	IconColor            int32
	IconEmojiID          string
	Creator              *Chat
	TopMessage           *Message
	UnreadCount          int32
	UnreadMentionsCount  int32
	UnreadReactionsCount int32
	UnreadPollVoteCount  int32
	IsMy                 bool
	IsClosed             bool
	IsPinned             bool
	IsShort              bool
	IsHidden             bool
	IsDeleted            bool
	ReadInboxMaxID       int32
	ReadOutboxMaxID      int32
}

// ForumTopicCreated represents the payload delivered when a new forum topic is created.
type ForumTopicCreated struct {
	ID            int32
	Title         string
	IconColor     int32
	CustomEmojiID string
}

// ForumTopicEdited represents the changes applied when a forum topic is edited,
// such as title, icon, or closed/hidden state.
type ForumTopicEdited struct {
	Title         string
	IconColor     int32
	CustomEmojiID string
	IsClosed      bool
	IsHidden      bool
}

// ParseForumTopic converts a TL ForumTopicClass into a ForumTopic.
// Returns nil if raw is nil or not a *tg.ForumTopic.
//
// Example:
//
//	topic := types.ParseForumTopic(rawTopic)
//	if topic != nil {
//	    fmt.Println(topic.Title, topic.ID)
//	}
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
		Date:                 time.Unix(int64(v.Date), 0),
		Title:                v.Title,
		IconColor:            v.IconColor,
		TopMessage:           &Message{ID: v.TopMessage},
		ReadInboxMaxID:       v.ReadInboxMaxID,
		ReadOutboxMaxID:      v.ReadOutboxMaxID,
		UnreadCount:          v.UnreadCount,
		UnreadMentionsCount:  v.UnreadMentionsCount,
		UnreadReactionsCount: v.UnreadReactionsCount,
		IsClosed:             v.Closed,
		IsPinned:             v.Pinned,
		IsShort:              v.Short,
		IsHidden:             v.Hidden,
		IsMy:                 v.My,
	}
	if v.IconEmojiID != 0 {
		out.IconEmojiID = fmt.Sprintf("%d", v.IconEmojiID)
	}
	return out
}

// ForumTopicClosed indicates that a forum topic was closed.
type ForumTopicClosed struct{}

// ForumTopicReopened indicates that a forum topic was reopened.
type ForumTopicReopened struct{}

// GeneralForumTopicHidden indicates that the general forum topic was hidden.
type GeneralForumTopicHidden struct{}

// GeneralForumTopicUnhidden indicates that the general forum topic was unhidden.
type GeneralForumTopicUnhidden struct{}
