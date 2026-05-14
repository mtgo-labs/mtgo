package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// InlineQueryHandler processes incoming inline queries sent when a user types
// the bot's username in any chat. Use this to provide inline results (articles,
// photos, GIFs, etc.) that the user can send without leaving the current chat.
type InlineQueryHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and InlineQuery when a client-type callback is provided.
	callbackClient func(*Client, *types.InlineQuery)
	// callbackFull is invoked with both the Context and InlineQuery when a full-type callback is provided.
	callbackFull func(*Context, *types.InlineQuery)
}

// NewInlineQueryHandler creates a handler for inline query updates.
// The callback must be one of:
//   - func(*Context):                     receives only the handler context
//   - func(*Client, *types.InlineQuery):  receives the client and the inline query
//   - func(*Context, *types.InlineQuery): receives both the context and the inline query
//
// Optional filters can be provided to further restrict which updates are handled.
//
// Example:
//
//	h := telegram.NewInlineQueryHandler(func(ctx *telegram.Context) {
//	    results := []tg.InputBotInlineResultClass{
//	        &tg.InputBotInlineResultPhoto{ID: "1", ...},
//	    }
//	    ctx.AnswerInlineQuery(results)
//	})
//	client.AddHandler(h)
func NewInlineQueryHandler(callback interface{}, filters ...Filter) *InlineQueryHandler {
	h := &InlineQueryHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.InlineQuery):
		h.callbackClient = fn
	case func(*Context, *types.InlineQuery):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains an InlineQuery field and
// passes the configured filters. Returns false if the update does not represent
// an inline query.
func (h *InlineQueryHandler) Check(update *Update) bool {
	if update.InlineQuery == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, InlineQuery: update.InlineQuery}
	return h.filters(ctx)
}

// Handle dispatches the inline query to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *InlineQueryHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.InlineQuery)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.InlineQuery)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
