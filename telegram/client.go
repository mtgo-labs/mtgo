// Package telegram provides a high-level Go client for the Telegram MTProto API.
//
// It offers connection management, authentication, RPC invocation, peer resolution,
// update handling, and a context-based API for responding to incoming events.
//
// Basic usage:
//
//	client, err := telegram.NewClient(apiID, apiHash)
//	if err := client.Connect(30 * time.Second); err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Disconnect()
package telegram

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/mtproxy"

	"github.com/mtgo-labs/mtgo/internal/storage"

	sessions "github.com/mtgo-labs/mtgo/session"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

var updatePool = sync.Pool{
	New: func() interface{} { return &Update{} },
}

type sessionKey struct {
	dcID    int
	isMedia bool
}

// UpdatePacket wraps a raw Telegram update together with the resolved user and chat maps
// extracted from the update, ready for dispatch to handlers.
type UpdatePacket struct {
	// Update is the raw Telegram UpdatesClass received from the server.
	Update tg.UpdatesClass
	// Users maps user IDs to their resolved User objects from this update batch.
	Users map[int64]*types.User
	// Chats maps chat IDs to their resolved Chat objects from this update batch.
	Chats map[int64]*types.Chat
}

// Dispatcher is the interface for an update dispatcher that enqueues, routes,
// and manages handler groups for incoming Telegram updates.
type Dispatcher interface {
	// Start begins dispatching updates using the specified number of worker goroutines.
	Start(workers int) error
	// Stop gracefully shuts down the dispatcher, waiting for in-flight handlers to finish.
	Stop() error
	// AddHandler registers a Handler in the given priority group.
	AddHandler(handler Handler, group int)
	// RemoveHandler removes a previously registered Handler from the given group.
	RemoveHandler(handler Handler, group int)
	// Enqueue submits an UpdatePacket for asynchronous dispatch to registered handlers.
	Enqueue(packet UpdatePacket) error
}

// Client is the main Telegram MTProto client. It manages connections, sessions,
// authentication state, peer resolution caches, and update dispatching.
//
// Create a new Client with NewClient, then call Connect to establish a session.
// Use the accessor methods (Me, Session, Storage, Config) to inspect client state,
// and the RPC methods (Invoke, Raw) to make arbitrary API calls.
type Client struct {
	cfg     Config
	mu      sync.RWMutex
	state   *connStateManager
	storage storage.Storage
	session *session.Session
	me      *types.User
	dialer  transport.Dialer
	Log     *Logger

	sessions           map[sessionKey]*session.Session
	sessionsMu         sync.Mutex
	dispatcher         Dispatcher
	handlerDispatcher  *HandlerDispatcher
	plugins            map[string]Plugin
	middlewares        []middlewareEntry
	mwCache            []Middleware
	invokerMiddlewares []InvokerMiddleware
	invokerCache       *tg.RPCClient

	peerCache          map[int64]tg.InputPeerClass
	peerCacheMu        sync.RWMutex
	usernameCache      map[string]int64
	peerCacheOrder     []int64
	usernameCacheOrder []string
	resolveCoalescer   resolveCoalescer

	stopCh chan struct{}

	reconnectMgr  *reconnectManager
	healthCheck   *healthChecker
	updateManager *updateManager

	autoConnectMu sync.Mutex

	secretChats           *SecretChatManager
	secretMsgHandlers     []SecretMessageHandler
	secretChatReqHandlers []SecretChatRequestHandler

	dcSessions *dcSessions

	testStorage  storage.Storage
	testSession  *session.Session
	testSessionF func(ctx context.Context, dcID int, addr string, port int, authKey []byte) (*session.Session, error)
	testInvoker  tg.Invoker
	testDialer   transport.Dialer
	testResolver PeerResolver

	// rng is a per-client random source, avoiding contention on the global
	// math/rand mutex under high concurrency.
	rng *rand.Rand

	// Booleans grouped at end to minimize padding on 64-bit.
	apiInit     bool
	mwSorted    bool
	migratingDC bool
}

