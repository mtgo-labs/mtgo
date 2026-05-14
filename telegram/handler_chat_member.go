package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// ChatMemberHandler processes updates about changes to the status of a chat member.
// Use this to track when users are kicked, banned, promoted, demoted, or leave/join
// a chat for which the bot has the appropriate administrator rights.
type ChatMemberHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and ChatMemberUpdated when a client-type callback is provided.
	callbackClient func(*Client, *types.ChatMemberUpdated)
	// callbackFull is invoked with both the Context and ChatMemberUpdated when a full-type callback is provided.
	callbackFull func(*Context, *types.ChatMemberUpdated)
}

// NewChatMemberHandler creates a handler for chat member status changes.
// The callback must be one of:
//   - func(*Context):                          receives only the handler context
//   - func(*Client, *types.ChatMemberUpdated): receives the client and the member update
//   - func(*Context, *types.ChatMemberUpdated): receives both the context and the member update
//
// Optional filters can be provided to further restrict which updates are handled.
func NewChatMemberHandler(callback interface{}, filters ...Filter) *ChatMemberHandler {
	h := &ChatMemberHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.ChatMemberUpdated):
		h.callbackClient = fn
	case func(*Context, *types.ChatMemberUpdated):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a ChatMember field and passes
// the configured filters. Returns false if the update does not represent a chat
// member status change.
func (h *ChatMemberHandler) Check(update *Update) bool {
	if update.ChatMember == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, ChatMember: update.ChatMember}
	return h.filters(ctx)
}

// Handle dispatches the chat member update to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *ChatMemberHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.ChatMember)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.ChatMember)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
