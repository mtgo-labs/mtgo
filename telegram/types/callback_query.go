package types

import (
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

// CallbackQuery represents an incoming callback query from an inline button
// press. When created by a Client, it carries a Binder so that Answer, Reply,
// and other convenience methods work without a direct Client reference.
//
// Example:
//
//	handler := telegram.NewHandler(func(ctx context.Context, update *telegram.Update) error {
//	    if cb := update.CallbackQuery; cb != nil {
//	        fmt.Printf("Button pressed: %s\n", string(cb.Data))
//	    }
//	    return nil
//	})
type CallbackQuery struct {
	// ID is the unique identifier of the callback query.
	ID int64
	// UserID is the ID of the user who pressed the button.
	UserID int64
	// ChatID is the chat where the originating message was sent. 0 for inline
	// messages.
	ChatID int64
	// MessageID is the message that contained the button. 0 for inline messages.
	MessageID int32
	// InlineMessage is true when the callback originates from an inline message
	// sent via a bot, rather than a regular message in a chat.
	InlineMessage bool
	// ChatInstance is an opaque identifier for the chat instance, used for
	// identifying the chat context of the callback.
	ChatInstance int64
	// Data is the button payload associated with the callback.
	Data []byte
	// GameShortName is the short name of the game when the callback was triggered
	// by a game button, or empty otherwise.
	GameShortName string
	binder        Binder
}

// InlineQuery represents an incoming inline query from a user typing a query in
// the @bot search box.
type InlineQuery struct {
	// ID is the unique identifier of the inline query.
	ID int64
	// UserID is the ID of the user who issued the query.
	UserID int64
	// Query is the text the user typed.
	Query string
	// Offset is the pagination offset for the next batch of results.
	Offset string
	binder Binder
}

// ParseCallbackQuery extracts a CallbackQuery from an MTProto update.
// It handles both regular bot callback queries and inline callback queries.
// Returns nil if raw is nil or is not a recognized callback query update.
func ParseCallbackQuery(raw tg.UpdateClass) *CallbackQuery {
	if raw == nil {
		return nil
	}
	switch r := raw.(type) {
	case *tg.UpdateBotCallbackQuery:
		q := &CallbackQuery{
			ID:           r.QueryID,
			UserID:       r.UserID,
			MessageID:    r.MsgID,
			ChatInstance: r.ChatInstance,
			Data:         r.Data,
		}
		if r.Peer != nil {
			q.ChatID = getBarePeerID(r.Peer)
		}
		if r.GameShortName != "" {
			q.GameShortName = r.GameShortName
		}
		return q
	case *tg.UpdateInlineBotCallbackQuery:
		q := &CallbackQuery{
			ID:            r.QueryID,
			UserID:        r.UserID,
			InlineMessage: true,
			ChatInstance:  r.ChatInstance,
			Data:          r.Data,
		}
		if r.GameShortName != "" {
			q.GameShortName = r.GameShortName
		}
		return q
	}
	return nil
}

// SetBinder injects the Binder that backs all bound convenience methods on this
// CallbackQuery. Called internally by the Client after constructing a
// CallbackQuery from an update.
func (c *CallbackQuery) SetBinder(b Binder) {
	c.binder = b
}

// Answer sends a simple toast notification to the user who pressed the button.
// The text appears briefly at the top of the screen.
//
// Example:
//
//	err := callback.Answer("Processing…")
func (c *CallbackQuery) Answer(text string) error {
	if c.binder == nil {
		return ErrNoBinder
	}
	return c.binder.BoundAnswerCallback(c.ID, text, false, "", 0)
}

// AnswerAlert shows a pop-up alert dialog to the user who pressed the button.
// Use this for important notifications that require explicit dismissal.
func (c *CallbackQuery) AnswerAlert(text string) error {
	if c.binder == nil {
		return ErrNoBinder
	}
	return c.binder.BoundAnswerCallback(c.ID, text, true, "", 0)
}

// AnswerURL opens a URL in the user's browser (or the Telegram in-app browser).
// Used for login URLs and game URLs.
func (c *CallbackQuery) AnswerURL(url string) error {
	if c.binder == nil {
		return ErrNoBinder
	}
	return c.binder.BoundAnswerCallback(c.ID, "", false, url, 0)
}

// Reply sends a text message in the chat where the callback button was pressed.
func (c *CallbackQuery) Reply(text string) (*Message, error) {
	if c.binder == nil {
		return nil, ErrNoBinder
	}
	return c.binder.BoundSend(c.ChatID, text, 0)
}

// EditMessage edits the text of the message that originated this callback query.
func (c *CallbackQuery) EditMessage(text string, opts ...*params.EditMessage) (*Message, error) {
	if c.binder == nil {
		return nil, ErrNoBinder
	}
	return c.binder.BoundEdit(c.ChatID, c.MessageID, text, opts...)
}

// EditCaption edits the caption of the media message that originated this callback
// query.
func (c *CallbackQuery) EditCaption(caption string, opts ...*params.EditMessage) (*Message, error) {
	if c.binder == nil {
		return nil, ErrNoBinder
	}
	return c.binder.BoundEditCaption(c.ChatID, c.MessageID, caption, opts...)
}

// EditMedia replaces the media content of the message that originated this
// callback query.
func (c *CallbackQuery) EditMedia(media tg.InputMediaClass) (*Message, error) {
	if c.binder == nil {
		return nil, ErrNoBinder
	}
	return c.binder.BoundEditMedia(c.ChatID, c.MessageID, media)
}

// EditReplyMarkup changes only the inline keyboard of the message that originated
// this callback query.
func (c *CallbackQuery) EditReplyMarkup(markup tg.ReplyMarkupClass) (*Message, error) {
	if c.binder == nil {
		return nil, ErrNoBinder
	}
	return c.binder.BoundEditReplyMarkup(c.ChatID, c.MessageID, markup)
}

// Delete removes the message that originated this callback query.
func (c *CallbackQuery) Delete() (int, error) {
	if c.binder == nil {
		return 0, ErrNoBinder
	}
	return c.binder.BoundDelete(c.ChatID, []int32{c.MessageID})
}

// ParseInlineQuery extracts an InlineQuery from an MTProto update.
// Returns nil if raw is nil or is not an inline query update.
func ParseInlineQuery(raw tg.UpdateClass) *InlineQuery {
	if raw == nil {
		return nil
	}
	if r, ok := raw.(*tg.UpdateBotInlineQuery); ok {
		return &InlineQuery{
			ID:     r.QueryID,
			UserID: r.UserID,
			Query:  r.Query,
			Offset: r.Offset,
		}
	}
	return nil
}

// SetBinder injects the Binder that backs all bound convenience methods on this
// InlineQuery. Called internally by the Client after constructing an InlineQuery
// from an update.
func (iq *InlineQuery) SetBinder(b Binder) {
	iq.binder = b
}

// Answer sends inline query results back to the user. Results is a slice of
// inline result objects. Optional params control gallery mode, caching, and
// pagination.
func (iq *InlineQuery) Answer(results []tg.InputBotInlineResultClass, opts ...*params.InlineQuery) error {
	if iq.binder == nil {
		return ErrNoInlineBinder
	}
	opt := params.GetOptDef(&params.InlineQuery{}, opts...)
	return iq.binder.BoundAnswerInline(
		iq.ID, results,
		opt.CacheTime, opt.Gallery, opt.Private,
		opt.NextOffset, opt.SwitchPM, opt.SwitchPMText,
	)
}
