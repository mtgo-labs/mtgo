package session

import (
	"sync/atomic"
	"time"
)

// MsgIDGenerator produces unique, monotonically increasing MTProto message IDs
// based on the server's Unix time. Message IDs are structured as
// (server_time << 32) | (counter << 2) to comply with the MTProto specification.
type MsgIDGenerator struct {
	serverTimeUnix atomic.Int64
	counter        atomic.Int64
}

// NewMsgIDGenerator returns a generator seeded with the given server time.
func NewMsgIDGenerator(serverTime time.Time) *MsgIDGenerator {
	g := &MsgIDGenerator{}
	g.serverTimeUnix.Store(serverTime.Unix())
	return g
}

// UpdateServerTime advances the internal clock to the given time if it is
// newer than the current reference, resetting the counter. Older times are
// ignored.
func (g *MsgIDGenerator) UpdateServerTime(t time.Time) {
	newUnix := t.Unix()
	for {
		cur := g.serverTimeUnix.Load()
		if newUnix <= cur {
			return
		}
		if g.serverTimeUnix.CompareAndSwap(cur, newUnix) {
			g.counter.Store(0)
			return
		}
	}
}

// Next returns the next unique message ID, guaranteed to be monotonically
// increasing. The lower bits encode a counter to ensure uniqueness within the
// same second.
func (g *MsgIDGenerator) Next() int64 {
	for {
		cur := g.serverTimeUnix.Load()
		c := g.counter.Add(1)
		perSec := int64(1 << 30) // ~1 billion messages/sec before needing a new timestamp

		// Fast path: counter still has room in the current second.
		// No syscall needed — the common case.
		if c <= perSec {
			return (cur << 32) | ((c - 1) << 2)
		}

		// Counter saturated — advance to next second.
		now := time.Now().Unix()
		if now > cur {
			if g.serverTimeUnix.CompareAndSwap(cur, now) {
				g.counter.Store(1)
				return (now << 32)
			}
		}
		// CAS failed (another goroutine advanced) or time didn't advance.
		// Retry the outer loop.
	}
}
