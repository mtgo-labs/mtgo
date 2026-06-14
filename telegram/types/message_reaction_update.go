package types

import "github.com/mtgo-labs/mtgo/tg"

// MessageReactionUpdate represents a reaction change on a message, delivered
// to the bot when a user adds/changes/removes a reaction in a chat where the
// bot is an administrator. Maps to the Bot API "message_reaction" update.
type MessageReactionUpdate struct {
	ChatID       int64
	MessageID    int64
	UserID       int64
	Date         int32
	OldReactions []Reaction
	NewReactions []Reaction
}

// ParseMessageReactionUpdate converts a TL updateBotMessageReaction into a
// MessageReactionUpdate, resolving the chat Peer and actor Peer to IDs.
// Returns nil if raw is nil.
func ParseMessageReactionUpdate(raw *tg.UpdateBotMessageReaction) *MessageReactionUpdate {
	if raw == nil {
		return nil
	}
	old := make([]Reaction, 0, len(raw.OldReactions))
	for _, r := range raw.OldReactions {
		if x := ParseReaction(r); x != nil {
			old = append(old, *x)
		}
	}
	newR := make([]Reaction, 0, len(raw.NewReactions))
	for _, r := range raw.NewReactions {
		if x := ParseReaction(r); x != nil {
			newR = append(newR, *x)
		}
	}
	return &MessageReactionUpdate{
		ChatID:       peerToChatID(raw.Peer),
		MessageID:    int64(raw.MsgID),
		UserID:       peerToUserID(raw.Actor),
		Date:         raw.Date,
		OldReactions: old,
		NewReactions: newR,
	}
}

// MessageReactionCountUpdate represents an anonymous reaction-count change on
// a message. Maps to the Bot API "message_reaction_count" update.
type MessageReactionCountUpdate struct {
	ChatID    int64
	MessageID int64
	Date      int32
	Reactions []Reaction // each carries Count + Emoji/CustomEmojiID
}

// ParseMessageReactionCountUpdate converts a TL updateBotMessageReactions.
func ParseMessageReactionCountUpdate(raw *tg.UpdateBotMessageReactions) *MessageReactionCountUpdate {
	if raw == nil {
		return nil
	}
	out := &MessageReactionCountUpdate{
		ChatID:    peerToChatID(raw.Peer),
		MessageID: int64(raw.MsgID),
		Date:      raw.Date,
	}
	for _, r := range raw.Reactions {
		if x := ParseReaction(r.Reaction); x != nil {
			x.Count = int(r.Count)
			out.Reactions = append(out.Reactions, *x)
		}
	}
	return out
}

// peerToChatID resolves a tg.PeerClass to a Bot API chat ID.
func peerToChatID(p tg.PeerClass) int64 {
	switch v := p.(type) {
	case *tg.PeerUser:
		return v.UserID
	case *tg.PeerChat:
		return -v.ChatID
	case *tg.PeerChannel:
		return -1_000_000_000_000 - v.ChannelID
	}
	return 0
}

// peerToUserID resolves a tg.PeerClass to a user ID (0 if not a user peer).
func peerToUserID(p tg.PeerClass) int64 {
	if u, ok := p.(*tg.PeerUser); ok {
		return u.UserID
	}
	return 0
}
