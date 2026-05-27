package session

import (
	"context"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

const (
	defaultSaltRefreshRatio = 0.75
	defaultSaltRefreshMin   = 30 * time.Second
	saltPrefetchRetryBase   = 5 * time.Second
	saltPrefetchRetryMax    = 30 * time.Second
	saltPrefetchMaxRetries  = 3
)

type nowFunc func() time.Time

type saltEntry struct {
	validSince int64
	validUntil int64
	salt       int64
}

func saltEntriesFromFuture(fs []*tg.FutureSalt) []saltEntry {
	entries := make([]saltEntry, 0, len(fs))
	for _, f := range fs {
		entries = append(entries, saltEntry{
			validSince: int64(f.ValidSince),
			validUntil: int64(f.ValidUntil),
			salt:       f.Salt,
		})
	}
	return entries
}

type saltManager struct {
	mu sync.Mutex

	now  nowFunc
	salt int64

	validSince int64
	validUntil int64

	pending []saltEntry

	refreshRatio float64
	refreshMin   time.Duration

	notify *sync.Cond
}

func newSaltManager(now nowFunc) *saltManager {
	if now == nil {
		now = time.Now
	}
	sm := &saltManager{
		now:          now,
		refreshRatio: defaultSaltRefreshRatio,
		refreshMin:   defaultSaltRefreshMin,
	}
	sm.notify = sync.NewCond(&sm.mu)
	return sm
}

func (m *saltManager) SetRefreshRatio(r float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r <= 0 || r > 1 {
		r = defaultSaltRefreshRatio
	}
	m.refreshRatio = r
}

func (m *saltManager) SetRefreshMin(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d < 0 {
		d = 0
	}
	m.refreshMin = d
}

func (m *saltManager) StoreSimple(salt int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storeSimpleLocked(salt)
	m.notify.Broadcast()
}

func (m *saltManager) storeSimpleLocked(salt int64) {
	now := m.now().Unix()
	m.salt = salt
	m.validSince = now
	m.validUntil = now + int64(2*time.Hour/time.Second)
	m.pending = nil
}

func (m *saltManager) StoreFromFutureSalts(salts []saltEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(salts) == 0 {
		return
	}
	now := m.now().Unix()

	best := salts[0]
	for _, s := range salts {
		if s.validSince <= now && s.validUntil > now {
			if s.validUntil > best.validUntil || best.validSince > now {
				best = s
			}
		}
	}

	if best.validSince <= now && best.validUntil > now {
		m.salt = best.salt
		m.validSince = best.validSince
		m.validUntil = best.validUntil
	} else {
		m.salt = salts[0].salt
		m.validSince = salts[0].validSince
		m.validUntil = salts[0].validUntil
	}

	var future []saltEntry
	for _, s := range salts {
		if s.validSince > now && s.validUntil > m.validUntil {
			future = append(future, s)
		}
	}
	if len(future) > 8 {
		future = future[:8]
	}
	m.pending = future
	m.notify.Broadcast()
}

func (m *saltManager) Load() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.promoteIfNeededLocked()
	return m.salt
}

func (m *saltManager) IsExpired() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.validUntil == 0 {
		return false
	}
	if m.now().Unix() < m.validUntil {
		return false
	}
	m.promoteIfNeededLocked()
	if m.validUntil == 0 {
		return false
	}
	return m.now().Unix() >= m.validUntil
}

// WaitForValid blocks until the salt manager has a non-expired salt or ctx is
// done. Returns true if a valid salt is available, false if the context was
// cancelled.
func (m *saltManager) WaitForValid(ctx context.Context) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for {
		if m.validUntil == 0 || m.now().Unix() < m.validUntil {
			return true
		}
		m.promoteIfNeededLocked()
		if m.validUntil == 0 || m.now().Unix() < m.validUntil {
			return true
		}
		if ctx.Err() != nil {
			return false
		}
		// Wait in a goroutine so we can check ctx concurrently.
		waitDone := make(chan struct{})
		go func() {
			m.mu.Lock()
			m.notify.Wait()
			m.mu.Unlock()
			close(waitDone)
		}()
		m.mu.Unlock()
		select {
		case <-waitDone:
		case <-ctx.Done():
			m.notify.Broadcast() // wake the waiter goroutine
			<-waitDone
			return false
		}
		m.mu.Lock()
	}
}

func (m *saltManager) HasValidSalt() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.salt == 0 {
		return false
	}
	if m.validUntil == 0 {
		return true
	}
	m.promoteIfNeededLocked()
	if m.validUntil == 0 {
		return m.salt != 0
	}
	return m.now().Unix() < m.validUntil
}

func (m *saltManager) ValidUntil() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.validUntil == 0 {
		return time.Time{}
	}
	return time.Unix(m.validUntil, 0)
}

func (m *saltManager) NextRefreshIn() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nextRefreshInLocked()
}

func (m *saltManager) nextRefreshInLocked() time.Duration {
	if m.validSince == 0 || m.validUntil == 0 {
		return defaultSaltRefreshMin
	}
	now := m.now().Unix()
	totalLifetime := m.validUntil - m.validSince
	if totalLifetime <= 0 {
		return defaultSaltRefreshMin
	}
	refreshAt := m.validSince + int64(float64(totalLifetime)*m.refreshRatio)
	remaining := refreshAt - now
	if remaining <= 0 {
		return 0
	}
	d := time.Duration(remaining) * time.Second
	if d < m.refreshMin {
		return m.refreshMin
	}
	return d
}

func (m *saltManager) promoteIfNeededLocked() {
	if m.validUntil == 0 || len(m.pending) == 0 {
		return
	}
	now := m.now().Unix()
	if now < m.validUntil {
		return
	}
	var best saltEntry
	bestIdx := -1
	for i, s := range m.pending {
		if s.validSince <= now && s.validUntil > now {
			if bestIdx == -1 || s.validUntil > best.validUntil {
				best = s
				bestIdx = i
			}
		}
	}
	if bestIdx < 0 {
		return
	}
	m.salt = best.salt
	m.validSince = best.validSince
	m.validUntil = best.validUntil
	if len(m.pending) > 1 {
		m.pending = append(m.pending[:bestIdx], m.pending[bestIdx+1:]...)
	} else {
		m.pending = nil
	}
}
