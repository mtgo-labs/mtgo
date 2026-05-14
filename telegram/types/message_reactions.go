package types

// MessageReactions holds the aggregated reaction state for a single message.
// Each entry in Reactions represents one emoji reaction with its total count.
type MessageReactions struct {
	// PeerID is the chat or channel ID where the reacted message was sent.
	PeerID int64
	// MessageID is the identifier of the message within the chat.
	MessageID int32
	// Reactions is the list of emoji reactions on the message with their counts.
	Reactions []ReactionCount
}

// ReactionCount represents a single emoji reaction on a message with its
// occurrence count and whether the current user has chosen it.
type ReactionCount struct {
	// Emoticon is the Unicode emoji character representing the reaction.
	Emoticon string
	// Count is the total number of times this reaction has been applied.
	Count int
	// Chosen indicates whether the current user has applied this reaction.
	Chosen bool
}
