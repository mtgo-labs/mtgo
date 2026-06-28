package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// GuestMessageHandler processes incoming messages sent by guest users pending
// approval in a Telegram Business chat. These are regular messages where
// IsFromPending is true.
type GuestMessageHandler struct {
	baseHandler
	callbackCtx    func(*Context)
	callbackClient func(*Client, *types.Message)
	callbackFull   func(*Context, *types.Message)
	callbackAll    func(*Context, *Client, *types.Message)
}

// NewGuestMessageHandler creates a new GuestMessageHandler with the given callback
// function and optional filters. A GuestMessage filter is automatically prepended.
// The callback must be one of four supported signatures.
//
// Example:
//
//	client.AddHandler(telegram.NewGuestMessageHandler(func(ctx *telegram.Context, msg *types.Message) {
//		fmt.Printf("guest message from user %d: %s\n", msg.From.ID, msg.Text)
//	}))
func NewGuestMessageHandler(callback any, filters ...Filter) *GuestMessageHandler {
	allFilters := append([]Filter{GuestMessage}, filters...)
	h := &GuestMessageHandler{baseHandler: baseHandler{filters: mergeFilters(allFilters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.Message):
		h.callbackClient = fn
	case func(*Context, *types.Message):
		h.callbackFull = fn
	case func(*Context, *Client, *types.Message):
		h.callbackAll = fn
	}
	return h
}

// Check reports whether the incoming update contains a guest message and passes
// the configured filters.
func (h *GuestMessageHandler) Check(update *Update) bool {
	if update.Message == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, Message: update.Message}
	return h.filters(ctx)
}

// Handle dispatches the guest message to whichever callback variant was provided
// at construction time.
//
// Example (the handler is invoked automatically by the dispatch loop):
//
//	// After registration via AddHandler, Handle is called for each matching update:
//	h := telegram.NewGuestMessageHandler(func(ctx *telegram.Context, msg *types.Message) {
//		ctx.Client.SendMessage(ctx, msg.Chat.ID, "Welcome, guest!")
//	})
//	client.AddHandler(h)
func (h *GuestMessageHandler) Handle(ctx *Context) {
	switch {
	case h.callbackAll != nil:
		h.callbackAll(ctx, ctx.Client, ctx.Message)
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.Message)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.Message)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
