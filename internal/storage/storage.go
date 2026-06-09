// Package storage defines interfaces and types used by the MTGO client
// to persist session data, peer cache entries, update state, and
// conversation state.
//
// It provides a self-contained [Memory] adapter for in-memory storage
// (used for testing and ephemeral sessions) and the [Adapter] interface
// that external implementations can satisfy for persistent backends.
package storage

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	extstorage "github.com/mtgo-labs/storage"
)

type (
	PeerType          = extstorage.PeerType
	Peer              = extstorage.Peer
	Session           = extstorage.Session
	Conversation      = extstorage.Conversation
	PeerStore         = extstorage.PeerStore
	SessionStore      = extstorage.SessionStore
	ConversationStore = extstorage.ConversationStore
	Adapter           = extstorage.Adapter
	DCAuthEntry       = extstorage.DCAuthEntry
	DCAuthStore       = extstorage.DCAuthStore
)

const (
	PeerTypeUser    PeerType = extstorage.PeerTypeUser
	PeerTypeChat    PeerType = extstorage.PeerTypeChat
	PeerTypeChannel PeerType = extstorage.PeerTypeChannel
)

// UpdateState holds the client's update sequence numbers.
type UpdateState struct {
	SessionID string `json:"session_id"`
	Pts       int32  `json:"pts"`
	Qts       int32  `json:"qts"`
	Date      int32  `json:"date"`
	Seq       int32  `json:"seq"`
}

// ChannelUpdateState holds the update state for a single channel.
type ChannelUpdateState struct {
	SessionID string `json:"session_id"`
	ChannelID int64  `json:"channel_id"`
	Pts       int32  `json:"pts"`
}

