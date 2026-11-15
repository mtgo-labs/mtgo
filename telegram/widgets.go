package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// SendWidget sends an interactive HTML widget message. The widget renders at
// the given pixel dimensions; opts control query support, shared-widget
// mode, per-chat and public data, and the usual send options.
func (c *MessageMethods) SendWidget(ctx context.Context, chatID int64, width, height int32, html string, opts ...*params.SendWidget) (*types.Message, error) {
	c.Log.Debugf("SendWidget chat_id=%d %dx%d", chatID, width, height)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	opt := params.GetOptDef(&params.SendWidget{}, opts...)
	media := &tg.InputMediaWidget{
		AllowQueries: opt.AllowQueries,
		SharedWidget: opt.SharedWidget,
		Width:        width,
		Height:       height,
		HTML:         html,
		PublicData:   opt.PublicData,
		ChatData:     opt.ChatData,
	}

	return c.sendMediaInternal(ctx, peer, media, "", opt.ToSendMsg())
}

// EditMessageWidgetData updates the public or per-chat data of a widget
// message in place and returns the updated message.
func (c *MessageMethods) EditMessageWidgetData(ctx context.Context, chatID int64, messageID int32, publicData, chatData string) (*types.Message, error) {
	c.Log.Debugf("EditMessageWidgetData chat_id=%d msg=%d", chatID, messageID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().MessagesEditMessageWidgetData(ctx, &tg.MessagesEditMessageWidgetDataRequest{
		Peer:       peer,
		ID:         messageID,
		PublicData: publicData,
		ChatData:   chatData,
	})
	if err != nil {
		return nil, fmt.Errorf("edit message widget data: %w", err)
	}
	return extractSingleMessage(result, c)
}

// EditWidgetData updates the public data of a shared widget by its widget ID.
// cacheTime bounds how long clients may cache the data.
func (c *MessageMethods) EditWidgetData(ctx context.Context, widgetID int64, publicData string, cacheTime int32) error {
	c.Log.Debugf("EditWidgetData widget=%d", widgetID)
	_, err := c.Raw().MessagesEditWidgetData(ctx, &tg.MessagesEditWidgetDataRequest{
		WidgetID:   widgetID,
		PublicData: publicData,
		CacheTime:  cacheTime,
	})
	if err != nil {
		return fmt.Errorf("edit widget data: %w", err)
	}
	return nil
}

// WidgetPublicData is the public state of a widget message. When NotModified
// is true the data is unchanged and Period is the server's refresh hint.
type WidgetPublicData struct {
	// Data is the widget's public data payload.
	Data string
	// Hash identifies the data revision.
	Hash int64
	// Period is the recommended polling interval in seconds.
	Period int32
	// NotModified reports that the data is unchanged since the given hash.
	NotModified bool
}

// GetWidgetPublicData fetches the public data of a widget message. Pass the
// previous hash to receive the NotModified short form.
func (c *MessageMethods) GetWidgetPublicData(ctx context.Context, chatID int64, messageID int32, hash int64) (*WidgetPublicData, error) {
	c.Log.Debugf("GetWidgetPublicData chat_id=%d msg=%d", chatID, messageID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().MessagesGetWidgetPublicData(ctx, &tg.MessagesGetWidgetPublicDataRequest{
		Peer:  peer,
		MsgID: messageID,
		Hash:  hash,
	})
	if err != nil {
		return nil, fmt.Errorf("get widget public data: %w", err)
	}
	switch v := result.(type) {
	case *tg.MessagesWidgetPublicData:
		return &WidgetPublicData{Data: v.Data, Hash: v.Hash, Period: v.Period}, nil
	case *tg.MessagesWidgetPublicDataNotModified:
		return &WidgetPublicData{NotModified: true, Period: v.Period}, nil
	default:
		return nil, fmt.Errorf("get widget public data: unexpected result %T", result)
	}
}

// AnswerWidgetQuery answers a widget query issued by a widget message of the
// logged-in bot. result carries the widget-visible reply payload.
func (c *BotsMethods) AnswerWidgetQuery(ctx context.Context, queryID int64, result string) error {
	c.Log.Debugf("AnswerWidgetQuery query=%d", queryID)
	_, err := c.Raw().MessagesSetBotWidgetQueryAnswer(ctx, &tg.MessagesSetBotWidgetQueryAnswerRequest{
		QueryID: queryID,
		Result:  result,
	})
	if err != nil {
		return fmt.Errorf("answer widget query: %w", err)
	}
	return nil
}
