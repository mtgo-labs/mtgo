package session

import (
	"sync"
	"time"
)

// MsgIDGenerator produces unique, monotonically increasing MTProto message IDs
// based on the server's Unix time. Message IDs are structured as
// (server_time << 32) | (counter << 2) to comply with the MTProto specification.
type MsgIDGenerator struct {
	serverTimeUnix int64
	counter        int64
	mu             sync.Mutex
}

// NewMsgIDGenerator returns a generator seeded with the given server time.
func NewMsgIDGenerator(serverTime time.Time) *MsgIDGenerator {
	return &MsgIDGenerator{
		serverTimeUnix: serverTime.Unix(),
	}
}

// UpdateServerTime advances the internal clock to the given time if it is
// newer than the current reference, resetting the counter. Older times are
// ignored.
func (g *MsgIDGenerator) UpdateServerTime(t time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if t.Unix() > g.serverTimeUnix {
		g.serverTimeUnix = t.Unix()
		g.counter = 0
	}
}

// Next returns the next unique message ID, guaranteed to be monotonically
// increasing. The lower bits encode a counter to ensure uniqueness within the
// same second.
func (g *MsgIDGenerator) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now().Unix()
	if now > g.serverTimeUnix {
		g.serverTimeUnix = now
		g.counter = 0
	}
	base := g.serverTimeUnix << 32
	g.counter++
	return base | (g.counter << 2)
}
