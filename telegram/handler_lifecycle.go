package telegram

// LifecycleHandler processes client lifecycle events such as connection, disconnection,
// start, and stop. Use this to perform initialization or cleanup tasks at specific
// points in the client's lifecycle, such as loading state on connect or persisting
// data on disconnect.
type LifecycleHandler struct {
	baseHandler
	// callbackCtx is invoked with the handler Context when the lifecycle event occurs.
	callbackCtx func(*Context)
	// kind identifies which lifecycle event this handler responds to: "connect",
	// "disconnect", "start", or "stop".
	kind string
}

// NewConnectHandler creates a handler that fires when the client successfully
// establishes a connection to the Telegram servers. Use this for post-connection
// initialization such as fetching initial state or sending queued messages.
func NewConnectHandler(callback func(*Context)) *LifecycleHandler {
	return &LifecycleHandler{baseHandler: baseHandler{}, callbackCtx: callback, kind: "connect"}
}

// NewDisconnectHandler creates a handler that fires when the client disconnects
// from the Telegram servers. Use this for cleanup tasks such as flushing buffers
// or releasing resources held during the connection.
func NewDisconnectHandler(callback func(*Context)) *LifecycleHandler {
	return &LifecycleHandler{baseHandler: baseHandler{}, callbackCtx: callback, kind: "disconnect"}
}

// NewStartHandler creates a handler that fires when the client starts up, before
// any connection attempt is made. Use this for one-time startup initialization.
//
// Example:
//
//	h := telegram.NewStartHandler(func(ctx *telegram.Context) {
//	    log.Println("Client is starting up...")
//	})
//	client.AddHandler(h)
func NewStartHandler(callback func(*Context)) *LifecycleHandler {
	return &LifecycleHandler{baseHandler: baseHandler{}, callbackCtx: callback, kind: "start"}
}

// NewStopHandler creates a handler that fires when the client is shutting down.
// Use this for final cleanup and graceful shutdown procedures.
func NewStopHandler(callback func(*Context)) *LifecycleHandler {
	return &LifecycleHandler{baseHandler: baseHandler{}, callbackCtx: callback, kind: "stop"}
}

// Check reports whether the incoming update matches the specific lifecycle event
// this handler is configured for. Returns true only when the update's corresponding
// lifecycle flag (Connected, Disconnected, Started, or Stopped) is set.
func (h *LifecycleHandler) Check(update *Update) bool {
	switch h.kind {
	case "connect":
		return update.Connected
	case "disconnect":
		return update.Disconnected
	case "start":
		return update.Started
	case "stop":
		return update.Stopped
	}
	return false
}

// Handle invokes the lifecycle callback with the current context when the
// corresponding lifecycle event is triggered.
func (h *LifecycleHandler) Handle(ctx *Context) {
	if h.callbackCtx != nil {
		h.callbackCtx(ctx)
	}
}
