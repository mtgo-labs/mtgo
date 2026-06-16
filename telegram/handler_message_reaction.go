package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// MessageReactionHandler processes updates when a reaction is added or removed from
// a message. Use this to track user engagement with reactions on messages in chats
// where the bot has the appropriate permissions.
type MessageReactionHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and MessageReactions when a client-type callback is provided.
	callbackClient func(*Client, *types.MessageReactionUpdate)
	// callbackFull is invoked with both the Context and MessageReactions when a full-type callback is provided.
	callbackFull func(*Context, *types.MessageReactionUpdate)
}

// NewMessageReactionHandler creates a handler for message reaction updates.
// The callback must be one of:
//   - func(*Context):                         receives only the handler context
//   - func(*Client, *types.MessageReactionUpdate): receives the client and the reaction update
//   - func(*Context, *types.MessageReactionUpdate): receives both the context and the reaction update
//
// Optional filters can be provided to further restrict which updates are handled.
func NewMessageReactionHandler(callback interface{}, filters ...Filter) *MessageReactionHandler {
	h := &MessageReactionHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.MessageReactionUpdate):
		h.callbackClient = fn
	case func(*Context, *types.MessageReactionUpdate):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a MessageReaction field and
// passes the configured filters. Returns false if the update does not represent a
// message reaction change.
func (h *MessageReactionHandler) Check(update *Update) bool {
	if update.MessageReaction == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, MessageReaction: update.MessageReaction}
	return h.filters(ctx)
}

// Handle dispatches the message reaction to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *MessageReactionHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.MessageReaction)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.MessageReaction)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
