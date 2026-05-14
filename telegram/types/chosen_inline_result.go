package types

// ChosenInlineResult represents a result chosen by the user from an inline query.
// Sent to the bot so it can react to the user's selection, e.g. to log queries
// or send a follow-up message.
type ChosenInlineResult struct {
	// ID is the unique identifier of the inline query that this result belongs to.
	ID int64
	// UserID is the Telegram user ID of the user who selected the result.
	UserID int64
	// ResultID is the identifier of the chosen result, matching the ID sent by the bot.
	ResultID string
	// Query is the original text the user typed to trigger the inline query.
	Query string
}
