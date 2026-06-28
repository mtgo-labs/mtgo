package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// ChatJoinRequestHandler processes incoming requests to join a chat, sent when a
// user taps the "Join" button on a chat with join approval enabled. Use this to
// approve or decline join requests.
type ChatJoinRequestHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and ChatJoinRequest when a client-type callback is provided.
	callbackClient func(*Client, *types.ChatJoinRequest)
	// callbackFull is invoked with both the Context and ChatJoinRequest when a full-type callback is provided.
	callbackFull func(*Context, *types.ChatJoinRequest)
}

// NewChatJoinRequestHandler creates a handler for chat join request updates.
// The callback must be one of:
//   - func(*Context):                          receives only the handler context
//   - func(*Client, *types.ChatJoinRequest):   receives the client and the join request
//   - func(*Context, *types.ChatJoinRequest):  receives both the context and the join request
//
// Optional filters can be provided to further restrict which updates are handled.
func NewChatJoinRequestHandler(callback any, filters ...Filter) *ChatJoinRequestHandler {
	h := &ChatJoinRequestHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.ChatJoinRequest):
		h.callbackClient = fn
	case func(*Context, *types.ChatJoinRequest):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a ChatJoinRequest field and
// passes the configured filters. Returns false if the update does not represent
// a join request.
func (h *ChatJoinRequestHandler) Check(update *Update) bool {
	if update.ChatJoinRequest == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, ChatJoinRequest: update.ChatJoinRequest}
	return h.filters(ctx)
}

// Handle dispatches the chat join request to whichever callback variant was
// provided at construction time. The full callback is preferred, followed by
// the client callback, then the context-only callback.
func (h *ChatJoinRequestHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.ChatJoinRequest)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.ChatJoinRequest)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
