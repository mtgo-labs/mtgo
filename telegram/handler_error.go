package telegram

// ErrorHandler processes error updates that occur during client operation. Use this
// to capture and respond to runtime errors, such as network failures, API errors,
// or unexpected conditions encountered while processing updates.
type ErrorHandler struct {
	baseHandler
	// callbackCtx is invoked with the handler Context containing the error details.
	callbackCtx func(*Context)
	// errTypes is an optional list of error types used to filter which errors trigger
	// this handler. When empty, all errors are handled.
	errTypes []error
}

// NewErrorHandler creates a handler for error updates. The callback receives a Context
// whose Error field is populated with the encountered error. Optional exception types
// can be provided to restrict the handler to specific error categories; when no
// exceptions are given the handler matches all errors.
//
// Example:
//
//	h := telegram.NewErrorHandler(func(ctx *telegram.Context) {
//	    log.Printf("bot error: %v", ctx.Error)
//	})
//	client.AddHandler(h)
func NewErrorHandler(callback func(*Context), exceptions ...error) *ErrorHandler {
	return &ErrorHandler{baseHandler: baseHandler{}, callbackCtx: callback, errTypes: exceptions}
}

// Check reports whether the incoming update contains an Error. Returns true if an
// error is present, regardless of the configured exception types (future filtering
// may refine this behavior).
func (h *ErrorHandler) Check(update *Update) bool {
	if update.Error == nil {
		return false
	}
	if len(h.errTypes) == 0 {
		return true
	}
	return true
}

// Handle invokes the error callback with the current context, which carries the
// error details in its Error field.
func (h *ErrorHandler) Handle(ctx *Context) {
	if h.callbackCtx != nil {
		h.callbackCtx(ctx)
	}
}
