package telegram

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/mtgo-labs/mtgo/internal/storage"
)

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
	dcID          int
	apiID         int32
	testMode      bool
	authKey       []byte
	sessionID     string
	userID        int64
	isBot         bool
	date          int
	serverAddress string
	port          int
	state         []byte
	firstName     string
	lastName      string
	username      string
	apiHash       string

	peers          map[int64]storage.Peer
	dcAuths        map[int]storage.DCAuthEntry
	updateStates   map[string]storage.UpdateState
	channelStates  map[string]map[int64]storage.ChannelUpdateState
	updateDedup    map[string]map[string]struct{}
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
		dcAuths:        make(map[int]storage.DCAuthEntry),
		updateStates:   make(map[string]storage.UpdateState),
		channelStates:  make(map[string]map[int64]storage.ChannelUpdateState),
		updateDedup:    make(map[string]map[string]struct{}),
		durableUpdates: make(map[string]map[string]storage.DurableUpdate),
	}
}

func (m *MemoryStorage) SessionID() (string, error)      { return m.sessionID, nil }
func (m *MemoryStorage) SetSessionID(v string) error     { m.sessionID = v; return nil }
func (m *MemoryStorage) DCID() (int, error)              { return m.dcID, nil }
func (m *MemoryStorage) SetDCID(v int) error             { m.dcID = v; return nil }
func (m *MemoryStorage) APIID() (int32, error)           { return m.apiID, nil }
func (m *MemoryStorage) SetAPIID(v int32) error          { m.apiID = v; return nil }
func (m *MemoryStorage) TestMode() (bool, error)         { return m.testMode, nil }
func (m *MemoryStorage) SetTestMode(v bool) error        { m.testMode = v; return nil }
func (m *MemoryStorage) AuthKey() ([]byte, error)        { return append([]byte(nil), m.authKey...), nil }
func (m *MemoryStorage) SetAuthKey(v []byte) error       { m.authKey = v; return nil }
func (m *MemoryStorage) UserID() (int64, error)          { return m.userID, nil }
func (m *MemoryStorage) SetUserID(v int64) error         { m.userID = v; return nil }
func (m *MemoryStorage) IsBot() (bool, error)            { return m.isBot, nil }
func (m *MemoryStorage) SetIsBot(v bool) error           { m.isBot = v; return nil }
func (m *MemoryStorage) FirstName() (string, error)      { return m.firstName, nil }
func (m *MemoryStorage) SetFirstName(v string) error     { m.firstName = v; return nil }
func (m *MemoryStorage) LastName() (string, error)       { return m.lastName, nil }
func (m *MemoryStorage) SetLastName(v string) error      { m.lastName = v; return nil }
func (m *MemoryStorage) Username() (string, error)       { return m.username, nil }
func (m *MemoryStorage) SetUsername(v string) error      { m.username = v; return nil }
func (m *MemoryStorage) APIHash() (string, error)        { return m.apiHash, nil }
func (m *MemoryStorage) SetAPIHash(v string) error       { m.apiHash = v; return nil }
func (m *MemoryStorage) Date() (int, error)              { return m.date, nil }
func (m *MemoryStorage) SetDate(v int) error             { m.date = v; return nil }
func (m *MemoryStorage) ServerAddress() (string, error)  { return m.serverAddress, nil }
func (m *MemoryStorage) SetServerAddress(v string) error { m.serverAddress = v; return nil }
func (m *MemoryStorage) Port() (int, error)              { return m.port, nil }
func (m *MemoryStorage) SetPort(v int) error             { m.port = v; return nil }
func (m *MemoryStorage) State() ([]byte, error)          { return append([]byte(nil), m.state...), nil }
func (m *MemoryStorage) SetState(v []byte) error         { m.state = v; return nil }

