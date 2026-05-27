package telegram

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type handlerEntry struct {
	handler Handler
	group   int
}

// HandlerDispatcher routes incoming Telegram updates to registered Handlers.
// Handlers are grouped by integer priority (lower group numbers run first).
// The dispatcher is safe for concurrent use: handlers may be added or removed
// while dispatch is in progress.
type HandlerDispatcher struct {
	// mu protects handlers, dirty, and sorted for concurrent access.
	mu sync.RWMutex

	// handlers holds all registered handlers in insertion order.
	handlers []handlerEntry

	// dirty is set to true whenever handlers is modified, signalling that
	// sorted needs to be recomputed on the next dispatch.
	dirty bool

	// sorted is the cached, group-sorted snapshot of handlers. It is rebuilt
	// lazily when dirty is true.
	sorted []handlerEntry
}

// NewHandlerDispatcher creates and returns a ready-to-use HandlerDispatcher
// with no registered handlers.
//
// Example:
//
//	disp := telegram.NewHandlerDispatcher()
//	disp.AddHandler(&pingHandler{}, 0)
func NewHandlerDispatcher() *HandlerDispatcher {
	return &HandlerDispatcher{}
}

// AddHandler registers h with the dispatcher. The optional group parameter
// controls execution order during dispatch: handlers in group 0 run first,
// then group 1, and so on. When group is omitted the handler is placed in
// group 0.
//
// Example:
//
//	disp := telegram.NewHandlerDispatcher()
//	disp.AddHandler(&loggingHandler{}, 0)
//	disp.AddHandler(&commandHandler{}, 1)
func (d *HandlerDispatcher) AddHandler(h Handler, group ...int) {
	g := 0
	if len(group) > 0 {
		g = group[0]
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers = append(d.handlers, handlerEntry{handler: h, group: g})
	d.dirty = true
}

// RemoveHandler removes the first occurrence of h from the dispatcher. If h
// is not found, RemoveHandler is a no-op.
func (d *HandlerDispatcher) RemoveHandler(h Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, he := range d.handlers {
		if he.handler == h {
			d.handlers = append(d.handlers[:i], d.handlers[i+1:]...)
			d.dirty = true
			return
		}
	}
}

// Dispatch delivers update to every registered handler whose Check method
// returns true, in ascending group order. It creates a fresh Context for each
// handler via Client.NewContext and populates it with fields extracted from the
// update. If any handler sets Context.Stopped, the loop terminates immediately
// and no further handlers are invoked.
//
// When client.cfg.HandlerTimeout is positive, the per-handler context carries a
// deadline; otherwise a plain Background context is used.
//
// Example:
//
//	disp := telegram.NewHandlerDispatcher()
//	disp.AddHandler(&myHandler{}, 0)
//	disp.Dispatch(client, update)
func (d *HandlerDispatcher) Dispatch(client *Client, update *Update) {
	var handlers []handlerEntry

	d.mu.RLock()
	if d.dirty {
		d.mu.RUnlock()
		d.mu.Lock()
		if d.dirty {
			sorted := make([]handlerEntry, len(d.handlers))
			copy(sorted, d.handlers)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].group < sorted[j].group
			})
			d.sorted = sorted
			d.dirty = false
		}
		handlers = d.sorted
		d.mu.Unlock()
	} else {
		handlers = d.sorted
		d.mu.RUnlock()
	}

	if client != nil && client.Log != nil {
		client.Log.Tracef("dispatching update to %d handlers", len(handlers))
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if client != nil && client.cfg.HandlerTimeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), client.cfg.HandlerTimeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	for _, he := range handlers {
		if he.handler.Check(update) {
			c := client.NewContext(ctx)
			c.Update = update
			populateContext(c, update)
			he.handler.Handle(c)
			if c.Stopped {
				return
			}
		}
	}
}

func (d *HandlerDispatcher) DispatchSafe(client *Client, update *Update) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: panic: %v", ErrUpdateHandlerFailed, r)
		}
	}()
	d.Dispatch(client, update)
	return nil
}

func populateContext(ctx *Context, update *Update) {
	ctx.Message = update.Message
	ctx.EditedMessage = update.EditedMessage
	ctx.BusinessMessage = update.BusinessMessage
	ctx.EditedBusinessMessage = update.EditedBusinessMessage
	ctx.DeletedMessages = update.DeletedMessages
	ctx.DeletedBusinessMessages = update.DeletedBusinessMessages
	ctx.CallbackQuery = update.CallbackQuery
	ctx.InlineQuery = update.InlineQuery
	ctx.ChosenInlineResult = update.ChosenInlineResult
	ctx.UserStatus = update.UserStatus
	ctx.ChatMember = update.ChatMember
	ctx.MessageReaction = update.MessageReaction
	ctx.MessageReactionCount = update.MessageReactionCount
	ctx.Poll = update.Poll
	ctx.BusinessConnection = update.BusinessConnection
	ctx.Story = update.Story
	ctx.ChatBoost = update.ChatBoost
	ctx.ChatJoinRequest = update.ChatJoinRequest
	ctx.PreCheckoutQuery = update.PreCheckoutQuery
	ctx.ShippingQuery = update.ShippingQuery
	ctx.PurchasedPaidMedia = update.PurchasedPaidMedia
	ctx.ManagedBot = update.ManagedBot
	ctx.Error = update.Error
	ctx.Connected = update.Connected
	ctx.Disconnected = update.Disconnected
	ctx.Started = update.Started
	ctx.SecretChat = update.SecretChat
	ctx.SecretMessage = update.SecretMessage
}
