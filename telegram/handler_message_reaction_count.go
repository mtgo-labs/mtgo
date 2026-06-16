package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// MessageReactionCountHandler processes updates about anonymous reaction count changes
// on messages. Unlike MessageReactionHandler, this provides only aggregate counts
// without identifying which user reacted, useful for privacy-conscious chats.
type MessageReactionCountHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and MessageReactions when a client-type callback is provided.
	callbackClient func(*Client, *types.MessageReactionCountUpdate)
	// callbackFull is invoked with both the Context and MessageReactions when a full-type callback is provided.
	callbackFull func(*Context, *types.MessageReactionCountUpdate)
}

// NewMessageReactionCountHandler creates a handler for anonymous reaction count updates.
// The callback must be one of:
//   - func(*Context):                         receives only the handler context
//   - func(*Client, *types.MessageReactionCountUpdate): receives the client and the reaction counts
//   - func(*Context, *types.MessageReactionCountUpdate): receives both the context and the reaction counts
//
// Optional filters can be provided to further restrict which updates are handled.
func NewMessageReactionCountHandler(callback interface{}, filters ...Filter) *MessageReactionCountHandler {
	h := &MessageReactionCountHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.MessageReactionCountUpdate):
		h.callbackClient = fn
	case func(*Context, *types.MessageReactionCountUpdate):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a MessageReactionCount field
// and passes the configured filters. Returns false if the update does not represent
// a reaction count change.
func (h *MessageReactionCountHandler) Check(update *Update) bool {
	if update.MessageReactionCount == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, MessageReactionCount: update.MessageReactionCount}
	return h.filters(ctx)
}

// Handle dispatches the reaction count update to whichever callback variant was
// provided at construction time. The full callback is preferred, followed by the
// client callback, then the context-only callback.
func (h *MessageReactionCountHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.MessageReactionCount)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.MessageReactionCount)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