func (m *MemoryStorage) ExportSessionString() (string, error) {
	if len(m.authKey) == 0 {
		return "", nil
	}

	var ip net.IP
	if m.serverAddress != "" {
		ip = net.ParseIP(m.serverAddress)
		if ip == nil {
			ip = net.ParseIP("0.0.0.0")
		}
	} else {
		ip = net.ParseIP("0.0.0.0")
	}
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}

	buf := new(bytes.Buffer)
	buf.WriteByte(uint8(m.dcID))
	buf.Write(ip)
	_ = binary.Write(buf, binary.BigEndian, uint16(m.port))
	buf.Write(m.authKey)

	return "1" + base64.URLEncoding.EncodeToString(buf.Bytes()), nil
}

func (m *MemoryStorage) Close() error { return nil }

func (m *MemoryStorage) SavePeer(p *storage.Peer) error {
	if m.peers == nil {
		m.peers = make(map[int64]storage.Peer)
	}
	m.peers[p.ID] = *p
	return nil
}

func (m *MemoryStorage) GetPeer(id int64) (*storage.Peer, error) {
	p, ok := m.peers[id]
	if !ok {
		return nil, nil
	}
	return &p, nil
}

func (m *MemoryStorage) GetPeerByUsername(username string) (*storage.Peer, error) {
	for _, p := range m.peers {
		if p.Username == username {
			return &p, nil
		}
	}
	return nil, nil
}

func (m *MemoryStorage) LoadPeers() ([]*storage.Peer, error) {
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
	delete(m.peers, id)
	return nil
}

func (m *MemoryStorage) SaveDCAuth(entry storage.DCAuthEntry) error {
	if m.dcAuths == nil {
		m.dcAuths = make(map[int]storage.DCAuthEntry)
	}
	m.dcAuths[entry.DCID] = entry
	return nil
}

func (m *MemoryStorage) LoadDCAuth(dcID int) (storage.DCAuthEntry, error) {
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
	state, ok := m.updateStates[sessionID]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (m *MemoryStorage) SaveUpdateState(state *storage.UpdateState) error {
	m.updateStates[state.SessionID] = *state
	return nil
}

func (m *MemoryStorage) LoadChannelUpdateState(sessionID string, channelID int64) (*storage.ChannelUpdateState, error) {
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
	if m.channelStates[state.SessionID] == nil {
		m.channelStates[state.SessionID] = make(map[int64]storage.ChannelUpdateState)
	}
	m.channelStates[state.SessionID][state.ChannelID] = *state
	return nil
}

// maxDedupKeysPerSession caps the number of dedup keys stored per session
// to prevent unbounded growth in long-running processes.
const maxDedupKeysPerSession = 10000

func (m *MemoryStorage) SaveUpdateDedupKey(sessionID string, key string) (bool, error) {
	set := m.updateDedup[sessionID]
	if set == nil {
		set = make(map[string]struct{})
		m.updateDedup[sessionID] = set
	}
	if _, ok := set[key]; ok {
		return false, nil
	}
	if len(set) >= maxDedupKeysPerSession {
		// Evict oldest half to prevent unbounded growth; map iteration
		// order is non-deterministic but provides approximate oldest-first
		// behavior for a fixed-capacity dedup set.
		half := maxDedupKeysPerSession / 2
		cleared := 0
		for k := range set {
			delete(set, k)
			cleared++
			if cleared >= half {
				break
			}
		}
	}
	set[key] = struct{}{}
	return true, nil
}

func (m *MemoryStorage) UpdateDedupKeyExists(sessionID string, key string) (bool, error) {
	_, ok := m.updateDedup[sessionID][key]
	return ok, nil
}

func (m *MemoryStorage) EnqueueDurableUpdate(update *storage.DurableUpdate) error {
	if m.durableUpdates[update.SessionID] == nil {
		m.durableUpdates[update.SessionID] = make(map[string]storage.DurableUpdate)
	}
	m.durableUpdates[update.SessionID][update.ID] = *update
	return nil
}

func (m *MemoryStorage) DeleteDurableUpdate(sessionID string, id string) error {
	delete(m.durableUpdates[sessionID], id)
	return nil
}

func (m *MemoryStorage) LoadDurableUpdates(sessionID string, limit int) ([]*storage.DurableUpdate, error) {
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
	item := m.durableUpdates[sessionID][id]
	item.Attempts = attempts
	item.LastError = lastErr
	m.durableUpdates[sessionID][id] = item
	return nil
}
