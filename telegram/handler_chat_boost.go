package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// ChatBoostHandler processes updates when a chat boost is added or changed.
// Use this to react to users boosting a chat, which can grant additional
// features or privileges within the chat.
type ChatBoostHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and ChatBoostUpdated when a client-type callback is provided.
	callbackClient func(*Client, *types.ChatBoostUpdated)
	// callbackFull is invoked with both the Context and ChatBoostUpdated when a full-type callback is provided.
	callbackFull func(*Context, *types.ChatBoostUpdated)
}

// NewChatBoostHandler creates a handler for chat boost updates.
// The callback must be one of:
//   - func(*Context):                          receives only the handler context
//   - func(*Client, *types.ChatBoostUpdated):  receives the client and the boost update
//   - func(*Context, *types.ChatBoostUpdated): receives both the context and the boost update
//
// Optional filters can be provided to further restrict which updates are handled.
func NewChatBoostHandler(callback any, filters ...Filter) *ChatBoostHandler {
	h := &ChatBoostHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.ChatBoostUpdated):
		h.callbackClient = fn
	case func(*Context, *types.ChatBoostUpdated):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a ChatBoost field and passes
// the configured filters. Returns false if the update does not represent a chat
// boost change.
func (h *ChatBoostHandler) Check(update *Update) bool {
	if update.ChatBoost == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, ChatBoost: update.ChatBoost}
	return h.filters(ctx)
}

// Handle dispatches the chat boost update to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *ChatBoostHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.ChatBoost)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.ChatBoost)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
