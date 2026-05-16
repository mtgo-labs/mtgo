package types

// ChatBoostUpdated represents a boost applied to a chat, including the boost
// details and associated star count.
//
// Example:
//
//	update := types.ParseChatBoostUpdated(chat, boost, stars)
//	fmt.Printf("Chat %s boosted, stars: %d\n", update.Chat.Title, update.StarCount)
type ChatBoostUpdated struct {
	Chat      *Chat
	Boost     *ChatBoost
	StarCount int64
}

// ParseChatBoostUpdated creates a ChatBoostUpdated from its components.
// Returns nil if both chat and boost are nil.
//
// Example:
//
//	update := types.ParseChatBoostUpdated(chat, boost, 500)
//	if update != nil {
//	    fmt.Println("Boost applied with", update.StarCount, "stars")
//	}
func ParseChatBoostUpdated(chat *Chat, boost *ChatBoost, stars int64) *ChatBoostUpdated {
	if chat == nil && boost == nil {
		return nil
	}
	return &ChatBoostUpdated{
		Chat:      chat,
		Boost:     boost,
		StarCount: stars,
	}
}
