package telegram

import (
	"context"
	"sync"
	"time"
)

// delayGate implements exponential-decay inter-request pacing, ported from
// mtcute's DownloadDelayGate and TDLib's DelayDispatcher. The first request
// waits initialDelay; each subsequent request's delay decays by decay factor
// until it reaches minDelay. All workers in a single transfer share one gate.
//
// Delay schedule (initial=50ms, decay=0.8, min=3ms):
//
//	50, 40, 32, 25.6, 20.5, 16.4, 13.1, 10.5, 8.4, 6.7, 5.4, 4.3, 3.4, 3, 3, 3 ...
//
// After ~14 requests, stabilizes at minDelay intervals.
type delayGate struct {
	mu      sync.Mutex
	nextAt  time.Time
	delay   time.Duration
	initial time.Duration
	min     time.Duration
	decay   float64
}

func newDownloadDelayGate() *delayGate {
	const (
		initial  = 50 * time.Millisecond
		minDelay = 3 * time.Millisecond
		decay    = 0.8
	)
	return &delayGate{
		delay:   initial,
		initial: initial,
		min:     minDelay,
		decay:   decay,
	}
}

// wait blocks until the next available request slot, enforcing the pacing
// delay between consecutive calls. Safe for concurrent use.
func (g *delayGate) wait() {
	if g == nil {
		return
	}
	g.mu.Lock()
	now := time.Now()
	at := g.nextAt
	if at.Before(now) {
		at = now
	}
	g.nextAt = at.Add(g.delay)
	g.delay = time.Duration(float64(g.delay) * g.decay)
	if g.delay < g.min {
		g.delay = g.min
	}
	g.mu.Unlock()

	if d := time.Until(at); d > 0 {
		time.Sleep(d)
	}
}

func (g *delayGate) reset() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.delay = g.initial
	g.nextAt = time.Time{}
	g.mu.Unlock()
}

// byteLimiter is a byte-based semaphore that caps total in-flight bytes per
// DC, ported from mtcute's ResourceLimiter and TDLib's ResourceManager.
// Download chunks acquire chunkSize bytes before sending; upload parts acquire
// len(data) bytes. Release happens after the RPC completes.
type byteLimiter struct {
	mu        sync.Mutex
	available int64
	max       int64
	notify    chan struct{}
}

func newByteLimiter(maxBytes int64) *byteLimiter {
	return &byteLimiter{
		available: maxBytes,
		max:       maxBytes,
		notify:    make(chan struct{}, 1),
	}
}

// acquire blocks until bytes capacity is available or ctx is cancelled.
func (l *byteLimiter) acquire(ctx context.Context, bytes int64) error {
	if l == nil || bytes <= 0 {
		return nil
	}
	if bytes > l.max {
		bytes = l.max
	}
	for {
		l.mu.Lock()
		if l.available >= bytes {
			l.available -= bytes
			l.mu.Unlock()
			return nil
		}
		l.mu.Unlock()

		select {
		case <-l.notify:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// release returns bytes to the pool and wakes one waiter.
func (l *byteLimiter) release(bytes int64) {
	if l == nil || bytes <= 0 {
		return
	}
	l.mu.Lock()
	l.available += bytes
	if l.available > l.max {
		l.available = l.max
	}
	l.mu.Unlock()
	select {
	case l.notify <- struct{}{}:
	default:
	}
}

// Default resource limits — tuned for 4-connection pools.
const (
	defaultDownloadByteLimit = 2 * 1024 * 1024 // 2 MB in-flight per DC
	defaultUploadByteLimit   = 8 * 1024 * 1024 // 8 MB in-flight
)
