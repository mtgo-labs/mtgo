package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// PollHandler processes updates about changes to poll state, such as when a user
// votes in a poll or a poll is closed. Use this to track poll progress and detect
// when final results are available.
type PollHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and PollUpdate when a client-type callback is provided.
	callbackClient func(*Client, *types.PollUpdated)
	// callbackFull is invoked with both the Context and PollUpdate when a full-type callback is provided.
	callbackFull func(*Context, *types.PollUpdated)
}

// NewPollHandler creates a handler for poll state updates.
// The callback must be one of:
//   - func(*Context):                     receives only the handler context
//   - func(*Client, *types.PollUpdated):   receives the client and the poll update
//   - func(*Context, *types.PollUpdated):  receives both the context and the poll update
//
// Optional filters can be provided to further restrict which updates are handled.
func NewPollHandler(callback interface{}, filters ...Filter) *PollHandler {
	h := &PollHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.PollUpdated):
		h.callbackClient = fn
	case func(*Context, *types.PollUpdated):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a Poll field and passes the
// configured filters. Returns false if the update does not represent a poll change.
func (h *PollHandler) Check(update *Update) bool {
	if update.Poll == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, Poll: update.Poll}
	return h.filters(ctx)
}

// Handle dispatches the poll update to whichever callback variant was provided at
// construction time. The full callback is preferred, followed by the client callback,
// then the context-only callback.
func (h *PollHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.Poll)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.Poll)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
