package telegram

import (
	"context"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// floodMaxRetries bounds total attempts (the initial call plus replays) a
// single invocation may make under the flood policy.
const floodMaxRetries = 5

// transferFloodThreshold is the flood wait transfers accept: any duration
// the server mandates.
const transferFloodThreshold = 24 * time.Hour

// floodRegistry remembers server-imposed per-method flood deadlines so new
// calls to a still-waiting method sleep locally BEFORE wasting a round trip.
type floodRegistry struct {
	mu    sync.Mutex
	until map[uint32]time.Time
}

func newFloodRegistry() *floodRegistry {
	return &floodRegistry{until: make(map[uint32]time.Time)}
}

// record notes that method (by TL constructor ID) is flooded until t.
func (f *floodRegistry) record(method uint32, t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.until[method]; !ok || t.After(existing) {
		f.until[method] = t
	}
}

// waitIfFlooded sleeps until the method's flood deadline passes. Returns
// ctx.Err() if the caller gives up first.
func (f *floodRegistry) waitIfFlooded(ctx context.Context, method uint32) error {
	for {
		f.mu.Lock()
		until, ok := f.until[method]
		if ok && !time.Now().Before(until) {
			// Deadline already passed: prune and proceed.
			delete(f.until, method)
			ok = false
		}
		f.mu.Unlock()
		if !ok {
			return nil
		}
		d := time.Until(until)
		if d < 0 {
			d = 0
		}
		if err := sleepCtx(ctx, d); err != nil {
			return err
		}
	}
}

// sleepCtx is the policy sleeper; tests may substitute an instant recorder.
var sleepCtx = sleepContext

// sleepContext sleeps for d, stopping early and returning ctx.Err() when the
// context is cancelled. Uses a stoppable timer so long gates do not pin
// timer entries until expiry.
func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// invokeWithFloodPolicy runs call under the unified flood-wait policy:
//
//   - before each attempt, sleep through any remembered per-method deadline
//   - on FLOOD_WAIT (or FLOOD_PREMIUM_WAIT) with a wait at or below threshold,
//     record the deadline, sleep (ctx-cancellable), and replay
//   - waits above the threshold surface to the caller untouched
//   - at most floodMaxRetries total attempts (initial call + replays)
//
// A non-positive threshold disables auto-sleeping entirely.
func invokeWithFloodPolicy(ctx context.Context, threshold time.Duration, method uint32, registry *floodRegistry, call func() error) error {
	// A non-positive threshold disables auto-sleeping entirely, including
	// the local early gate.
	if threshold <= 0 {
		return call()
	}
	if registry != nil {
		if err := registry.waitIfFlooded(ctx, method); err != nil {
			return err
		}
	}
	for attempt := 0; ; attempt++ {
		err := call()
		if err == nil {
			return nil
		}
		wait, ok := tgerr.AsFloodWait(err)
		if !ok {
			return err
		}
		if attempt+1 >= floodMaxRetries {
			return err
		}
		if wait > threshold {
			return err
		}
		if registry != nil {
			registry.record(method, time.Now().Add(wait))
		}
		if err := sleepCtx(ctx, wait); err != nil {
			return err
		}
		if registry != nil {
			if err := registry.waitIfFlooded(ctx, method); err != nil {
				return err
			}
		}
	}
}

// constructorOf returns the TL constructor ID of a query for registry keys.
// Init-connection wrappers are unwrapped so first-call floods are recorded
// under the real method the caller will use next time.
func constructorOf(query tg.TLObject) uint32 {
	if query == nil {
		return 0
	}
	for {
		if wrapped, ok := query.(*tg.InvokeWithLayerRequest); ok && wrapped.Query != nil {
			query = wrapped.Query
			continue
		}
		return query.ConstructorID()
	}
}

// invokeFlood is the Client-bound form of invokeWithFloodPolicy, deriving
// the threshold, method key, and registry from the client and call context.
func (c *Client) invokeFlood(ctx context.Context, query tg.TLObject, call func() error) error {
	return invokeWithFloodPolicy(ctx, floodThresholdFor(ctx, c.config()), constructorOf(query), c.floodReg(), call)
}
