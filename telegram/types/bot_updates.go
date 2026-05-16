package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// ChatShared represents a chat that was shared with the bot via a keyboard button.
// Delivered as a service message when the user selects a chat to share.
type ChatShared struct {
	RequestID int64
	ChatID    int64
	Title     string
	ButtonID  int
	Chat      *Chat
}

// ParseChatShared converts a TL RequestedPeerChat into a ChatShared.
// Returns nil if raw is nil.
func ParseChatShared(raw *tg.RequestedPeerChat) *ChatShared {
	if raw == nil {
		return nil
	}
	s := &ChatShared{
		ChatID: raw.ChatID,
	}
	if raw.Title != "" {
		s.Title = raw.Title
	}
	return s
}

// UsersShared represents users that were shared with the bot via a keyboard button.
// Delivered as a service message when the user selects one or more users to share.
type UsersShared struct {
	RequestID int64
	UserIDs   []int64
	ButtonID  int
	Users     []*User
}

// MessageReactionUpdated represents a reaction change on a message observed by the bot.
// Delivered when a user adds or removes a reaction on a message the bot can see.
type MessageReactionUpdated struct {
	Chat        *Chat
	MessageID   int32
	User        *User
	ActorChat   *Chat
	Date        time.Time
	OldReaction []Reaction
	NewReaction []Reaction
	Reactions   []BotReactionCount
}

// BotReactionCount represents a single reaction type with its count for bot reaction updates.
type BotReactionCount struct {
	// Type is the emoji or custom emoji reaction identifier.
	Type string
	// Count is the number of times this reaction has been applied.
	Count int32
}

// MessageReactionCountUpdated represents the aggregated reaction count update on a message.
// Delivered when the total reaction counts change, without identifying individual users.
type MessageReactionCountUpdated struct {
	Chat      *Chat
	MessageID int32
	Date      time.Time
	Reactions []BotReactionCount
}

// ParseMessageReactionUpdated converts a TL UpdateBotMessageReaction into a MessageReactionUpdated.
// Returns nil if raw is nil.
func reactionString(r tg.ReactionClass) string {
	if r == nil {
		return ""
	}
	switch v := r.(type) {
	case *tg.ReactionEmpty:
		return ""
	case *tg.ReactionEmoji:
		return v.Emoticon
	case *tg.ReactionCustomEmoji:
		return "custom"
	case *tg.ReactionPaid:
		return "paid"
	}
	return ""
}

// ParseMessageReactionUpdated converts a TL UpdateBotMessageReaction into a
// MessageReactionUpdated, parsing old and new reaction lists. Returns nil if raw is nil.
//
// Example:
//
//	update := types.ParseMessageReactionUpdated(rawUpdate)
//	fmt.Printf("Reactions changed: %v -> %v\n", update.OldReaction, update.NewReaction)
func ParseMessageReactionUpdated(raw *tg.UpdateBotMessageReaction) *MessageReactionUpdated {
	if raw == nil {
		return nil
	}
	u := &MessageReactionUpdated{
		MessageID: raw.MsgID,
		Date:      time.Unix(int64(raw.Date), 0),
	}
	for _, r := range raw.OldReactions {
		if parsed := ParseReaction(r); parsed != nil {
			u.OldReaction = append(u.OldReaction, *parsed)
		}
	}
	for _, r := range raw.NewReactions {
		if parsed := ParseReaction(r); parsed != nil {
			u.NewReaction = append(u.NewReaction, *parsed)
		}
		u.Reactions = append(u.Reactions, BotReactionCount{
			Type:  reactionString(r),
			Count: 1,
		})
	}
	return u
}

// ParseMessageReactionCountUpdated converts a TL UpdateBotMessageReactions into a MessageReactionCountUpdated.
// Returns nil if raw is nil.
func ParseMessageReactionCountUpdated(raw *tg.UpdateBotMessageReactions) *MessageReactionCountUpdated {
	if raw == nil {
		return nil
	}
	u := &MessageReactionCountUpdated{
		MessageID: raw.MsgID,
		Date:      time.Unix(int64(raw.Date), 0),
	}
	for _, r := range raw.Reactions {
		if r != nil {
			u.Reactions = append(u.Reactions, BotReactionCount{
				Type:  reactionString(r.Reaction),
				Count: r.Count,
			})
		}
	}
	return u
}

// BotAccessSettings describes the access settings of a bot.
type BotAccessSettings struct{}

// CallbackGame represents a placeholder for a game callback button.
type CallbackGame struct{}

// SentGuestMessage represents a message sent by a guest user via inline mode.
type SentGuestMessage struct {
	InlineMessageID string
}

// ParseSentGuestMessage converts an MTProto inline message ID returned by
// messages.setBotGuestChatResult into a SentGuestMessage.
func ParseSentGuestMessage(raw tg.InputBotInlineMessageIDClass) *SentGuestMessage {
	if raw == nil {
		return nil
	}
	return &SentGuestMessage{InlineMessageID: formatInlineMessageID(raw)}
}

// KeyboardButtonRequestManagedBot represents a keyboard button that requests a managed bot.
type KeyboardButtonRequestManagedBot struct {
	RequestID         int64
	ButtonID          int
	SuggestedName     string
	SuggestedUsername string
}
