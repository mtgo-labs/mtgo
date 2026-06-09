package telegram

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/internal/storage"
)

type dedupEntry struct {
	key string
	ts  time.Time
}

// MemoryStorage implements the storage interface with in-memory maps, suitable for testing
// and ephemeral sessions. It stores peers, DC auth entries, update states, and deduplication
// data in Go maps. All data is lost when the process exits.
//
// Example:
//
//	store := telegram.NewMemoryStorage()
//	client, err := telegram.NewClient(telegram.Config{
//	    Storage: store,
//	    APIID:   12345,
//	    APIHash: "your_api_hash",
//	})
type MemoryStorage struct {
	mu        sync.RWMutex
	dcID      int
	apiID     int32
	testMode  bool
	authKey   []byte
	sessionID string
	userID    int64
	isBot     bool
	date      int
	state     []byte
	firstName string
	lastName  string
	username  string
	apiHash   string

	peers          map[int64]storage.Peer
	peerUsernames  map[string]int64
	dcAuths        map[int]storage.DCAuthEntry
	updateStates   map[string]storage.UpdateState
	channelStates  map[string]map[int64]storage.ChannelUpdateState
	updateDedup    map[string]map[string]time.Time
	dedupOrder     map[string][]string
	durableUpdates map[string]map[string]storage.DurableUpdate
}

// NewMemoryStorage creates and returns a new initialized MemoryStorage with all internal
// maps pre-allocated. Pass the result to a Client config to use an in-memory session store.
//
// Example:
//
//	store := telegram.NewMemoryStorage()
//	_ = store.SetSessionID("my-session")
//	_ = store.SetAPIID(12345)
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		peers:          make(map[int64]storage.Peer),
		peerUsernames:  make(map[string]int64),
		dcAuths:        make(map[int]storage.DCAuthEntry),
		updateStates:   make(map[string]storage.UpdateState),
		channelStates:  make(map[string]map[int64]storage.ChannelUpdateState),
		updateDedup:    make(map[string]map[string]time.Time),
		dedupOrder:     make(map[string][]string),
		durableUpdates: make(map[string]map[string]storage.DurableUpdate),
	}
}

func (m *MemoryStorage) SessionID() (string, error)  { return m.sessionID, nil }
func (m *MemoryStorage) SetSessionID(v string) error { m.sessionID = v; return nil }
func (m *MemoryStorage) DCID() (int, error)          { return m.dcID, nil }
func (m *MemoryStorage) SetDCID(v int) error         { m.dcID = v; return nil }
func (m *MemoryStorage) APIID() (int32, error)       { return m.apiID, nil }
func (m *MemoryStorage) SetAPIID(v int32) error      { m.apiID = v; return nil }
func (m *MemoryStorage) TestMode() (bool, error)     { return m.testMode, nil }
func (m *MemoryStorage) SetTestMode(v bool) error    { m.testMode = v; return nil }
func (m *MemoryStorage) AuthKey() ([]byte, error)    { return append([]byte(nil), m.authKey...), nil }
func (m *MemoryStorage) SetAuthKey(v []byte) error   { m.authKey = v; return nil }
func (m *MemoryStorage) UserID() (int64, error)      { return m.userID, nil }
func (m *MemoryStorage) SetUserID(v int64) error     { m.userID = v; return nil }
func (m *MemoryStorage) IsBot() (bool, error)        { return m.isBot, nil }
func (m *MemoryStorage) SetIsBot(v bool) error       { m.isBot = v; return nil }
func (m *MemoryStorage) FirstName() (string, error)  { return m.firstName, nil }
func (m *MemoryStorage) SetFirstName(v string) error { m.firstName = v; return nil }
func (m *MemoryStorage) LastName() (string, error)   { return m.lastName, nil }
func (m *MemoryStorage) SetLastName(v string) error  { m.lastName = v; return nil }
func (m *MemoryStorage) Username() (string, error)   { return m.username, nil }
func (m *MemoryStorage) SetUsername(v string) error  { m.username = v; return nil }
func (m *MemoryStorage) APIHash() (string, error)    { return m.apiHash, nil }
func (m *MemoryStorage) SetAPIHash(v string) error   { m.apiHash = v; return nil }
func (m *MemoryStorage) Date() (int, error)          { return m.date, nil }
func (m *MemoryStorage) SetDate(v int) error         { m.date = v; return nil }
func (m *MemoryStorage) State() ([]byte, error)      { return append([]byte(nil), m.state...), nil }
func (m *MemoryStorage) SetState(v []byte) error     { m.state = v; return nil }

