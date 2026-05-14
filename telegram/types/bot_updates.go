package types

import "github.com/mtgo-labs/mtgo/tg"

// ChatShared represents a chat that was shared with the bot via a keyboard button.
// Delivered as a service message when the user selects a chat to share.
type ChatShared struct {
	// RequestID matches the request_id from the KeyboardButtonRequestChat that triggered the share.
	RequestID int64
	// ChatID is the ID of the shared chat.
	ChatID int64
	// Title is the title of the shared chat, if available.
	Title string
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
	// RequestID matches the request_id from the KeyboardButtonRequestUsers that triggered the share.
	RequestID int64
	// UserIDs is the list of Telegram user IDs of the shared users.
	UserIDs []int64
}

// MessageReactionUpdated represents a reaction change on a message observed by the bot.
// Delivered when a user adds or removes a reaction on a message the bot can see.
type MessageReactionUpdated struct {
	// ChatID is the chat where the reacted message was sent.
	ChatID int64
	// MsgID is the message ID that received the reaction change.
	MsgID int32
	// Date is the Unix timestamp when the reaction was updated.
	Date int32
	// Reactions is the list of reactions on the message after the change.
	Reactions []BotReactionCount
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
	// ChatID is the chat where the reacted message was sent.
	ChatID int64
	// MsgID is the message ID that received the reaction count update.
	MsgID int32
	// Date is the Unix timestamp when the reaction counts were updated.
	Date int32
	// Reactions is the full list of reaction counts on the message.
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

func ParseMessageReactionUpdated(raw *tg.UpdateBotMessageReaction) *MessageReactionUpdated {
	if raw == nil {
		return nil
	}
	u := &MessageReactionUpdated{
		ChatID: GetPeerID(raw.Peer),
		MsgID:  raw.MsgID,
		Date:   raw.Date,
	}
	for _, r := range raw.NewReactions {
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
		ChatID: GetPeerID(raw.Peer),
		MsgID:  raw.MsgID,
		Date:   raw.Date,
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
