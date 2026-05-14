package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// ManagedBotHandler processes updates about changes to a managed bot's connection
// state. Use this when operating as a business or managed bot to track when the
// bot is connected or disconnected by the managing application.
type ManagedBotHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and ManagedBotUpdated when a client-type callback is provided.
	callbackClient func(*Client, *types.ManagedBotUpdated)
	// callbackFull is invoked with both the Context and ManagedBotUpdated when a full-type callback is provided.
	callbackFull func(*Context, *types.ManagedBotUpdated)
}

// NewManagedBotHandler creates a handler for managed bot connection updates.
// The callback must be one of:
//   - func(*Context):                            receives only the handler context
//   - func(*Client, *types.ManagedBotUpdated):   receives the client and the managed bot update
//   - func(*Context, *types.ManagedBotUpdated):  receives both the context and the managed bot update
//
// Optional filters can be provided to further restrict which updates are handled.
func NewManagedBotHandler(callback interface{}, filters ...Filter) *ManagedBotHandler {
	h := &ManagedBotHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.ManagedBotUpdated):
		h.callbackClient = fn
	case func(*Context, *types.ManagedBotUpdated):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a ManagedBot field and passes
// the configured filters. Returns false if the update does not represent a managed
// bot change.
func (h *ManagedBotHandler) Check(update *Update) bool {
	if update.ManagedBot == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, ManagedBot: update.ManagedBot}
	return h.filters(ctx)
}

// Handle dispatches the managed bot update to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *ManagedBotHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.ManagedBot)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.ManagedBot)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
