package telegram

import (
	"context"
	"sync"
	"time"
)

// FloodLimit is a single sliding-window rule for FloodControl.
type FloodLimit struct {
	Count    int
	WindowMs int64
}

// FloodControl is a sliding-window event limiter. Ported from TDLib's
// FloodControlStrict (tdutils/td/utils/FloodControlStrict.h). Multiple rules
// compose: the strictest wakeup time across all rules wins.
//
// Unlike exponential backoff (which adds fixed delay per attempt), FloodControl
// tracks the actual distribution of events in time and only delays when a
// window is genuinely saturated. This prevents connection storms that trigger
// server-side transport 429 errors.
type FloodControl struct {
	mu     sync.Mutex
	rules  []FloodLimit
	events []int64 // perf-now ms timestamps, ascending
	maxWin int64   // widest window across all rules
}

// NewFloodControl creates a limiter from the given rules. At least one rule is
// required.
func NewFloodControl(rules []FloodLimit) *FloodControl {
	var maxWin int64
	for _, r := range rules {
		if r.WindowMs > maxWin {
			maxWin = r.WindowMs
		}
	}
	return &FloodControl{rules: rules, maxWin: maxWin}
}

// wakeupAt returns the earliest absolute perf-now ms at which a new event is
// permitted, or 0 if an event is allowed now. now is perf-now ms.
func (f *FloodControl) wakeupAt(now int64) int64 {
	var wakeup int64
	for _, r := range f.rules {
		cutoff := now - r.WindowMs
		inWindow := 0
		oldestIdx := -1
		for i := len(f.events) - 1; i >= 0; i-- {
			if f.events[i] <= cutoff {
				break
			}
			inWindow++
			oldestIdx = i
		}
		if inWindow >= r.Count {
			t := f.events[oldestIdx] + r.WindowMs
			if t > wakeup {
				wakeup = t
			}
		}
	}
	return wakeup
}

// addEvent records an event at the given perf-now ms and prunes old entries.
func (f *FloodControl) addEvent(now int64) {
	f.events = append(f.events, now)
	cutoff := now - f.maxWin
	i := 0
	for i < len(f.events) && f.events[i] < cutoff {
		i++
	}
	if i > 0 {
		f.events = f.events[i:]
	}
}

// wait blocks until an event is permitted under all rules, then records it.
// Returns ctx.Err() if the context is cancelled while waiting.
func (f *FloodControl) wait(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		now := time.Now().UnixMilli()
		f.mu.Lock()
		wakeup := f.wakeupAt(now)
		if wakeup <= now {
			f.addEvent(now)
			f.mu.Unlock()
			return nil
		}
		f.mu.Unlock()

		delay := time.Duration(wakeup-now) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (f *FloodControl) reset() {
	f.mu.Lock()
	f.events = f.events[:0]
	f.mu.Unlock()
}

// TDLib defaults from ConnectionCreator.cpp:97-108.
var (
	defaultFloodLimits = []FloodLimit{
		{Count: 1, WindowMs: 1_000},
		{Count: 4, WindowMs: 2_000},
		{Count: 8, WindowMs: 3_000},
	}
	defaultSanityLimits = []FloodLimit{
		{Count: 5, WindowMs: 10_000},
	}
)

// connectionFloodGate bundles three FloodControl limiters to gate connection
// attempts, modeled after TDLib's ConnectionCreator client state. The three
// limiters compose: the strictest wakeup time wins.
//
//   - sanity: absolute ceiling, always applied
//   - flood: burst limit on connection attempts
//   - mtprotoError: ticks only on MTProto-level errors (transport 429)
type connectionFloodGate struct {
	sanity       *FloodControl
	flood        *FloodControl
	mtprotoError *FloodControl
}

func newConnectionFloodGate() *connectionFloodGate {
	return &connectionFloodGate{
		sanity:       NewFloodControl(defaultSanityLimits),
		flood:        NewFloodControl(defaultFloodLimits),
		mtprotoError: NewFloodControl(defaultFloodLimits),
	}
}

// wait blocks until a connection attempt is permitted under all limiters,
// then records the attempt in sanity + flood. Returns ctx.Err() on cancel.
func (g *connectionFloodGate) wait(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		now := time.Now().UnixMilli()
		g.sanity.mu.Lock()
		g.flood.mu.Lock()

		wakeup := g.sanity.wakeupAt(now)
		if w := g.flood.wakeupAt(now); w > wakeup {
			wakeup = w
		}
		if w := g.mtprotoError.wakeupAt(now); w > wakeup {
			wakeup = w
		}

		if wakeup <= now {
			g.sanity.addEvent(now)
			g.flood.addEvent(now)
			g.flood.mu.Unlock()
			g.sanity.mu.Unlock()
			return nil
		}

		g.flood.mu.Unlock()
		g.sanity.mu.Unlock()

		delay := time.Duration(wakeup-now) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// recordMtprotoError ticks the MTProto error limiter. Called when a transport
// 429 or similar MTProto-level error is received.
func (g *connectionFloodGate) recordMtprotoError() {
	now := time.Now().UnixMilli()
	g.mtprotoError.mu.Lock()
	g.mtprotoError.addEvent(now)
	g.mtprotoError.mu.Unlock()
}

// notifyNetworkUp resets the sanity limiter after a network transition, so
// stale events from the previous network don't throttle fresh attempts.
func (g *connectionFloodGate) notifyNetworkUp() {
	g.sanity.reset()
}
