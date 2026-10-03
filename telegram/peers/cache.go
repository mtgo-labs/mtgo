package peers

import (
	"github.com/mtgo-labs/mtgo/internal/peerid"
	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
)

// Cached looks up a previously cached input peer by numeric ID without
// performing RPC calls. When a persistent store is configured, it is
// consulted on a memory miss and hits are promoted to the in-memory cache.
// Returns [ErrNotFound] when the ID is unknown.
func (m *Manager) Cached(id int64) (tg.InputPeerClass, error) {
	m.mu.RLock()
	for _, lookupID := range peerLookupIDs(id) {
		if p, ok := m.byID[lookupID]; ok {
			m.mu.RUnlock()
			return p, nil
		}
	}
	m.mu.RUnlock()

	if m.savePeers() {
		if ps := m.store(); ps != nil {
			for _, lookupID := range peerLookupIDs(id) {
				p, err := ps.GetPeer(lookupID)
				if err != nil || p == nil {
					continue
				}
				peer, convErr := peerFromStorageEntry(p)
				if convErr != nil {
					return nil, convErr
				}
				if ch, ok := peer.(*tg.InputPeerChannel); ok {
					if raw, ok2 := peerid.UnmarkChannel(ch.ChannelID); ok2 {
						ch.ChannelID = raw
					}
				}
				m.Cache(lookupID, peer)
				if p.Username != "" {
					m.CacheUsername(p.Username, p.ID)
				}
				if p.PhoneNumber != "" {
					m.CachePhone(p.PhoneNumber, p.ID)
				}
				return peer, nil
			}
		}
	}
	return nil, ErrNotFound
}

// Cache stores an input peer under the given numeric ID. Canonical IDs are
// derived from the peer itself; a zero access hash never overwrites a cached
// non-zero one; the FIFO eviction limit is enforced; and the peer is
// persisted when configured.
func (m *Manager) Cache(id int64, peer tg.InputPeerClass) {
	id = canonicalPeerID(id, peer)
	m.mu.Lock()

	// Don't let a zero-hash min entity overwrite a known full hash.
	if existing, ok := m.byID[id]; ok {
		peer = preserveAccessHash(existing, peer)
	}

	if _, exists := m.byID[id]; !exists {
		m.idOrder = append(m.idOrder, id)
	}
	m.byID[id] = peer
	m.evictOldestLocked()

	save := m.savePeers()
	m.mu.Unlock()

	if save {
		if ps := m.store(); ps != nil {
			entry := &storage.Peer{ID: id}
			switch p := peer.(type) {
			case *tg.InputPeerUser:
				entry.Type = storage.PeerTypeUser
				entry.AccessHash = p.AccessHash
			case *tg.InputPeerChat:
				entry.Type = storage.PeerTypeChat
			case *tg.InputPeerChannel:
				entry.Type = storage.PeerTypeChannel
				entry.ID = p.ChannelID
				entry.AccessHash = p.AccessHash
			default:
				return
			}
			_ = ps.SavePeer(entry)
		}
	}
}

// CacheUsername records a username→ID mapping for cache-only lookups.
// The reverse index is maintained so username lookup by ID stays O(1).
func (m *Manager) CacheUsername(username string, id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheUsernameLocked(username, id)
	limit := m.cacheSize()
	if limit <= 0 || len(m.usernameToID) <= limit {
		return
	}
	for len(m.usernameToID) > limit && len(m.usernameOrder) > 0 {
		oldest := m.usernameOrder[0]
		m.deleteUsernameLocked(oldest)
		copy(m.usernameOrder, m.usernameOrder[1:])
		m.usernameOrder[len(m.usernameOrder)-1] = ""
		m.usernameOrder = m.usernameOrder[:len(m.usernameOrder)-1]
	}
}

func (m *Manager) cacheUsernameLocked(username string, id int64) {
	if _, exists := m.usernameToID[username]; !exists {
		m.usernameOrder = append(m.usernameOrder, username)
	}
	// Drop any username previously bound to this ID so the reverse index
	// never goes stale (usernames can change).
	if prev, ok := m.idToUsername[id]; ok && prev != username {
		delete(m.usernameToID, prev)
	}
	m.usernameToID[username] = id
	m.idToUsername[id] = username
}

