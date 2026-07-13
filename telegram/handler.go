package telegram

// Handler is the interface for processing incoming Telegram updates. Implementations
// define both a predicate (Check) to decide whether an update is relevant, and an
// action (Handle) to execute when it is. Register handlers with
// HandlerDispatcher.AddHandler; the dispatcher calls Check on each registered
// handler and invokes Handle for every match.
//
// Example:
//
//	type pingHandler struct{}
//
//	func (h *pingHandler) Check(u *telegram.Update) bool {
//	    return u.Message != nil && u.Message.Text == "/ping"
//	}
//
//	func (h *pingHandler) Handle(ctx *telegram.Context) {
//	    fmt.Println("pong")
//	}
type Handler interface {
	// Check reports whether this handler should be invoked for the given update.
	// Return true to signal that Handle should be called.
	Check(update *Update) bool

	// Handle executes the handler logic for a matched update. The provided Context
	// carries the Client, the raw Update, and any populated convenience fields.
	// Set Context.StopPropagation to true to short-circuit the dispatch loop and prevent
	// lower-priority handlers from running.
	Handle(ctx *Context)
}

// Filter is a predicate over a dispatch Context. Use filters to narrow which
// updates a handler responds to (e.g. only text messages, only messages from a
// specific chat). Filters compose with And, Or, and Not.
//
// Example:
//
//	IsText := func(ctx *telegram.Context) bool {
//	    return ctx.Message != nil && ctx.Message.Text != ""
//	}
//	IsPrivate := func(ctx *telegram.Context) bool {
//	    return ctx.Message != nil && !ctx.Message.Group
//	}
//	combined := IsText.And(IsPrivate)
type Filter func(*Context) bool

// And returns a Filter that matches only when both the receiver and other match.
// Use it to require multiple conditions simultaneously, e.g. IsPrivate.And(IsText).
//
// Example:
//
//	onlyPrivateText := IsPrivate.And(IsText)
func (f Filter) And(other Filter) Filter {
	return func(ctx *Context) bool {
		return f(ctx) && other(ctx)
	}
}

// Or returns a Filter that matches when either the receiver or other matches.
// Use it to accept updates that satisfy at least one of several alternatives.
//
// Example:
//
//	textOrCommand := IsText.Or(IsCommand)
func (f Filter) Or(other Filter) Filter {
	return func(ctx *Context) bool {
		return f(ctx) || other(ctx)
	}
}

// Not returns a Filter that negates the receiver. Use it to exclude updates that
// would otherwise match, e.g. IsGroup.Not() to match only non-group messages.
//
// Example:
//
//	notBot := IsBot.Not()
func (f Filter) Not() Filter {
	return func(ctx *Context) bool {
		return !f(ctx)
	}
}

type baseHandler struct {
	filters Filter
}

func mergeFilters(filters []Filter) Filter {
	switch len(filters) {
	case 0:
		return nil
	case 1:
		return filters[0]
	default:
		return func(ctx *Context) bool {
			for _, f := range filters {
				if !f(ctx) {
					return false
				}
			}
			return true
		}
	}
}

// FuncHandler wraps a function as a Handler. Check always returns true.
// Useful for testing and for quick handler construction.
//
// Example:
//
//	client.AddHandler(&tg.FuncHandler{Fn: func(ctx *tg.Context) {
//	    ctx.Reply("pong")
//	}})
type FuncHandler struct {
	Fn func(*Context)
}

func (h *FuncHandler) Check(_ *Update) bool { return true }
func (h *FuncHandler) Handle(ctx *Context)  { h.Fn(ctx) }
