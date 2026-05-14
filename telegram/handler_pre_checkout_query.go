package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// PreCheckoutQueryHandler processes pre-checkout queries sent when a user confirms
// payment in a third-party payment provider. Use this to verify order details and
// respond to the Telegram payment system before finalizing the transaction.
type PreCheckoutQueryHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and PreCheckoutQuery when a client-type callback is provided.
	callbackClient func(*Client, *types.PreCheckoutQuery)
	// callbackFull is invoked with both the Context and PreCheckoutQuery when a full-type callback is provided.
	callbackFull func(*Context, *types.PreCheckoutQuery)
}

// NewPreCheckoutQueryHandler creates a handler for pre-checkout payment queries.
// The callback must be one of:
//   - func(*Context):                           receives only the handler context
//   - func(*Client, *types.PreCheckoutQuery):   receives the client and the pre-checkout query
//   - func(*Context, *types.PreCheckoutQuery):  receives both the context and the pre-checkout query
//
// Optional filters can be provided to further restrict which updates are handled.
func NewPreCheckoutQueryHandler(callback interface{}, filters ...Filter) *PreCheckoutQueryHandler {
	h := &PreCheckoutQueryHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.PreCheckoutQuery):
		h.callbackClient = fn
	case func(*Context, *types.PreCheckoutQuery):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a PreCheckoutQuery field and
// passes the configured filters. Returns false if the update does not represent a
// pre-checkout query.
func (h *PreCheckoutQueryHandler) Check(update *Update) bool {
	if update.PreCheckoutQuery == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, PreCheckoutQuery: update.PreCheckoutQuery}
	return h.filters(ctx)
}

// Handle dispatches the pre-checkout query to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *PreCheckoutQueryHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.PreCheckoutQuery)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.PreCheckoutQuery)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
