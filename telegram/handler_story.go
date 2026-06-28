package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// StoryHandler processes updates about new or changed Telegram stories posted by
// users the bot is connected to. Use this to monitor story activity, such as
// detecting when a user posts a new story or edits an existing one.
type StoryHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and Story when a client-type callback is provided.
	callbackClient func(*Client, *types.Story)
	// callbackFull is invoked with both the Context and Story when a full-type callback is provided.
	callbackFull func(*Context, *types.Story)
}

// NewStoryHandler creates a handler for story updates.
// The callback must be one of:
//   - func(*Context):                 receives only the handler context
//   - func(*Client, *types.Story):    receives the client and the story
//   - func(*Context, *types.Story):   receives both the context and the story
//
// Optional filters can be provided to further restrict which updates are handled.
func NewStoryHandler(callback any, filters ...Filter) *StoryHandler {
	h := &StoryHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.Story):
		h.callbackClient = fn
	case func(*Context, *types.Story):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a Story field and passes the
// configured filters. Returns false if the update does not represent a story change.
func (h *StoryHandler) Check(update *Update) bool {
	if update.Story == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, Story: update.Story}
	return h.filters(ctx)
}

// Handle dispatches the story update to whichever callback variant was provided at
// construction time. The full callback is preferred, followed by the client callback,
// then the context-only callback.
func (h *StoryHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.Story)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.Story)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
