package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// DeletedBusinessMessagesHandler processes updates about messages that were deleted
// through a business connection. Use this to track message removals performed on
// behalf of a business account via the bot.
type DeletedBusinessMessagesHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and DeletedMessages when a client-type callback is provided.
	callbackClient func(*Client, *types.DeletedMessages)
	// callbackFull is invoked with both the Context and DeletedMessages when a full-type callback is provided.
	callbackFull func(*Context, *types.DeletedMessages)
}

// NewDeletedBusinessMessagesHandler creates a handler for deleted business message updates.
// The callback must be one of:
//   - func(*Context):                        receives only the handler context
//   - func(*Client, *types.DeletedMessages): receives the client and the deleted messages info
//   - func(*Context, *types.DeletedMessages): receives both the context and the deleted messages info
//
// Optional filters can be provided to further restrict which updates are handled.
func NewDeletedBusinessMessagesHandler(callback interface{}, filters ...Filter) *DeletedBusinessMessagesHandler {
	h := &DeletedBusinessMessagesHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.DeletedMessages):
		h.callbackClient = fn
	case func(*Context, *types.DeletedMessages):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a DeletedBusinessMessages field
// and passes the configured filters. Returns false if the update does not represent
// deleted business messages.
func (h *DeletedBusinessMessagesHandler) Check(update *Update) bool {
	if update.DeletedBusinessMessages == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, DeletedBusinessMessages: update.DeletedBusinessMessages}
	return h.filters(ctx)
}

// Handle dispatches the deleted business messages to whichever callback variant was
// provided at construction time. The full callback is preferred, followed by the
// client callback, then the context-only callback.
func (h *DeletedBusinessMessagesHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.DeletedBusinessMessages)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.DeletedBusinessMessages)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
