package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// AnswerCallbackQuery sends an answer to a callback query originated from an
// inline button press. text is shown as a notification to the user (or an alert
// when showAlert is true). url is an optional deep-linking URL to open. cacheTime
// is the maximum time in seconds the answer may be cached by the client.
// Returns an error if the underlying RPC call fails.
//
// Example:
//
//	err := client.AnswerCallbackQuery(ctx, queryID, "Processing...", false, "", 0)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID int64, text string, showAlert bool, url string, cacheTime int) error {
	c.Log.Debugf("AnswerCallbackQuery query_id=%d", callbackQueryID)
	req := &tg.MessagesSetBotCallbackAnswerRequest{
		Alert:     showAlert,
		QueryID:   callbackQueryID,
		Message:   text,
		URL:       url,
		CacheTime: int32(cacheTime),
	}

	_, err := c.Raw().MessagesSetBotCallbackAnswer(ctx, req)
	return err
}

// AnswerWebAppQuery sends the result of an interaction with a Web App to the
// Telegram server. queryID is the unique identifier of the query received from
// the Web App. result is the inline result to send. Returns the sent WebView
// message or an error if the request fails.
//
// Example:
//
//	sent, err := client.AnswerWebAppQuery(ctx, "AAQFXXXXXX", inlineResult)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Web app message sent: %v\n", sent)
func (c *Client) AnswerWebAppQuery(ctx context.Context, queryID string, result tg.InputBotInlineResultClass) (*tg.WebViewMessageSent, error) {
	c.Log.Debugf("AnswerWebAppQuery query_id=%s", queryID)
	rpc := c.Raw()
	return rpc.MessagesSendWebViewResultMessage(ctx, &tg.MessagesSendWebViewResultMessageRequest{
		BotQueryID: queryID,
		Result:     result,
	})
}

// RequestCallbackAnswer requests the bot's callback answer for a message in a
// chat. chatID identifies the chat, messageID is the target message, and data is
// the callback payload bytes. Returns the bot callback answer or an error if the
// chat cannot be resolved or the RPC call fails.
func (c *Client) RequestCallbackAnswer(ctx context.Context, chatID int64, messageID int64, data []byte) (*tg.MessagesBotCallbackAnswer, error) {
	c.Log.Debugf("RequestCallbackAnswer chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve chat: %w", err)
	}

	var flags tg.Fields
	if data != nil {
		flags.Set(0)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetBotCallbackAnswer(ctx, &tg.MessagesGetBotCallbackAnswerRequest{
		Flags: flags,
		Peer:  peer,
		MsgID: int32(messageID),
		Data:  data,
	})
	if err != nil {
		return nil, fmt.Errorf("get bot callback answer: %w", err)
	}
	return result, nil
}
