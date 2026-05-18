package telegram

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

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

// AnswerInlineQuery sends results for an inline query originated from the @bot
// search box. Use this when a bot receives an inline query update.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - queryID: unique identifier of the inline query to answer
//   - results: slice of inline result objects to present to the user
//   - opts: optional [AnswerInlineQueryOption] configuration
//
// Returns an error if the RPC call fails.
//
// Example:
//
//	err := client.AnswerInlineQuery(ctx, queryID, results, &telegram.AnswerInlineQueryOption{
//	    Gallery:   true,
//	    CacheTime: 300,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) AnswerInlineQuery(ctx context.Context, queryID int64, results []tg.InputBotInlineResultClass, opts ...*AnswerInlineQueryOption) error {
	c.Log.Debugf("AnswerInlineQuery query_id=%d", queryID)
	opt := getOptDef(&AnswerInlineQueryOption{}, opts...)

	var flags tg.Fields
	if opt.Gallery {
		flags.Set(0)
	}
	if opt.Private {
		flags.Set(1)
	}
	if opt.NextOffset != "" {
		flags.Set(2)
	}
	if opt.SwitchPM != nil {
		flags.Set(3)
	}
	if opt.SwitchWebview != nil {
		flags.Set(4)
	}

	req := &tg.MessagesSetInlineBotResultsRequest{
		Flags:         flags,
		Gallery:       opt.Gallery,
		Private:       opt.Private,
		QueryID:       queryID,
		Results:       results,
		CacheTime:     opt.CacheTime,
		NextOffset:    opt.NextOffset,
		SwitchPm:      opt.SwitchPM,
		SwitchWebview: opt.SwitchWebview,
	}

	_, err := c.Raw().MessagesSetInlineBotResults(ctx, req)
	return err
}

// AnswerGuestQuery sends an inline result as a reply to a received guest
// message. guestQueryID is the string identifier from Message.GuestQueryID.
//
// Returns the sent guest message or an error if the query ID is invalid, the
// RPC call fails, or Telegram returns an unexpected result type.
func (c *Client) AnswerGuestQuery(ctx context.Context, guestQueryID string, result tg.InputBotInlineResultClass) (*types.SentGuestMessage, error) {
	c.Log.Debugf("AnswerGuestQuery query_id=%s", guestQueryID)

	queryID, err := strconv.ParseInt(guestQueryID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse guest query ID: %w", err)
	}

	raw, err := c.Raw().Invoke(ctx, &tg.MessagesSetBotGuestChatResultRequest{
		QueryID: queryID,
		Result:  result,
	}, tg.ReadTLObject)
	if err != nil {
		return nil, fmt.Errorf("answer guest query: %w", err)
	}

	inlineMessageID, ok := raw.(tg.InputBotInlineMessageIDClass)
	if !ok {
		return nil, fmt.Errorf("answer guest query: unexpected result type %T", raw)
	}

	return types.ParseSentGuestMessage(inlineMessageID), nil
}

// GetInlineBotResults requests inline results from a bot for the given query.
// The results can then be sent using [Client.SendInlineBotResult].
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - bot: user ID of the inline bot to query
//   - chatID: chat where the inline query originates
//   - query: the search text entered by the user
//   - offset: pagination offset string (empty for first page)
//
// Returns the bot results containing inline result items, or an error if the
// chat or bot peer cannot be resolved or the RPC call fails.
//
// Example:
//
//	results, err := client.GetInlineBotResults(ctx, botID, chatID, "cats", "")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, r := range results.Results {
//	    fmt.Printf("Result: %T\n", r)
//	}
func (c *Client) GetInlineBotResults(ctx context.Context, bot int64, chatID int64, query, offset string) (*tg.MessagesBotResults, error) {
	c.Log.Debug("GetInlineBotResults")
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve chat: %w", err)
	}
	user, err := resolveUserID(c, bot)
	if err != nil {
		return nil, fmt.Errorf("resolve bot: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetInlineBotResults(ctx, &tg.MessagesGetInlineBotResultsRequest{
		Bot:    user,
		Peer:   peer,
		Query:  query,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("get inline bot results: %w", err)
	}
	return result, nil
}

// SendInlineBotResult sends a specific inline bot result to a chat. Obtain the
// queryID and resultID from [Client.GetInlineBotResults].
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: target chat identifier to send the result to
//   - queryID: unique identifier of the inline query
//   - resultID: unique identifier of the chosen inline result
//   - opts: optional [SendInlineBotResultOption] configuration
//
// Returns the sent message or an error if the chat cannot be resolved or the
// RPC call fails.
//
// Example:
//
//	msg, err := client.SendInlineBotResult(ctx, chatID, queryID, "1", &telegram.SendInlineBotResultOption{
//	    HideVia: true,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Sent message ID: %d\n", msg.ID)
func (c *Client) SendInlineBotResult(ctx context.Context, chatID int64, queryID int64, resultID string, opts ...*SendInlineBotResultOption) (*types.Message, error) {
	c.Log.Debugf("SendInlineBotResult chat_id=%d", chatID)
	opt := getOptDef(&SendInlineBotResultOption{}, opts...)

	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve chat: %w", err)
	}

	var flags tg.Fields
	if opt.Silent {
		flags.Set(5)
	}
	if opt.HideVia {
		flags.Set(11)
	}
	if opt.ClearDraft {
		flags.Set(7)
	}
	if opt.ScheduleDate != nil {
		flags.Set(10)
	}

	req := &tg.MessagesSendInlineBotResultRequest{
		Flags:    flags,
		Silent:   opt.Silent,
		HideVia:  opt.HideVia,
		Peer:     peer,
		RandomID: c.RandomID(),
		QueryID:  queryID,
		ID:       resultID,
	}
	if opt.ReplyTo != 0 {
		req.Flags |= 1 << 0
		req.ReplyTo = &tg.InputReplyToMessage{ReplyToMsgID: int32(opt.ReplyTo)}
	}
	if opt.ScheduleDate != nil {
		req.ScheduleDate = *opt.ScheduleDate
	}

	result, err := c.Raw().MessagesSendInlineBotResult(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("send inline bot result: %w", err)
	}

	return extractSingleMessage(result, c)
}