// NewClient creates a new Telegram client with the given API credentials and optional configuration.
//
// apiID and apiHash are the Telegram API credentials obtained from https://my.telegram.org.
// A *Config can be passed as the third argument to customize behavior.
//
// The client is not connected after construction; call Connect or Start to establish a session.
//
// Example:
//
//	client, err := telegram.NewClient(12345, "your_api_hash", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Disconnect()
func NewClient(apiID int32, apiHash string, cfg *Config) (*Client, error) {
	if apiID == 0 {
		return nil, ErrAPIIDRequired
	}
	if apiHash == "" {
		return nil, ErrAPIHashRequired
	}

	c := DefaultConfig
	c.APIID = apiID
	c.APIHash = apiHash
	if cfg != nil {
		if cfg.SessionName != "" {
			c.SessionName = cfg.SessionName
		}
		if cfg.BotToken != "" {
			c.BotToken = cfg.BotToken
		}

		if cfg.PhoneNumber != "" {
			c.PhoneNumber = cfg.PhoneNumber
		}
		if cfg.PhoneCode != "" {
			c.PhoneCode = cfg.PhoneCode
		}
		if cfg.Password != "" {
			c.Password = cfg.Password
		}
		if cfg.WorkDir != "" {
			c.WorkDir = cfg.WorkDir
		}
		if cfg.InMemory {
			c.InMemory = true
		}
		if cfg.Proxy != nil {
			c.Proxy = cfg.Proxy
		}
		if cfg.TestMode {
			c.TestMode = true
		}
		if cfg.IPv6 {
			c.IPv6 = true
		}
		if cfg.NoUpdates {
			c.NoUpdates = true
		}
		if cfg.AutoConnect {
			c.AutoConnect = true
		}
		if cfg.SkipUpdates {
			c.SkipUpdates = true
		}
		if cfg.SleepThreshold != 0 {
			c.SleepThreshold = cfg.SleepThreshold
		}
		if cfg.HandlerTimeout != 0 {
			c.HandlerTimeout = cfg.HandlerTimeout
		}
		if cfg.Timeout != 0 {
			c.Timeout = cfg.Timeout
		}
		if cfg.ReqTimeout != 0 {
			c.ReqTimeout = cfg.ReqTimeout
		}
		if cfg.MaxConcurrentTrans != 0 {
			c.MaxConcurrentTrans = cfg.MaxConcurrentTrans
		}
		if cfg.MaxMessageCacheSize != 0 {
			c.MaxMessageCacheSize = cfg.MaxMessageCacheSize
		}
		if cfg.MaxTopicCacheSize != 0 {
			c.MaxTopicCacheSize = cfg.MaxTopicCacheSize
		}
		if cfg.ParseMode != "" {
			c.ParseMode = cfg.ParseMode
		}

		if cfg.HidePassword {
			c.HidePassword = true
		}
		if cfg.Takeout {
			c.Takeout = true
		}
		if cfg.LinkPreviewOptions != nil {
			c.LinkPreviewOptions = cfg.LinkPreviewOptions
		}
		if cfg.FetchReplies {
			c.FetchReplies = true
		}
		if cfg.FetchTopics {
			c.FetchTopics = true
		}
		if cfg.FetchStories {
			c.FetchStories = true
		}
		if cfg.FetchStickers {
			c.FetchStickers = true
		}
		if cfg.Device.DeviceModel != "" || cfg.Device.AppVersion != "" {
			if cfg.Device.DeviceModel != "" {
				c.Device.DeviceModel = cfg.Device.DeviceModel
			}
			if cfg.Device.SystemVersion != "" {
				c.Device.SystemVersion = cfg.Device.SystemVersion
			}
			if cfg.Device.AppVersion != "" {
				c.Device.AppVersion = cfg.Device.AppVersion
			}
			if cfg.Device.LangCode != "" {
				c.Device.LangCode = cfg.Device.LangCode
			}
			if cfg.Device.LangPack != "" {
				c.Device.LangPack = cfg.Device.LangPack
			}
			if cfg.Device.SystemLangCode != "" {
				c.Device.SystemLangCode = cfg.Device.SystemLangCode
			}
			if cfg.Device.TZOffset != 0 {
				c.Device.TZOffset = cfg.Device.TZOffset
			}
			if cfg.Device.ClientPlatform != "" {
				c.Device.ClientPlatform = cfg.Device.ClientPlatform
			}
		}
		// Deprecated top-level fields override Device for backwards compat.
		if cfg.AppVersion != "" {
			c.Device.AppVersion = cfg.AppVersion
		}
		if cfg.DeviceModel != "" {
			c.Device.DeviceModel = cfg.DeviceModel
		}
		if cfg.SystemVersion != "" {
			c.Device.SystemVersion = cfg.SystemVersion
		}
		if cfg.LangCode != "" {
			c.Device.LangCode = cfg.LangCode
		}
		if cfg.LangPack != "" {
			c.Device.LangPack = cfg.LangPack
		}
		if cfg.SystemLangCode != "" {
			c.Device.SystemLangCode = cfg.SystemLangCode
		}
		if cfg.TZOffset != 0 {
			c.Device.TZOffset = cfg.TZOffset
		}
		if cfg.ClientPlatform != "" {
			c.Device.ClientPlatform = cfg.ClientPlatform
		}
		if cfg.TransportMode != "" {
			c.TransportMode = cfg.TransportMode
		}
		if cfg.Storage != nil {
			c.Storage = cfg.Storage
		}
		if cfg.SessionString != "" {
			c.SessionString = cfg.SessionString
		}
		if cfg.MTProxy != nil {
			c.MTProxy = cfg.MTProxy
		}
		if cfg.SavePeers {
			c.SavePeers = true
		}
		if cfg.WebSocket {
			c.WebSocket = true
		}
		if cfg.WebSocketTLS {
			c.WebSocketTLS = true
		}
		if cfg.DC != 0 {
			c.DC = cfg.DC
		}
		if cfg.ServerAddr != "" {
			c.ServerAddr = cfg.ServerAddr
		}
		if cfg.LocalAddr != "" {
			c.LocalAddr = cfg.LocalAddr
		}
		c.ReconnectEnabled = cfg.ReconnectEnabled
		if cfg.ReconnectBaseDelay != 0 {
			c.ReconnectBaseDelay = cfg.ReconnectBaseDelay
		}
		if cfg.ReconnectMaxDelay != 0 {
			c.ReconnectMaxDelay = cfg.ReconnectMaxDelay
		}
		if cfg.ReconnectMaxAttempts != 0 {
			c.ReconnectMaxAttempts = cfg.ReconnectMaxAttempts
		}
		c.HealthEnabled = cfg.HealthEnabled
		if cfg.HealthPingInterval != 0 {
			c.HealthPingInterval = cfg.HealthPingInterval
		}
		if cfg.HealthPongTimeout != 0 {
			c.HealthPongTimeout = cfg.HealthPongTimeout
		}
		if cfg.UpdateQueueSize != 0 {
			c.UpdateQueueSize = cfg.UpdateQueueSize
		}
		if cfg.DurableUpdateQueue {
			c.DurableUpdateQueue = true
		}
		if cfg.MaxUpdateHandlerRetry != 0 {
			c.MaxUpdateHandlerRetry = cfg.MaxUpdateHandlerRetry
		}
		c.UpdateRecoveryEnabled = cfg.UpdateRecoveryEnabled
		c.Log = cfg.Log
	}
	if _, err := newTCPTransport(c.TransportMode, nil); err != nil {
		return nil, err
	}

	var logger *Logger
	if c.Log.Logger != nil {
		logger = c.Log.Logger
	} else {
		logger = NewLogger("mtgo")
		if c.Log.Level != 0 {
			logger.SetLevel(c.Log.Level)
		}
		if c.Log.File != "" {
			if err := logger.SetFile(c.Log.File, c.Log.MaxSize); err != nil {
				return nil, fmt.Errorf("setup log file: %w", err)
			}
		}
	}

	dialer := transport.Dialer(&transport.NetDialer{LocalAddr: c.LocalAddr})
	if c.Proxy != nil {
		dialer = newProxyDialer(c.Proxy, dialer)
	}

	client := &Client{
		cfg:               c,
		state:             newConnectionState(),
		sessions:          make(map[sessionKey]*session.Session),
		dialer:            dialer,
		peerCache:         make(map[int64]tg.InputPeerClass),
		usernameCache:     make(map[string]int64),
		handlerDispatcher: NewHandlerDispatcher(),
		dcSessions:        newDCSessions(),
		Log:               logger,
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	client.initSecretChats()
	client.reconnectMgr = newReconnectManager(client, client.backoffConfig())
	registerClient(client)

	return client, nil
}

func (c *Client) backoffConfig() backoffConfig {
	cfg := defaultBackoffConfig
	if c.cfg.ReconnectBaseDelay != 0 {
		cfg.BaseDelay = c.cfg.ReconnectBaseDelay
	}
	if c.cfg.ReconnectMaxDelay != 0 {
		cfg.MaxDelay = c.cfg.ReconnectMaxDelay
	}
	if c.cfg.ReconnectMaxAttempts != 0 {
		cfg.MaxAttempts = c.cfg.ReconnectMaxAttempts
	}
	return cfg
}

func (c *Client) healthConfig() healthCheckConfig {
	cfg := defaultHealthCheckConfig
	if c.cfg.HealthPingInterval != 0 {
		cfg.PingInterval = c.cfg.HealthPingInterval
	}
	if c.cfg.HealthPongTimeout != 0 {
		cfg.PongTimeout = c.cfg.HealthPongTimeout
	}
	return cfg
}

// IsConnected reports whether the client has an active connection to Telegram.
func (c *Client) IsConnected() bool {
	return c.state.isConnected()
}

// RandomID returns a cryptographically-non-secure random int64 suitable for
// Telegram message random_id fields. Uses a per-client random source to avoid
// contention on the global math/rand mutex under high concurrency.
func (c *Client) RandomID() int64 {
	if c.rng == nil {
		return rand.Int63()
	}
	return c.rng.Int63()
}

// Me returns the currently authenticated user. If the user has not been cached yet
// and the client is connected, it fetches the user from the server. Returns nil if
// not connected or if the fetch fails.
func (c *Client) Me() *types.User {
	c.mu.RLock()
	me := c.me
	c.mu.RUnlock()
	if me != nil {
		return me
	}
	if !c.IsConnected() {
		return nil
	}
	me, _ = c.GetMe(context.Background())
	return me
}

// Session returns the primary MTProto session, or nil if not connected.
func (c *Client) Session() *session.Session {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.session
}

// Storage returns the persistent storage backend used by the client, or nil if not connected.
func (c *Client) Storage() storage.Storage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.storage
}

// Adapter returns the underlying storage.Adapter, or nil if unavailable.
// Use this for session, peer, and conversation access.
func (c *Client) Adapter() storage.Adapter {
	if a, ok := c.storage.(storage.Adapter); ok {
		return a
	}
	return nil
}

// ConversationStore returns the conversation store, or nil if unavailable.
func (c *Client) ConversationStore() storage.ConversationStore {
	if cs, ok := c.storage.(storage.ConversationStore); ok {
		return cs
	}
	return nil
}

// UpdateState returns the update state store, or nil if unavailable.
func (c *Client) UpdateState() storage.UpdateStateStore {
	if us, ok := c.storage.(storage.UpdateStateStore); ok {
		return us
	}
	return nil
}

// --- Session ---

// LoadSession loads the session from storage.
func (c *Client) LoadSession() (*storage.Session, error) {
	if a := c.Adapter(); a != nil {
		return a.LoadSession()
	}
	return nil, nil
}

// SaveSession persists the session to storage.
func (c *Client) SaveSession(s *storage.Session) error {
	if a := c.Adapter(); a != nil {
		return a.SaveSession(s)
	}
	return nil
}

// --- Peers ---

// SavePeer persists a peer to the storage backend.
func (c *Client) SavePeer(p *storage.Peer) error {
	if a := c.Adapter(); a != nil {
		return a.SavePeer(p)
	}
	return nil
}

// GetPeer retrieves a cached peer by ID from the storage backend.
func (c *Client) GetPeer(id int64) (*storage.Peer, error) {
	if a := c.Adapter(); a != nil {
		return a.GetPeer(id)
	}
	return nil, nil
}

// GetPeerByUsername retrieves a cached peer by username from the storage backend.
func (c *Client) GetPeerByUsername(username string) (*storage.Peer, error) {
	if a := c.Adapter(); a != nil {
		return a.GetPeerByUsername(username)
	}
	return nil, nil
}

// LoadPeers returns all cached peers from the storage backend.
func (c *Client) LoadPeers() ([]*storage.Peer, error) {
	if a := c.Adapter(); a != nil {
		return a.LoadPeers()
	}
	return nil, nil
}

// DeletePeer removes a cached peer from the storage backend.
func (c *Client) DeletePeer(id int64) error {
	if a := c.Adapter(); a != nil {
		return a.DeletePeer(id)
	}
	return nil
}

// --- Conversations ---

// SaveConversation persists a conversation to the storage backend.
func (c *Client) SaveConversation(conv *storage.Conversation) error {
	if cs := c.ConversationStore(); cs != nil {
		return cs.SaveConversation(conv)
	}
	return nil
}

// LoadConversation retrieves a conversation by chat and user ID.
func (c *Client) LoadConversation(chatID, userID int64) (*storage.Conversation, error) {
	if cs := c.ConversationStore(); cs != nil {
		return cs.LoadConversation(chatID, userID)
	}
	return nil, nil
}

