package session

import (
	"io"
	"sync"
	"time"
)

// PoolEntry is a cached warm transport connection.
type PoolEntry struct {
	DCOption  DataCenter
	Conn      io.Closer
	CreatedAt time.Time
}

// ConnectionPool caches recently successful raw connections to avoid redundant
// TCP handshakes on reconnect. Entries are consumed on first use (single-use).
// Ported from td/td/telegram/net/ConnectionCreator.cpp (ready_connections, READY_CONNECTIONS_TIMEOUT).
type ConnectionPool struct {
	entries []PoolEntry
	ttl     time.Duration
	timer   *time.Timer
	timerID uint64
	closed  bool
	mu      sync.Mutex
}

// NewConnectionPool creates a connection pool with the given TTL.
func NewConnectionPool(ttl time.Duration) *ConnectionPool {
	return &ConnectionPool{
		ttl: ttl,
	}
}

// Get returns a cached connection for the given endpoint if it exists and has
// not expired. The entry is consumed (removed) on cache hit. Returns nil, false
// on cache miss or expiry.
func (p *ConnectionPool) Get(dcID int, option DataCenter) (io.Closer, bool) {
	p.mu.Lock()
	now := time.Now()
	for i, entry := range p.entries {
		if entry.DCOption.ID == dcID && entry.DCOption == option {
			p.removeLocked(i)
			if now.Sub(entry.CreatedAt) < p.ttl {
				p.scheduleExpiryLocked(now)
				p.mu.Unlock()
				return entry.Conn, true
			}
			p.scheduleExpiryLocked(now)
			p.mu.Unlock()
			closeConnections([]io.Closer{entry.Conn})
			return nil, false
		}
	}
	p.mu.Unlock()
	return nil, false
}

// Put caches a successful connection for the given endpoint.
func (p *ConnectionPool) Put(dcID int, option DataCenter, conn io.Closer) {
	if conn == nil {
		return
	}

	p.mu.Lock()
	if p.closed || p.ttl <= 0 {
		p.mu.Unlock()
		closeConnections([]io.Closer{conn})
		return
	}

	now := time.Now()
	for i := range p.entries {
		if p.entries[i].DCOption.ID == dcID && p.entries[i].DCOption == option {
			old := p.entries[i].Conn
			p.entries[i] = PoolEntry{DCOption: option, Conn: conn, CreatedAt: now}
			p.scheduleExpiryLocked(now)
			p.mu.Unlock()
			closeConnections([]io.Closer{old})
			return
		}
	}

	p.entries = append(p.entries, PoolEntry{
		DCOption:  option,
		Conn:      conn,
		CreatedAt: now,
	})
	p.scheduleExpiryLocked(now)
	p.mu.Unlock()
}

// Evict removes a specific cached entry (e.g., on connection failure).
func (p *ConnectionPool) Evict(dcID int, option DataCenter) {
	p.mu.Lock()
	for i, entry := range p.entries {
		if entry.DCOption.ID == dcID && entry.DCOption == option {
			p.removeLocked(i)
			p.scheduleExpiryLocked(time.Now())
			p.mu.Unlock()
			closeConnections([]io.Closer{entry.Conn})
			return
		}
	}
	p.mu.Unlock()
}

// Purge removes and closes all expired entries immediately. Entries also expire
// automatically through the pool's single shared timer.
func (p *ConnectionPool) Purge() int {
	p.mu.Lock()
	now := time.Now()
	kept := p.entries[:0]
	closed := make([]io.Closer, 0, len(p.entries))
	for _, entry := range p.entries {
		if now.Sub(entry.CreatedAt) >= p.ttl {
			closed = append(closed, entry.Conn)
			continue
		}
		kept = append(kept, entry)
	}
	clear(p.entries[len(kept):])
	p.entries = kept
	p.scheduleExpiryLocked(now)
	p.mu.Unlock()
	closeConnections(closed)
	return len(closed)
}

// Count returns the number of cached entries.
func (p *ConnectionPool) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.entries)
}

// Clear closes and removes every cached connection while keeping the pool reusable.
func (p *ConnectionPool) Clear() {
	p.mu.Lock()
	entries := p.takeAllLocked()
	p.mu.Unlock()
	closePoolEntries(entries)
}

// Close closes every cached connection and permanently disables the pool.
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	p.closed = true
	entries := p.takeAllLocked()
	p.mu.Unlock()
	closePoolEntries(entries)
}

func (p *ConnectionPool) expire(timerID uint64) {
	p.mu.Lock()
	if timerID != p.timerID {
		p.mu.Unlock()
		return
	}
	if p.closed {
		p.timer = nil
		p.mu.Unlock()
		return
	}
	p.timer = nil
	now := time.Now()
	kept := p.entries[:0]
	closed := make([]io.Closer, 0, len(p.entries))
	for _, entry := range p.entries {
		if now.Sub(entry.CreatedAt) >= p.ttl {
			closed = append(closed, entry.Conn)
			continue
		}
		kept = append(kept, entry)
	}
	clear(p.entries[len(kept):])
	p.entries = kept
	p.scheduleExpiryLocked(now)
	p.mu.Unlock()
	closeConnections(closed)
}

func (p *ConnectionPool) scheduleExpiryLocked(now time.Time) {
	if p.closed || p.ttl <= 0 || len(p.entries) == 0 {
		if p.timer != nil {
			p.timerID++
			p.timer.Stop()
			p.timer = nil
		}
		return
	}
	if p.timer != nil {
		return
	}

	expiresAt := p.entries[0].CreatedAt.Add(p.ttl)
	for _, entry := range p.entries[1:] {
		if candidate := entry.CreatedAt.Add(p.ttl); candidate.Before(expiresAt) {
			expiresAt = candidate
		}
	}
	delay := expiresAt.Sub(now)
	if delay < 0 {
		delay = 0
	}
	p.timerID++
	timerID := p.timerID
	p.timer = time.AfterFunc(delay, func() {
		p.expire(timerID)
	})
}

func (p *ConnectionPool) removeLocked(i int) {
	copy(p.entries[i:], p.entries[i+1:])
	p.entries[len(p.entries)-1] = PoolEntry{}
	p.entries = p.entries[:len(p.entries)-1]
}

func (p *ConnectionPool) takeAllLocked() []PoolEntry {
	p.timerID++
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
	entries := p.entries
	p.entries = nil
	return entries
}

func closePoolEntries(entries []PoolEntry) {
	connections := make([]io.Closer, 0, len(entries))
	for _, entry := range entries {
		connections = append(connections, entry.Conn)
	}
	closeConnections(connections)
}

func closeConnections(connections []io.Closer) {
	for _, conn := range connections {
		if conn != nil {
			_ = conn.Close()
		}
	}
}
