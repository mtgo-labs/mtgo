package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// ChosenInlineResultHandler processes updates when a user selects a result from an
// inline query. Use this to track which inline results users actually choose, allowing
// the bot to learn user preferences or log inline interaction analytics.
type ChosenInlineResultHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and ChosenInlineResult when a client-type callback is provided.
	callbackClient func(*Client, *types.ChosenInlineResult)
	// callbackFull is invoked with both the Context and ChosenInlineResult when a full-type callback is provided.
	callbackFull func(*Context, *types.ChosenInlineResult)
}

// NewChosenInlineResultHandler creates a handler for chosen inline result updates.
// The callback must be one of:
//   - func(*Context):                            receives only the handler context
//   - func(*Client, *types.ChosenInlineResult):  receives the client and the chosen result
//   - func(*Context, *types.ChosenInlineResult): receives both the context and the chosen result
//
// Optional filters can be provided to further restrict which updates are handled.
func NewChosenInlineResultHandler(callback any, filters ...Filter) *ChosenInlineResultHandler {
	h := &ChosenInlineResultHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.ChosenInlineResult):
		h.callbackClient = fn
	case func(*Context, *types.ChosenInlineResult):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a ChosenInlineResult field and
// passes the configured filters. Returns false if the update does not represent a
// user choosing an inline result.
func (h *ChosenInlineResultHandler) Check(update *Update) bool {
	if update.ChosenInlineResult == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, ChosenInlineResult: update.ChosenInlineResult}
	return h.filters(ctx)
}

// Handle dispatches the chosen inline result to whichever callback variant was
// provided at construction time. The full callback is preferred, followed by the
// client callback, then the context-only callback.
func (h *ChosenInlineResultHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.ChosenInlineResult)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.ChosenInlineResult)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