// DeleteConversation removes a conversation from the storage backend.
func (c *Client) DeleteConversation(chatID, userID int64) error {
	if cs := c.ConversationStore(); cs != nil {
		return cs.DeleteConversation(chatID, userID)
	}
	return nil
}

// --- Update State ---

// LoadUpdateState loads the update state for a session.
func (c *Client) LoadUpdateState(sessionID string) (*storage.UpdateState, error) {
	if us := c.UpdateState(); us != nil {
		return us.LoadUpdateState(sessionID)
	}
	return nil, nil
}

// SaveUpdateState persists the update state.
func (c *Client) SaveUpdateState(s *storage.UpdateState) error {
	if us := c.UpdateState(); us != nil {
		return us.SaveUpdateState(s)
	}
	return nil
}

// LoadChannelUpdateState loads the channel update state.
func (c *Client) LoadChannelUpdateState(sessionID string, channelID int64) (*storage.ChannelUpdateState, error) {
	if us := c.UpdateState(); us != nil {
		return us.LoadChannelUpdateState(sessionID, channelID)
	}
	return nil, nil
}

// LoadAllChannelUpdateStates loads all channel update states for a session.
func (c *Client) LoadAllChannelUpdateStates(sessionID string) ([]*storage.ChannelUpdateState, error) {
	if us := c.UpdateState(); us != nil {
		return us.LoadAllChannelUpdateStates(sessionID)
	}
	return nil, nil
}

// SaveChannelUpdateState persists a channel update state.
func (c *Client) SaveChannelUpdateState(s *storage.ChannelUpdateState) error {
	if us := c.UpdateState(); us != nil {
		return us.SaveChannelUpdateState(s)
	}
	return nil
}

// SaveUpdateDedupKey inserts a dedup key, returns true if it was new.
func (c *Client) SaveUpdateDedupKey(sessionID string, key string) (bool, error) {
	if us := c.UpdateState(); us != nil {
		return us.SaveUpdateDedupKey(sessionID, key)
	}
	return false, nil
}

// UpdateDedupKeyExists checks if a dedup key exists.
func (c *Client) UpdateDedupKeyExists(sessionID string, key string) (bool, error) {
	if us := c.UpdateState(); us != nil {
		return us.UpdateDedupKeyExists(sessionID, key)
	}
	return false, nil
}

// EnqueueDurableUpdate enqueues a durable update for retry.
func (c *Client) EnqueueDurableUpdate(u *storage.DurableUpdate) error {
	if us := c.UpdateState(); us != nil {
		return us.EnqueueDurableUpdate(u)
	}
	return nil
}

// DeleteDurableUpdate removes a durable update.
func (c *Client) DeleteDurableUpdate(sessionID string, id string) error {
	if us := c.UpdateState(); us != nil {
		return us.DeleteDurableUpdate(sessionID, id)
	}
	return nil
}

// LoadDurableUpdates loads pending durable updates.
func (c *Client) LoadDurableUpdates(sessionID string, limit int) ([]*storage.DurableUpdate, error) {
	if us := c.UpdateState(); us != nil {
		return us.LoadDurableUpdates(sessionID, limit)
	}
	return nil, nil
}

// MarkDurableUpdateFailed marks a durable update as failed.
func (c *Client) MarkDurableUpdateFailed(sessionID string, id string, attempts int, lastErr string) error {
	if us := c.UpdateState(); us != nil {
		return us.MarkDurableUpdateFailed(sessionID, id, attempts, lastErr)
	}
	return nil
}

// Config returns a copy of the client's current configuration.
func (c *Client) Config() Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg
}

// SetDispatcher replaces the update dispatcher used to route incoming updates to handlers.
func (c *Client) SetDispatcher(d Dispatcher) {
	c.mu.Lock()
	c.dispatcher = d
	c.mu.Unlock()
}

func (c *Client) setTestStorage(s storage.Storage) {
	c.mu.Lock()
	c.testStorage = s
	c.mu.Unlock()
}

func (c *Client) setTestSession(s *session.Session) {
	c.mu.Lock()
	c.testSession = s
	c.mu.Unlock()
}

func (c *Client) setTestSessionFactory(f func(ctx context.Context, dcID int, addr string, port int, authKey []byte) (*session.Session, error)) {
	c.mu.Lock()
	c.testSessionF = f
	c.mu.Unlock()
}

func (c *Client) setTestDialer(d transport.Dialer) {
	c.mu.Lock()
	c.testDialer = d
	c.mu.Unlock()
}

// ensureConnected checks whether the client is connected. When AutoConnect is
// enabled and the client is not connected (and not closed), it serialises the
// connection attempt behind autoConnectMu so that exactly one goroutine dials
// while any others wait and then observe the resulting state.
//
// When AutoConnect is false, this is equivalent to state.requireConnected().
func (c *Client) ensureConnected() error {
	if err := c.state.requireConnected(); err == nil {
		return nil
	}
	if !c.cfg.AutoConnect {
		return ErrNotConnected
	}
	if c.state.IsClosed() {
		return ErrClientClosed
	}

	c.autoConnectMu.Lock()
	defer c.autoConnectMu.Unlock()

	if err := c.state.requireConnected(); err == nil {
		return nil
	}
	if c.state.IsClosed() {
		return ErrClientClosed
	}

	timeout := c.cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if err := c.connectTransport(timeout); err != nil {
		return err
	}
	return nil
}

