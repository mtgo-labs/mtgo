package telegram

import (
	"context"

	"github.com/mtgo-labs/mtgo/tg"
)

// EditInlineText edits the text content of an inline message sent via the bot.
//
// Example:
//
//	ok, err := client.EditInlineText(ctx, inlineMsgID, "updated text",
//		&telegram.EditInlineOpts{NoWebpage: true},
//	)
func (c *Client) EditInlineText(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, text string, opts ...*EditInlineOpts) (bool, error) {
	c.Log.Debugf("EditInlineText")
	opt := getEditInlineOpts(opts...)

	var flags tg.Fields
	if opt.NoWebpage {
		flags.Set(1)
	}
	if text != "" {
		flags.Set(11)
	}
	if opt.InvertMedia {
		flags.Set(16)
	}
	if opt.ReplyMarkup != nil {
		flags.Set(2)
	}
	if len(opt.Entities) > 0 {
		flags.Set(3)
	}

	req := &tg.MessagesEditInlineBotMessageRequest{
		Flags:       flags,
		NoWebpage:   opt.NoWebpage,
		InvertMedia: opt.InvertMedia,
		ID:          inlineMessageID,
		Message:     text,
		ReplyMarkup: opt.ReplyMarkup,
		Entities:    opt.Entities,
	}

	rpc, err := c.inlineEditRPC(ctx, inlineMessageID)
	if err != nil {
		return false, err
	}
	result, err := rpc.MessagesEditInlineBotMessage(ctx, req)
	if err != nil {
		return false, err
	}
	return result, nil
}

// EditInlineCaption edits the caption of an inline media message sent via the bot.
//
// Example:
//
//	ok, err := client.EditInlineCaption(ctx, inlineMsgID, "new caption")
func (c *Client) EditInlineCaption(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, caption string, opts ...*EditInlineOpts) (bool, error) {
	c.Log.Debugf("EditInlineCaption")
	opt := getEditInlineOpts(opts...)

	var flags tg.Fields
	flags.Set(11)
	if opt.InvertMedia {
		flags.Set(16)
	}
	if opt.ReplyMarkup != nil {
		flags.Set(2)
	}

	req := &tg.MessagesEditInlineBotMessageRequest{
		Flags:       flags,
		InvertMedia: opt.InvertMedia,
		ID:          inlineMessageID,
		Message:     caption,
		ReplyMarkup: opt.ReplyMarkup,
	}

	rpc, err := c.inlineEditRPC(ctx, inlineMessageID)
	if err != nil {
		return false, err
	}
	result, err := rpc.MessagesEditInlineBotMessage(ctx, req)
	if err != nil {
		return false, err
	}
	return result, nil
}

// EditInlineMedia edits the media attachment of an inline message sent via the bot.
//
// Example:
//
//	media := &tg.InputMediaPhoto{ID: &tg.InputPhoto{ID: photoID}}
//	ok, err := client.EditInlineMedia(ctx, inlineMsgID, media)
func (c *Client) EditInlineMedia(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, media tg.InputMediaClass, opts ...*EditInlineOpts) (bool, error) {
	c.Log.Debugf("EditInlineMedia")
	opt := getEditInlineOpts(opts...)

	var flags tg.Fields
	flags.Set(14)
	if opt.InvertMedia {
		flags.Set(16)
	}
	if opt.ReplyMarkup != nil {
		flags.Set(2)
	}

	req := &tg.MessagesEditInlineBotMessageRequest{
		Flags:       flags,
		InvertMedia: opt.InvertMedia,
		ID:          inlineMessageID,
		Media:       media,
		ReplyMarkup: opt.ReplyMarkup,
	}

	rpc, err := c.inlineEditRPC(ctx, inlineMessageID)
	if err != nil {
		return false, err
	}
	result, err := rpc.MessagesEditInlineBotMessage(ctx, req)
	if err != nil {
		return false, err
	}
	return result, nil
}

// EditInlineReplyMarkup replaces the inline keyboard of an inline message.
//
// Example:
//
//	keyboard := &tg.ReplyInlineMarkup{Rows: rows}
//	ok, err := client.EditInlineReplyMarkup(ctx, inlineMsgID, keyboard)
func (c *Client) EditInlineReplyMarkup(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, replyMarkup tg.ReplyMarkupClass) (bool, error) {
	c.Log.Debugf("EditInlineReplyMarkup")

	var flags tg.Fields
	flags.Set(2)

	req := &tg.MessagesEditInlineBotMessageRequest{
		Flags:       flags,
		ID:          inlineMessageID,
		ReplyMarkup: replyMarkup,
	}

	rpc, err := c.inlineEditRPC(ctx, inlineMessageID)
	if err != nil {
		return false, err
	}
	result, err := rpc.MessagesEditInlineBotMessage(ctx, req)
	if err != nil {
		return false, err
	}
	return result, nil
}

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
}

func getEditInlineOpts(opts ...*EditInlineOpts) *EditInlineOpts {
	if len(opts) == 0 || opts[0] == nil {
		return &EditInlineOpts{}
	}
	return opts[0]
}

// inlineMessageDC extracts the origin DC encoded in an inline message ID.
// Telegram requires messages.editInlineBotMessage to be invoked on that DC:
// edits sent to any other DC fail with MESSAGE_ID_INVALID instead of a 303
// migration error, so the routing cannot be recovered reactively.
func inlineMessageDC(inlineMessageID tg.InputBotInlineMessageIDClass) (int32, bool) {
	switch id := inlineMessageID.(type) {
	case *tg.InputBotInlineMessageID:
		return id.DCID, true
	case *tg.InputBotInlineMessageID64:
		return id.DCID, true
	default:
		return 0, false
	}
}

// inlineEditRPC returns the RPC client for the DC hosting the inline message.
// Foreign-DC edits use an auxiliary authorized session; same-DC edits (or IDs
// without a usable DC) use the main invoker.
func (c *Client) inlineEditRPC(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass) (*tg.RPCClient, error) {
	dcID, ok := inlineMessageDC(inlineMessageID)
	if !ok || dcID <= 0 {
		return c.Raw(), nil
	}
	return c.dcRPC(ctx, int(dcID))
}
