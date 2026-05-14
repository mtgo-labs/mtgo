package telegram

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// Answer sends a callback query acknowledgment to Telegram. This must be called for every
// callback query to stop the loading indicator on the user's device. If the context has no
// active callback query, the call is silently ignored.
//
// Parameters:
//   - text: optional notification text shown to the user (up to 200 characters)
//   - showAlert: if true, the text is shown as a popup alert instead of a toast
//
// Returns:
//   - error: non-nil if the answer could not be sent
//
// Example:
//
//	client.OnCallbackQuery(func(ctx *telegram.Context) {
//	    ctx.Answer("Processing your request...", false)
//	})
func (c *Context) Answer(text string, showAlert bool) error {
	if c.CallbackQuery == nil {
		return nil
	}
	return c.Client.AnswerCallbackQuery(c.Ctx, c.CallbackQuery.ID, text, showAlert, "", 0)
}

// AnswerCallbackQuery is an alias for [Context.Answer] with the same parameters.
// Provided for naming consistency with the Telegram Bot API terminology.
//
// Example:
//
//	client.OnCallbackQuery(func(ctx *telegram.Context) {
//	    ctx.AnswerCallbackQuery("Action completed!", false)
//	})
func (c *Context) AnswerCallbackQuery(text string, showAlert bool) error {
	return c.Answer(text, showAlert)
}

// AnswerCallback is a short alias for [Context.Answer].
func (c *Context) AnswerCallback(text string, showAlert bool) error {
	return c.Answer(text, showAlert)
}

// CallbackEditText edits the text of the message that originated the current callback query.
// This cannot be used for inline messages (those sent via inline mode), as they lack a
// persistent message ID.
//
// Parameters:
//   - text: the new message text (Markdown or HTML formatted)
//   - opts: optional [params.EditMessage] parameters for additional configuration
//
// Returns:
//   - *types.Message: the edited message
//   - error: non-nil if there is no callback query, the message is inline, or the edit fails
func (c *Context) CallbackEditText(text string, opts ...*params.EditMessage) (*types.Message, error) {
	if c.CallbackQuery == nil {
		return nil, fmt.Errorf("context: no callback query")
	}
	if c.CallbackQuery.InlineMessage {
		return nil, fmt.Errorf("context: cannot edit inline message by ID")
	}
	chatID := c.CallbackQuery.ChatID
	return c.Client.EditMessageText(c.Ctx, chatID, c.CallbackQuery.MessageID, text, opts...)
}

// CallbackEditCaption edits the caption of the media message that originated the current
// callback query. Cannot be used for inline messages.
//
// Parameters:
//   - caption: the new caption text
//   - opts: optional [params.EditMessage] parameters for additional configuration
//
// Returns:
//   - *types.Message: the edited message
//   - error: non-nil if there is no callback query, the message is inline, or the edit fails
func (c *Context) CallbackEditCaption(caption string, opts ...*params.EditMessage) (*types.Message, error) {
	if c.CallbackQuery == nil {
		return nil, fmt.Errorf("context: no callback query")
	}
	if c.CallbackQuery.InlineMessage {
		return nil, fmt.Errorf("context: cannot edit inline message by ID")
	}
	chatID := c.CallbackQuery.ChatID
	return c.Client.EditMessageCaption(c.Ctx, chatID, c.CallbackQuery.MessageID, caption, opts...)
}

// CallbackEditMedia replaces the media attachment of the message that originated the
// current callback query. Cannot be used for inline messages.
//
// Parameters:
//   - media: the new media to replace the existing attachment
//   - opts: optional [params.EditMessage] parameters for additional configuration
//
// Returns:
//   - *types.Message: the edited message
//   - error: non-nil if there is no callback query, the message is inline, or the edit fails
func (c *Context) CallbackEditMedia(media tg.InputMediaClass, opts ...*params.EditMessage) (*types.Message, error) {
	if c.CallbackQuery == nil {
		return nil, fmt.Errorf("context: no callback query")
	}
	if c.CallbackQuery.InlineMessage {
		return nil, fmt.Errorf("context: cannot edit inline message by ID")
	}
	chatID := c.CallbackQuery.ChatID
	return c.Client.EditMessageMedia(c.Ctx, chatID, c.CallbackQuery.MessageID, media, opts...)
}

// CallbackEditReplyMarkup replaces the inline keyboard of the message that originated the
// current callback query. Use this to update button states after a callback is handled.
// Cannot be used for inline messages.
//
// Parameters:
//   - replyMarkup: the new inline keyboard markup
//
// Returns:
//   - *types.Message: the edited message
//   - error: non-nil if there is no callback query, the message is inline, or the edit fails
func (c *Context) CallbackEditReplyMarkup(replyMarkup tg.ReplyMarkupClass) (*types.Message, error) {
	if c.CallbackQuery == nil {
		return nil, fmt.Errorf("context: no callback query")
	}
	if c.CallbackQuery.InlineMessage {
		return nil, fmt.Errorf("context: cannot edit inline message by ID")
	}
	chatID := c.CallbackQuery.ChatID
	return c.Client.EditMessageReplyMarkup(c.Ctx, chatID, c.CallbackQuery.MessageID, replyMarkup)
}
