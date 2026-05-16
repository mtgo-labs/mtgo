package telegram

import (
	"context"
	"fmt"
	"io"
	"maps"
	"sort"

	"github.com/mtgo-labs/mtgo/tg"
)

// Plugin defines the lifecycle interface for client plugins. Implementations receive
// the Client on Start and must release resources on Stop.
//
// Example:
//
//	type LoggingPlugin struct{}
//
//	func (p *LoggingPlugin) Name() string { return "logging" }
//	func (p *LoggingPlugin) Start(ctx context.Context, client *telegram.Client) error {
//		client.Log.Info("logging plugin started")
//		return nil
//	}
//	func (p *LoggingPlugin) Stop(ctx context.Context) error {
//		return nil
//	}
//
//	client.Use(&LoggingPlugin{})
type Plugin interface {
	Name() string
	Start(ctx context.Context, client *Client) error
	Stop(ctx context.Context) error
}

// Use registers a plugin with the client. Plugins are started when the client connects.
//
// Example:
//
//	client.Use(&MyPlugin{})
//	client.Use(&AnotherPlugin{})
//	// Plugins start automatically when client.Connect() is called.
func (c *Client) Use(plugin Plugin) {
	c.mu.Lock()
	if c.plugins == nil {
		c.plugins = make(map[string]Plugin)
	}
	c.plugins[plugin.Name()] = plugin
	c.mu.Unlock()
	if c.Log != nil {
		c.Log.Infof("plugin loaded: %s", plugin.Name())
	}
}

func (c *Client) startPlugins(ctx context.Context) error {
	c.mu.RLock()
	plugins := make(map[string]Plugin, len(c.plugins))
	maps.Copy(plugins, c.plugins)
	c.mu.RUnlock()
	for name, p := range plugins {
		if err := p.Start(ctx, c); err != nil {
			return fmt.Errorf("plugin %s: %w", name, err)
		}
		if c.Log != nil {
			c.Log.Infof("plugin started: %s", name)
		}
	}
	return nil
}

func (c *Client) stopPlugins(ctx context.Context) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for name, p := range c.plugins {
		if err := p.Stop(ctx); err != nil && c.Log != nil {
			c.Log.Errorf("plugin %s stop: %v", name, err)
		}
	}
}

// Middleware wraps a Handler, returning a new Handler that can intercept,
// modify, or short-circuit the update processing. Middleware is composed
// in priority order: lower priority values run first (outermost).
//
// To stop propagation, set ctx.Stopped = true and return from Handle.
type Middleware func(Handler) Handler

// Chain composes multiple middleware into a single Middleware.
// The resulting middleware applies mws in order: mws[0] wraps mws[1]
// which wraps mws[2], etc. This means mws[0] runs first on the way in
// and last on the way out.
func Chain(mws ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

// InvokerMiddleware wraps a tg.Invoker, returning a new one that can
// intercept, modify, or retry RPC calls.
type InvokerMiddleware func(next tg.Invoker) tg.Invoker

// UseInvokerMiddleware registers an invoker-level middleware that
// intercepts all RPC calls. Middleware is applied in registration
// order: the first registered wraps all subsequent ones.
func (c *Client) UseInvokerMiddleware(mw InvokerMiddleware) {
	c.mu.Lock()
	c.invokerMiddlewares = append(c.invokerMiddlewares, mw)
	c.invokerCache = nil
	c.mu.Unlock()
}

// InvokeWithMiddleware wraps base with all registered invoker middleware
// and performs the RPC call.
func (c *Client) InvokeWithMiddleware(base tg.Invoker, ctx context.Context, input tg.TLObject, decode func(io.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	c.mu.RLock()
	mws := c.invokerMiddlewares
	c.mu.RUnlock()
	invoker := base
	for i := len(mws) - 1; i >= 0; i-- {
		invoker = mws[i](invoker)
	}
	return invoker.RPCInvoke(ctx, input, decode)
}

// middlewareEntry pairs a Middleware with an integer priority.
// Lower values run first (are outermost in the chain).
type middlewareEntry struct {
	mw       Middleware
	priority int
}

// UseMiddleware registers mw. The optional priority parameter controls
// execution order: lower values wrap the handler first and therefore see
// the request before higher values. When omitted, the middleware is placed
// at priority 0.
//
// Middleware is applied around the entire handler dispatch. The chain
// is: priority -10 → priority 0 → priority 5 → actual handler.
//
// Example:
//
//	// Logging runs before auth check (lower priority = outermost)
//	client.UseMiddleware(loggingMW, -10)
//	client.UseMiddleware(authMW, 0)
func (c *Client) UseMiddleware(mw Middleware, priority ...int) {
	p := 0
	if len(priority) > 0 {
		p = priority[0]
	}
	c.mu.Lock()
	c.middlewares = append(c.middlewares, middlewareEntry{mw: mw, priority: p})
	c.mwSorted = false
	c.mu.Unlock()
}

// sortedMiddlewares returns the middleware chain sorted by ascending
// priority. The result is cached and only recomputed when new middleware
// is added.
func (c *Client) sortedMiddlewares() []Middleware {
	c.mu.Lock()
	if !c.mwSorted {
		sorted := make([]middlewareEntry, len(c.middlewares))
		copy(sorted, c.middlewares)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].priority < sorted[j].priority
		})
		c.mwCache = make([]Middleware, len(sorted))
		for i, e := range sorted {
			c.mwCache[i] = e.mw
		}
		c.mwSorted = true
	}
	cache := c.mwCache
	c.mu.Unlock()
	return cache
}

// applyMiddleware wraps h with all registered middleware. The sorted list
// is in ascending priority order (lowest first). We iterate in reverse so
// the lowest-priority middleware wraps last and becomes outermost:
//
//	sorted = [C(-5), A(0), B(10)]
//	wrap order: B, A, C → execution: C → A → B → handler
func (c *Client) applyMiddleware(h Handler) Handler {
	mws := c.sortedMiddlewares()
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func (c *Client) dispatchUpdate(d *HandlerDispatcher, update *Update) {
	mws := c.sortedMiddlewares()

	if len(mws) == 0 {
		d.DispatchSafe(c, update)
		return
	}

	inner := &dispatchAllHandler{d: d, c: c, update: update}
	wrapped := c.applyMiddleware(Handler(inner))

	cctx := c.NewContext(context.TODO())
	cctx.Update = update
	populateContext(cctx, update)
	wrapped.Handle(cctx)
}

type dispatchAllHandler struct {
	d      *HandlerDispatcher
	c      *Client
	update *Update
}

func (h *dispatchAllHandler) Check(update *Update) bool {
	return true
}

func (h *dispatchAllHandler) Handle(ctx *Context) {
	h.d.DispatchSafe(h.c, h.update)
}
