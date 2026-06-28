package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// BusinessConnectionHandler processes updates about changes to the bot's business
// connection status with a Business account. Use this to track when a business
// account connects or disconnects the bot, enabling or disabling business features.
type BusinessConnectionHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and BusinessConnection when a client-type callback is provided.
	callbackClient func(*Client, *types.BusinessConnection)
	// callbackFull is invoked with both the Context and BusinessConnection when a full-type callback is provided.
	callbackFull func(*Context, *types.BusinessConnection)
}

// NewBusinessConnectionHandler creates a handler for business connection updates.
// The callback must be one of:
//   - func(*Context):                        receives only the handler context
//   - func(*Client, *types.BusinessConnection): receives the client and the business connection
//   - func(*Context, *types.BusinessConnection): receives both the context and the business connection
//
// Optional filters can be provided to further restrict which updates are handled.
func NewBusinessConnectionHandler(callback any, filters ...Filter) *BusinessConnectionHandler {
	h := &BusinessConnectionHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.BusinessConnection):
		h.callbackClient = fn
	case func(*Context, *types.BusinessConnection):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a BusinessConnection field
// and passes the configured filters. Returns false if the update does not
// represent a business connection change.
func (h *BusinessConnectionHandler) Check(update *Update) bool {
	if update.BusinessConnection == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, BusinessConnection: update.BusinessConnection}
	return h.filters(ctx)
}

// Handle dispatches the business connection update to whichever callback variant
// was provided at construction time. The full callback is preferred, followed by
// the client callback, then the context-only callback.
func (h *BusinessConnectionHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.BusinessConnection)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.BusinessConnection)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
