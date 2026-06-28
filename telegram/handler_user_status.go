package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// UserStatusHandler processes updates about changes to a user's online status, such
// as going online, offline, or changing their last-seen visibility. Use this to track
// user presence for features like activity monitoring or conditional messaging.
type UserStatusHandler struct {
	baseHandler
	// callbackCtx is invoked with only the handler Context when a Context-only callback is provided.
	callbackCtx func(*Context)
	// callbackClient is invoked with the Client and UserStatusUpdated when a client-type callback is provided.
	callbackClient func(*Client, *types.UserStatusUpdated)
	// callbackFull is invoked with both the Context and UserStatusUpdated when a full-type callback is provided.
	callbackFull func(*Context, *types.UserStatusUpdated)
}

// NewUserStatusHandler creates a handler for user status change updates.
// The callback must be one of:
//   - func(*Context):                            receives only the handler context
//   - func(*Client, *types.UserStatusUpdated):   receives the client and the status update
//   - func(*Context, *types.UserStatusUpdated):  receives both the context and the status update
//
// Optional filters can be provided to further restrict which updates are handled.
func NewUserStatusHandler(callback any, filters ...Filter) *UserStatusHandler {
	h := &UserStatusHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.UserStatusUpdated):
		h.callbackClient = fn
	case func(*Context, *types.UserStatusUpdated):
		h.callbackFull = fn
	}
	return h
}

// Check reports whether the incoming update contains a UserStatus field and passes
// the configured filters. Returns false if the update does not represent a user
// status change.
func (h *UserStatusHandler) Check(update *Update) bool {
	if update.UserStatus == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, UserStatus: update.UserStatus}
	return h.filters(ctx)
}

// Handle dispatches the user status update to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *UserStatusHandler) Handle(ctx *Context) {
	switch {
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.UserStatus)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.UserStatus)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
