package types

// BotAccessSettings describes the access settings of a bot.
type BotAccessSettings struct{}

// CallbackGame represents a placeholder for a game callback button.
type CallbackGame struct{}

// SentGuestMessage represents a message sent by a guest user via inline mode.
type SentGuestMessage struct {
	InlineMessageID string
}

// KeyboardButtonRequestManagedBot represents a keyboard button that requests a managed bot.
type KeyboardButtonRequestManagedBot struct {
	RequestID int64
}