// Connect initializes storage, loads or creates a session, and marks the client as connected.
//
// The timeout parameter is reserved for future use in connection establishment.
// If InMemory mode is enabled, an in-memory storage backend is used.
// If Storage is nil and InMemory is false, Connect returns an error.
// Returns ErrAlreadyConnected if the client is already connected.
//
// After connecting, use the auth methods (SendCode, SignIn, etc.) to authenticate,
// or provide a BotToken option for bot authentication.
//
// Example:
//
//	client, _ := telegram.NewClient(apiID, apiHash, &telegram.Config{InMemory: true})
//	if err := client.Connect(30 * time.Second); err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Disconnect()
func (c *Client) Connect(timeout time.Duration) error {
	if timeout <= 0 {
		timeout = c.cfg.Timeout
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return c.connectTransport(timeout)
}

// Start connects the client and then blocks until Stop is called. This is the
// simplest way to run a long-lived client: it handles connection setup and then
// keeps the process alive to receive updates.
//
// Example:
//
//		client, _ := telegram.NewClient(apiID, apiHash, &telegram.Config{BotToken: "123:ABC"})
//		go func() {
//		    time.Sleep(5 * time.Minute)
//		    client.Stop()
//	}()
//
//		if err := client.Start(); err != nil {
//		    log.Fatal(err)
//	}
func (c *Client) Start() error {
	timeout := c.cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if err := c.connectTransport(timeout); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	if c.stopCh == nil {
		c.stopCh = make(chan struct{})
	}

	<-c.stopCh
	return nil
}

func (c *Client) Stop() {
	if c.stopCh != nil {
		select {
		case <-c.stopCh:
		default:
			close(c.stopCh)
		}
	}
	_ = c.Disconnect()
}

func (c *Client) Idle() {
	if c.stopCh == nil {
		c.stopCh = make(chan struct{})
	}
	<-c.stopCh
}

func (c *Client) connectToDC(dcID int, timeout time.Duration) error {
	c.cfg.DC = dcID
	c.migratingDC = true
	return c.connectTransport(timeout)
}

func (c *Client) initialDCID(st storage.Storage) int {
	if c.cfg.DC != 0 {
		return c.cfg.DC
	}
	if st != nil {
		if dcID, err := st.DCID(); err == nil && dcID != 0 {
			return dcID
		}
	}
	return 2
}

func (c *Client) connectTransport(timeout time.Duration) error {
	c.mu.Lock()
	locked := true
	defer func() {
		if locked {
			c.mu.Unlock()
		}
	}()

	dcID := c.initialDCID(c.testStorage)
	if err := c.state.SetConnecting(dcID); err != nil {
		return err
	}

	st := c.testStorage
	if st == nil {
		if c.cfg.Storage != nil {
			st = c.cfg.Storage
		} else if c.cfg.InMemory {
			st = NewMemoryStorage()
		} else if c.cfg.SessionName != "" {
			s, err := newDefaultStorage(c.cfg.SessionName)
			if err != nil {
				return fmt.Errorf("telegram: auto-create storage: %w", err)
			}
			st = s
		} else {
			return ErrNoStorage
		}
	}
	c.storage = st

	if c.cfg.SessionName != "" {
		if err := st.SetSessionID(c.cfg.SessionName); err != nil {
			return fmt.Errorf("set session id %q: %w", c.cfg.SessionName, err)
		}
	}

	// If SessionString is set, decode and copy session fields into storage.
	if c.cfg.SessionString != "" {
		src, err := sessions.StringSession(c.cfg.SessionString)
		if err != nil {
			return fmt.Errorf("telegram: decode session string: %w", err)
		}
		if dc, _ := src.DCID(); dc > 0 {
			_ = st.SetDCID(dc)
		}
		if addr, _ := src.ServerAddress(); addr != "" {
			_ = st.SetServerAddress(addr)
		}
		if p, _ := src.Port(); p > 0 {
			_ = st.SetPort(p)
		}
		if key, _ := src.AuthKey(); len(key) > 0 {
			_ = st.SetAuthKey(key)
		}
	}

	c.loadPeersFromStorage()

	sess := c.testSession
	if sess == nil {
		dcID := c.initialDCID(st)
		dc := session.DataCenter{
			ID:       dcID,
			TestMode: c.cfg.TestMode,
			IPv6:     c.cfg.IPv6,
		}
		if dc.Address() == "" {
			return fmt.Errorf("unknown dc_id: %d", dc.ID)
		}
		s, err := session.NewSession(dc, st, c.cfg.Device.DeviceModel, c.cfg.Device.AppVersion, c.cfg.Device.SystemLangCode, c.cfg.Device.LangCode)
		if err != nil {
			return fmt.Errorf("create session: %w", err)
		}
		sess = s
		if err := st.SetDCID(dcID); err != nil {
			return fmt.Errorf("save dc_id: %w", err)
		}
	}
	c.session = sess

	dc := sess.DC()

	var sessionTp *sessionTransport

	if useWebSocket(c.cfg) {
		wsAddr := wsDCAddress(dc.ID, dc.TestMode, c.cfg.WebSocketTLS)
		wsCtx, wsCancel := dialerCtx(timeout)
		defer wsCancel()
		wsConn, err := transport.DialWebsocket(wsCtx, wsAddr)
		if err != nil {
			return fmt.Errorf("ws dial %s: %w", wsAddr, err)
		}
		tp := transport.NewTCPIntermediateNoHeader(wsConn)
		if err := tp.Connect(); err != nil {
			wsConn.Close()
			return fmt.Errorf("ws transport handshake: %w", err)
		}
		sessionTp = newSessionTransport(tp, wsConn)
	} else if c.cfg.MTProxy != nil {
		mpConn, err := mtproxy.Dial(c.cfg.MTProxy.Addr, c.cfg.MTProxy.Secret, dc.ID, timeout)
		if err != nil {
			return fmt.Errorf("mtproxy dial: %w", err)
		}
		tp := transport.NewTCPIntermediateNoHeader(mpConn)
		if err := tp.Connect(); err != nil {
			mpConn.Close()
			return fmt.Errorf("mtproxy transport handshake: %w", err)
		}
		sessionTp = newSessionTransport(tp, mpConn)
	} else {
		addr := fmt.Sprintf("%s:%d", dc.Address(), dc.Port())
		if c.cfg.ServerAddr != "" {
			addr = c.cfg.ServerAddr
		}

		d := c.dialer
		if c.testDialer != nil {
			d = c.testDialer
		}
		conn, err := d.Dial("tcp", addr, timeout)
		if err != nil {
			return fmt.Errorf("dial %s: %w", addr, err)
		}

		tp, err := newTCPTransport(c.cfg.TransportMode, conn)
		if err != nil {
			conn.Close()
			return err
		}
		if err := tp.Connect(); err != nil {
			conn.Close()
			return fmt.Errorf("transport handshake: %w", err)
		}
		sessionTp = newSessionTransport(tp, conn)
	}

	authKey, _ := st.AuthKey()
	needDH := len(authKey) == 0 || c.migratingDC
	c.migratingDC = false
	if needDH {
		c.Log.Debug("auth key missing; starting DH exchange with DC ", dc.ID)
		auth := &session.Auth{
			DC:       dc.ID,
			TestMode: dc.TestMode,
		}
		c.mu.Unlock()
		locked = false
		result, err := auth.Create(sessionTp)
		if err != nil {
			sessionTp.Close()
			return fmt.Errorf("DH key exchange: %w", err)
		}
		c.mu.Lock()
		locked = true
		sess.SetAuthKey(result.AuthKey)
		sess.SetServerSalt(result.ServerSalt)
		sess.SetServerTime(time.Unix(int64(result.ServerTime), 0))
		if err := st.SetAuthKey(result.AuthKey); err != nil {
			return fmt.Errorf("save auth key: %w", err)
		}
		if err := st.SetAPIID(c.cfg.APIID); err != nil {
			c.Log.Warnf("save api_id: %v", err)
		}
		if err := st.SetAPIHash(c.cfg.APIHash); err != nil {
			c.Log.Warnf("save api_hash: %v", err)
		}
		if err := st.SetServerAddress(dc.Address()); err != nil {
			c.Log.Warnf("save server address: %v", err)
		}
		if err := st.SetPort(dc.Port()); err != nil {
			c.Log.Warnf("save port: %v", err)
		}
		if err := st.SetTestMode(dc.TestMode); err != nil {
			c.Log.Warnf("save test mode: %v", err)
		}
		c.Log.Debug("DH exchange complete; auth_key=", len(result.AuthKey), " bytes")
	} else {
		c.Log.Debug("loaded auth key from session; auth_key=", len(authKey), " bytes")
	}

	sess.SetUpdateHandler(func(obj tg.TLObject) {
		c.processRawUpdate(obj)
	})
	sess.SetOnDisconnect(func(err error) {
		c.Log.Warnf("session transport error: %v", err)
		if c.state.IsConnected() {
			c.triggerReconnect(err)
		}
	})
	sess.SetOnPanic(func(r any) {
		c.Log.Errorf("session dispatch panic: %v", r)
	})

	c.apiInit = false
	c.Log.Debug("starting encrypted session")
	if err := sess.Connect(sessionTp, timeout); err != nil {
		sessionTp.Close()
		return fmt.Errorf("session start: %w", err)
	}
	c.Log.Info("encrypted session started")

	// Release c.mu before making RPC calls (bot auth, UpdatesGetState)
	// to avoid deadlock with Invoke's c.mu.RLock. Mark connected first
	// so requireConnected() passes for the RPC invocations.
	c.state.SetConnected()
	c.state.SetDC(dcID)
	c.mu.Unlock()
	locked = false

	if botToken := c.cfg.BotToken; botToken != "" {
		alreadyAuthorized := false
		isUserAccount := false
		if st != nil {
			if uid, err := st.UserID(); err == nil && uid != 0 {
				if isBot, err := st.IsBot(); err == nil && isBot {
					alreadyAuthorized = true
				} else {
					isUserAccount = true
				}
			}
		}

		if isUserAccount {
			c.Log.Debug("session is a user account; skipping bot auth import (remove BOT_TOKEN for userbots)")
			if uid, _ := st.UserID(); uid != 0 {
				c.me = &types.User{
					ID:        uid,
					FirstName: func() string { v, _ := st.FirstName(); return v }(),
					LastName:  func() string { v, _ := st.LastName(); return v }(),
					Username:  func() string { v, _ := st.Username(); return v }(),
				}
				c.Log.Debug("user account restored: id=", c.me.ID, " username=", c.me.Username)
			}
		}

		if !alreadyAuthorized && !isUserAccount {
			c.Log.Info("importing bot authorization")
			rpc := c.Raw()
			authResult, err := rpc.AuthImportBotAuthorization(context.Background(), &tg.AuthImportBotAuthorizationRequest{
				Flags:        0,
				APIID:        c.cfg.APIID,
				APIHash:      c.cfg.APIHash,
				BotAuthToken: botToken,
			})
			if err != nil {
				var rpcErr *tgerr.Error
				if errors.As(err, &rpcErr) && rpcErr.Code == 303 && rpcErr.Type == "USER_MIGRATE" {
					c.cleanupSessions(false)
					c.Log.Debug("migrating to DC ", rpcErr.Argument)
					return c.connectToDC(rpcErr.Argument, timeout)
				}
				c.cleanupSessions()
				return fmt.Errorf("bot auth: %w", err)
			}
			if auth, ok := authResult.(*tg.AuthAuthorization); ok {
				if auth.User != nil {
					if u, ok := auth.User.(*tg.User); ok && u != nil {
						c.me = types.ParseUser(u)
						c.Log.Info("bot user: id=", c.me.ID, " username=", c.me.Username)
						if st != nil {
							if err := st.SetUserID(c.me.ID); err != nil {
								c.Log.Warnf("save user id: %v", err)
							}
							if err := st.SetIsBot(true); err != nil {
								c.Log.Warnf("save is_bot: %v", err)
							}
							if err := st.SetFirstName(c.me.FirstName); err != nil {
								c.Log.Warnf("save first_name: %v", err)
							}
							if err := st.SetLastName(c.me.LastName); err != nil {
								c.Log.Warnf("save last_name: %v", err)
							}
							if err := st.SetUsername(c.me.Username); err != nil {
								c.Log.Warnf("save username: %v", err)
							}
						}
					} else {
						c.Log.Warn("auth.User is not *tg.User or nil pointer: ", fmt.Sprintf("%T", auth.User))
					}
				} else {
					c.Log.Warn("auth.User is nil")
				}
			}
			c.Log.Info("bot authorization imported")
		}
	}

	if c.me == nil && st != nil {
		if uid, err := st.UserID(); err == nil && uid != 0 {
			isBot, _ := st.IsBot()
			c.me = &types.User{
				ID:        uid,
				IsBot:     isBot,
				FirstName: func() string { v, _ := st.FirstName(); return v }(),
				LastName:  func() string { v, _ := st.LastName(); return v }(),
				Username:  func() string { v, _ := st.Username(); return v }(),
			}
			c.Log.Debug("user restored from storage: id=", c.me.ID, " username=", c.me.Username)
		}
	}

	if !c.cfg.NoUpdates {
		c.Log.Debug("fetching updates state")
		rpc := c.Raw()
		_, err := rpc.UpdatesGetState(context.Background())
		if err != nil {
			if rpcErr, ok := tgerr.As(err); ok && rpcErr.Code == 401 {
				c.Log.Debug("updates state fetch skipped: not authorized (", rpcErr.Type, ")")
			} else {
				c.cleanupSessions()
				return fmt.Errorf("get state: %w", err)
			}
		} else {
			c.Log.Info("updates state fetched")
		}
	}

	if err := c.startPlugins(context.Background()); err != nil {
		c.Log.Errorf("plugin start: %v", err)
	}

	if c.cfg.UpdateRecoveryEnabled && !c.cfg.NoUpdates {
		mgr := newUpdateManager(c, c.storage, updateManagerConfig{
			QueueSize:       c.cfg.UpdateQueueSize,
			DurableQueue:    c.cfg.DurableUpdateQueue,
			MaxHandlerRetry: c.cfg.MaxUpdateHandlerRetry,
		})
		if err := mgr.Start(context.Background()); err != nil {
			c.Log.Warnf("update recovery start: %v", err)
		} else {
			mgr.SetRPC(c.Raw())
			c.updateManager = mgr
			c.Log.Info("update recovery enabled")
		}
	}

	if c.cfg.HealthEnabled {
		if c.healthCheck != nil {
			c.healthCheck.Stop()
		}
		c.healthCheck = newHealthChecker(c, c.healthConfig())
		c.healthCheck.Start(context.Background())
	}

	return nil
}

func (c *Client) processRawUpdate(obj tg.TLObject) {
	updates, ok := obj.(tg.UpdatesClass)
	if !ok {
		return
	}
	if c.updateManager != nil {
		if err := c.updateManager.EnqueueLive(updates); err != nil {
			c.Log.Warnf("enqueue live update: %v", err)
		}
		return
	}
	c.HandleUpdates(updates)
}

// Disconnect closes all sessions (main and exported), releases storage, and marks the client
// as disconnected. It is safe to call Disconnect on an already-disconnected client.
// cleanupSessions stops all sessions without requiring the client to be in a
// connected state. It releases storage unless closeStorage is explicitly false
// for paths that immediately retry with the same configured storage instance.
func (c *Client) cleanupSessions(closeStorage ...bool) {
	shouldCloseStorage := true
	if len(closeStorage) > 0 {
		shouldCloseStorage = closeStorage[0]
	}

	if c.updateManager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		c.updateManager.Stop(ctx)
		cancel()
		c.updateManager = nil
	}
	if c.reconnectMgr != nil {
		c.reconnectMgr.Stop()
	}
	if c.healthCheck != nil {
		c.healthCheck.Stop()
	}

	c.sessionsMu.Lock()
	for key, sess := range c.sessions {
		if sess != nil {
			sess.Stop()
		}
		delete(c.sessions, key)
	}
	c.sessionsMu.Unlock()

	if c.dcSessions != nil {
		c.dcSessions.cleanup()
	}

	c.mu.Lock()
	sess := c.session
	c.session = nil
	c.me = nil
	c.apiInit = false
	c.mu.Unlock()

	if sess != nil {
		sess.Stop()
	}

	if shouldCloseStorage && c.storage != nil {
		c.storage.Close()
		c.mu.Lock()
		c.storage = nil
		c.mu.Unlock()
	}

	c.state.setConnected(false)
}

