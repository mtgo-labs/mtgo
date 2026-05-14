package telegram

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

type trackingHandler struct {
	id  string
	mu  *sync.Mutex
	log *[]string
}

func (h *trackingHandler) Check(_ *Update) bool { return true }

func (h *trackingHandler) Handle(_ *Context) {
	h.mu.Lock()
	*h.log = append(*h.log, h.id)
	h.mu.Unlock()
}

func middlewareTracker(label string, mu *sync.Mutex, log *[]string) Middleware {
	return func(next Handler) Handler {
		return &trackerMiddleware{label: label, next: next, mu: mu, log: log}
	}
}

type trackerMiddleware struct {
	label string
	next  Handler
	mu    *sync.Mutex
	log   *[]string
}

func (m *trackerMiddleware) Check(u *Update) bool { return m.next.Check(u) }

func (m *trackerMiddleware) Handle(ctx *Context) {
	m.mu.Lock()
	*m.log = append(*m.log, m.label+":before")
	m.mu.Unlock()
	m.next.Handle(ctx)
	m.mu.Lock()
	*m.log = append(*m.log, m.label+":after")
	m.mu.Unlock()
}

func TestMiddlewarePriority(t *testing.T) {
	var mu sync.Mutex
	var log []string

	client, err := NewClient(1, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}

	client.UseMiddleware(middlewareTracker("A", &mu, &log), 0)
	client.UseMiddleware(middlewareTracker("B", &mu, &log), 10)
	client.UseMiddleware(middlewareTracker("C", &mu, &log), -5)

	handler := &trackingHandler{id: "handler", mu: &mu, log: &log}
	wrapped := client.applyMiddleware(handler)
	wrapped.Handle(&Context{Update: &Update{}})

	mu.Lock()
	defer mu.Unlock()

	expected := []string{"C:before", "A:before", "B:before", "handler", "B:after", "A:after", "C:after"}
	if len(log) != len(expected) {
		t.Fatalf("expected %d entries, got %d: %v", len(expected), len(log), log)
	}
	for i, want := range expected {
		if log[i] != want {
			t.Errorf("entry %d: want %q, got %q", i, want, log[i])
		}
	}
}

func TestMiddlewareDefaultPriority(t *testing.T) {
	var mu sync.Mutex
	var log []string

	client, err := NewClient(1, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}

	client.UseMiddleware(middlewareTracker("first", &mu, &log))
	client.UseMiddleware(middlewareTracker("second", &mu, &log))

	handler := &trackingHandler{id: "handler", mu: &mu, log: &log}
	wrapped := client.applyMiddleware(handler)
	wrapped.Handle(&Context{Update: &Update{}})

	mu.Lock()
	defer mu.Unlock()

	expected := []string{"first:before", "second:before", "handler", "second:after", "first:after"}
	if len(log) != len(expected) {
		t.Fatalf("expected %d entries, got %d: %v", len(expected), len(log), log)
	}
	for i, want := range expected {
		if log[i] != want {
			t.Errorf("entry %d: want %q, got %q", i, want, log[i])
		}
	}
}

func TestMiddlewareStopPropagation(t *testing.T) {
	var mu sync.Mutex
	var log []string

	stopMW := func(next Handler) Handler {
		return &stopMiddleware{mu: &mu, log: &log}
	}

	client, err := NewClient(1, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}

	client.UseMiddleware(stopMW, -10)
	client.UseMiddleware(middlewareTracker("outer", &mu, &log), 0)

	handler := &trackingHandler{id: "handler", mu: &mu, log: &log}
	wrapped := client.applyMiddleware(handler)
	wrapped.Handle(&Context{Update: &Update{}})

	mu.Lock()
	defer mu.Unlock()

	if len(log) != 1 || log[0] != "stop:before" {
		t.Fatalf("expected only stop:before, got %v", log)
	}
}

type stopMiddleware struct {
	mu  *sync.Mutex
	log *[]string
}

func (m *stopMiddleware) Check(u *Update) bool { return true }

func (m *stopMiddleware) Handle(ctx *Context) {
	m.mu.Lock()
	*m.log = append(*m.log, "stop:before")
	m.mu.Unlock()
	ctx.Stopped = true
}

func TestMiddlewareNoMiddleware(t *testing.T) {
	client, err := NewClient(1, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}

	called := false
	handler := &simpleHandler{fn: func(_ *Context) { called = true }}
	wrapped := client.applyMiddleware(handler)
	wrapped.Handle(&Context{Update: &Update{}})

	if !called {
		t.Error("handler should have been called when no middleware registered")
	}
}

type simpleHandler struct {
	fn func(*Context)
}

func (h *simpleHandler) Check(_ *Update) bool { return true }
func (h *simpleHandler) Handle(ctx *Context)  { h.fn(ctx) }