// DurableUpdate represents an update that must be delivered even
// across handler failures or process restarts.
type DurableUpdate struct {
	SessionID string `json:"session_id"`
	ID        string `json:"id"`
	Payload   []byte `json:"payload"`
	Attempts  int    `json:"attempts"`
	LastError string `json:"last_error"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// UpdateStateStore is the interface used by the update manager to
// persist and query update-related state.
type UpdateStateStore interface {
	LoadUpdateState(sessionID string) (*UpdateState, error)
	SaveUpdateState(state *UpdateState) error
	LoadChannelUpdateState(sessionID string, channelID int64) (*ChannelUpdateState, error)
	LoadAllChannelUpdateStates(sessionID string) ([]*ChannelUpdateState, error)
	SaveChannelUpdateState(state *ChannelUpdateState) error
	SaveUpdateDedupKey(sessionID string, key string) (bool, error)
	UpdateDedupKeyExists(sessionID string, key string) (bool, error)
	EnqueueDurableUpdate(update *DurableUpdate) error
	DeleteDurableUpdate(sessionID string, id string) error
	LoadDurableUpdates(sessionID string, limit int) ([]*DurableUpdate, error)
	MarkDurableUpdateFailed(sessionID string, id string, attempts int, lastErr string) error
}

// SessionIDAware is an optional interface that adapters can implement
// to receive the session name from the client.
type SessionIDAware interface {
	SetSessionName(name string)
}

// Storage is the full interface the Telegram client uses to read and
// write session fields.
type Storage interface {
	SessionID() (string, error)
	SetSessionID(string) error
	DCID() (int, error)
	SetDCID(int) error
	APIID() (int32, error)
	SetAPIID(int32) error
	APIHash() (string, error)
	SetAPIHash(string) error
	TestMode() (bool, error)
	SetTestMode(bool) error
	AuthKey() ([]byte, error)
	SetAuthKey([]byte) error
	UserID() (int64, error)
	SetUserID(int64) error
	IsBot() (bool, error)
	SetIsBot(bool) error
	FirstName() (string, error)
	SetFirstName(string) error
	LastName() (string, error)
	SetLastName(string) error
	Username() (string, error)
	SetUsername(string) error
	Date() (int, error)
	SetDate(int) error
	State() ([]byte, error)
	SetState([]byte) error
	ExportSessionString() (string, error)
	Close() error
}

// --- Memory adapter ---

// NewMemory returns a Storage backed entirely by in-memory maps.
// All data is lost when the process exits.
func NewMemory() Storage {
	return NewAdapter(newMemoryAdapter())
}

type memoryAdapter struct {
	mu          sync.RWMutex
	sessionName string
	sess        *Session

	peers          map[int64]*Peer
	peerByUsername map[string]int64

	updateStates   map[string]*UpdateState
	channelStates  map[string]map[int64]*ChannelUpdateState
	updateDedup    map[string]map[string]struct{}
	durableUpdates map[string]map[string]*DurableUpdate

	conversations map[conversationKey]*Conversation
}

type conversationKey struct {
	ChatID int64
	UserID int64
}

var (
	_ Adapter           = (*memoryAdapter)(nil)
	_ ConversationStore = (*memoryAdapter)(nil)
	_ UpdateStateStore  = (*memoryAdapter)(nil)
	_ SessionIDAware    = (*memoryAdapter)(nil)
)

func newMemoryAdapter() *memoryAdapter {
	return &memoryAdapter{
		peers:          make(map[int64]*Peer),
		peerByUsername: make(map[string]int64),
		updateStates:   make(map[string]*UpdateState),
		channelStates:  make(map[string]map[int64]*ChannelUpdateState),
		updateDedup:    make(map[string]map[string]struct{}),
		durableUpdates: make(map[string]map[string]*DurableUpdate),
		conversations:  make(map[conversationKey]*Conversation),
	}
}

func (m *memoryAdapter) SetSessionName(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionName = name
}

func (m *memoryAdapter) LoadSession() (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.sess == nil {
		return nil, nil
	}
	cp := *m.sess
	return &cp, nil
}

func (m *memoryAdapter) SaveSession(s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *s
	m.sess = &cp
	return nil
}

func (m *memoryAdapter) SavePeer(p *Peer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *p
	m.peers[p.ID] = &cp
	if p.Username != "" {
		m.peerByUsername[p.Username] = p.ID
	}
	return nil
}

func (m *memoryAdapter) GetPeer(id int64) (*Peer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.peers[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (m *memoryAdapter) GetPeerByUsername(username string) (*Peer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.peerByUsername[username]
	if !ok {
		return nil, nil
	}
	p, ok := m.peers[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (m *memoryAdapter) LoadPeers() ([]*Peer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Peer, 0, len(m.peers))
	for _, p := range m.peers {
		cp := *p
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memoryAdapter) DeletePeer(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.peers[id]; ok {
		if p.Username != "" {
			delete(m.peerByUsername, p.Username)
		}
		delete(m.peers, id)
	}
	return nil
}

func (m *memoryAdapter) SaveConversation(c *Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := c.UpdatedAt
	if now == 0 {
		now = time.Now().Unix()
	}
	createdAt := c.CreatedAt
	if createdAt == 0 {
		createdAt = now
	}
	cp := *c
	cp.CreatedAt = createdAt
	cp.UpdatedAt = now
	m.conversations[conversationKey{ChatID: c.ChatID, UserID: c.UserID}] = &cp
	return nil
}

func (m *memoryAdapter) LoadConversation(chatID, userID int64) (*Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.conversations[conversationKey{ChatID: chatID, UserID: userID}]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (m *memoryAdapter) DeleteConversation(chatID, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.conversations, conversationKey{ChatID: chatID, UserID: userID})
	return nil
}

func (m *memoryAdapter) LoadUpdateState(sessionID string) (*UpdateState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.updateStates[sessionID]
	if !ok {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (m *memoryAdapter) SaveUpdateState(s *UpdateState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *s
	m.updateStates[s.SessionID] = &cp
	return nil
}

func (m *memoryAdapter) LoadChannelUpdateState(sessionID string, channelID int64) (*ChannelUpdateState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channels := m.channelStates[sessionID]
	if channels == nil {
		return nil, nil
	}
	s, ok := channels[channelID]
	if !ok {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (m *memoryAdapter) LoadAllChannelUpdateStates(sessionID string) ([]*ChannelUpdateState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	channels := m.channelStates[sessionID]
	if channels == nil {
		return nil, nil
	}
	out := make([]*ChannelUpdateState, 0, len(channels))
	for _, s := range channels {
		cp := *s
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memoryAdapter) SaveChannelUpdateState(s *ChannelUpdateState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.channelStates[s.SessionID] == nil {
		m.channelStates[s.SessionID] = make(map[int64]*ChannelUpdateState)
	}
	cp := *s
	m.channelStates[s.SessionID][s.ChannelID] = &cp
	return nil
}

func (m *memoryAdapter) SaveUpdateDedupKey(sessionID string, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateDedup[sessionID] == nil {
		m.updateDedup[sessionID] = make(map[string]struct{})
	}
	if _, ok := m.updateDedup[sessionID][key]; ok {
		return false, nil
	}
	m.updateDedup[sessionID][key] = struct{}{}
	return true, nil
}

func (m *memoryAdapter) UpdateDedupKeyExists(sessionID string, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.updateDedup[sessionID][key]
	return ok, nil
}

func (m *memoryAdapter) EnqueueDurableUpdate(u *DurableUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.durableUpdates[u.SessionID] == nil {
		m.durableUpdates[u.SessionID] = make(map[string]*DurableUpdate)
	}
	cp := *u
	m.durableUpdates[u.SessionID][u.ID] = &cp
	return nil
}

func (m *memoryAdapter) DeleteDurableUpdate(sessionID string, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.durableUpdates[sessionID], id)
	return nil
}

func (m *memoryAdapter) LoadDurableUpdates(sessionID string, limit int) ([]*DurableUpdate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*DurableUpdate
	for _, item := range m.durableUpdates[sessionID] {
		cp := *item
		out = append(out, &cp)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *memoryAdapter) MarkDurableUpdateFailed(sessionID string, id string, attempts int, lastErr string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item := m.durableUpdates[sessionID][id]
	if item == nil {
		return nil
	}
	item.Attempts = attempts
	item.LastError = lastErr
	return nil
}

func (m *memoryAdapter) Close() error { return nil }

// --- Adapter wrapper ---

var _ PeerStore = (*adapterWrapper)(nil)

// adapterWrapper wraps an [Adapter] to satisfy the [Storage] interface.
type adapterWrapper struct {
	mu   sync.Mutex
	ext  Adapter
	sess *Session
}

// NewAdapter wraps an [Adapter] so it can be used as a Storage in the
// client config.
func NewAdapter(a Adapter) *adapterWrapper {
	return &adapterWrapper{ext: a}
}

// load initializes a.sess from the external adapter. Caller must hold a.mu.
func (a *adapterWrapper) load() error {
	if a.sess != nil {
		return nil
	}
	s, err := a.ext.LoadSession()
	if err != nil {
		return err
	}
	if s == nil {
		s = &Session{}
	}
	a.sess = s
	return nil
}

// save persists the current session to the external adapter. Caller must hold a.mu.
func (a *adapterWrapper) save() error {
	return a.ext.SaveSession(a.sess)
}

func (a *adapterWrapper) SessionID() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return "", err
	}
	return a.sess.SessionID, nil
}

func (a *adapterWrapper) SetSessionID(v string) error {
	if sa, ok := a.ext.(SessionIDAware); ok {
		sa.SetSessionName(v)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sess = nil
	if err := a.load(); err != nil {
		return fmt.Errorf("load after SetSessionName: %w", err)
	}
	a.sess.SessionID = v
	if err := a.save(); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

func (a *adapterWrapper) DCID() (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return 0, err
	}
	return a.sess.DC, nil
}

func (a *adapterWrapper) SetDCID(v int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sess == nil {
		if err := a.load(); err != nil {
			return err
		}
	}
	a.sess.DC = v
	return a.save()
}

func (a *adapterWrapper) APIID() (int32, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return 0, err
	}
	return a.sess.APIID, nil
}

func (a *adapterWrapper) SetAPIID(v int32) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.APIID = v
	return a.save()
}

func (a *adapterWrapper) APIHash() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return "", err
	}
	return a.sess.APIHash, nil
}

func (a *adapterWrapper) SetAPIHash(v string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.APIHash = v
	return a.save()
}

func (a *adapterWrapper) TestMode() (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return false, err
	}
	return a.sess.TestMode, nil
}

func (a *adapterWrapper) SetTestMode(v bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.TestMode = v
	return a.save()
}

func (a *adapterWrapper) AuthKey() ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return nil, err
	}
	return a.sess.AuthKey, nil
}

func (a *adapterWrapper) SetAuthKey(v []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.AuthKey = v
	return a.save()
}

func (a *adapterWrapper) UserID() (int64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return 0, err
	}
	return a.sess.UserID, nil
}

func (a *adapterWrapper) SetUserID(v int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.UserID = v
	return a.save()
}

func (a *adapterWrapper) IsBot() (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return false, err
	}
	return a.sess.IsBot, nil
}

func (a *adapterWrapper) SetIsBot(v bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.IsBot = v
	return a.save()
}

func (a *adapterWrapper) FirstName() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return "", err
	}
	return a.sess.FirstName, nil
}

func (a *adapterWrapper) SetFirstName(v string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.FirstName = v
	return a.save()
}

func (a *adapterWrapper) LastName() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return "", err
	}
	return a.sess.LastName, nil
}

func (a *adapterWrapper) SetLastName(v string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.LastName = v
	return a.save()
}

func (a *adapterWrapper) Username() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return "", err
	}
	return a.sess.Username, nil
}

func (a *adapterWrapper) SetUsername(v string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.Username = v
	return a.save()
}

func (a *adapterWrapper) Date() (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return 0, err
	}
	return a.sess.Date, nil
}

func (a *adapterWrapper) SetDate(v int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.Date = v
	return a.save()
}

func (a *adapterWrapper) State() ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return nil, err
	}
	return a.sess.State, nil
}

func (a *adapterWrapper) SetState(v []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return err
	}
	a.sess.State = v
	return a.save()
}

func (a *adapterWrapper) ExportSessionString() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.load(); err != nil {
		return "", err
	}
	if len(a.sess.AuthKey) == 0 {
		return "", nil
	}
	if a.sess.APIID == 0 {
		return "", fmt.Errorf("telegram: cannot export session: api_id not stored")
	}

	buf := make([]byte, 0, 271)
	buf = append(buf, byte(a.sess.DC))
	apiID := make([]byte, 4)
	binary.BigEndian.PutUint32(apiID, uint32(a.sess.APIID))
	buf = append(buf, apiID...)
	testMode := byte(0)
	if a.sess.TestMode {
		testMode = 1
	}
	buf = append(buf, testMode)
	buf = append(buf, a.sess.AuthKey...)
	userID := make([]byte, 8)
	binary.BigEndian.PutUint64(userID, uint64(a.sess.UserID))
	buf = append(buf, userID...)
	isBot := byte(0)
	if a.sess.IsBot {
		isBot = 1
	}
	buf = append(buf, isBot)

	encoded := base64.URLEncoding.EncodeToString(buf)
	encoded = strings.TrimRight(encoded, "=")
	return encoded, nil
}

var prodDCAddresses = map[int]string{
	1: "149.154.175.53",
	2: "149.154.167.51",
	3: "149.154.175.100",
	4: "149.154.167.91",
	5: "149.154.171.5",
}

func dcIPv4Address(id int) string {
	return prodDCAddresses[id]
}
func (a *adapterWrapper) Close() error           { return a.ext.Close() }
func (a *adapterWrapper) SavePeer(p *Peer) error { return a.ext.SavePeer(p) }
func (a *adapterWrapper) SavePeers(peers []*Peer) error {
	for _, p := range peers {
		if err := a.ext.SavePeer(p); err != nil {
			return err
		}
	}
	return nil
}

func (a *adapterWrapper) LoadPeers() ([]*Peer, error) {
	return a.ext.LoadPeers()
}

func (a *adapterWrapper) GetPeer(id int64) (*Peer, error) { return a.ext.GetPeer(id) }

func (a *adapterWrapper) GetPeerByUsername(username string) (*Peer, error) {
	return a.ext.GetPeerByUsername(username)
}

func (a *adapterWrapper) DeletePeer(id int64) error { return a.ext.DeletePeer(id) }

func (a *adapterWrapper) cs() ConversationStore {
	if s, ok := a.ext.(ConversationStore); ok {
		return s
	}
	return nil
}

func (a *adapterWrapper) SaveConversation(c *Conversation) error {
	if s := a.cs(); s != nil {
		return s.SaveConversation(c)
	}
	return nil
}

func (a *adapterWrapper) LoadConversation(chatID, userID int64) (*Conversation, error) {
	if s := a.cs(); s != nil {
		return s.LoadConversation(chatID, userID)
	}
	return nil, nil
}

func (a *adapterWrapper) DeleteConversation(chatID, userID int64) error {
	if s := a.cs(); s != nil {
		return s.DeleteConversation(chatID, userID)
	}
	return nil
}

func (a *adapterWrapper) uss() UpdateStateStore {
	if s, ok := a.ext.(UpdateStateStore); ok {
		return s
	}
	return nil
}

func (a *adapterWrapper) LoadUpdateState(sid string) (*UpdateState, error) {
	if s := a.uss(); s != nil {
		return s.LoadUpdateState(sid)
	}
	return nil, nil
}

func (a *adapterWrapper) SaveUpdateState(state *UpdateState) error {
	if s := a.uss(); s != nil {
		return s.SaveUpdateState(state)
	}
	return nil
}

func (a *adapterWrapper) LoadChannelUpdateState(sid string, cid int64) (*ChannelUpdateState, error) {
	if s := a.uss(); s != nil {
		return s.LoadChannelUpdateState(sid, cid)
	}
	return nil, nil
}

func (a *adapterWrapper) LoadAllChannelUpdateStates(sid string) ([]*ChannelUpdateState, error) {
	if s := a.uss(); s != nil {
		return s.LoadAllChannelUpdateStates(sid)
	}
	return nil, nil
}

func (a *adapterWrapper) SaveChannelUpdateState(state *ChannelUpdateState) error {
	if s := a.uss(); s != nil {
		return s.SaveChannelUpdateState(state)
	}
	return nil
}

func (a *adapterWrapper) SaveUpdateDedupKey(sid, key string) (bool, error) {
	if s := a.uss(); s != nil {
		return s.SaveUpdateDedupKey(sid, key)
	}
	return true, nil
}

func (a *adapterWrapper) UpdateDedupKeyExists(sid, key string) (bool, error) {
	if s := a.uss(); s != nil {
		return s.UpdateDedupKeyExists(sid, key)
	}
	return false, nil
}

func (a *adapterWrapper) EnqueueDurableUpdate(u *DurableUpdate) error {
	if s := a.uss(); s != nil {
		return s.EnqueueDurableUpdate(u)
	}
	return nil
}

func (a *adapterWrapper) DeleteDurableUpdate(sid, key string) error {
	if s := a.uss(); s != nil {
		return s.DeleteDurableUpdate(sid, key)
	}
	return nil
}

func (a *adapterWrapper) LoadDurableUpdates(sid string, limit int) ([]*DurableUpdate, error) {
	if s := a.uss(); s != nil {
		return s.LoadDurableUpdates(sid, limit)
	}
	return nil, nil
}

func (a *adapterWrapper) MarkDurableUpdateFailed(sid, key string, attempts int, lastErr string) error {
	if s := a.uss(); s != nil {
		return s.MarkDurableUpdateFailed(sid, key, attempts, lastErr)
	}
	return nil
}
