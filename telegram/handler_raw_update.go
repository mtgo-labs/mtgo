package telegram

// RawUpdateHandler processes every incoming update without filtering by update type.
// Use this as a catch-all handler when you need to inspect or log all raw updates
// received from Telegram, regardless of their specific type.
type RawUpdateHandler struct {
	baseHandler
	// callbackCtx is invoked with the handler Context for every update that passes the filters.
	callbackCtx func(*Context)
}

// NewRawUpdateHandler creates a handler that fires for all incoming updates.
// The callback must be of type func(*Context). Optional filters can be provided
// to restrict which updates trigger the handler, but unlike typed handlers this
// accepts all update types by default.
func NewRawUpdateHandler(callback any, filters ...Filter) *RawUpdateHandler {
	h := &RawUpdateHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	if fn, ok := callback.(func(*Context)); ok {
		h.callbackCtx = fn
	}
	return h
}

// Check reports whether the incoming update passes the configured filters. Unlike
// typed handlers, this always returns true when no filters are set because it
// matches all update types.
func (h *RawUpdateHandler) Check(update *Update) bool {
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update}
	return h.filters(ctx)
}

// Handle invokes the callback with the current context for every update that
// passes the filter check.
func (h *RawUpdateHandler) Handle(ctx *Context) {
	if h.callbackCtx != nil {
		h.callbackCtx(ctx)
	}
}