// UpdateHealth returns a snapshot of the update manager's health metrics.
func (c *Client) UpdateHealth() UpdateHealth {
	if c.updateManager == nil {
		return UpdateHealth{}
	}
	return c.updateManager.Health()
}

// Disconnect closes all sessions (main and exported), releases storage, and marks the client
// as disconnected. It is safe to call Disconnect on an already-disconnected client.
// Returns ErrNotConnected if the client was never connected.
func (c *Client) Disconnect() error {
	if err := c.state.requireConnected(); err != nil {
		return err
	}
	c.stopPlugins(context.Background())
	c.cleanupSessions()
	return nil
}

// Close permanently closes the client, stopping all reconnect and health-check goroutines.
// After Close, the client cannot be reconnected; create a new Client instead.
// It is safe to call Close on an already-closed client.
func (c *Client) Close() {
	c.stopPlugins(context.Background())
	c.cleanupSessions()
	c.state.SetClosed()
}

func (c *Client) Health() HealthStatus {
	return c.state.Health()
}

func (c *Client) handleMigrationError(rpcErr *tgerr.Error, query tg.TLObject) (tg.TLObject, error) {
	targetDC := rpcErr.Argument
	if targetDC <= 0 {
		return nil, &MigrationError{TargetDC: targetDC, Err: ErrMigrationUnknown}
	}

	if rpcErr.Code != 303 {
		return nil, rpcErr
	}

	c.Log.Infof("DC migration required: %s -> DC %d", rpcErr.Type, targetDC)

	c.mu.Lock()
	st := c.storage
	c.mu.Unlock()
	if st == nil {
		return nil, &MigrationError{TargetDC: targetDC, Err: ErrNotConnected}
	}

	switch rpcErr.Type {
	case "PHONE_MIGRATE", "NETWORK_MIGRATE", "USER_MIGRATE":
		return c.migrateAndRetry(targetDC, query, st)
	case "FILE_MIGRATE", "STATS_MIGRATE":
		return c.migrateExportImport(targetDC, query, st)
	default:
		return nil, &MigrationError{TargetDC: targetDC, Err: fmt.Errorf("unsupported migration type: %s", rpcErr.Type)}
	}
}

var idempotentConstructors = map[uint32]bool{
	tg.InvokeWithLayerTypeID:         true,
	tg.InitConnectionTypeID:          true,
	tg.AuthExportAuthorizationTypeID: true,
	tg.AuthImportAuthorizationTypeID: true,
}

func isIdempotent(query tg.TLObject) bool {
	if query == nil {
		return false
	}
	return idempotentConstructors[query.ConstructorID()]
}

func (c *Client) migrateAndRetry(targetDC int, query tg.TLObject, st storage.Storage) (tg.TLObject, error) {
	if !isIdempotent(query) {
		return nil, &UnsafeMigrationError{TargetDC: targetDC, Method: fmt.Sprintf("%T", query)}
	}

	c.cleanupSessions(false)

	c.mu.Lock()
	c.migratingDC = true
	c.mu.Unlock()

	if err := st.SetDCID(targetDC); err != nil {
		return nil, &MigrationError{TargetDC: targetDC, Err: fmt.Errorf("save dc_id: %w", err)}
	}

	if err := c.connectTransport(30 * time.Second); err != nil {
		return nil, &MigrationError{TargetDC: targetDC, Err: err}
	}

	retries := c.cfg.Retries
	if retries < 1 {
		retries = 1
	}
	return c.Invoke(context.Background(), query, retries, 30*time.Second)
}

func (c *Client) migrateExportImport(targetDC int, query tg.TLObject, _ storage.Storage) (tg.TLObject, error) {
	ctx := context.Background()

	rpc, err := c.dcRPC(ctx, targetDC)
	if err != nil {
		return nil, &MigrationError{TargetDC: targetDC, Err: err}
	}

	c.Log.Infof("DC migration to DC %d complete via dcRPC", targetDC)

	return rpc.Invoke(ctx, query, nil)
}

// Invoke sends a TLObject query through the primary session with the given retry count and timeout.
// The provided context is used for cancellation: when cancelled after the message has been sent,
// an RPCDropAnswerRequest is sent to the server. It wraps errors from the session with a
// "client: invoke:" prefix.
//
// Returns ErrNotConnected if the client is not connected.
func (c *Client) Invoke(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	sess := c.session
	c.mu.RUnlock()

	if sess == nil {
		return nil, ErrNotConnected
	}

	result, err := sess.Invoke(ctx, query, retries, timeout)
	if err != nil {
		var rpcErr *tgerr.Error
		if errors.As(err, &rpcErr) && rpcErr.Code == 303 {
			return c.handleMigrationError(rpcErr, query)
		}
		return nil, fmt.Errorf("client: invoke: %w", err)
	}
	return result, nil
}

