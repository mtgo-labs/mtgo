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
		flags.Set(0)
	}
	if text != "" {
		flags.Set(11)
	}
	if opt.InvertMedia {
		flags.Set(26)
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

	rpc := c.Raw()
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
		flags.Set(26)
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

	rpc := c.Raw()
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
	flags.Set(13)
	if opt.InvertMedia {
		flags.Set(26)
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

	rpc := c.Raw()
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

	rpc := c.Raw()
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
