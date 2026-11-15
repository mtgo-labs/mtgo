package types

import (
	"context"
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
	// FromUser is the resolved user who triggered the callback.
	FromUser *User
	// InlineMessageID is the inline message identifier, set when the callback
	// originates from an inline message.
	InlineMessageID tg.InputBotInlineMessageIDClass
	binder          CallbackQueryBinder
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
	// Geo is the user's geolocation attached to the query, if shared.
	Geo tg.GeoPointClass
	// PeerType describes the chat the inline query originated from; maps to the
	// Bot API inline-query chat_type.
	PeerType tg.InlineQueryPeerTypeClass
	binder   InlineQueryBinder
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
			ID:              r.QueryID,
			UserID:          r.UserID,
			InlineMessage:   true,
			InlineMessageID: r.MsgID,
			ChatInstance:    r.ChatInstance,
			Data:            r.Data,
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
func (c *CallbackQuery) SetBinder(b CallbackQueryBinder) {
	c.binder = b
}

// Answer sends a simple toast notification to the user who pressed the button.
// The text appears briefly at the top of the screen.
//
// Example:
//
//	err := callback.Answer("Processing…")
func (c *CallbackQuery) Answer(text string) error {
	return c.AnswerContext(context.Background(), text)
}

func (c *CallbackQuery) AnswerContext(ctx context.Context, text string) error {
	if c.binder == nil {
		return ErrNoBinder
	}
	return c.binder.AnswerCallbackQuery(ctx, c.ID, text, false, "", 0)
}

// AnswerAlert shows a pop-up alert dialog to the user who pressed the button.
// Use this for important notifications that require explicit dismissal.
func (c *CallbackQuery) AnswerAlert(text string) error {
	return c.AnswerAlertContext(context.Background(), text)
}

func (c *CallbackQuery) AnswerAlertContext(ctx context.Context, text string) error {
	if c.binder == nil {
		return ErrNoBinder
	}
	return c.binder.AnswerCallbackQuery(ctx, c.ID, text, true, "", 0)
}

// AnswerURL opens a URL in the user's browser (or the Telegram in-app browser).
// Used for login URLs and game URLs.
func (c *CallbackQuery) AnswerURL(url string) error {
	return c.AnswerURLContext(context.Background(), url)
}

func (c *CallbackQuery) AnswerURLContext(ctx context.Context, url string) error {
	if c.binder == nil {
		return ErrNoBinder
	}
	return c.binder.AnswerCallbackQuery(ctx, c.ID, "", false, url, 0)
}

// Reply sends a text message in the chat where the callback button was pressed.
func (c *CallbackQuery) Reply(text string) (*Message, error) {
	return c.ReplyContext(context.Background(), text)
}

func (c *CallbackQuery) ReplyContext(ctx context.Context, text string) (*Message, error) {
	if c.binder == nil {
		return nil, ErrNoBinder
	}
	return c.binder.SendMessage(ctx, c.ChatID, text)
}

func (c *CallbackQuery) EditMessageText(text string, opts ...*params.EditMessage) (*Message, bool, error) {
	return c.EditMessageTextContext(context.Background(), text, opts...)
}

func (c *CallbackQuery) EditMessageTextContext(ctx context.Context, text string, opts ...*params.EditMessage) (*Message, bool, error) {
	if c.binder == nil {
		return nil, false, ErrNoBinder
	}
	if c.InlineMessageID != nil {
		ok, err := c.binder.EditInlineText(ctx, c.InlineMessageID, text, editInlineOptsFromParams(opts...)...)
		return nil, ok, err
	}
	opt := params.GetOptDef(&params.EditMessage{}, opts...)
	msg, err := c.binder.EditMessageText(ctx, c.ChatID, c.MessageID, text, opt)
	return msg, false, err
}

func (c *CallbackQuery) EditMessage(text string, opts ...*params.EditMessage) (*Message, bool, error) {
	return c.EditMessageText(text, opts...)
}

func (c *CallbackQuery) EditMessageCaption(caption string, opts ...*params.EditMessage) (*Message, bool, error) {
	return c.EditMessageCaptionContext(context.Background(), caption, opts...)
}

func (c *CallbackQuery) EditMessageCaptionContext(ctx context.Context, caption string, opts ...*params.EditMessage) (*Message, bool, error) {
	if c.binder == nil {
		return nil, false, ErrNoBinder
	}
	if c.InlineMessageID != nil {
		ok, err := c.binder.EditInlineCaption(ctx, c.InlineMessageID, caption, editInlineOptsFromParams(opts...)...)
		return nil, ok, err
	}
	opt := params.GetOptDef(&params.EditMessage{}, opts...)
	msg, err := c.binder.EditMessageCaption(ctx, c.ChatID, c.MessageID, caption, opt)
	return msg, false, err
}

func (c *CallbackQuery) EditCaption(caption string, opts ...*params.EditMessage) (*Message, error) {
	msg, _, err := c.EditMessageCaption(caption, opts...)
	return msg, err
}

func (c *CallbackQuery) EditMessageMedia(media tg.InputMediaClass) (*Message, bool, error) {
	return c.EditMessageMediaContext(context.Background(), media)
}

func (c *CallbackQuery) EditMessageMediaContext(ctx context.Context, media tg.InputMediaClass) (*Message, bool, error) {
	if c.binder == nil {
		return nil, false, ErrNoBinder
	}
	if c.InlineMessageID != nil {
		ok, err := c.binder.EditInlineMedia(ctx, c.InlineMessageID, media)
		return nil, ok, err
	}
	msg, err := c.binder.EditMessageMedia(ctx, c.ChatID, c.MessageID, media)
	return msg, false, err
}

func (c *CallbackQuery) EditMedia(media tg.InputMediaClass) (*Message, error) {
	msg, _, err := c.EditMessageMedia(media)
	return msg, err
}

func (c *CallbackQuery) EditMessageReplyMarkup(markup tg.ReplyMarkupClass) (*Message, bool, error) {
	return c.EditMessageReplyMarkupContext(context.Background(), markup)
}

func (c *CallbackQuery) EditMessageReplyMarkupContext(ctx context.Context, markup tg.ReplyMarkupClass) (*Message, bool, error) {
	if c.binder == nil {
		return nil, false, ErrNoBinder
	}
	if c.InlineMessageID != nil {
		ok, err := c.binder.EditInlineReplyMarkup(ctx, c.InlineMessageID, markup)
		return nil, ok, err
	}
	msg, err := c.binder.EditMessageReplyMarkup(ctx, c.ChatID, c.MessageID, markup)
	return msg, false, err
}

func (c *CallbackQuery) EditReplyMarkup(markup tg.ReplyMarkupClass) (*Message, error) {
	msg, _, err := c.EditMessageReplyMarkup(markup)
	return msg, err
}

// EditRich replaces the originating message content with a rich message. Works
// for both inline messages (InlineMessageID set) and regular chat messages.
func (c *CallbackQuery) EditRich(rm tg.InputRichMessageClass) (*Message, bool, error) {
	return c.EditRichContext(context.Background(), rm)
}

func (c *CallbackQuery) EditRichContext(ctx context.Context, rm tg.InputRichMessageClass) (*Message, bool, error) {
	if c.binder == nil {
		return nil, false, ErrNoBinder
	}
	if c.InlineMessageID != nil {
		ok, err := c.binder.EditInlineRichMessage(ctx, c.InlineMessageID, rm)
		return nil, ok, err
	}
	msg, err := c.binder.EditMessageText(ctx, c.ChatID, c.MessageID, "", &params.EditMessage{RichMessage: rm})
	return msg, false, err
}

// Delete removes the message that originated this callback query.
func (c *CallbackQuery) Delete() (int, error) {
	return c.DeleteContext(context.Background())
}

func (c *CallbackQuery) DeleteContext(ctx context.Context) (int, error) {
	if c.binder == nil {
		return 0, ErrNoBinder
	}
	return c.binder.DeleteMessages(ctx, c.ChatID, []int32{c.MessageID})
}

// ParseInlineQuery extracts an InlineQuery from an MTProto update.
// Returns nil if raw is nil or is not an inline query update.
func ParseInlineQuery(raw tg.UpdateClass) *InlineQuery {
	if raw == nil {
		return nil
	}
	if r, ok := raw.(*tg.UpdateBotInlineQuery); ok {
		return &InlineQuery{
			ID:       r.QueryID,
			UserID:   r.UserID,
			Query:    r.Query,
			Offset:   r.Offset,
			Geo:      r.Geo,
			PeerType: r.PeerType,
		}
	}
	return nil
}

// SetBinder injects the Binder that backs all bound convenience methods on this
// InlineQuery. Called internally by the Client after constructing an InlineQuery
// from an update.
func (iq *InlineQuery) SetBinder(b InlineQueryBinder) {
	iq.binder = b
}

// Answer sends inline query results back to the user. Results is a slice of
// inline result objects. Optional params control gallery mode, caching, and
// pagination.
func (iq *InlineQuery) Answer(results []tg.InputBotInlineResultClass, opts ...*params.InlineQuery) error {
	return iq.AnswerContext(context.Background(), results, opts...)
}

func (iq *InlineQuery) AnswerContext(ctx context.Context, results []tg.InputBotInlineResultClass, opts ...*params.InlineQuery) error {
	if iq.binder == nil {
		return ErrNoInlineBinder
	}
	o := params.GetOptDef(&params.InlineQuery{}, opts...)
	op := &params.AnswerInlineQueryOption{
		Gallery:    o.Gallery,
		Private:    o.Private,
		CacheTime:  int32(o.CacheTime),
		NextOffset: o.NextOffset,
	}
	if o.SwitchPM != "" {
		op.SwitchPM = &tg.InlineBotSwitchPm{
			Text:       o.SwitchPMText,
			StartParam: o.SwitchPM,
		}
	}
	return iq.binder.AnswerInlineQuery(ctx, iq.ID, results, op)
}

func (iq *InlineQuery) AnswerResults(results []InlineResultBuilder, opts ...*params.InlineQuery) error {
	tlResults, err := buildInlineResults(results)
	if err != nil {
		return err
	}
	return iq.Answer(tlResults, opts...)
}

// AnswerResult answers the query with a single builder-produced result. Any
// InlineResultBuilder works: InlineArticle, InlinePhoto, InlineLocation,
// InlineRich, and so on.
func (iq *InlineQuery) AnswerResult(result InlineResultBuilder, opts ...*params.InlineQuery) error {
	return iq.AnswerResults([]InlineResultBuilder{result}, opts...)
}

func (iq *InlineQuery) AnswerArticle(id, title, text string, opts ...*params.InlineQuery) error {
	return iq.AnswerResults([]InlineResultBuilder{
		&InlineArticle{ID: id, Title: title, Text: text},
	}, opts...)
}

func (iq *InlineQuery) AnswerArticles(articles []*InlineArticle, opts ...*params.InlineQuery) error {
	results := make([]InlineResultBuilder, len(articles))
	for i, a := range articles {
		results[i] = a
	}
	return iq.AnswerResults(results, opts...)
}

func (iq *InlineQuery) AnswerPhoto(id string, photoID, accessHash int64, text string, opts ...*params.InlineQuery) error {
	return iq.AnswerResults([]InlineResultBuilder{
		&InlinePhoto{ID: id, PhotoID: photoID, AccessHash: accessHash, Text: text},
	}, opts...)
}

func (iq *InlineQuery) AnswerPhotos(photos []*InlinePhoto, opts ...*params.InlineQuery) error {
	results := make([]InlineResultBuilder, len(photos))
	for i, p := range photos {
		results[i] = p
	}
	return iq.AnswerResults(results, opts...)
}

func (iq *InlineQuery) AnswerDocument(id string, docID, accessHash int64, text string, opts ...*params.InlineQuery) error {
	return iq.AnswerResults([]InlineResultBuilder{
		&InlineDocument{ID: id, DocumentID: docID, AccessHash: accessHash, Text: text},
	}, opts...)
}

func (iq *InlineQuery) AnswerDocuments(docs []*InlineDocument, opts ...*params.InlineQuery) error {
	results := make([]InlineResultBuilder, len(docs))
	for i, d := range docs {
		results[i] = d
	}
	return iq.AnswerResults(results, opts...)
}

func (iq *InlineQuery) AnswerGame(id, shortName string, opts ...*params.InlineQuery) error {
	return iq.AnswerResults([]InlineResultBuilder{
		&InlineGame{ID: id, ShortName: shortName},
	}, opts...)
}