// InvokeRaw sends a TLObject query through the primary session, returning the raw response
// without wrapping errors. This is useful when the caller needs to inspect the original error.
// The provided context is used for cancellation.
//
// Returns ErrNotConnected if the client is not connected.
func (c *Client) InvokeRaw(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	sess := c.session
	c.mu.RUnlock()

	if sess == nil {
		return nil, ErrNotConnected
	}

	return sess.Invoke(ctx, query, retries, timeout)
}

// InvokeWithRawResult sends a TLObject query and returns the raw MTProto
// rpc_result result:Object payload bytes. The returned bytes are not decoded
// into a Go struct and are not gzip-unpacked; if the server returned
// gzip_packed, the bytes start with the gzip_packed constructor.
func (c *Client) InvokeWithRawResult(ctx context.Context, query tg.TLObject) ([]byte, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	sess := c.session
	c.mu.RUnlock()

	if sess == nil {
		return nil, ErrNotConnected
	}

	timeout := c.cfg.ReqTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	retries := c.cfg.Retries
	if retries < 1 {
		retries = 1
	}

	return sess.InvokeRaw(ctx, query, retries, timeout)
}

// InvokeWithRawByte is deprecated. Use [Client.InvokeWithRawResult].
func (c *Client) InvokeWithRawByte(ctx context.Context, query tg.TLObject) ([]byte, error) {
	return c.InvokeWithRawResult(ctx, query)
}

// HandleUpdates processes an incoming Telegram UpdatesClass by flattening it
// into individual updates and dispatching them to registered handlers.
//
// Example:
//
//	client.HandleUpdates(updates)
//	// updates is a tg.UpdatesClass received from the server,
//	// e.g. *tg.UpdatesTL containing a batch of UpdateClass items.
func (c *Client) HandleUpdates(updates tg.UpdatesClass) {
	if c.cfg.NoUpdates || !c.IsConnected() {
		return
	}

	c.mu.RLock()
	disp := c.dispatcher
	hdisp := c.handlerDispatcher
	c.mu.RUnlock()

	parsedUsers, parsedChats, rawUpdates := c.flattenUpdates(updates)
	c.cachePeersFromUpdates(parsedUsers, parsedChats)
	userMap := buildUserMap(parsedUsers)
	chatMap := buildChatMap(parsedChats)
	pm := types.NewPeerMapFromClasses(parsedUsers, parsedChats)

	for _, rawUpd := range rawUpdates {
		upd := c.toUpdate(rawUpd, userMap, chatMap, pm)

		if disp != nil {
			pkt := UpdatePacket{
				Update: updates,
				Users:  userMap,
				Chats:  chatMap,
			}
			if err := disp.Enqueue(pkt); err != nil {
				c.Log.Warnf("enqueue update: %v", err)
			}
		}

		if hdisp != nil {
			c.dispatchUpdate(hdisp, upd)
		}

		upd.reset()
		updatePool.Put(upd)
	}

	if len(rawUpdates) == 0 {
		upd := updatePool.Get().(*Update)
		upd.Raw = updates
		upd.Users = userMap
		upd.Chats = chatMap
		if disp != nil {
			if err := disp.Enqueue(UpdatePacket{Update: updates, Users: userMap, Chats: chatMap}); err != nil {
				c.Log.Warnf("enqueue update: %v", err)
			}
		}
		if hdisp != nil {
			c.dispatchUpdate(hdisp, upd)
		}
		upd.reset()
		updatePool.Put(upd)
	}
}

func (c *Client) flattenUpdates(updates tg.UpdatesClass) ([]tg.UserClass, []tg.ChatClass, []tg.UpdateClass) {
	switch v := updates.(type) {
	case *tg.Updates:
		return v.Users, v.Chats, v.Updates
	case *tg.UpdatesCombined:
		return v.Users, v.Chats, v.Updates
	case *tg.UpdateShort:
		return nil, nil, []tg.UpdateClass{v.Update}
	case *tg.UpdateShortMessage:
		var fromID, peerID tg.PeerClass
		if v.Out {
			if c.me != nil {
				fromID = &tg.PeerUser{UserID: c.me.ID}
			}
			peerID = &tg.PeerUser{UserID: v.UserID}
		} else {
			fromID = &tg.PeerUser{UserID: v.UserID}
			peerID = &tg.PeerUser{UserID: v.UserID}
		}
		msg := &tg.Message{
			ID:        v.ID,
			Message:   v.Message,
			Date:      v.Date,
			Out:       v.Out,
			Mentioned: v.Mentioned,
			Silent:    v.Silent,
			FromID:    fromID,
			PeerID:    peerID,
		}
		if v.ReplyTo != nil {
			msg.ReplyTo = v.ReplyTo
		}
		if v.Entities != nil {
			msg.Entities = v.Entities
		}
		if v.FwdFrom != nil {
			msg.FwdFrom = v.FwdFrom
		}
		if v.ViaBotID != 0 {
			msg.ViaBotID = v.ViaBotID
		}
		upd := &tg.UpdateNewMessage{
			Message:  msg,
			PTS:      v.PTS,
			PTSCount: v.PTSCount,
		}
		return nil, nil, []tg.UpdateClass{upd}
	case *tg.UpdateShortChatMessage:
		msg := &tg.Message{
			ID:        v.ID,
			Message:   v.Message,
			Date:      v.Date,
			Out:       v.Out,
			Mentioned: v.Mentioned,
			Silent:    v.Silent,
			FromID:    &tg.PeerUser{UserID: v.FromID},
			PeerID:    &tg.PeerChat{ChatID: v.ChatID},
		}
		if v.ReplyTo != nil {
			msg.ReplyTo = v.ReplyTo
		}
		if v.Entities != nil {
			msg.Entities = v.Entities
		}
		if v.FwdFrom != nil {
			msg.FwdFrom = v.FwdFrom
		}
		if v.ViaBotID != 0 {
			msg.ViaBotID = v.ViaBotID
		}
		upd := &tg.UpdateNewMessage{
			Message:  msg,
			PTS:      v.PTS,
			PTSCount: v.PTSCount,
		}
		return nil, nil, []tg.UpdateClass{upd}
	default:
		return nil, nil, nil
	}
}

