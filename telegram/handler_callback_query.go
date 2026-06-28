package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// CallbackQueryHandler processes callback queries originating from inline keyboard
// button presses. Use this to respond to user interactions with inline buttons
// attached to messages sent by the bot.
type CallbackQueryHandler struct {
	baseHandler
	callbackCtx    func(*Context)
	callbackClient func(*Client, *types.CallbackQuery)
	callbackFull   func(*Context, *types.CallbackQuery)
	callbackAll    func(*Context, *Client, *types.CallbackQuery)
}

// NewCallbackQueryHandler creates a new CallbackQueryHandler with the given callback
// function and optional filters. The callback must be one of four supported signatures.
//
// Example:
//
//	client.AddHandler(telegram.NewCallbackQueryHandler(func(ctx *telegram.Context) {
//		fmt.Println("callback data:", ctx.CallbackQuery.Data)
//	}))
//
// With filters:
//
//	client.AddHandler(telegram.NewCallbackQueryHandler(
//		func(ctx *telegram.Context, cb *types.CallbackQuery) {
//			fmt.Println("button:", cb.Data)
//		},
//		telegram.PrefixFilter("action_"),
//	))
func NewCallbackQueryHandler(callback any, filters ...Filter) *CallbackQueryHandler {
	h := &CallbackQueryHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.CallbackQuery):
		h.callbackClient = fn
	case func(*Context, *types.CallbackQuery):
		h.callbackFull = fn
	case func(*Context, *Client, *types.CallbackQuery):
		h.callbackAll = fn
	}
	return h
}

// Check reports whether the incoming update contains a CallbackQuery field and
// passes the configured filters. Returns false if the update does not represent
// a callback query from an inline keyboard.
func (h *CallbackQueryHandler) Check(update *Update) bool {
	if update.CallbackQuery == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, CallbackQuery: update.CallbackQuery}
	return h.filters(ctx)
}

// Handle dispatches the callback query to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *CallbackQueryHandler) Handle(ctx *Context) {
	switch {
	case h.callbackAll != nil:
		h.callbackAll(ctx, ctx.Client, ctx.CallbackQuery)
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.CallbackQuery)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.CallbackQuery)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
