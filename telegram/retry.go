package telegram

import (
	"context"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// floodMaxRetries bounds how many times a single call may sleep through a
// FLOOD_WAIT and replay within the configured threshold.
const floodMaxRetries = 5

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
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return ctx.Err()
		}
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
	if registry != nil {
		if err := registry.waitIfFlooded(ctx, method); err != nil {
			return err
		}
	}
	if threshold <= 0 {
		return call()
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
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
		if registry != nil {
			if err := registry.waitIfFlooded(ctx, method); err != nil {
				return err
			}
		}
	}
}

// constructorOf returns the TL constructor ID of a query for registry keys.
func constructorOf(query tg.TLObject) uint32 {
	if query == nil {
		return 0
	}
	return query.ConstructorID()
}
