package types

import "github.com/mtgo-labs/mtgo/tg"

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
	// MsgID is the inline message identifier, set when the chosen result was
	// sent via inline mode (to a chat the bot isn't a member of). Nil when the
	// result was sent to a regular chat the bot is in.
	MsgID tg.InputBotInlineMessageIDClass
}

// ParseChosenInlineResult converts a TL updateBotInlineSend into a
// ChosenInlineResult. Returns nil if raw is nil.
func ParseChosenInlineResult(raw *tg.UpdateBotInlineSend) *ChosenInlineResult {
	if raw == nil {
		return nil
	}
	return &ChosenInlineResult{
		UserID:   raw.UserID,
		Query:    raw.Query,
		ResultID: raw.ID,
		MsgID:    raw.MsgID,
	}
}