func (m *MemoryStorage) ExportSessionString() (string, error) {
	if len(m.authKey) == 0 {
		return "", nil
	}
	if m.apiID == 0 {
		return "", fmt.Errorf("telegram: cannot export session: api_id not stored")
	}

	buf := make([]byte, 0, 271)
	buf = append(buf, byte(m.dcID))
	apiID := make([]byte, 4)
	binary.BigEndian.PutUint32(apiID, uint32(m.apiID))
	buf = append(buf, apiID...)
	testMode := byte(0)
	if m.testMode {
		testMode = 1
	}
	buf = append(buf, testMode)
	buf = append(buf, m.authKey...)
	userID := make([]byte, 8)
	binary.BigEndian.PutUint64(userID, uint64(m.userID))
	buf = append(buf, userID...)
	isBot := byte(0)
	if m.isBot {
		isBot = 1
	}
	buf = append(buf, isBot)

	encoded := base64.URLEncoding.EncodeToString(buf)
	encoded = strings.TrimRight(encoded, "=")
	return encoded, nil
}

func (m *MemoryStorage) Close() error { return nil }

func (m *MemoryStorage) SavePeer(p *storage.Peer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.peers == nil {
		m.peers = make(map[int64]storage.Peer)
	}
	if m.peerUsernames == nil {
		m.peerUsernames = make(map[string]int64)
	}
	if old, ok := m.peers[p.ID]; ok && old.Username != "" {
		delete(m.peerUsernames, strings.ToLower(old.Username))
	}
	if p.Username != "" {
		m.peerUsernames[strings.ToLower(p.Username)] = p.ID
	}
	m.peers[p.ID] = *p
	return nil
}

func (m *MemoryStorage) GetPeer(id int64) (*storage.Peer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.peers[id]
	if !ok {
		return nil, nil
	}
	return &p, nil
}

func (m *MemoryStorage) GetPeerByUsername(username string) (*storage.Peer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.peerUsernames[strings.ToLower(username)]
	if !ok {
		return nil, nil
	}
	p, ok := m.peers[id]
	if !ok {
		return nil, nil
	}
	return &p, nil
}

func (m *MemoryStorage) LoadPeers() ([]*storage.Peer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.peers == nil {
		return nil, nil
	}
	result := make([]*storage.Peer, 0, len(m.peers))
	for _, p := range m.peers {
		cp := p
		result = append(result, &cp)
	}
	return result, nil
}

func (m *MemoryStorage) DeletePeer(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.peers[id]; ok && old.Username != "" {
		delete(m.peerUsernames, strings.ToLower(old.Username))
	}
	delete(m.peers, id)
	return nil
}

func (m *MemoryStorage) SaveDCAuth(entry storage.DCAuthEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.dcAuths == nil {
		m.dcAuths = make(map[int]storage.DCAuthEntry)
	}
	m.dcAuths[entry.DCID] = entry
	return nil
}

func (m *MemoryStorage) LoadDCAuth(dcID int) (storage.DCAuthEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.dcAuths == nil {
		return storage.DCAuthEntry{}, fmt.Errorf("dc auth not found: %d", dcID)
	}
	entry, ok := m.dcAuths[dcID]
	if !ok {
		return storage.DCAuthEntry{}, fmt.Errorf("dc auth not found: %d", dcID)
	}
	return entry, nil
}

