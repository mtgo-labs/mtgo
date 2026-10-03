package telegram

import (
	"context"
	"sync"
)

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
