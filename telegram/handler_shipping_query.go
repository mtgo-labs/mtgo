package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// ShippingQueryHandler processes shipping queries sent when a user submits payment
// information that requires shipping details. Use this to provide available shipping
// options and their costs before the user finalizes the payment.
type ShippingQueryHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and ShippingQuery when a client-type callback is provided.
	callbackClient func(*Client, *types.ShippingQuery)
	// callbackFull is invoked with both the Context and ShippingQuery when a full-type callback is provided.
	callbackFull func(*Context, *types.ShippingQuery)
}

// NewShippingQueryHandler creates a handler for shipping query updates received
// during the Telegram payment flow.
// The callback must be one of:
//   - func(*Context):                        receives only the handler context
//   - func(*Client, *types.ShippingQuery):   receives the client and the shipping query
//   - func(*Context, *types.ShippingQuery):  receives both the context and the shipping query
//
// Optional filters can be provided to further restrict which updates are handled.
func NewShippingQueryHandler(callback interface{}, filters ...Filter) *ShippingQueryHandler {
	h := &ShippingQueryHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.ShippingQuery):
		h.callbackClient = fn
	case func(*Context, *types.ShippingQuery):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a ShippingQuery field and
// passes the configured filters. Returns false if the update does not represent a
// shipping query.
func (h *ShippingQueryHandler) Check(update *Update) bool {
	if update.ShippingQuery == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, ShippingQuery: update.ShippingQuery}
	return h.filters(ctx)
}

// Handle dispatches the shipping query to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *ShippingQueryHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.ShippingQuery)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.ShippingQuery)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
