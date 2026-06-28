package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// EditedBusinessMessageHandler processes updates about messages that were edited
// through a business connection. Use this to react to edits made on messages sent
// on behalf of a business account.
type EditedBusinessMessageHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and Message when a client-type callback is provided.
	callbackClient func(*Client, *types.Message)
	// callbackFull is invoked with both the Context and Message when a full-type callback is provided.
	callbackFull func(*Context, *types.Message)
}

// NewEditedBusinessMessageHandler creates a handler for edited business message updates.
// The callback must be one of:
//   - func(*Context):                  receives only the handler context
//   - func(*Client, *types.Message):   receives the client and the edited business message
//   - func(*Context, *types.Message):   receives both the context and the edited business message
//
// Optional filters can be provided to further restrict which updates are handled.
func NewEditedBusinessMessageHandler(callback any, filters ...Filter) *EditedBusinessMessageHandler {
	h := &EditedBusinessMessageHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
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

// Check reports whether the incoming update contains an EditedBusinessMessage field
// and passes the configured filters. Returns false if the update does not represent
// an edited business message.
func (h *EditedBusinessMessageHandler) Check(update *Update) bool {
	if update.EditedBusinessMessage == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, EditedBusinessMessage: update.EditedBusinessMessage}
	return h.filters(ctx)
}

// Handle dispatches the edited business message to whichever callback variant was
// provided at construction time. The full callback is preferred, followed by the
// client callback, then the context-only callback.
func (h *EditedBusinessMessageHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.EditedBusinessMessage)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.EditedBusinessMessage)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
