package peers

import (
	"sync"

	"github.com/mtgo-labs/mtgo/tg"
)

// Ref is an opaque peer reference: exactly one field carries the identity.
// It is the value type behind telegram.ChatRef and telegram.UserRef; the
// zero ID resolves to the current user.
type Ref struct {
	// ID is a marked or raw numeric peer ID (0 means self).
	ID int64
	// Username is a Telegram username without the leading '@'.
	Username string
	// Phone is a phone number; normalized automatically.
	Phone string
	// Peer is a pre-built input peer, used as-is.
	Peer tg.InputPeerClass
}

// Deps are the Manager's runtime dependencies. Each is read at call time so
// that lazy storage creation, runtime config changes, and post-login bot
// status are always honored. All fields are optional; nil functions disable
// the corresponding behavior.
type Deps struct {
	// Invoker performs RPC calls (e.g. *tg.RPCClient, from
	// telegram.Client.Raw). Required before any network resolution.
	Invoker func() *tg.RPCClient
	// Store returns the persistent peer store adapter, or nil when no
	// storage is configured.
	Store func() Store
	// CacheSize returns the in-memory cache limit; 0 or negative disables
	// eviction.
	CacheSize func() int
	// SavePeers reports whether resolved peers should be persisted.
	SavePeers func() bool
	// IsBot reports whether the client is authorized as a bot (changes the
	// numeric-ID fallback strategy and enables hash completion).
	IsBot func() bool
	// Debugf receives debug-level log lines.
	Debugf func(format string, args ...any)
}

// Manager resolves peer references into input peers, maintaining an
// in-memory cache and an optional persistent store. Construct it with
// [NewManager]; it is safe for concurrent use.
type Manager struct {
	deps Deps

	mu            sync.RWMutex
	byID          map[int64]tg.InputPeerClass
	idOrder       []int64
	usernameToID  map[string]int64
	idToUsername  map[int64]string
	usernameOrder []string
	phoneToID     map[string]int64
	phoneOrder    []string
	coalescer     coalescer
}

// NewManager creates a Manager from the given dependencies.
func NewManager(deps Deps) *Manager {
	return &Manager{
		deps:         deps,
		byID:         make(map[int64]tg.InputPeerClass),
		usernameToID: make(map[string]int64),
		idToUsername: make(map[int64]string),
		phoneToID:    make(map[string]int64),
	}
}

func (m *Manager) invoker() *tg.RPCClient {
	if m.deps.Invoker == nil {
		return nil
	}
	return m.deps.Invoker()
}

func (m *Manager) store() Store {
	if m.deps.Store == nil {
		return nil
	}
	return m.deps.Store()
}

func (m *Manager) cacheSize() int {
	if m.deps.CacheSize == nil {
		return 0
	}
	return m.deps.CacheSize()
}

func (m *Manager) savePeers() bool {
	return m.deps.SavePeers != nil && m.deps.SavePeers()
}

func (m *Manager) isBot() bool {
	return m.deps.IsBot != nil && m.deps.IsBot()
}

func (m *Manager) debugf(format string, args ...any) {
	if m.deps.Debugf != nil {
		m.deps.Debugf(format, args...)
	}
}

// Reset drops all in-memory cache state. Used when the underlying
// authorization is lost and cached access hashes can no longer be trusted.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID = make(map[int64]tg.InputPeerClass)
	m.idOrder = nil
	m.usernameToID = make(map[string]int64)
	m.idToUsername = make(map[int64]string)
	m.usernameOrder = nil
	m.phoneToID = make(map[string]int64)
	m.phoneOrder = nil
}
