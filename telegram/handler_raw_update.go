package telegram

import (
	"reflect"
)

// RawUpdateHandler processes incoming raw TL updates. It supports two modes:
//
//   - Catch-all: callback is func(*Context) — fires for every update.
//   - Typed: callback is func(T), func(*Context, T), or func(*Client, T)
//     where T is a concrete *tg.UpdateXxx type — fires only when the raw
//     update matches that type.
//
// Use the typed form to handle specific update types without manual
// type-switching:
//
//	client.OnRawUpdate(func(upd *tg.UpdatePhoneCallSignalingData) {
//	    log.Printf("phone call signaling: %x", upd.Data)
//	})
//
// With context:
//
//	client.OnRawUpdate(func(ctx *telegram.Context, upd *tg.UpdateUserTyping) {
//	    ctx.Reply("I see you typing!")
//	})
type RawUpdateHandler struct {
	baseHandler
	// callbackCtx is invoked for catch-all handlers: func(*Context).
	callbackCtx func(*Context)
	// updateType is the reflect.Type of the concrete *tg.UpdateXxx for typed
	// handlers. nil for catch-all.
	updateType reflect.Type
	// typedDispatch calls the typed callback with the matched update.
	typedDispatch func(ctx *Context)
}

var contextPtrType = reflect.TypeOf((*Context)(nil))

// NewRawUpdateHandler creates a raw update handler. The callback determines
// the mode:
//
//   - func(*Context): catch-all, fires for every update (backward compatible).
//   - func(T): typed, fires only when Raw is of type T.
//   - func(*Context, T): typed with context access.
//   - func(*Client, T): typed with client access.
//
// where T is a concrete *tg.UpdateXxx type.
//
// Optional filters further restrict which updates trigger the handler.
func NewRawUpdateHandler(callback any, filters ...Filter) *RawUpdateHandler {
	h := &RawUpdateHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}

	// Catch-all: func(*Context)
	if fn, ok := callback.(func(*Context)); ok {
		h.callbackCtx = fn
		return h
	}

	// Typed callbacks via reflection
	if callback == nil {
		return h
	}
	rv := reflect.ValueOf(callback)
	rt := rv.Type()
	if rt.Kind() != reflect.Func || rt.NumOut() != 0 || rt.NumIn() < 1 || rt.NumIn() > 2 {
		return h
	}

	var updType reflect.Type
	var buildArgs func(ctx *Context) []reflect.Value

	switch rt.NumIn() {
	case 1:
		// func(T)
		updType = rt.In(0)
		buildArgs = func(ctx *Context) []reflect.Value {
			return []reflect.Value{reflect.ValueOf(ctx.Update.Raw)}
		}
	case 2:
		// func(*Context, T) or func(*Client, T)
		first := rt.In(0)
		updType = rt.In(1)
		buildArgs = func(ctx *Context) []reflect.Value {
			var firstVal reflect.Value
			if first == contextPtrType {
				firstVal = reflect.ValueOf(ctx)
			} else {
				firstVal = reflect.ValueOf(ctx.Client)
			}
			return []reflect.Value{firstVal, reflect.ValueOf(ctx.Update.Raw)}
		}
	}

	// updateType must be a pointer (e.g. *tg.UpdatePhoneCall)
	if updType.Kind() != reflect.Ptr {
		return h
	}

	h.updateType = updType
	h.typedDispatch = func(ctx *Context) {
		rv.Call(buildArgs(ctx))
	}
	return h
}

// Check reports whether the incoming update matches the handler's type filter
// (for typed handlers) and passes any configured filters.
func (h *RawUpdateHandler) Check(update *Update) bool {
	// Typed handler: verify raw update matches the expected type.
	if h.updateType != nil {
		if update.Raw == nil || reflect.TypeOf(update.Raw) != h.updateType {
			return false
		}
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update}
	return h.filters(ctx)
}

// Handle invokes the callback. For typed handlers the matched update is passed
// directly; for catch-all handlers the Context is passed.
func (h *RawUpdateHandler) Handle(ctx *Context) {
	if h.typedDispatch != nil {
		h.typedDispatch(ctx)
		return
	}
	if h.callbackCtx != nil {
		h.callbackCtx(ctx)
	}
}