func (m *MemoryStorage) LoadUpdateState(sessionID string) (*storage.UpdateState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.updateStates[sessionID]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (m *MemoryStorage) SaveUpdateState(state *storage.UpdateState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateStates[state.SessionID] = *state
	return nil
}

func (m *MemoryStorage) LoadChannelUpdateState(sessionID string, channelID int64) (*storage.ChannelUpdateState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channels := m.channelStates[sessionID]
	if channels == nil {
		return nil, nil
	}
	state, ok := channels[channelID]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (m *MemoryStorage) LoadAllChannelUpdateStates(sessionID string) ([]*storage.ChannelUpdateState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channels := m.channelStates[sessionID]
	if channels == nil {
		return nil, nil
	}
	var out []*storage.ChannelUpdateState
	for _, s := range channels {
		cp := s
		out = append(out, &cp)
	}
	return out, nil
}

func (m *MemoryStorage) SaveChannelUpdateState(state *storage.ChannelUpdateState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.channelStates[state.SessionID] == nil {
		m.channelStates[state.SessionID] = make(map[int64]storage.ChannelUpdateState)
	}
	m.channelStates[state.SessionID][state.ChannelID] = *state
	return nil
}

// maxDedupKeysPerSession caps the number of dedup keys stored per session
// to prevent unbounded growth in long-running processes.
const maxDedupKeysPerSession = 10000

// dedupExpiry is the maximum age of a dedup key before it is eligible for
// timestamp-based eviction.
const dedupExpiry = 30 * time.Minute

func (m *MemoryStorage) SaveUpdateDedupKey(sessionID string, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	set := m.updateDedup[sessionID]
	if set == nil {
		set = make(map[string]time.Time)
		m.updateDedup[sessionID] = set
		m.dedupOrder[sessionID] = nil
	}
	if _, ok := set[key]; ok {
		return false, nil
	}
	if len(set) >= maxDedupKeysPerSession {
		order := m.dedupOrder[sessionID]
		now := time.Now()
		half := maxDedupKeysPerSession / 2
		cleared := 0
		for len(order) > 0 && cleared < half {
			oldest := order[0]
			order = order[1:]
			ts, ok := set[oldest]
			if !ok {
				continue
			}
			delete(set, oldest)
			cleared++
			if now.Sub(ts) < dedupExpiry && cleared >= 1 {
				break
			}
		}
		m.dedupOrder[sessionID] = order
	}
	set[key] = time.Now()
	m.dedupOrder[sessionID] = append(m.dedupOrder[sessionID], key)
	return true, nil
}

func (m *MemoryStorage) UpdateDedupKeyExists(sessionID string, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.updateDedup[sessionID]
	if !ok {
		return false, nil
	}
	_, ok = m.updateDedup[sessionID][key]
	return ok, nil
}

func (m *MemoryStorage) EnqueueDurableUpdate(update *storage.DurableUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.durableUpdates[update.SessionID] == nil {
		m.durableUpdates[update.SessionID] = make(map[string]storage.DurableUpdate)
	}
	m.durableUpdates[update.SessionID][update.ID] = *update
	return nil
}

func (m *MemoryStorage) DeleteDurableUpdate(sessionID string, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.durableUpdates[sessionID], id)
	return nil
}

func (m *MemoryStorage) LoadDurableUpdates(sessionID string, limit int) ([]*storage.DurableUpdate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*storage.DurableUpdate
	for _, item := range m.durableUpdates[sessionID] {
		cp := item
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *MemoryStorage) MarkDurableUpdateFailed(sessionID string, id string, attempts int, lastErr string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inner := m.durableUpdates[sessionID]
	if inner == nil {
		return nil
	}
	item, ok := inner[id]
	if !ok {
		return nil
	}
	item.Attempts = attempts
	item.LastError = lastErr
	inner[id] = item
	return nil
}
