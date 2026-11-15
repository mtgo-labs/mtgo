package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

// EditInlineText edits the text content of an inline message sent via the bot.
//
// Example:
//
//	ok, err := client.EditInlineText(ctx, inlineMsgID, "updated text",
//		&telegram.EditInlineOpts{NoWebpage: true},
//	)
func (c *MessageMethods) EditInlineText(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, text string, opts ...*EditInlineOpts) (bool, error) {
	c.Log.Debugf("EditInlineText")
	opt := getEditInlineOpts(opts...)

	// Parse formatted text when no pre-built entities are provided.
	sendText := text
	var parsedEntities []tg.MessageEntityClass
	if len(opt.Entities) == 0 {
		parsed, entities, err := c.parseText(text, opt.ParseMode)
		if err != nil {
			return false, fmt.Errorf("parse text: %w", err)
		}
		sendText = parsed
		parsedEntities = entities
	}

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
	entities := opt.Entities
	if len(entities) == 0 {
		entities = parsedEntities
	}
	if len(entities) > 0 {
		flags.Set(3)
	}
	if opt.RichMessage != nil {
		flags.Set(23)
	}

	req := &tg.MessagesEditInlineBotMessageRequest{
		Flags:       flags,
		NoWebpage:   opt.NoWebpage,
		InvertMedia: opt.InvertMedia,
		ID:          inlineMessageID,
		Message:     sendText,
		ReplyMarkup: opt.ReplyMarkup,
		Entities:    entities,
	}
	if opt.RichMessage != nil {
		req.RichMessage = opt.RichMessage
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
func (c *MessageMethods) EditInlineCaption(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, caption string, opts ...*EditInlineOpts) (bool, error) {
	c.Log.Debugf("EditInlineCaption")
	opt := getEditInlineOpts(opts...)

	// Parse formatted caption when no pre-built entities are provided.
	sendCaption := caption
	var parsedEntities []tg.MessageEntityClass
	if len(opt.Entities) == 0 {
		parsed, entities, err := c.parseText(caption, opt.ParseMode)
		if err != nil {
			return false, fmt.Errorf("parse caption: %w", err)
		}
		sendCaption = parsed
		parsedEntities = entities
	}

	var flags tg.Fields
	flags.Set(11)
	if opt.InvertMedia {
		flags.Set(16)
	}
	if opt.ReplyMarkup != nil {
		flags.Set(2)
	}
	entities := opt.Entities
	if len(entities) == 0 {
		entities = parsedEntities
	}
	if len(entities) > 0 {
		flags.Set(3)
	}
	if opt.RichMessage != nil {
		flags.Set(23)
	}

	req := &tg.MessagesEditInlineBotMessageRequest{
		Flags:       flags,
		InvertMedia: opt.InvertMedia,
		ID:          inlineMessageID,
		Message:     sendCaption,
		ReplyMarkup: opt.ReplyMarkup,
		Entities:    entities,
	}
	if opt.RichMessage != nil {
		req.RichMessage = opt.RichMessage
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
func (c *MessageMethods) EditInlineMedia(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, media tg.InputMediaClass, opts ...*EditInlineOpts) (bool, error) {
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
func (c *MessageMethods) EditInlineReplyMarkup(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, replyMarkup tg.ReplyMarkupClass) (bool, error) {
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

// EditInlineRichMessage replaces the content of an inline message with a rich
// message (Bot API 10.1 InputRichMessageContent editing).
//
// Example:
//
//	ok, err := client.EditInlineRichMessage(ctx, inlineMsgID,
//		telegram.RichMessageMarkdown("**updated**", false))
func (c *MessageMethods) EditInlineRichMessage(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, rm tg.InputRichMessageClass, opts ...*EditInlineOpts) (bool, error) {
	c.Log.Debugf("EditInlineRichMessage")
	if rm == nil {
		return false, fmt.Errorf("edit inline rich message: rich message is nil")
	}
	opt := getEditInlineOpts(opts...)

	var flags tg.Fields
	flags.Set(23)
	if opt.ReplyMarkup != nil {
		flags.Set(2)
	}

	req := &tg.MessagesEditInlineBotMessageRequest{
		Flags:       flags,
		ID:          inlineMessageID,
		ReplyMarkup: opt.ReplyMarkup,
		RichMessage: rm,
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
	// ParseMode parses the text/caption into entities when Entities is empty.
	ParseMode params.ParseMode
	// RichMessage replaces the inline message content with a rich message
	// (flag 23).
	RichMessage tg.InputRichMessageClass
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