func TestChain(t *testing.T) {
	var mu sync.Mutex
	var log []string

	chained := Chain(middlewareTracker("mw1", &mu, &log), middlewareTracker("mw2", &mu, &log))
	handler := &trackingHandler{id: "handler", mu: &mu, log: &log}
	wrapped := chained(handler)
	wrapped.Handle(&Context{Update: &Update{}})

	mu.Lock()
	defer mu.Unlock()

	expected := []string{"mw1:before", "mw2:before", "handler", "mw2:after", "mw1:after"}
	if len(log) != len(expected) {
		t.Fatalf("expected %d entries, got %d: %v", len(expected), len(log), log)
	}
	for i, want := range expected {
		if log[i] != want {
			t.Errorf("entry %d: want %q, got %q", i, want, log[i])
		}
	}
}

func TestChainEmpty(t *testing.T) {
	called := false
	chained := Chain()
	wrapped := chained(&simpleHandler{fn: func(_ *Context) { called = true }})
	wrapped.Handle(&Context{Update: &Update{}})

	if !called {
		t.Error("handler should be called with empty chain")
	}
}

func TestMiddlewareDispatchUpdate(t *testing.T) {
	client, err := NewClient(1, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var log []string
	var mwLog []string

	client.UseMiddleware(func(next Handler) Handler {
		return &dispatchTracker{next: next, mu: &mu, log: &mwLog, label: "mw"}
	}, 0)

	d := NewHandlerDispatcher()
	d.AddHandler(&simpleHandler{fn: func(_ *Context) {
		mu.Lock()
		log = append(log, "dispatched")
		mu.Unlock()
	}})

	client.dispatchUpdate(d, &Update{})

	mu.Lock()
	defer mu.Unlock()

	if len(mwLog) == 0 {
		t.Error("middleware should have been called")
	}
	if len(log) == 0 {
		t.Error("handler should have been dispatched")
	}
}

type dispatchTracker struct {
	next  Handler
	mu    *sync.Mutex
	log   *[]string
	label string
}

func (d *dispatchTracker) Check(u *Update) bool { return d.next.Check(u) }

func (d *dispatchTracker) Handle(ctx *Context) {
	d.mu.Lock()
	*d.log = append(*d.log, d.label+":before")
	d.mu.Unlock()
	d.next.Handle(ctx)
	d.mu.Lock()
	*d.log = append(*d.log, d.label+":after")
	d.mu.Unlock()
}

func TestMiddlewareCacheInvalidation(t *testing.T) {
	client, err := NewClient(1, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}

	var sortCount int
	mw := func(next Handler) Handler { return next }

	client.UseMiddleware(mw, 0)
	_ = client.sortedMiddlewares()
	sortCount++

	_ = client.sortedMiddlewares()
	// Cache hit — sortCount stays the same since we just check the field directly.

	// Adding new middleware invalidates cache.
	client.UseMiddleware(mw, 5)

	// Verify the cache was invalidated by checking the internal state.
	client.mu.Lock()
	cached := client.mwSorted
	client.mu.Unlock()

	if cached {
		t.Error("adding middleware should invalidate cache")
	}
}

func TestInvokerMiddlewareChain(t *testing.T) {
	client, err := NewClient(1, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}

	var order []string
	var mu sync.Mutex

	track := func(label string) InvokerMiddleware {
		return func(next tg.Invoker) tg.Invoker {
			return tg.InvokerFunc(func(ctx context.Context, input tg.TLObject, decode func(io.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
				mu.Lock()
				order = append(order, label)
				mu.Unlock()
				return next.RPCInvoke(ctx, input, decode)
			})
		}
	}

	client.UseInvokerMiddleware(track("first"))
	client.UseInvokerMiddleware(track("second"))
	client.UseInvokerMiddleware(track("third"))

	// Raw() should cache and return the same client.
	rpc1 := client.Raw()
	rpc2 := client.Raw()
	if rpc1 != rpc2 {
		t.Error("Raw() should return cached RPCClient")
	}

	// The middleware chain should wrap: first → second → third → base.
	// Since first was registered first, it's outermost.
	_, _ = rpc1.RPC().RPCInvoke(context.Background(), nil, nil)

	mu.Lock()
	got := order
	mu.Unlock()

	want := []string{"first", "second", "third"}
	if len(got) != len(want) {
		t.Fatalf("expected %d calls, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("call %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestInvokerMiddlewareCacheInvalidation(t *testing.T) {
	client, err := NewClient(1, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}

	mw := func(next tg.Invoker) tg.Invoker { return next }

	client.UseInvokerMiddleware(mw)
	rpc1 := client.Raw()

	// Adding new invoker middleware should invalidate cache.
	client.UseInvokerMiddleware(mw)
	rpc2 := client.Raw()

	if rpc1 == rpc2 {
		t.Error("adding invoker middleware should invalidate cache")
	}
}
