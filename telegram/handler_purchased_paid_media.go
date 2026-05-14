package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// PurchasedPaidMediaHandler processes updates when a user purchases paid media
// in a chat. Use this to fulfill paid media purchases, grant access to purchased
// content, or log transaction records for paid media features.
type PurchasedPaidMediaHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and PurchasedPaidMedia when a client-type callback is provided.
	callbackClient func(*Client, *types.PurchasedPaidMedia)
	// callbackFull is invoked with both the Context and PurchasedPaidMedia when a full-type callback is provided.
	callbackFull func(*Context, *types.PurchasedPaidMedia)
}

// NewPurchasedPaidMediaHandler creates a handler for paid media purchase updates.
// The callback must be one of:
//   - func(*Context):                            receives only the handler context
//   - func(*Client, *types.PurchasedPaidMedia):  receives the client and the purchase details
//   - func(*Context, *types.PurchasedPaidMedia): receives both the context and the purchase details
//
// Optional filters can be provided to further restrict which updates are handled.
func NewPurchasedPaidMediaHandler(callback interface{}, filters ...Filter) *PurchasedPaidMediaHandler {
	h := &PurchasedPaidMediaHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.PurchasedPaidMedia):
		h.callbackClient = fn
	case func(*Context, *types.PurchasedPaidMedia):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a PurchasedPaidMedia field
// and passes the configured filters. Returns false if the update does not represent
// a paid media purchase.
func (h *PurchasedPaidMediaHandler) Check(update *Update) bool {
	if update.PurchasedPaidMedia == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, PurchasedPaidMedia: update.PurchasedPaidMedia}
	return h.filters(ctx)
}

// Handle dispatches the paid media purchase to whichever callback variant was
// provided at construction time. The full callback is preferred, followed by the
// client callback, then the context-only callback.
func (h *PurchasedPaidMediaHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.PurchasedPaidMedia)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.PurchasedPaidMedia)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
