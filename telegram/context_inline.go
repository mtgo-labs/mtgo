package telegram

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// AnswerInlineQuery sends a list of inline results to the user in response to the current
// inline query. If the context has no active inline query, the call is silently ignored.
// Results are displayed to the user as they type in any chat via the inline mode.
//
// Parameters:
//   - results: slice of inline results to present to the user
//   - opts: optional [AnswerInlineQueryOption] parameters for gallery mode, caching, and pagination
//
// Returns:
//   - error: non-nil if the results could not be sent
//
// Example:
//
//	client.OnInlineQuery(func(ctx *telegram.Context) {
//	    results := []tg.InputBotInlineResultClass{
//	        &tg.InputBotInlineResultPhoto{
//	            ID:    "1",
//	            Title: "Example Photo",
//	            Photo: &tg.InputPhoto{ID: 12345},
//	        },
//	    }
//	    ctx.AnswerInlineQuery(results)
//	})
func (c *Context) AnswerInlineQuery(results []tg.InputBotInlineResultClass, opts ...*AnswerInlineQueryOption) error {
	if c.InlineQuery == nil {
		return nil
	}
	return c.Client.AnswerInlineQuery(c.Ctx, c.InlineQuery.ID, results, opts...)
}

// AnswerInline is a short alias for [Context.AnswerInlineQuery].
func (c *Context) AnswerInline(results []tg.InputBotInlineResultClass, opts ...*AnswerInlineQueryOption) error {
	return c.AnswerInlineQuery(results, opts...)
}

// AnswerShipping responds to a shipping query received during a Telegram Stars or regular
// payment flow. Set ok to true with shipping options to confirm, or false to report that
// shipping is unavailable.
//
// Parameters:
//   - queryID: the shipping query ID to respond to
//   - ok: true if shipping is available, false to decline
//   - options: available shipping options (only used when ok is true)
//
// Returns:
//   - error: non-nil if the context has no client or the response could not be sent
func (c *Context) AnswerShipping(queryID int64, ok bool, options []*tg.ShippingOption) error {
	if c.Client == nil {
		return fmt.Errorf("context: no client")
	}
	return c.Client.AnswerShippingQuery(c.Ctx, queryID, ok, options)
}

// AnswerPreCheckout responds to a pre-checkout query, confirming or rejecting the payment
// before it is finalized. Set ok to true to confirm, or false with an errorMessage to
// explain why the payment was rejected.
//
// Parameters:
//   - queryID: the pre-checkout query ID to respond to
//   - ok: true to confirm the payment, false to reject
//   - errorMessage: human-readable reason for rejection (only used when ok is false)
//
// Returns:
//   - error: non-nil if the context has no client or the response could not be sent
func (c *Context) AnswerPreCheckout(queryID int64, ok bool, errorMessage string) error {
	if c.Client == nil {
		return fmt.Errorf("context: no client")
	}
	return c.Client.AnswerPreCheckoutQuery(c.Ctx, queryID, ok, errorMessage)
}
