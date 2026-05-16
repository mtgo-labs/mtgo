package types

// MessageReactions holds the aggregated reaction state for a single message.
// Each entry in Reactions represents one emoji reaction with its total count.
type MessageReactions struct {
	Reactions            []Reaction
	AreTags              bool
	PaidReactors         []*PaidReactor
	CanGetAddedReactions bool
}
