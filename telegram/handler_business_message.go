package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// BusinessMessageHandler processes new messages sent through a business connection
// on behalf of a business account. Use this when the bot is acting as a business
// bot and needs to handle messages created in that context.
type BusinessMessageHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and Message when a client-type callback is provided.
	callbackClient func(*Client, *types.Message)
	// callbackFull is invoked with both the Context and Message when a full-type callback is provided.
	callbackFull func(*Context, *types.Message)
}

// NewBusinessMessageHandler creates a handler for new business messages.
// The callback must be one of:
//   - func(*Context):                  receives only the handler context
//   - func(*Client, *types.Message):   receives the client and the business message
//   - func(*Context, *types.Message):   receives both the context and the business message
//
// Optional filters can be provided to further restrict which updates are handled.
func NewBusinessMessageHandler(callback interface{}, filters ...Filter) *BusinessMessageHandler {
	h := &BusinessMessageHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.Message):
		h.callbackClient = fn
	case func(*Context, *types.Message):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a BusinessMessage field
// and passes the configured filters. Returns false if the update does not
// represent a new business message.
func (h *BusinessMessageHandler) Check(update *Update) bool {
	if update.BusinessMessage == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, BusinessMessage: update.BusinessMessage}
	return h.filters(ctx)
}

// Handle dispatches the business message to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *BusinessMessageHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.BusinessMessage)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.BusinessMessage)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