func (m *Manager) deleteUsernameLocked(username string) {
	if id, ok := m.usernameToID[username]; ok {
		delete(m.idToUsername, id)
	}
	delete(m.usernameToID, username)
}

// CachePhone records a normalized phone→ID mapping for cache-only lookups.
// A reverse index is maintained so invalidation by ID stays O(1).
func (m *Manager) CachePhone(phone string, id int64) {
	phone = NormalizePhone(phone)
	if phone == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.phoneToID[phone]; !exists {
		m.phoneOrder = append(m.phoneOrder, phone)
	}
	// Drop any phone previously bound to this ID so the reverse index never
	// goes stale.
	if prev, ok := m.idToPhone[id]; ok && prev != phone {
		delete(m.phoneToID, prev)
	}
	m.phoneToID[phone] = id
	m.idToPhone[id] = phone
	limit := m.cacheSize()
	if limit <= 0 || len(m.phoneToID) <= limit {
		return
	}
	for len(m.phoneToID) > limit && len(m.phoneOrder) > 0 {
		oldest := m.phoneOrder[0]
		delete(m.phoneToID, oldest)
		copy(m.phoneOrder, m.phoneOrder[1:])
		m.phoneOrder[len(m.phoneOrder)-1] = ""
		m.phoneOrder = m.phoneOrder[:len(m.phoneOrder)-1]
	}
}

// LookupUsername returns the cached username for a peer ID, or "".
func (m *Manager) LookupUsername(peerID int64) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.idToUsername[peerID]
}

func (m *Manager) cachedByUsername(username string) (tg.InputPeerClass, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if cachedID, ok := m.usernameToID[username]; ok {
		if p, ok2 := m.byID[cachedID]; ok2 {
			return p, true
		}
	}
	return nil, false
}

// cachedByPhone returns the cached peer for a normalized phone number.
func (m *Manager) cachedByPhone(phone string) (tg.InputPeerClass, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if cachedID, ok := m.phoneToID[phone]; ok {
		if p, ok2 := m.byID[cachedID]; ok2 {
			return p, true
		}
	}
	return nil, false
}

func (m *Manager) evictOldestLocked() {
	limit := m.cacheSize()
	if limit <= 0 || len(m.byID) <= limit {
		return
	}
	for len(m.byID) > limit && len(m.idOrder) > 0 {
		oldest := m.idOrder[0]
		delete(m.byID, oldest)
		if cachedUsername, ok := m.idToUsername[oldest]; ok {
			m.deleteUsernameLocked(cachedUsername)
		}
		copy(m.idOrder, m.idOrder[1:])
		m.idOrder[len(m.idOrder)-1] = 0
		m.idOrder = m.idOrder[:len(m.idOrder)-1]
	}
}

// preserveAccessHash copies a non-zero access hash from the existing cached
// peer when the incoming peer has a zero hash. This prevents min entities
// (which carry no usable access hash) from poisoning a previously-good cache
// entry. The storage backend already merges correctly via storage.MergePeer;
// this brings the in-memory cache to the same guarantee.
func preserveAccessHash(existing, incoming tg.InputPeerClass) tg.InputPeerClass {
	switch e := existing.(type) {
	case *tg.InputPeerChannel:
		if c, ok := incoming.(*tg.InputPeerChannel); ok && c.AccessHash == 0 && e.AccessHash != 0 {
			return &tg.InputPeerChannel{ChannelID: c.ChannelID, AccessHash: e.AccessHash}
		}
	case *tg.InputPeerUser:
		if u, ok := incoming.(*tg.InputPeerUser); ok && u.AccessHash == 0 && e.AccessHash != 0 {
			return &tg.InputPeerUser{UserID: u.UserID, AccessHash: e.AccessHash}
		}
	}
	return incoming
}

func peerLookupIDs(id int64) []int64 {
	if raw, ok := peerid.UnmarkChannel(id); ok {
		return []int64{raw, id}
	}
	return []int64{id}
}

func canonicalPeerID(id int64, peer tg.InputPeerClass) int64 {
	if p, ok := peer.(*tg.InputPeerChannel); ok && p.ChannelID != 0 {
		return p.ChannelID
	}
	if raw, ok := peerid.UnmarkChannel(id); ok {
		return raw
	}
	return id
}