func (c *Client) toUpdate(raw tg.UpdateClass, users map[int64]*types.User, chats map[int64]*types.Chat, pm *types.PeerMap) *Update {
	upd := updatePool.Get().(*Update)
	upd.Users = users
	upd.Chats = chats
	upd.Raw = raw

	switch v := raw.(type) {
	case *tg.UpdateNewMessage:
		upd.Message = types.ParseMessage(v.Message, pm)
		bindMessage(upd.Message, c)
		c.resolveMessagePeers(upd.Message, users, chats)
	case *tg.UpdateNewChannelMessage:
		upd.Message = types.ParseMessage(v.Message, pm)
		bindMessage(upd.Message, c)
		c.resolveMessagePeers(upd.Message, users, chats)
	case *tg.UpdateEditMessage:
		upd.EditedMessage = types.ParseMessage(v.Message, pm)
		bindMessage(upd.EditedMessage, c)
		c.resolveMessagePeers(upd.EditedMessage, users, chats)
	case *tg.UpdateEditChannelMessage:
		upd.EditedMessage = types.ParseMessage(v.Message, pm)
		bindMessage(upd.EditedMessage, c)
		c.resolveMessagePeers(upd.EditedMessage, users, chats)
	case *tg.UpdateDeleteMessages:
		upd.DeletedMessages = &types.DeletedMessages{Messages: v.Messages}
	case *tg.UpdateDeleteChannelMessages:
		upd.DeletedMessages = &types.DeletedMessages{Messages: v.Messages, ChatID: v.ChannelID}
	case *tg.UpdateBotCallbackQuery:
		upd.CallbackQuery = types.ParseCallbackQuery(v)
		if upd.CallbackQuery != nil {
			upd.CallbackQuery.SetBinder(c)
		}
	case *tg.UpdateBotInlineQuery:
		upd.InlineQuery = &types.InlineQuery{ID: v.QueryID, UserID: v.UserID, Query: v.Query, Offset: v.Offset}
	case *tg.UpdateUserStatus:
		upd.UserStatus = &types.UserStatusUpdated{UserID: v.UserID}
	case *tg.UpdateUserName:
		if len(v.Usernames) > 0 {
			c.cacheUsernameLocked(v.Usernames[0].Username, v.UserID)
			if ps, ok := c.storage.(storage.PeerStore); ok {
				_ = ps.SavePeer(&storage.Peer{
					ID:        v.UserID,
					Type:      storage.PeerTypeUser,
					Username:  v.Usernames[0].Username,
					FirstName: v.FirstName,
					LastName:  v.LastName,
				})
			}
		}
	case *tg.UpdateChatParticipant:
		upd.ChatMember = &types.ChatMemberUpdated{}
	case *tg.UpdateChannelParticipant:
		upd.ChatMember = &types.ChatMemberUpdated{}
	case *tg.UpdateMessageReactions:
		upd.MessageReaction = &types.MessageReactions{}
	case *tg.UpdateMessagePoll:
		upd.Poll = &types.PollUpdate{PollID: v.PollID}
	case *tg.UpdateBotPrecheckoutQuery:
		upd.PreCheckoutQuery = &types.PreCheckoutQuery{
			ID:          v.QueryID,
			UserID:      v.UserID,
			Currency:    v.Currency,
			TotalAmount: v.TotalAmount,
		}
		if v.ShippingOptionID != "" {
			upd.PreCheckoutQuery.ShippingOptionID = v.ShippingOptionID
		}
		if v.Info != nil {
			info := &types.OrderInfo{}
			if v.Info.Name != "" {
				info.Name = v.Info.Name
			}
			if v.Info.Phone != "" {
				info.Phone = v.Info.Phone
			}
			if v.Info.Email != "" {
				info.Email = v.Info.Email
			}
			if v.Info.ShippingAddress != nil {
				info.ShippingAddress = &types.ShippingAddress{
					CountryCode: v.Info.ShippingAddress.CountryIso2,
					State:       v.Info.ShippingAddress.State,
					City:        v.Info.ShippingAddress.City,
					StreetLine1: v.Info.ShippingAddress.StreetLine1,
					StreetLine2: v.Info.ShippingAddress.StreetLine2,
					PostCode:    v.Info.ShippingAddress.PostCode,
				}
			}
			upd.PreCheckoutQuery.OrderInfo = info
		}
		upd.PreCheckoutQuery.SetBinder(c)
	case *tg.UpdateBotShippingQuery:
		upd.ShippingQuery = &types.ShippingQuery{
			ID:     v.QueryID,
			UserID: v.UserID,
		}
		if v.ShippingAddress != nil {
			upd.ShippingQuery.Address = &types.ShippingAddress{
				CountryCode: v.ShippingAddress.CountryIso2,
				State:       v.ShippingAddress.State,
				City:        v.ShippingAddress.City,
				StreetLine1: v.ShippingAddress.StreetLine1,
				StreetLine2: v.ShippingAddress.StreetLine2,
				PostCode:    v.ShippingAddress.PostCode,
			}
		}
		upd.ShippingQuery.SetBinder(c)
	case *tg.UpdateBotChatInviteRequester:
		upd.ChatJoinRequest = types.ParseChatJoinRequest(v, users, chats)
		if upd.ChatJoinRequest != nil {
			upd.ChatJoinRequest.SetBinder(c)
		}
	case *tg.UpdateStory:
		story := types.ParseStory(v.Story, pm)
		if story != nil {
			if p, ok := v.Peer.(*tg.PeerUser); ok {
				story.FromID = p.UserID
			}
			story.SetBinder(c)
		}
		upd.Story = story
	}

	return upd
}

func bindMessage(msg *types.Message, binder types.Binder) {
	if msg != nil {
		msg.SetBinder(binder)
	}
}

func (c *Client) resolveMessagePeers(msg *types.Message, users map[int64]*types.User, chats map[int64]*types.Chat) {
	if msg == nil {
		return
	}
	if msg.Sender == nil && msg.FromID > 0 {
		if u, ok := users[msg.FromID]; ok {
			u.SetBinder(c)
			msg.Sender = u
		}
	} else if msg.Sender != nil {
		msg.Sender.SetBinder(c)
	}
	if msg.Chat == nil && msg.ChatID != 0 {
		if ch, ok := chats[msg.ChatID]; ok {
			msg.Chat = ch
		}
	}
}

func buildUserMap(users []tg.UserClass) map[int64]*types.User {
	m := make(map[int64]*types.User, len(users))
	for _, u := range users {
		parsed := types.ParseUser(u)
		if parsed != nil {
			m[parsed.ID] = parsed
		}
	}
	return m
}

func buildChatMap(chats []tg.ChatClass) map[int64]*types.Chat {
	m := make(map[int64]*types.Chat, len(chats))
	for _, ch := range chats {
		parsed := types.ParseChatFromChat(ch)
		if parsed != nil {
			m[parsed.ID] = parsed
		}
	}
	return m
}

// ResolvePeer resolves a ChatRef (ID, username, or InputPeer) into an InputPeerClass
// suitable for use in API calls.
//
// Returns ErrNotConnected if the client is not connected, or ErrPeerNotFound if the
// peer cannot be resolved.
//
// Example:
//
//		ctx := context.Background()
//		peer, err := client.ResolvePeer(ctx, "@durov")
//		if err != nil {
//		    log.Fatal(err)
//	}
//
//	fmt.Println(peer)
func (c *Client) ResolvePeer(ctx context.Context, peerID interface{}) (tg.InputPeerClass, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	switch p := peerID.(type) {
	case tg.InputPeerClass:
		return p, nil
	case int64:
		return ChatID(p).resolve(ctx, c)
	case int:
		return ChatID(int64(p)).resolve(ctx, c)
	case string:
		return ChatRefFrom(p).resolve(ctx, c)
	case ChatRef:
		return p.resolve(ctx, c)
	default:
		return nil, fmt.Errorf("%w: unsupported peer type %T", ErrPeerNotFound, peerID)
	}
}

// GetSession returns or creates a session for the specified data center. If isMedia or isCDN
// is false and the requested dcID matches the main session's DC, the main session is returned.
//
// Sessions are cached by (dcID, isMedia) key; subsequent calls with the same parameters
// return the cached session.
//
// Returns ErrNotConnected if the client is not connected, or an error if the dcID is unknown
// or session creation fails.
func (c *Client) GetSession(ctx context.Context, dcID int, isMedia bool, isCDN bool) (*session.Session, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	if !isMedia && !isCDN {
		c.mu.RLock()
		st := c.storage
		mainSess := c.session
		c.mu.RUnlock()

		if st != nil {
			storedDC, err := st.DCID()
			if err == nil && storedDC == dcID && mainSess != nil {
				return mainSess, nil
			}
		}
	}

	key := sessionKey{dcID: dcID, isMedia: isMedia}

	c.sessionsMu.Lock()
	if sess, ok := c.sessions[key]; ok {
		c.sessionsMu.Unlock()
		return sess, nil
	}
	c.sessionsMu.Unlock()

	addr := ResolveDCAddress(dcID, c.cfg.TestMode)
	if addr == "" {
		return nil, fmt.Errorf("unknown dc_id: %d", dcID)
	}
	port := DefaultDCPort(c.cfg.TestMode)

	var sess *session.Session
	var err error

	if c.testSessionF != nil {
		sess, err = c.testSessionF(ctx, dcID, addr, port, nil)
	} else {
		dc := session.DataCenter{
			ID:       dcID,
			TestMode: c.cfg.TestMode,
			IPv6:     c.cfg.IPv6,
		}
		c.mu.RLock()
		st := c.storage
		c.mu.RUnlock()
		if st == nil {
			st = NewMemoryStorage()
		}
		sess, err = session.NewSession(dc, st, c.cfg.DeviceModel, c.cfg.AppVersion, c.cfg.SystemLangCode, c.cfg.LangCode)
	}
	if err != nil {
		return nil, fmt.Errorf("create session for dc %d: %w", dcID, err)
	}

	c.sessionsMu.Lock()
	if existing, ok := c.sessions[key]; ok {
		c.sessionsMu.Unlock()
		sess.Stop()
		return existing, nil
	}
	c.sessions[key] = sess
	c.sessionsMu.Unlock()

	return sess, nil
}

// ExportSessionString exports the current session as an encoded string that can be stored
// and later passed to WithSessionString to resume the session.
//
// Returns ErrNotConnected if the client has no active storage.
func (c *Client) ExportSessionString() (string, error) {
	c.mu.RLock()
	st := c.storage
	c.mu.RUnlock()
	if st == nil {
		return "", ErrNotConnected
	}
	return st.ExportSessionString()
}

// LogOut disconnects the client. It does not revoke the session on the server;
// for full server-side logout use the auth.SignOut method.
//
// Returns ErrNotConnected if the client is not connected.
func (c *Client) LogOut() error {
	if !c.IsConnected() {
		return ErrNotConnected
	}
	return c.Disconnect()
}

// SetMe sets the authenticated user on the client. This is typically called internally
// after a successful sign-in.
func (c *Client) SetMe(user *types.User) {
	c.mu.Lock()
	c.me = user
	c.mu.Unlock()
}

