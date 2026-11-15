package params

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// EditInlineOpts provides optional parameters for inline message editing operations.
//
// Example:
//
//	opts := &telegram.EditInlineOpts{
//		NoWebpage:   true,
//		InvertMedia: false,
//		Entities:    nil,
//	}
//	ok, err := client.EditInlineText(ctx, msgID, "hello", opts)
type EditInlineOpts struct {
	NoWebpage   bool
	InvertMedia bool
	ReplyMarkup tg.ReplyMarkupClass
	Entities    []tg.MessageEntityClass
	// ParseMode parses the text/caption into entities when Entities is empty.
	ParseMode ParseMode
	// RichMessage replaces the inline message content with a rich message
	// (flag 23).
	RichMessage tg.InputRichMessageClass
}

// AnswerInlineQueryOption configures how inline query results are presented to
// the user. Use these options to control gallery mode, caching, pagination, and
// deep-link prompts.
type AnswerInlineQueryOption struct {
	// Gallery displays results as a gallery (grid) instead of a vertical list.
	Gallery bool
	// Private restricts results to the bot's private chats only.
	Private bool
	// CacheTime is the maximum number of seconds the client should cache the
	// results.
	CacheTime int32
	// NextOffset is the pagination offset string for the next batch of results.
	// Set this to enable result pagination.
	NextOffset string
	// SwitchPM provides a "Switch to PM" button that opens a private chat with
	// the bot, passing the given start parameter.
	SwitchPM *tg.InlineBotSwitchPm
	// SwitchWebview provides a button to open a web view with the bot.
	SwitchWebview *tg.InlineBotWebView
}

// SendInlineBotResultOption configures optional parameters when sending an
// inline bot result to a chat.
type SendInlineBotResultOption struct {
	// Silent sends the message without triggering a notification.
	Silent bool
	// HideVia hides the "via @bot" label on the sent message.
	HideVia bool
	// ReplyTo is the message ID to reply to (0 for no reply).
	ReplyTo int64
	// ScheduleDate schedules the message to be sent at the given Unix timestamp.
	ScheduleDate *int32
	// ClearDraft clears the chat's draft message after sending.
	ClearDraft bool
}

// SendDiceOption configures the animated emoji used in a dice message. When Emoticon
// is empty, the default game die (🎲) is used. Other supported values include
// 🎯 (dart), 🏀 (basketball), ⚽ (football), 🎰 (slot machine), and 🎳 (bowling).
type SendDiceOption struct {
	// Emoticon is the Unicode emoji representing the animated dice type.
	// Leave empty to use the default 🎲.
	Emoticon string
}

// GetPaymentFormOption holds optional parameters for GetPaymentForm.
type GetPaymentFormOption struct {
	ThemeParams *string
}

// SendPaymentFormOption holds optional parameters for SendPaymentForm.
type SendPaymentFormOption struct {
	RequestedInfoID  string
	ShippingOptionID string
	TipAmount        *int64
}