func (c *Client) saveMeToStorage(user *types.User) {
	if user == nil {
		return
	}
	c.mu.RLock()
	st := c.storage
	c.mu.RUnlock()
	if st == nil {
		return
	}
	_ = st.SetUserID(user.ID)
	_ = st.SetIsBot(user.IsBot)
	_ = st.SetFirstName(user.FirstName)
	_ = st.SetLastName(user.LastName)
	_ = st.SetUsername(user.Username)
}

// ServerTime returns the current estimated server time adjusted by the configured timezone offset.
func (c *Client) ServerTime() int32 {
	return ServerTime(c.cfg.Device.TZOffset)
}

// APIID returns the Telegram API ID configured for this client.
func (c *Client) APIID() int32 { return c.cfg.APIID }

// APIHash returns the Telegram API hash configured for this client.
func (c *Client) APIHash() string { return c.cfg.APIHash }

// DC returns the configured preferred data center ID, or zero when automatic.
func (c *Client) DC() int { return c.cfg.DC }

// ServerAddr returns the manually configured server address, or empty for auto-resolution.
func (c *Client) ServerAddr() string { return c.cfg.ServerAddr }

// LocalAddr returns the local address binding for outbound connections, or empty for default.
func (c *Client) LocalAddr() string { return c.cfg.LocalAddr }

// SessionName returns the session name used for storage file naming.
func (c *Client) SessionName() string { return c.cfg.SessionName }

// BotToken returns the bot token if one was configured, or an empty string for user accounts.
func (c *Client) BotToken() string { return c.cfg.BotToken }

// TestMode reports whether the client is configured to connect to Telegram's test DC.
func (c *Client) TestMode() bool { return c.cfg.TestMode }

// AutoConnect reports whether the client will automatically connect before the first
// operation that requires an active connection.
func (c *Client) AutoConnect() bool { return c.cfg.AutoConnect }

// IPv6 reports whether IPv6 connections are preferred.
func (c *Client) IPv6() bool { return c.cfg.IPv6 }

// NoUpdates reports whether update processing is disabled.
func (c *Client) NoUpdates() bool { return c.cfg.NoUpdates }

// ParseMode returns the default message parsing mode.
func (c *Client) ParseMode() ParseMode { return c.cfg.ParseMode }

// SleepThreshold returns the flood-wait threshold; requests with shorter waits are automatically retried.
func (c *Client) SleepThreshold() time.Duration { return c.cfg.SleepThreshold }

// Timeout returns the TCP connection timeout used when dialing Telegram servers.
func (c *Client) Timeout() time.Duration { return c.cfg.Timeout }

// ReqTimeout returns the default RPC request timeout applied when no context deadline is set.
func (c *Client) ReqTimeout() time.Duration { return c.cfg.ReqTimeout }

// MaxConcurrentTransmissions returns the maximum number of concurrent RPC transmissions allowed.
func (c *Client) MaxConcurrentTransmissions() int { return c.cfg.MaxConcurrentTrans }

// MaxMessageCacheSize returns the maximum number of messages retained in the message cache.
func (c *Client) MaxMessageCacheSize() int { return c.cfg.MaxMessageCacheSize }

// MaxTopicCacheSize returns the maximum number of forum topics retained in the topic cache.
func (c *Client) MaxTopicCacheSize() int { return c.cfg.MaxTopicCacheSize }

// LinkPreviewOptions returns the global link preview defaults, or nil if none are set.
func (c *Client) LinkPreviewOptions() *types.LinkPreviewOptions { return c.cfg.LinkPreviewOptions }

// Takeout reports whether the client is configured to use a takeout session for data export.
func (c *Client) Takeout() bool { return c.cfg.Takeout }

// IsBot reports whether the client is authenticated as a bot. It checks the stored
// session state first, falling back to whether a BotToken was configured.
func (c *Client) IsBot() bool {
	if c.storage == nil {
		return c.cfg.BotToken != ""
	}
	isBot, _ := c.storage.IsBot()
	return isBot
}

// SetBotToken updates the bot token in the client configuration.
func (c *Client) SetBotToken(token string) { c.cfg.BotToken = token }

// ResolvePeerCache looks up a previously cached InputPeer by its numeric ID.
// Returns the cached peer or ErrPeerNotFound if not present.
func (c *Client) ResolvePeerCache(id int64) (tg.InputPeerClass, error) {
	c.peerCacheMu.RLock()
	defer c.peerCacheMu.RUnlock()
	if p, ok := c.peerCache[id]; ok {
		return p, nil
	}
	return nil, ErrPeerNotFound
}

// CachePeer stores an InputPeer in the client's peer cache keyed by numeric ID.
// Cached peers are used by ResolvePeerCache to avoid redundant RPC calls.
// When PeerCacheSize is set and the cache exceeds the limit, the oldest entry
// is evicted (FIFO) to prevent unbounded growth.
func (c *Client) CachePeer(id int64, peer tg.InputPeerClass) {
	c.peerCacheMu.Lock()
	if _, exists := c.peerCache[id]; !exists {
		c.peerCacheOrder = append(c.peerCacheOrder, id)
	}
	c.peerCache[id] = peer
	c.evictOldestPeerLocked()
	c.peerCacheMu.Unlock()
}

func (c *Client) evictOldestPeerLocked() {
	limit := c.cfg.PeerCacheSize
	if limit <= 0 || len(c.peerCache) <= limit {
		return
	}
	for len(c.peerCache) > limit && len(c.peerCacheOrder) > 0 {
		oldest := c.peerCacheOrder[0]
		c.peerCacheOrder = c.peerCacheOrder[1:]
		delete(c.peerCache, oldest)
	}
}

func (c *Client) cacheUsernameLocked(username string, userID int64) {
	if _, exists := c.usernameCache[username]; !exists {
		c.usernameCacheOrder = append(c.usernameCacheOrder, username)
	}
	c.usernameCache[username] = userID
	limit := c.cfg.PeerCacheSize
	if limit <= 0 || len(c.usernameCache) <= limit {
		return
	}
	for len(c.usernameCache) > limit && len(c.usernameCacheOrder) > 0 {
		oldest := c.usernameCacheOrder[0]
		c.usernameCacheOrder = c.usernameCacheOrder[1:]
		delete(c.usernameCache, oldest)
	}
}

func (c *Client) clientPeerResolver() PeerResolver {
	if c.testResolver != nil {
		return c.testResolver
	}
	return c
}

func (c *Client) cachePeersFromUpdates(users []tg.UserClass, chats []tg.ChatClass) {
	var entries []*storage.Peer
	for _, u := range users {
		user, ok := u.(*tg.User)
		if !ok || user.AccessHash == 0 {
			continue
		}
		c.CachePeer(user.ID, &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash})
		username := user.Username
		if username != "" {
			c.cacheUsernameLocked(username, user.ID)
		}
		entries = append(entries, &storage.Peer{
			ID:         user.ID,
			Type:       storage.PeerTypeUser,
			AccessHash: user.AccessHash,
			Username:   username,
			FirstName:  user.FirstName,
			LastName:   user.LastName,
		})
	}
	for _, ch := range chats {
		switch v := ch.(type) {
		case *tg.Chat:
			c.CachePeer(v.ID, &tg.InputPeerChat{ChatID: v.ID})
			entries = append(entries, &storage.Peer{
				ID:   v.ID,
				Type: storage.PeerTypeChat,
			})
		case *tg.Channel:
			accessHash := v.AccessHash
			c.CachePeer(v.ID, &tg.InputPeerChannel{ChannelID: v.ID, AccessHash: accessHash})
			username := v.Username
			if username != "" {
				c.cacheUsernameLocked(username, v.ID)
			}
			entries = append(entries, &storage.Peer{
				ID:         v.ID,
				Type:       storage.PeerTypeChannel,
				AccessHash: accessHash,
				Username:   username,
			})
		}
	}
	if c.cfg.SavePeers && len(entries) > 0 {
		if ps, ok := c.storage.(storage.PeerStore); ok {
			for _, entry := range entries {
				_ = ps.SavePeer(entry)
			}
		}
	}
}

func (c *Client) loadPeersFromStorage() {
	if !c.cfg.SavePeers {
		return
	}
	ps, ok := c.storage.(storage.PeerStore)
	if !ok {
		return
	}
	peers, err := ps.LoadPeers()
	if err != nil {
		return
	}
	for _, p := range peers {
		var peer tg.InputPeerClass
		switch p.Type {
		case storage.PeerTypeUser:
			peer = &tg.InputPeerUser{UserID: p.ID, AccessHash: p.AccessHash}
		case storage.PeerTypeChat:
			peer = &tg.InputPeerChat{ChatID: p.ID}
		case storage.PeerTypeChannel:
			peer = &tg.InputPeerChannel{ChannelID: p.ID, AccessHash: p.AccessHash}
		default:
			continue
		}
		c.CachePeer(p.ID, peer)
		if p.Username != "" {
			c.cacheUsernameLocked(p.Username, p.ID)
		}
	}
}
