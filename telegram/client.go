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
	"math/rand/v2"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
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
	New: func() any { return &Update{} },
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
	cfgMu   sync.RWMutex
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
	hooksMu             sync.RWMutex
	updateReceivedHooks []UpdateReceivedHook
	sessionLoadedHooks  []SessionLoadedHook
	connectedHooks      []ConnectedHook
	reconnectHooks      []ReconnectHook

	peerCache          map[int64]tg.InputPeerClass
	peerCacheMu        sync.RWMutex
	usernameCache      map[string]int64
	peerCacheOrder     []int64
	usernameCacheOrder []string
	resolveCoalescer   resolveCoalescer

	stopCh   chan struct{}
	stopOnce sync.Once

	reconnectMgr  *reconnectManager
	updateManager *updateManager

	autoConnectMu sync.Mutex

	sessionWg sync.WaitGroup

	secretChats           *SecretChatManager
	secretMsgHandlers     []SecretMessageHandler
	secretChatReqHandlers []SecretChatRequestHandler

	dcSessions *dcSessions

	// uploadPool is a lazily-created pool of sessions for upload traffic
	// isolation. Each session has its own TCP connection but shares the main
	// session's auth key, so upload.saveBigFilePart traffic doesn't compete
	// with API calls and updates on the main session.
	uploadPool      []*sideSession
	uploadSessionMu sync.Mutex

	// dcOptionPool manages candidate endpoints per DC with health scoring.
	// Ported from td/td/telegram/net/DcOptionsSet.h.
	dcOptionPool *session.DCOptionPool
	// connPool caches warm connections to avoid redundant TCP handshakes.
	// Ported from td/td/telegram/net/ConnectionCreator.cpp.
	connPool *session.ConnectionPool
	// sessionRouter routes queries to the appropriate session slot per DC.
	// Ported from td/td/telegram/net/NetQueryDispatcher.h.
	sessionRouter *session.SessionRouter
	// dcAuthManager tracks exported authorization state for non-main DCs.
	// Ported from td/td/telegram/net/DcAuthManager.h.
	dcAuthManager *session.DcAuthManager

	// keySet is the trusted server RSA key store for DH fingerprint verification
	// and rotation. When nil, the bundled static keys are used (backward compat).
	// Constructed when Config.RSAKeyRotationInterval > 0.
	keySet *crypto.RSAKeySet
	// keyWatchdog periodically refreshes server RSA keys when non-nil.
	keyWatchdog       *crypto.PublicRsaKeyWatchdog
	keyWatchdogCancel context.CancelFunc
	// overloadController gates RPC admission by priority when non-nil.
	// Constructed when Config.MaxInFlightRPCs > 0.
	overloadController *OverloadController

	testStorage  storage.Storage
	testSession  *session.Session
	testSessionF func(ctx context.Context, dcID int, addr string, port int, authKey []byte) (*session.Session, error)
	testInvoker  tg.Invoker
	testDialer   transport.Dialer
	testResolver PeerResolver

	// rng is a per-client random source, avoiding contention on the global
	// math/rand mutex under high concurrency.
	rng   *rand.Rand
	rngMu sync.Mutex

	// Booleans grouped at end to minimize padding on 64-bit.
	apiInit     bool
	mwSorted    bool
	migratingDC atomic.Bool
}

func (c *Client) config() Config {
	c.cfgMu.RLock()
	cfg := c.cfg
	c.cfgMu.RUnlock()
	return cfg
}

func (c *Client) updateConfig(fn func(cfg *Config)) {
	c.cfgMu.Lock()
	fn(&c.cfg)
	c.cfgMu.Unlock()
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
	hasSessionString := cfg != nil && cfg.SessionString != ""
	if apiID == 0 && !hasSessionString {
		return nil, ErrAPIIDRequired
	}
	if apiHash == "" && !hasSessionString {
		return nil, ErrAPIHashRequired
	}

	c := DefaultConfig
	c.APIID = apiID
	c.APIHash = apiHash
	if cfg != nil {
		c.mergeConfig(cfg)
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
		resolveCoalescer:  resolveCoalescer{inFlight: make(map[string][]chan resolveResult)},
		handlerDispatcher: NewHandlerDispatcher(),
		dcSessions:        newDCSessions(),
		dcOptionPool:      session.NewDCOptionPool(2, c.EndpointCoolDown),
		connPool:          session.NewConnectionPool(c.ConnPoolTTL),
		sessionRouter:     session.NewSessionRouter(5 * time.Minute),
		Log:               logger,
		rng:               rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())^0x9E3779B97F4A7C15)),
	}

	client.dcAuthManager = session.NewDcAuthManager(2, func(ctx context.Context, fromDC, toDC int) (*tg.AuthExportedAuthorization, error) {
		return client.Raw().AuthExportAuthorization(ctx, &tg.AuthExportAuthorizationRequest{
			DCID: int32(toDC),
		})
	}, nil, logger)

	client.initSecretChats()
	client.reconnectMgr = newReconnectManager(client, client.backoffConfig())

	// Production hardening: enable RSA key rotation watchdog when configured.
	if c.RSAKeyRotationInterval > 0 {
		client.keySet = crypto.NewRSAKeySet()
		client.keyWatchdog = crypto.NewPublicRsaKeyWatchdog(crypto.WatchdogConfig{
			KeySet:   client.keySet,
			Interval: c.RSAKeyRotationInterval,
			FetchFn:  nil, // No key-distribution RPC wired yet; watchdog ticks harmlessly.
			Log: func(format string, args ...any) {
				logger.Debugf("rsa watchdog: "+format, args...)
			},
		})
	}

	// Production hardening: enable overload control when configured.
	if c.MaxInFlightRPCs > 0 {
		client.overloadController = NewOverloadController(OverloadConfig{
			Ceilings: ResourceCeilings{
				MaxInFlightRPCs: c.MaxInFlightRPCs,
			},
			AdmissionDeadline: c.AdmissionDeadline,
			Log:               logger,
		})
	}


	client.initDeviceStorage()
	registerClient(client)

	return client, nil
}

func (c *Client) backoffConfig() backoffConfig {
	cfg := defaultBackoffConfig
	if c.config().ReconnectBaseDelay != 0 {
		cfg.BaseDelay = c.config().ReconnectBaseDelay
	}
	if c.config().ReconnectMaxDelay != 0 {
		cfg.MaxDelay = c.config().ReconnectMaxDelay
	}
	if c.config().ReconnectMaxAttempts != 0 {
		cfg.MaxAttempts = c.config().ReconnectMaxAttempts
	}
	return cfg
}

// mergeConfig copies non-zero fields from src into c. Deprecated top-level
// device fields override Device.* for backwards compatibility.
func (c *Config) mergeConfig(src *Config) {
	if src.SessionName != "" {
		c.SessionName = src.SessionName
	}
	if src.BotToken != "" {
		c.BotToken = src.BotToken
	}
	if src.PhoneNumber != "" {
		c.PhoneNumber = src.PhoneNumber
	}
	if src.PhoneCode != "" {
		c.PhoneCode = src.PhoneCode
	}
	if src.Password != "" {
		c.Password = src.Password
	}
	if src.CodeFunc != nil {
		c.CodeFunc = src.CodeFunc
	}
	if src.PasswordFunc != nil {
		c.PasswordFunc = src.PasswordFunc
	}
	if src.WorkDir != "" {
		c.WorkDir = src.WorkDir
	}
	if src.InMemory {
		c.InMemory = true
	}
	if src.Proxy != nil {
		c.Proxy = src.Proxy
	}
	if src.TestMode {
		c.TestMode = true
	}
	if src.IPv6 {
		c.IPv6 = true
	}
	if src.NoUpdates {
		c.NoUpdates = true
	}
	if src.AutoConnect {
		c.AutoConnect = true
	}
	if src.SkipUpdates {
		c.SkipUpdates = true
	}
	if src.SleepThreshold != 0 {
		c.SleepThreshold = src.SleepThreshold
	}
	if src.HandlerTimeout != 0 {
		c.HandlerTimeout = src.HandlerTimeout
	}
	if src.Timeout != 0 {
		c.Timeout = src.Timeout
	}
	if src.ReqTimeout != 0 {
		c.ReqTimeout = src.ReqTimeout
	}
	if src.Retries != 0 {
		c.Retries = src.Retries
	}
	if src.MaxConcurrentTrans != 0 {
		c.MaxConcurrentTrans = src.MaxConcurrentTrans
	}
	if src.DispatchWorkers != 0 {
		c.DispatchWorkers = src.DispatchWorkers
	}
	if src.DispatchQueueSize != 0 {
		c.DispatchQueueSize = src.DispatchQueueSize
	}
	if src.MaxMessageCacheSize != 0 {
		c.MaxMessageCacheSize = src.MaxMessageCacheSize
	}
	if src.MaxTopicCacheSize != 0 {
		c.MaxTopicCacheSize = src.MaxTopicCacheSize
	}
	if src.ParseMode != "" {
		c.ParseMode = src.ParseMode
	}
	if src.HidePassword {
		c.HidePassword = true
	}
	if src.Takeout {
		c.Takeout = true
	}
	if src.LinkPreviewOptions != nil {
		c.LinkPreviewOptions = src.LinkPreviewOptions
	}
	if src.FetchReplies {
		c.FetchReplies = true
	}
	if src.FetchTopics {
		c.FetchTopics = true
	}
	if src.FetchStories {
		c.FetchStories = true
	}
	if src.FetchStickers {
		c.FetchStickers = true
	}
	if src.Device.DeviceModel != "" || src.Device.AppVersion != "" {
		if src.Device.DeviceModel != "" {
			c.Device.DeviceModel = src.Device.DeviceModel
		}
		if src.Device.SystemVersion != "" {
			c.Device.SystemVersion = src.Device.SystemVersion
		}
		if src.Device.AppVersion != "" {
			c.Device.AppVersion = src.Device.AppVersion
		}
		if src.Device.LangCode != "" {
			c.Device.LangCode = src.Device.LangCode
		}
		if src.Device.LangPack != "" {
			c.Device.LangPack = src.Device.LangPack
		}
		if src.Device.SystemLangCode != "" {
			c.Device.SystemLangCode = src.Device.SystemLangCode
		}
		if src.Device.TZOffset != 0 {
			c.Device.TZOffset = src.Device.TZOffset
		}
		if src.Device.ClientPlatform != "" {
			c.Device.ClientPlatform = src.Device.ClientPlatform
		}
	}
	// Deprecated top-level fields override Device for backwards compat.
	if src.AppVersion != "" {
		c.Device.AppVersion = src.AppVersion
	}
	if src.DeviceModel != "" {
		c.Device.DeviceModel = src.DeviceModel
	}
	if src.SystemVersion != "" {
		c.Device.SystemVersion = src.SystemVersion
	}
	if src.LangCode != "" {
		c.Device.LangCode = src.LangCode
	}
	if src.LangPack != "" {
		c.Device.LangPack = src.LangPack
	}
	if src.SystemLangCode != "" {
		c.Device.SystemLangCode = src.SystemLangCode
	}
	if src.TZOffset != 0 {
		c.Device.TZOffset = src.TZOffset
	}
	if src.ClientPlatform != "" {
		c.Device.ClientPlatform = src.ClientPlatform
	}
	if src.TransportMode != "" {
		c.TransportMode = src.TransportMode
	}
	if src.Storage != nil {
		c.Storage = src.Storage
	}
	if src.SessionString != "" {
		c.SessionString = src.SessionString
	}
	if src.MTProxy != nil {
		c.MTProxy = src.MTProxy
	}
	if src.SavePeers {
		c.SavePeers = true
	}
	if src.WebSocket {
		c.WebSocket = true
	}
	if src.WebSocketTLS {
		c.WebSocketTLS = true
	}
	if src.WSDialer != nil {
		c.WSDialer = src.WSDialer
	}
	if src.DC != 0 {
		c.DC = src.DC
	}
	if src.ServerAddr != "" {
		c.ServerAddr = src.ServerAddr
	}
	if src.LocalAddr != "" {
		c.LocalAddr = src.LocalAddr
	}
	c.ReconnectEnabled = src.ReconnectEnabled
	if src.ReconnectBaseDelay != 0 {
		c.ReconnectBaseDelay = src.ReconnectBaseDelay
	}
	if src.ReconnectMaxDelay != 0 {
		c.ReconnectMaxDelay = src.ReconnectMaxDelay
	}
	if src.ReconnectMaxAttempts != 0 {
		c.ReconnectMaxAttempts = src.ReconnectMaxAttempts
	}
	c.HealthEnabled = src.HealthEnabled
	if src.HealthPingInterval != 0 {
		c.HealthPingInterval = src.HealthPingInterval
	}
	if src.HealthPongTimeout != 0 {
		c.HealthPongTimeout = src.HealthPongTimeout
	}
	if src.UpdateQueueSize != 0 {
		c.UpdateQueueSize = src.UpdateQueueSize
	}
	if src.DurableUpdateQueue {
		c.DurableUpdateQueue = true
	}
	if src.MaxUpdateHandlerRetry != 0 {
		c.MaxUpdateHandlerRetry = src.MaxUpdateHandlerRetry
	}
	c.UpdateRecoveryEnabled = src.UpdateRecoveryEnabled
	c.Log = src.Log
	if src.AlwaysObfuscate {
		c.AlwaysObfuscate = true
	}
	if src.MaxChannelDiffConcurrency != 0 {
		c.MaxChannelDiffConcurrency = src.MaxChannelDiffConcurrency
	}
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
		return rand.Int64()
	}
	c.rngMu.Lock()
	id := c.rng.Int64()
	c.rngMu.Unlock()
	return id
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
	if ps := c.peerStore(); ps != nil {
		return ps.SavePeer(p)
	}
	return nil
}

// GetPeer retrieves a cached peer by ID from the storage backend.
func (c *Client) GetPeer(id int64) (*storage.Peer, error) {
	if ps := c.peerStore(); ps != nil {
		return ps.GetPeer(id)
	}
	return nil, nil
}

// GetPeerByUsername retrieves a cached peer by username from the storage backend.
func (c *Client) GetPeerByUsername(username string) (*storage.Peer, error) {
	if ps := c.peerStore(); ps != nil {
		return ps.GetPeerByUsername(username)
	}
	return nil, nil
}

// LoadPeers returns all cached peers from the storage backend.
func (c *Client) LoadPeers() ([]*storage.Peer, error) {
	if ps := c.peerStore(); ps != nil {
		return ps.LoadPeers()
	}
	return nil, nil
}

// DeletePeer removes a cached peer from the storage backend.
func (c *Client) DeletePeer(id int64) error {
	if ps := c.peerStore(); ps != nil {
		return ps.DeletePeer(id)
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

func configureSessionDispatch(sess *session.Session, cfg Config, log *Logger) {
	if sess == nil {
		return
	}
	if cfg.DispatchWorkers > 0 {
		sess.SetDispatchConfig(cfg.DispatchWorkers, cfg.DispatchQueueSize)
	}
	if log != nil {
		sess.SetLogger(log)
	}
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
	if !c.config().AutoConnect {
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

	timeout := c.config().Timeout
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
		timeout = c.config().Timeout
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
	timeout := c.config().Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if err := c.connectTransport(timeout); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	c.mu.Lock()
	if c.stopCh == nil {
		c.stopCh = make(chan struct{})
	}
	stopCh := c.stopCh
	c.mu.Unlock()

	<-stopCh
	return nil
}

func (c *Client) Stop() {
	unregisterClient(c)
	c.mu.Lock()
	if c.stopCh == nil {
		c.stopCh = make(chan struct{})
	}
	stopCh := c.stopCh
	c.mu.Unlock()

	c.stopOnce.Do(func() { close(stopCh) })
	_ = c.Disconnect()
}

func (c *Client) Idle() {
	c.mu.Lock()
	if c.stopCh == nil {
		c.stopCh = make(chan struct{})
	}
	stopCh := c.stopCh
	c.mu.Unlock()
	<-stopCh
}

func (c *Client) connectToDC(dcID int, timeout time.Duration) error {
	c.updateConfig(func(cfg *Config) { cfg.DC = dcID })
	c.migratingDC.Store(true)
	return c.connectTransport(timeout)
}

func (c *Client) initialDCID(st storage.Storage) int {
	if c.config().DC != 0 {
		return c.config().DC
	}
	if st != nil {
		if dcID, err := st.DCID(); err == nil && dcID != 0 {
			return dcID
		}
	}
	return 2
}

func (c *Client) connectTransport(timeout time.Duration) error {
	st, migratingDC, err := c.initStorage()
	if err != nil {
		return err
	}
	defer c.migratingDC.Store(false)
	testSession := c.testSession
	testDialer := c.testDialer

	if err := c.importSessionString(st); err != nil {
		return err
	}
	c.loadPeersFromStorage()

	sess, err := c.initSession(st, testSession)
	if err != nil {
		return err
	}

	dc := sess.DC()
	sessionTp, err := c.dialTransport(dc, timeout, testDialer)
	if err != nil {
		return err
	}

	if err := c.performDHExchange(sess, st, dc, sessionTp, migratingDC); err != nil {
		return err
	}

	if err := c.performPFS(sess, st, dc, sessionTp); err != nil {
		return err
	}

	if err := c.startSession(sess, sessionTp, timeout); err != nil {
		return err
	}

	if err := c.bindPFS(sess); err != nil {
		return err
	}

	if err := c.authenticateUser(st, timeout); err != nil {
		return err
	}

	c.postConnect()
	return nil
}

// initStorage resolves the storage backend, sets the connecting state, and
// returns the storage, whether a DC migration is in progress, and any error.
func (c *Client) initStorage() (storage.Storage, bool, error) {
	c.mu.Lock()
	dcID := c.initialDCID(c.testStorage)
	if err := c.state.SetConnecting(dcID); err != nil {
		c.mu.Unlock()
		return nil, false, err
	}

	st := c.testStorage
	if st == nil {
		if c.storage != nil {
			st = c.storage
		} else if c.config().Storage != nil {
			st = c.config().Storage
		} else if c.config().InMemory {
			st = NewMemoryStorage()
		} else if c.config().SessionName != "" {
			s, err := newDefaultStorage(c.config().SessionName)
			if err != nil {
				c.mu.Unlock()
				return nil, false, fmt.Errorf("telegram: auto-create storage: %w", err)
			}
			st = s
		} else {
			c.mu.Unlock()
			return nil, false, ErrNoStorage
		}
	}
	c.storage = st
	migratingDC := c.migratingDC.Load()
	c.mu.Unlock()

	if c.config().SessionName != "" {
		if err := st.SetSessionID(c.config().SessionName); err != nil {
			return nil, false, fmt.Errorf("set session id %q: %w", c.config().SessionName, err)
		}
	}
	return st, migratingDC, nil
}

// importSessionString decodes the SessionString config field and copies its
// session fields (DC, auth key, API ID) into the storage backend.
func (c *Client) importSessionString(st storage.Storage) error {
	if c.config().SessionString == "" {
		if c.config().APIID == 0 {
			return fmt.Errorf("telegram: apiID is required (not found in session string)")
		}
		return nil
	}
	src, err := sessions.StringSession(c.config().SessionString)
	if err != nil {
		return fmt.Errorf("telegram: decode session string: %w", err)
	}
	if dc, _ := src.DCID(); dc > 0 {
		if err := st.SetDCID(dc); err != nil {
			return fmt.Errorf("telegram: import session dc_id: %w", err)
		}
	}
	if key, _ := src.AuthKey(); len(key) > 0 {
		if err := st.SetAuthKey(key); err != nil {
			return fmt.Errorf("telegram: import session auth key: %w", err)
		}
	}
	if c.config().APIID == 0 {
		if appID, _ := src.APIID(); appID > 0 {
			c.updateConfig(func(cfg *Config) { cfg.APIID = appID })
		}
	}
	if c.config().APIID == 0 {
		return fmt.Errorf("telegram: apiID is required (not found in session string)")
	}
	if err := st.SetAPIID(c.config().APIID); err != nil {
		return fmt.Errorf("telegram: import session api_id: %w", err)
	}
	if uid, err := src.UserID(); err == nil && uid != 0 {
		if err := st.SetUserID(uid); err != nil {
			return fmt.Errorf("telegram: import session user_id: %w", err)
		}
	}
	if isBot, err := src.IsBot(); err == nil {
		if err := st.SetIsBot(isBot); err != nil {
			c.Log.Warnf("import session is_bot: %v", err)
		}
	}
	return nil
}

// initSession creates or restores the MTProto session for the given storage
// backend. If a test session is provided it is used directly.
func (c *Client) initSession(st storage.Storage, testSession *session.Session) (*session.Session, error) {
	if testSession != nil {
		configureSessionDispatch(testSession, c.cfg, c.Log)
		return testSession, nil
	}
	dcID := c.initialDCID(st)
	dc := session.DataCenter{
		ID:       dcID,
		TestMode: c.config().TestMode,
		IPv6:     c.config().IPv6,
	}
	if dc.Address() == "" {
		return nil, fmt.Errorf("%w: %d", ErrUnknownDC, dc.ID)
	}
	sess, err := session.NewSession(dc, st, c.config().Device.DeviceModel, c.config().Device.AppVersion, c.config().Device.SystemLangCode, c.config().Device.LangCode)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if err := st.SetDCID(dcID); err != nil {
		return nil, fmt.Errorf("save dc_id: %w", err)
	}
	configureSessionDispatch(sess, c.cfg, c.Log)
	return sess, nil
}

// dialTransport establishes the underlying transport connection (TCP,
// WebSocket, or MTProxy) to the given data center.
func (c *Client) dialTransport(dc session.DataCenter, timeout time.Duration, testDialer transport.Dialer) (*sessionTransport, error) {
	// Add this endpoint to the pool if not already present.
	c.dcOptionPool.AddOption(dc)

	// Check connection pool first (warm cache).
	if cached, ok := c.connPool.Get(dc.ID, dc); ok {
		if st, ok := cached.(*sessionTransport); ok {
			c.Log.Debug("reusing cached connection for ", dc)
			c.dcOptionPool.RecordSuccess(dc)
			return st, nil
		}
	}

	if useWebSocket(c.cfg) {
		wsAddr := wsDCAddress(dc.ID, dc.TestMode, c.config().WebSocketTLS)
		wsCtx, wsCancel := dialerCtx(timeout)
		defer wsCancel()
		var wsConn net.Conn
		var err error
		if c.config().WSDialer != nil {
			wsConn, err = c.config().WSDialer(wsCtx, wsAddr)
		} else {
			wsConn, err = transport.DialWebsocket(wsCtx, wsAddr)
		}
		if err != nil {
			c.dcOptionPool.RecordFailure(dc)
			return nil, fmt.Errorf("ws dial %s: %w", wsAddr, err)
		}
		tp := transport.NewTCPIntermediateNoHeader(wsConn)
		if err := tp.Connect(); err != nil {
			wsConn.Close()
			c.dcOptionPool.RecordFailure(dc)
			return nil, fmt.Errorf("ws transport handshake: %w", err)
		}
		st := newSessionTransport(tp, wsConn)
		c.dcOptionPool.RecordSuccess(dc)
		return st, nil
	}
	if c.config().MTProxy != nil {
		mpConn, err := mtproxy.Dial(c.config().MTProxy.Addr, c.config().MTProxy.Secret, dc.ID, timeout)
		if err != nil {
			c.dcOptionPool.RecordFailure(dc)
			return nil, fmt.Errorf("mtproxy dial: %w", err)
		}
		tp := transport.NewTCPIntermediateNoHeader(mpConn)
		if err := tp.Connect(); err != nil {
			mpConn.Close()
			c.dcOptionPool.RecordFailure(dc)
			return nil, fmt.Errorf("mtproxy transport handshake: %w", err)
		}
		st := newSessionTransport(tp, mpConn)
		c.dcOptionPool.RecordSuccess(dc)
		return st, nil
	}

	addr := fmt.Sprintf("%s:%d", dc.Address(), dc.Port())
	if c.config().ServerAddr != "" {
		addr = c.config().ServerAddr
	}
	d := c.dialer
	if testDialer != nil {
		d = testDialer
	}
	conn, err := d.Dial("tcp", addr, timeout)
	if err != nil {
		c.dcOptionPool.RecordFailure(dc)
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	tp, err := c.createTransport(conn)
	if err != nil {
		conn.Close()
		c.dcOptionPool.RecordFailure(dc)
		return nil, err
	}
	if err := tp.Connect(); err != nil {
		conn.Close()
		c.dcOptionPool.RecordFailure(dc)
		return nil, fmt.Errorf("transport handshake: %w", err)
	}
	st := newSessionTransport(tp, conn)
	c.dcOptionPool.RecordSuccess(dc)

	// Cache the connection for potential reuse.
	c.connPool.Put(dc.ID, dc, st)

	return st, nil
}

// performDHExchange runs the MTProto DH key exchange if no auth key exists or
// a DC migration is in progress. On success the auth key and salt are saved to
// storage.
func (c *Client) performDHExchange(sess *session.Session, st storage.Storage, dc session.DataCenter, sessionTp *sessionTransport, migratingDC bool) error {
	authKey, _ := st.AuthKey()
	if len(authKey) != 0 && !migratingDC {
		c.Log.Debug("loaded auth key from session; auth_key=", len(authKey), " bytes")
		return nil
	}

	c.Log.Debug("auth key missing; starting DH exchange with DC ", dc.ID)
	auth := &session.Auth{
		DC:       dc.ID,
		TestMode: dc.TestMode,
	}
	if c.keySet != nil {
		auth.SetKeySet(c.keySet)
	}
	result, err := auth.Create(sessionTp)
	if err != nil {
		sessionTp.Close()
		return fmt.Errorf("DH key exchange: %w", err)
	}
	sess.SetAuthKey(result.AuthKey)
	sess.SetServerSalt(result.ServerSalt)
	sess.SetServerTime(time.Unix(int64(result.ServerTime), 0))
	if err := st.SetAuthKey(result.AuthKey); err != nil {
		return fmt.Errorf("save auth key: %w", err)
	}
	if err := st.SetAPIID(c.config().APIID); err != nil {
		c.Log.Warnf("save api_id: %v", err)
	}
	if err := st.SetAPIHash(c.config().APIHash); err != nil {
		c.Log.Warnf("save api_hash: %v", err)
	}
	if err := st.SetTestMode(dc.TestMode); err != nil {
		c.Log.Warnf("save test mode: %v", err)
	}
	c.Log.Debug("DH exchange complete; auth_key=", len(result.AuthKey), " bytes")
	syncStorage(st)
	return nil
}

// performPFS generates a temporary auth key for Perfect Forward Secrecy.
// Matches MadelineProto's always-on PFS approach.
//
// On first connect: generates a new temp key via unencrypted DH exchange.
// On reconnect: reuses the existing temp key if still valid (avoids DH overhead).
// Falls back to permanent key on failure (non-fatal).
func (c *Client) performPFS(sess *session.Session, st storage.Storage, dc session.DataCenter, sessionTp *sessionTransport) error {
	if !c.config().PFS {
		return nil
	}

	// Check if session already has a valid temp key from a previous connection.
	if existing := sess.PFS(); existing != nil {
		if !existing.NeedsRotation() && existing.IsBound() {
			tempKey, _ := existing.GetKey()
			if len(tempKey) > 0 {
				c.Log.Debug("PFS: reusing existing temp key (still valid)")
				return nil
			}
		}
		c.Log.Debug("PFS: temp key expired or unbound, generating new one")
	}

	permKey, _ := st.AuthKey()
	if len(permKey) == 0 {
		return nil
	}

	c.Log.Debug("PFS: generating temporary auth key (24h expiry)")

	mgr := session.NewTempKeyManager(dc.ID, dc.TestMode, permKey, true, true, st)
	if err := mgr.Generate(sessionTp); err != nil {
		c.Log.Warnf("PFS: temp key generation failed, continuing with perm key: %v", err)
		return nil // non-fatal
	}

	tempKey, _ := mgr.GetKey()
	sess.SwapAuthKey(tempKey)
	sess.SetPFS(mgr)

	c.Log.Debug("PFS: temp key generated, session swapped to temp key")
	return nil
}

// bindPFS sends auth.bindTempAuthKey to bind the temp key to the permanent
// key. This must be called after the encrypted session has started (the bind
// request goes through the encrypted channel using the temp key).
//
// On success, sends initConnection to rewrite client info as required by the
// PFS spec. On failure, falls back to the permanent key.
func (c *Client) bindPFS(sess *session.Session) error {
	pfs := sess.PFS()
	if pfs == nil {
		return nil
	}

	c.Log.Debug("PFS: binding temp key to permanent key")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := pfs.Bind(ctx, sess.SessionID(), func(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error) {
		return sess.Invoke(ctx, query, retries, timeout)
	})
	if err != nil {
		c.Log.Warnf("PFS: bind failed, falling back to permanent key: %v", err)
		sess.SwapAuthKey(pfs.PermKey())
		sess.SetPFS(nil)
		return nil // non-fatal
	}

	if pfs.NeedsInitConnection() {
		c.Log.Debug("PFS: rewriting client info via initConnection")
		rpc := c.Raw()
		_, icErr := rpc.HelpGetConfig(ctx)
		if icErr != nil {
			c.Log.Warnf("PFS: initConnection (help.getConfig) failed: %v", icErr)
		}
	}

	c.Log.Info("PFS: temp key bound successfully")
	return nil
}

// startSession registers the update handler, starts the encrypted session, and
// marks the client as connected.
func (c *Client) startSession(sess *session.Session, sessionTp *sessionTransport, timeout time.Duration) error {
	sess.SetUpdateHandler(func(obj tg.TLObject) {
		c.processRawUpdate(obj)
	})
	sess.SetOnPanic(func(r any) {
		c.Log.Errorf("session dispatch panic: %v", r)
	})

	c.mu.Lock()
	c.apiInit = false
	c.mu.Unlock()

	if c.config().HealthPingInterval > 0 {
		sess.SetPingInterval(c.config().HealthPingInterval)
	}
	if c.config().HealthPongTimeout > 0 {
		sess.SetPongTimeout(c.config().HealthPongTimeout)
	}

	c.Log.Debug("starting encrypted session")
	if err := sess.Connect(sessionTp, timeout); err != nil {
		sessionTp.Close()
		return fmt.Errorf("session start: %w", err)
	}
	c.Log.Info("encrypted session started")

	c.sessionWg.Go(func() {
		<-sess.SessionDone()
		if c.state.IsConnected() {
			c.triggerReconnect(fmt.Errorf("session exited"))
		}
	})

	c.mu.Lock()
	c.session = sess
	c.state.SetConnected()
	c.state.SetDC(c.initialDCID(c.storage))
	c.mu.Unlock()
	return nil
}

// authenticateUser handles bot authorization import, user restore from storage,
// and phone login flow. Returns an error only for fatal failures that should
// abort the connection.
func (c *Client) authenticateUser(st storage.Storage, timeout time.Duration) error {
	if botToken := c.config().BotToken; botToken != "" {
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
				me := &types.User{
					ID:        uid,
					FirstName: func() string { v, _ := st.FirstName(); return v }(),
					LastName:  func() string { v, _ := st.LastName(); return v }(),
					Username:  func() string { v, _ := st.Username(); return v }(),
				}
				c.mu.Lock()
				c.me = me
				c.mu.Unlock()
				c.Log.Debug("user account restored: id=", me.ID, " username=", me.Username)
			}
		}

		if !alreadyAuthorized && !isUserAccount {
			c.Log.Info("importing bot authorization")
			rpc := c.Raw()
			authResult, err := rpc.AuthImportBotAuthorization(context.Background(), &tg.AuthImportBotAuthorizationRequest{
				Flags:        0,
				APIID:        c.config().APIID,
				APIHash:      c.config().APIHash,
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
						me := types.ParseUser(u)
						c.mu.Lock()
						c.me = me
						c.mu.Unlock()
						c.Log.Info("bot user: id=", me.ID, " username=", me.Username)
						if st != nil {
							if err := st.SetUserID(me.ID); err != nil {
								c.Log.Warnf("save user id: %v", err)
							}
							if err := st.SetIsBot(true); err != nil {
								c.Log.Warnf("save is_bot: %v", err)
							}
							if err := st.SetFirstName(me.FirstName); err != nil {
								c.Log.Warnf("save first_name: %v", err)
							}
							if err := st.SetLastName(me.LastName); err != nil {
								c.Log.Warnf("save last_name: %v", err)
							}
							if err := st.SetUsername(me.Username); err != nil {
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

	if err := c.restoreAuthorizedUser(st); err != nil {
		c.Log.Debugf("user restore skipped: %v", err)
	}

	needLogin := !c.isAuthorized() && c.config().PhoneNumber != "" && c.config().BotToken == "" && c.config().SessionString == "" && !c.migratingDC.Load()
	if needLogin {
		c.Log.Info("session not authorized; starting phone login flow")
		if err := c.loginUser(context.Background()); err != nil {
			c.cleanupSessions()
			return fmt.Errorf("phone login: %w", err)
		}
	}
	return nil
}

func (c *Client) restoreAuthorizedUser(st storage.Storage) error {
	c.mu.RLock()
	meSet := c.me != nil
	c.mu.RUnlock()
	if meSet || st == nil {
		return nil
	}

	if uid, err := st.UserID(); err == nil && uid != 0 {
		isBot, _ := st.IsBot()
		me := &types.User{
			ID:        uid,
			IsBot:     isBot,
			FirstName: func() string { v, _ := st.FirstName(); return v }(),
			LastName:  func() string { v, _ := st.LastName(); return v }(),
			Username:  func() string { v, _ := st.Username(); return v }(),
		}
		c.mu.Lock()
		c.me = me
		c.mu.Unlock()
		c.Log.Debug("user restored from storage: id=", me.ID, " username=", me.Username)
		return nil
	}

	authKey, err := st.AuthKey()
	if err != nil {
		return err
	}
	if len(authKey) == 0 {
		return nil
	}

	timeout := c.config().ReqTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	me, err := c.GetMe(ctx)
	if err != nil {
		// AUTH_KEY_UNREGISTERED (401) means the auth key has no associated
		// user session. This is expected for freshly created keys (DH exchange
		// without prior auth.signIn/importBotAuthorization). Treat as a non-error
		// so the caller can proceed to the login flow or return to the user.
		if rpcErr, ok := tgerr.As(err); ok && rpcErr.Code == 401 {
			c.Log.Debug("auth key has no user session; skipping user restore")
			return nil
		}
		return err
	}
	c.saveMeToStorage(me)
	c.Log.Debug("user restored from auth key: id=", me.ID, " username=", me.Username)
	return nil
}

// postConnect runs post-connection setup: fetching update state, starting
// postConnect runs initialization steps after the session is connected: fetching
// updates state, plugins, and enabling update recovery.
func (c *Client) postConnect() {
	// Start the RSA key rotation watchdog if configured.
	if c.keyWatchdog != nil {
		ctx, cancel := context.WithCancel(context.Background())
		c.keyWatchdogCancel = cancel
		c.keyWatchdog.Start(ctx)
		c.Log.Debug("rsa key rotation watchdog started")
	}

	// Enable outbound container packing if configured.
	if c.config().OutboundBatchEnabled {
		c.mu.RLock()
		sess := c.session
		c.mu.RUnlock()
		if sess != nil {
			sess.EnableOutboundBatching(
				c.config().OutboundMaxContainerBytes,
				c.config().OutboundCoalesceWindow,
			)
			c.Log.Debug("outbound container packing enabled")
		}
	}
	// Notify session-loaded hooks before any update processing.
	c.fireSessionLoaded()

	if !c.config().NoUpdates {
		c.Log.Debug("fetching updates state")
		rpc := c.Raw()
		_, err := rpc.UpdatesGetState(context.Background())
		if err != nil {
			if rpcErr, ok := tgerr.As(err); ok && rpcErr.Code == 401 {
				c.Log.Debug("updates state fetch skipped: not authorized (", rpcErr.Type, ")")
			} else {
				c.cleanupSessions()
				c.Log.Errorf("get state: %v", err)
			}
		} else {
			c.Log.Info("updates state fetched")
		}
	}

	if err := c.startPlugins(context.Background()); err != nil {
		c.Log.Errorf("plugin start: %v", err)
	}

	if c.config().UpdateRecoveryEnabled && !c.config().NoUpdates {
		umCfg := updateManagerConfig{
			QueueSize:                 c.config().UpdateQueueSize,
			DurableQueue:              c.config().DurableUpdateQueue,
			MaxHandlerRetry:           c.config().MaxUpdateHandlerRetry,
			MaxChannelDiffConcurrency: c.config().MaxChannelDiffConcurrency,
		}
		// Enable the inbound dispatch queue when InboundQueueDepth > 0.
		if c.config().InboundQueueDepth > 0 {
			umCfg.InboundQueue = NewInboundUpdateQueue(InboundQueueConfig{
				MaxDepth:    c.config().InboundQueueDepth,
				Workers:     c.config().InboundQueueWorkers,
				StallBudget: c.config().InboundStallBudget,
				Log:         c.Log,
			})
		}
		mgr := newUpdateManager(c, c.storage, umCfg)
		if err := mgr.Start(context.Background()); err != nil {
			c.Log.Warnf("update recovery start: %v", err)
		} else {
			mgr.SetRPC(c.Raw())
			c.mu.Lock()
			c.updateManager = mgr
			c.mu.Unlock()
			c.Log.Info("update recovery enabled")
		}
	}

	// Notify connected hooks after all post-connect setup is done.
	c.fireConnected()
}

func (c *Client) processRawUpdate(obj tg.TLObject) {
	updates, ok := obj.(tg.UpdatesClass)
	if !ok {
		return
	}

	// Notify lifecycle hooks before routing. Plugins use this for state
	// tracking and gap detection. Hooks must be non-blocking.
	c.fireUpdateReceived(updates)

	c.mu.RLock()
	um := c.updateManager
	c.mu.RUnlock()
	if um != nil {
		if err := um.EnqueueLive(updates); err != nil {
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

	c.mu.Lock()
	if c.updateManager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		c.updateManager.Stop(ctx)
		cancel()
		c.updateManager = nil
	}
	c.mu.Unlock()
	if c.reconnectMgr != nil {
		c.reconnectMgr.Stop()
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

	c.stopUploadSession()

	c.mu.Lock()
	sess := c.session
	c.session = nil
	c.me = nil
	c.apiInit = false
	c.mu.Unlock()

	if sess != nil {
		sess.CloseOutboundBatching()
		sess.Stop()
	}
	c.sessionWg.Wait()

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
	c.mu.RLock()
	um := c.updateManager
	c.mu.RUnlock()
	if um == nil {
		return UpdateHealth{}
	}
	return um.Health()
}

// Disconnect closes all sessions (main and exported), releases storage, and marks the client
// as disconnected. It is safe to call Disconnect on an already-disconnected client.
// Returns ErrNotConnected if the client was never connected.
func (c *Client) Disconnect() error {
	if err := c.state.requireConnected(); err != nil {
		if !errors.Is(err, ErrNotConnected) || !c.hasActiveResources() {
			return err
		}
	}
	c.stopPlugins(context.Background())
	c.cleanupSessions()
	return nil
}

func (c *Client) hasActiveResources() bool {
	c.mu.RLock()
	hasMain := c.session != nil || c.storage != nil || c.updateManager != nil
	c.mu.RUnlock()
	if hasMain {
		return true
	}

	c.sessionsMu.Lock()
	hasSessions := len(c.sessions) > 0
	c.sessionsMu.Unlock()
	if hasSessions {
		return true
	}

	if c.dcSessions != nil {
		c.dcSessions.mu.Lock()
		hasDCSessions := len(c.dcSessions.entries) > 0 || len(c.dcSessions.pools) > 0
		c.dcSessions.mu.Unlock()
		return hasDCSessions
	}
	return false
}

// Close permanently closes the client, stopping all reconnect and health-check goroutines.
// After Close, the client cannot be reconnected; create a new Client instead.
// It is safe to call Close on an already-closed client.
func (c *Client) Close() {
	c.stopPlugins(context.Background())
	c.cleanupSessions()
	// Stop the RSA key rotation watchdog (no goroutine leak — Principle V).
	if c.keyWatchdogCancel != nil {
		c.keyWatchdogCancel()
	}
	if c.keyWatchdog != nil {
		c.keyWatchdog.Wait()
	}
	c.state.SetClosed()
}

func (c *Client) Health() HealthStatus {
	return c.state.Health()
}

func (c *Client) handleMigrationError(ctx context.Context, rpcErr *tgerr.Error, query tg.TLObject) (tg.TLObject, error) {
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
		return c.migrateAndRetry(ctx, targetDC, query, st)
	case "FILE_MIGRATE", "STATS_MIGRATE":
		return c.migrateExportImport(ctx, targetDC, query, st)
	default:
		return nil, &MigrationError{TargetDC: targetDC, Err: fmt.Errorf("unsupported migration type: %s", rpcErr.Type)}
	}
}

var idempotentConstructors = map[uint32]bool{
	tg.InvokeWithLayerTypeID:         true,
	tg.InitConnectionTypeID:          true,
	tg.AuthExportAuthorizationTypeID: true,
	tg.AuthImportAuthorizationTypeID: true,
	tg.AuthSendCodeTypeID:            true,
}

func isIdempotent(query tg.TLObject) bool {
	if query == nil {
		return false
	}
	return idempotentConstructors[query.ConstructorID()]
}

func (c *Client) migrateAndRetry(ctx context.Context, targetDC int, query tg.TLObject, st storage.Storage) (tg.TLObject, error) {
	if !isIdempotent(query) {
		return nil, &UnsafeMigrationError{TargetDC: targetDC, Method: fmt.Sprintf("%T", query)}
	}

	fmt.Fprintf(os.Stderr, "migrating to DC %d...\n", targetDC)

	c.cleanupSessions(false)

	c.migratingDC.Store(true)

	if err := st.SetDCID(targetDC); err != nil {
		return nil, &MigrationError{TargetDC: targetDC, Err: fmt.Errorf("save dc_id: %w", err)}
	}
	if err := st.SetAuthKey(nil); err != nil {
		return nil, &MigrationError{TargetDC: targetDC, Err: fmt.Errorf("clear auth key: %w", err)}
	}
	if err := st.SetUserID(0); err != nil {
		return nil, &MigrationError{TargetDC: targetDC, Err: fmt.Errorf("clear user id: %w", err)}
	}
	c.updateConfig(func(cfg *Config) { cfg.DC = targetDC })

	if err := c.connectTransport(30 * time.Second); err != nil {
		return nil, &MigrationError{TargetDC: targetDC, Err: err}
	}

	retries := max(c.config().Retries, 1)
	result, err := c.Invoke(ctx, query, retries, 30*time.Second)
	if err != nil {
		return nil, &MigrationError{TargetDC: targetDC, Err: err}
	}
	if rpcErr, ok := result.(*tg.RPCError); ok {
		parsed := tgerr.New(int(rpcErr.ErrorCode), rpcErr.ErrorMessage)
		return nil, &MigrationError{TargetDC: targetDC, Err: parsed}
	}
	return result, nil
}

func (c *Client) migrateExportImport(ctx context.Context, targetDC int, query tg.TLObject, _ storage.Storage) (tg.TLObject, error) {
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

	// Route query to the appropriate session slot.
	// Currently all slots share the same session, but the routing
	// infrastructure is in place for future multi-session support.
	slotType := session.RouteQuery(query)
	_ = slotType // routing decision logged for future use

	c.mu.RLock()
	sess := c.session
	oc := c.overloadController
	c.mu.RUnlock()

	if sess == nil {
		return nil, ErrNotConnected
	}

	// Overload control: gate admission by priority (FR-018). When disabled
	// (MaxInFlightRPCs == 0), Admit is a no-op (backward compat).
	if oc != nil && oc.Enabled() {
		priority := int(session.RoutePriority(query))
		release, err := oc.Admit(ctx, priority)
		if err != nil {
			return nil, err
		}
		defer release()
	}

	result, err := sess.Invoke(ctx, query, retries, timeout)
	if err != nil {
		var rpcErr *tgerr.Error
		if errors.As(err, &rpcErr) && rpcErr.Code == 303 {
			return c.handleMigrationError(ctx, rpcErr, query)
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

	timeout := c.config().ReqTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if d := time.Until(deadline); d < timeout {
			timeout = d
		}
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if timeout < time.Second {
		timeout = time.Second
	}
	retries := max(c.config().Retries, 1)

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
	if c.config().NoUpdates || !c.IsConnected() {
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
	c.backfillMinAccessHashes(chatMap, userMap)

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
			c.mu.RLock()
			me := c.me
			c.mu.RUnlock()
			if me != nil {
				fromID = &tg.PeerUser{UserID: me.ID}
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
	case *tg.UpdateBotInlineSend:
		upd.ChosenInlineResult = types.ParseChosenInlineResult(v)
	case *tg.UpdateUserStatus:
		upd.UserStatus = &types.UserStatusUpdated{UserID: v.UserID}
	case *tg.UpdateUserName:
		if len(v.Usernames) > 0 {
			c.cacheUsername(v.Usernames[0].Username, v.UserID)
			if ps := c.peerStore(); ps != nil {
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
		upd.ChatMember = types.ParseChatMemberUpdated(v, userClassesFromPeerMap(pm), pm)
	case *tg.UpdateChannelParticipant:
		upd.ChatMember = types.ParseChatMemberUpdated(v, userClassesFromPeerMap(pm), pm)
	case *tg.UpdateBotMessageReaction:
		upd.MessageReaction = types.ParseMessageReactionUpdate(v)
	case *tg.UpdateBotMessageReactions:
		upd.MessageReactionCount = types.ParseMessageReactionCountUpdate(v)
	case *tg.UpdateMessagePoll:
		upd.Poll = types.ParsePollUpdated(v)
	case *tg.UpdateMessagePollVote:
		upd.PollAnswer = types.ParsePollAnswerUpdate(v)
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
	case *tg.UpdateBotBusinessConnect:
		upd.BusinessConnection = types.ParseBusinessConnection(v.Connection, nil)
	case *tg.UpdateBotNewBusinessMessage:
		upd.BusinessMessage = types.ParseMessage(v.Message, pm)
		bindMessage(upd.BusinessMessage, c)
		c.resolveMessagePeers(upd.BusinessMessage, users, chats)
	case *tg.UpdateBotEditBusinessMessage:
		upd.EditedBusinessMessage = types.ParseMessage(v.Message, pm)
		bindMessage(upd.EditedBusinessMessage, c)
		c.resolveMessagePeers(upd.EditedBusinessMessage, users, chats)
	case *tg.UpdateBotDeleteBusinessMessage:
		upd.DeletedBusinessMessages = &types.DeletedMessages{Messages: v.Messages}
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
	case *tg.UpdateBotChatBoost:
		var chatID int64
		switch p := v.Peer.(type) {
		case *tg.PeerChat:
			chatID = -p.ChatID
		case *tg.PeerChannel:
			chatID = -1_000_000_000_000 - p.ChannelID
		case *tg.PeerUser:
			chatID = p.UserID
		}
		boost := types.ParseChatBoost(v.Boost, pm)
		var chat *types.Chat
		if ch, ok := chats[chatID]; ok {
			chat = ch
		}
		upd.ChatBoost = types.ParseChatBoostUpdated(chat, boost, v.Boost.Stars)
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

func userClassesFromPeerMap(pm *types.PeerMap) map[int64]tg.UserClass {
	if pm == nil || len(pm.Users) == 0 {
		return nil
	}
	users := make(map[int64]tg.UserClass, len(pm.Users))
	for id, user := range pm.Users {
		users[id] = user
	}
	return users
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
func (c *Client) ResolvePeer(ctx context.Context, peerID any) (tg.InputPeerClass, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}
	var (
		peer tg.InputPeerClass
		err  error
	)
	switch p := peerID.(type) {
	case tg.InputPeerClass:
		peer = p
	case int64:
		peer, err = c.resolveNumericPeer(ctx, p)
	case int:
		peer, err = c.resolveNumericPeer(ctx, int64(p))
	case string:
		peer, err = ChatRefFrom(p).resolve(ctx, c)
	case ChatRef:
		peer, err = p.resolve(ctx, c)
	default:
		return nil, fmt.Errorf("%w: unsupported peer type %T", ErrPeerNotFound, peerID)
	}
	if err != nil {
		return nil, err
	}
	if c.IsBot() {
		return c.resolveBotPeerAccessHash(ctx, peer)
	}
	return peer, nil
}

func (c *Client) resolveNumericPeer(ctx context.Context, id int64) (tg.InputPeerClass, error) {
	peer, err := ChatID(id).resolve(ctx, c)
	if err == nil && hasAccessHash(peer) {
		return peer, nil
	}
	if c.IsBot() {
		return c.resolveNumericPeerForBot(ctx, id)
	}
	return c.resolveNumericPeerForAccount(ctx, id)
}

// hasAccessHash returns false when peer is a channel or user with a zero
// access hash. Such peers (typically cached from min entities) are unusable
// for API calls like channels.getFullChannel and must be re-resolved.
func hasAccessHash(peer tg.InputPeerClass) bool {
	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		return p.AccessHash != 0
	case *tg.InputPeerUser:
		return p.AccessHash != 0
	default:
		return true
	}
}

func (c *Client) resolveNumericPeerForBot(ctx context.Context, id int64) (tg.InputPeerClass, error) {
	if peer, ok := inputPeerFromBareChatID(id); ok {
		return peer, nil
	}
	if raw, ok := rawChannelID(id); ok {
		peer, err := c.resolveBotPeerAccessHash(ctx, &tg.InputPeerChannel{ChannelID: raw})
		if err != nil {
			return nil, fmt.Errorf("could not resolve chat: %w", err)
		}
		return peer, nil
	}
	if id > 0 {
		peer, err := c.resolveBotPeerAccessHash(ctx, &tg.InputPeerUser{UserID: id})
		if err != nil {
			return nil, fmt.Errorf("could not resolve chat: %w", err)
		}
		return peer, nil
	}
	return nil, fmt.Errorf("could not resolve chat: %w", ErrPeerNotFound)
}

func (c *Client) resolveBotPeerAccessHash(ctx context.Context, peer tg.InputPeerClass) (tg.InputPeerClass, error) {
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		if p.AccessHash != 0 {
			return peer, nil
		}
		return c.resolveBotUserAccessHash(ctx, p.UserID)
	case *tg.InputPeerChannel:
		if p.AccessHash != 0 {
			return peer, nil
		}
		return c.resolveBotChannelAccessHash(ctx, p.ChannelID)
	default:
		return peer, nil
	}
}

func (c *Client) resolveBotUserAccessHash(ctx context.Context, userID int64) (tg.InputPeerClass, error) {
	rpc := c.Raw()
	result, err := rpc.UsersGetUsers(ctx, &tg.UsersGetUsersRequest{
		ID: []tg.InputUserClass{
			&tg.InputUser{UserID: userID, AccessHash: 0},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: get user %d: %w", ErrPeerNotFound, userID, err)
	}
	users := usersFromUsersGetUsers(result)
	c.cachePeersFromUpdates(users, nil)
	for _, u := range users {
		user, ok := u.(*tg.User)
		if ok && user.ID == userID && user.AccessHash != 0 {
			peer := &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}
			c.CachePeer(user.ID, peer)
			return peer, nil
		}
	}
	return nil, ErrPeerNotFound
}

func (c *Client) resolveBotChannelAccessHash(ctx context.Context, channelID int64) (tg.InputPeerClass, error) {
	rpc := c.Raw()
	result, err := rpc.ChannelsGetChannels(ctx, &tg.ChannelsGetChannelsRequest{
		ID: []tg.InputChannelClass{
			&tg.InputChannel{ChannelID: channelID, AccessHash: 0},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: get channel %d: %w", ErrPeerNotFound, channelID, err)
	}
	chats := chatsFromChatsClass(result)
	c.cachePeersFromUpdates(nil, chats)
	for _, ch := range chats {
		channel, ok := ch.(*tg.Channel)
		if ok && channel.ID == channelID && channel.AccessHash != 0 {
			peer := &tg.InputPeerChannel{ChannelID: channel.ID, AccessHash: channel.AccessHash}
			c.CachePeer(channel.ID, peer)
			return peer, nil
		}
	}
	return nil, ErrPeerNotFound
}

func (c *Client) resolveNumericPeerForAccount(ctx context.Context, id int64) (tg.InputPeerClass, error) {
	if peer, ok := inputPeerFromBareChatID(id); ok {
		return peer, nil
	}
	if preloadErr := c.preloadDialogPeer(ctx, id); preloadErr != nil && !errors.Is(preloadErr, ErrPeerNotFound) {
		return nil, preloadErr
	}
	peer, err := ChatID(id).resolve(ctx, c)
	if err != nil {
		return nil, err
	}
	// If dialog preload couldn't find a full hash, try username resolution
	// as a last resort before returning a zero-hash peer.
	if !hasAccessHash(peer) {
		if resolved, err := c.resolvePeerByUsername(ctx, id); err == nil {
			return resolved, nil
		}
	}
	return peer, nil
}

// resolvePeerByUsername attempts to resolve a peer via its cached username
// when the access hash is unknown (min entity). This is the fallback path
// after dialog preload fails to find the peer.
func (c *Client) resolvePeerByUsername(ctx context.Context, id int64) (tg.InputPeerClass, error) {
	username := c.lookupUsername(id)
	if username == "" {
		return nil, ErrPeerNotFound
	}
	return c.ResolveUsername(ctx, username)
}

func (c *Client) lookupUsername(peerID int64) string {
	c.peerCacheMu.RLock()
	defer c.peerCacheMu.RUnlock()
	for username, id := range c.usernameCache {
		if id == peerID {
			return username
		}
	}
	return ""
}

func inputPeerFromBareChatID(id int64) (tg.InputPeerClass, bool) {
	if id < 0 {
		if _, ok := rawChannelID(id); !ok {
			return &tg.InputPeerChat{ChatID: -id}, true
		}
	}
	return nil, false
}

func (c *Client) preloadDialogPeer(ctx context.Context, id int64) error {
	const (
		dialogPageLimit = 100
		maxDialogPages  = 20
	)

	rpc := c.Raw()
	offsetPeer := tg.InputPeerClass(&tg.InputPeerEmpty{})
	var offsetDate int32
	var offsetID int32

	for page := 0; page < maxDialogPages; page++ {
		result, err := rpc.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetDate: offsetDate,
			OffsetID:   offsetID,
			OffsetPeer: offsetPeer,
			Limit:      dialogPageLimit,
		})
		if err != nil {
			return err
		}

		dialogs, messages, users, chats, ok := unpackDialogs(result)
		if !ok {
			return ErrPeerNotFound
		}

		c.cachePeersFromUpdates(users, chats)
		if _, err := c.ResolvePeerCache(id); err == nil {
			return nil
		}
		if len(dialogs) == 0 {
			break
		}

		nextPeer, nextID, nextDate, ok := c.nextDialogOffset(dialogs, messages)
		if !ok {
			break
		}
		offsetPeer = nextPeer
		offsetID = nextID
		offsetDate = nextDate
	}

	return ErrPeerNotFound
}

func unpackDialogs(result tg.DialogsClass) ([]tg.DialogClass, []tg.MessageClass, []tg.UserClass, []tg.ChatClass, bool) {
	switch v := result.(type) {
	case *tg.MessagesDialogs:
		return v.Dialogs, v.Messages, v.Users, v.Chats, true
	case *tg.MessagesDialogsSlice:
		return v.Dialogs, v.Messages, v.Users, v.Chats, true
	case *tg.MessagesDialogsNotModified:
		return nil, nil, nil, nil, false
	default:
		return nil, nil, nil, nil, false
	}
}

func chatsFromChatsClass(result tg.ChatsClass) []tg.ChatClass {
	switch v := result.(type) {
	case *tg.MessagesChats:
		return v.Chats
	case *tg.MessagesChatsSlice:
		return v.Chats
	default:
		return nil
	}
}

func usersFromUsersGetUsers(result tg.TLObject) []tg.UserClass {
	vector, ok := result.(*tg.GenericVector)
	if !ok {
		return nil
	}
	users := make([]tg.UserClass, 0, len(vector.Items))
	for _, item := range vector.Items {
		if user, ok := item.(tg.UserClass); ok {
			users = append(users, user)
		}
	}
	return users
}

func (c *Client) nextDialogOffset(dialogs []tg.DialogClass, messages []tg.MessageClass) (tg.InputPeerClass, int32, int32, bool) {
	last, ok := dialogs[len(dialogs)-1].(*tg.Dialog)
	if !ok || last.Peer == nil {
		return nil, 0, 0, false
	}
	peerID, ok := peerClassID(last.Peer)
	if !ok {
		return nil, 0, 0, false
	}
	peer, err := c.ResolvePeerCache(peerID)
	if err != nil {
		return nil, 0, 0, false
	}
	return peer, last.TopMessage, messageDate(messages, last.TopMessage), true
}

func peerClassID(peer tg.PeerClass) (int64, bool) {
	switch p := peer.(type) {
	case *tg.PeerUser:
		return p.UserID, true
	case *tg.PeerChat:
		return p.ChatID, true
	case *tg.PeerChannel:
		return p.ChannelID, true
	default:
		return 0, false
	}
}

func messageDate(messages []tg.MessageClass, id int32) int32 {
	for _, msg := range messages {
		switch m := msg.(type) {
		case *tg.Message:
			if m.ID == id {
				return m.Date
			}
		case *tg.MessageService:
			if m.ID == id {
				return m.Date
			}
		}
	}
	return 0
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

	c.mu.RLock()
	testFactory := c.testSessionF
	c.mu.RUnlock()
	if err := c.ensureConnected(); err != nil && testFactory == nil {
		return nil, err
	}

	addr := ResolveDCAddress(dcID, c.config().TestMode)
	if addr == "" {
		return nil, fmt.Errorf("%w: %d", ErrUnknownDC, dcID)
	}
	port := DefaultDCPort(c.config().TestMode)

	var sess *session.Session
	var err error

	if testFactory != nil {
		sess, err = testFactory(ctx, dcID, addr, port, nil)
	} else {
		dc := session.DataCenter{
			ID:       dcID,
			TestMode: c.config().TestMode,
			IPv6:     c.config().IPv6,
		}
		c.mu.RLock()
		st := c.storage
		c.mu.RUnlock()
		if st == nil {
			st = NewMemoryStorage()
		}
		sess, err = session.NewSession(dc, st, c.config().DeviceModel, c.config().AppVersion, c.config().SystemLangCode, c.config().LangCode)
	}
	if err != nil {
		return nil, fmt.Errorf("create session for dc %d: %w", dcID, err)
	}
	c.mu.RLock()
	cfg := c.cfg
	log := c.Log
	c.mu.RUnlock()
	configureSessionDispatch(sess, cfg, log)

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
	syncStorage(st)
}

// syncStorage flushes pending session changes to durable storage. It is a
// no-op for storage backends that do not implement a Sync method (e.g.
// in-memory storage).
func syncStorage(st storage.Storage) {
	type syncer interface{ Sync() error }
	if s, ok := st.(syncer); ok {
		_ = s.Sync()
	}
}

// ServerTime returns the current estimated server time adjusted by the configured timezone offset.
func (c *Client) ServerTime() int32 {
	return ServerTime(c.config().Device.TZOffset)
}

// APIID returns the Telegram API ID configured for this client.
func (c *Client) APIID() int32 { return c.config().APIID }

// APIHash returns the Telegram API hash configured for this client.
func (c *Client) APIHash() string { return c.config().APIHash }

// DC returns the configured preferred data center ID, or zero when automatic.
func (c *Client) DC() int { return c.config().DC }

// ServerAddr returns the manually configured server address, or empty for auto-resolution.
func (c *Client) ServerAddr() string { return c.config().ServerAddr }

// LocalAddr returns the local address binding for outbound connections, or empty for default.
func (c *Client) LocalAddr() string { return c.config().LocalAddr }

// SessionName returns the session name used for storage file naming.
func (c *Client) SessionName() string { return c.config().SessionName }

// BotToken returns the bot token if one was configured, or an empty string for user accounts.
func (c *Client) BotToken() string { return c.config().BotToken }

// TestMode reports whether the client is configured to connect to Telegram's test DC.
func (c *Client) TestMode() bool { return c.config().TestMode }

// AutoConnect reports whether the client will automatically connect before the first
// operation that requires an active connection.
func (c *Client) AutoConnect() bool { return c.config().AutoConnect }

// IPv6 reports whether IPv6 connections are preferred.
func (c *Client) IPv6() bool { return c.config().IPv6 }

// NoUpdates reports whether update processing is disabled.
func (c *Client) NoUpdates() bool { return c.config().NoUpdates }

// ParseMode returns the default message parsing mode.
func (c *Client) ParseMode() ParseMode { return c.config().ParseMode }

// SleepThreshold returns the flood-wait threshold; requests with shorter waits are automatically retried.
func (c *Client) SleepThreshold() time.Duration { return c.config().SleepThreshold }

// Timeout returns the TCP connection timeout used when dialing Telegram servers.
func (c *Client) Timeout() time.Duration { return c.config().Timeout }

// ReqTimeout returns the default RPC request timeout applied when no context deadline is set.
func (c *Client) ReqTimeout() time.Duration { return c.config().ReqTimeout }

// MaxConcurrentTransmissions returns the maximum number of concurrent RPC transmissions allowed.
func (c *Client) MaxConcurrentTransmissions() int { return c.config().MaxConcurrentTrans }

// MaxMessageCacheSize returns the maximum number of messages retained in the message cache.
func (c *Client) MaxMessageCacheSize() int { return c.config().MaxMessageCacheSize }

// MaxTopicCacheSize returns the maximum number of forum topics retained in the topic cache.
func (c *Client) MaxTopicCacheSize() int { return c.config().MaxTopicCacheSize }

// LinkPreviewOptions returns the global link preview defaults, or nil if none are set.
func (c *Client) LinkPreviewOptions() *types.LinkPreviewOptions { return c.config().LinkPreviewOptions }

// Takeout reports whether the client is configured to use a takeout session for data export.
func (c *Client) Takeout() bool { return c.config().Takeout }

// IsBot reports whether the client is authenticated as a bot. It checks the stored
// session state first, falling back to whether a BotToken was configured.
func (c *Client) IsBot() bool {
	if c.storage == nil {
		return c.config().BotToken != ""
	}
	isBot, _ := c.storage.IsBot()
	return isBot
}

// SetBotToken updates the bot token in the client configuration.
func (c *Client) SetBotToken(token string) {
	c.updateConfig(func(cfg *Config) { cfg.BotToken = token })
}

// ResolvePeerCache looks up a previously cached InputPeer by its numeric ID.
// Returns the cached peer or ErrPeerNotFound if not present.
func (c *Client) ResolvePeerCache(id int64) (tg.InputPeerClass, error) {
	c.peerCacheMu.RLock()
	for _, lookupID := range peerLookupIDs(id) {
		if p, ok := c.peerCache[lookupID]; ok {
			c.peerCacheMu.RUnlock()
			return p, nil
		}
	}
	c.peerCacheMu.RUnlock()

	if c.config().SavePeers {
		if ps := c.peerStore(); ps != nil {
			for _, lookupID := range peerLookupIDs(id) {
				p, err := ps.GetPeer(lookupID)
				if err != nil || p == nil {
					continue
				}
				var peer tg.InputPeerClass
				switch p.Type {
				case storage.PeerTypeUser:
					peer = &tg.InputPeerUser{UserID: p.ID, AccessHash: p.AccessHash}
				case storage.PeerTypeChat:
					peer = &tg.InputPeerChat{ChatID: p.ID}
				case storage.PeerTypeChannel:
					channelID := p.ID
					if raw, ok := rawChannelID(channelID); ok {
						channelID = raw
					}
					peer = &tg.InputPeerChannel{ChannelID: channelID, AccessHash: p.AccessHash}
				default:
					return nil, ErrPeerNotFound
				}
				c.CachePeer(lookupID, peer)
				if p.Username != "" {
					c.cacheUsername(p.Username, p.ID)
				}
				return peer, nil
			}
		}
	}
	return nil, ErrPeerNotFound
}

func (c *Client) CachePeer(id int64, peer tg.InputPeerClass) {
	id = canonicalPeerID(id, peer)
	c.peerCacheMu.Lock()
	defer c.peerCacheMu.Unlock()

	// Don't let a zero-hash min entity overwrite a known full hash.
	if existing, ok := c.peerCache[id]; ok {
		peer = preserveAccessHash(existing, peer)
	}

	if _, exists := c.peerCache[id]; !exists {
		c.peerCacheOrder = append(c.peerCacheOrder, id)
	}
	c.peerCache[id] = peer
	c.evictOldestPeerLocked()

	if c.config().SavePeers {
		if ps := c.peerStore(); ps != nil {
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

// preserveAccessHash copies a non-zero access hash from the existing cached
// peer when the incoming peer has a zero hash. This prevents min entities
// (which carry no usable access hash) from poisoning a previously-good cache
// entry. The storage backend already merges correctly via mergePeer; this
// brings the in-memory cache to the same guarantee.
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
	if raw, ok := rawChannelID(id); ok {
		return []int64{raw, id}
	}
	return []int64{id}
}

func canonicalPeerID(id int64, peer tg.InputPeerClass) int64 {
	if p, ok := peer.(*tg.InputPeerChannel); ok && p.ChannelID != 0 {
		return p.ChannelID
	}
	if raw, ok := rawChannelID(id); ok {
		return raw
	}
	return id
}

func rawChannelID(id int64) (int64, bool) {
	const channelChatIDPrefix int64 = -1000000000000
	if id <= channelChatIDPrefix {
		return channelChatIDPrefix - id, true
	}
	return 0, false
}

func (c *Client) evictOldestPeerLocked() {
	limit := c.config().PeerCacheSize
	if limit <= 0 || len(c.peerCache) <= limit {
		return
	}
	for len(c.peerCache) > limit && len(c.peerCacheOrder) > 0 {
		oldest := c.peerCacheOrder[0]
		delete(c.peerCache, oldest)
		if cachedID, ok := c.reverseUsernameCache(oldest); ok {
			delete(c.usernameCache, cachedID)
		}
		copy(c.peerCacheOrder, c.peerCacheOrder[1:])
		c.peerCacheOrder[len(c.peerCacheOrder)-1] = 0
		c.peerCacheOrder = c.peerCacheOrder[:len(c.peerCacheOrder)-1]
	}
}

func (c *Client) reverseUsernameCache(peerID int64) (string, bool) {
	for username, id := range c.usernameCache {
		if id == peerID {
			return username, true
		}
	}
	return "", false
}

func (c *Client) cacheUsername(username string, userID int64) {
	c.peerCacheMu.Lock()
	defer c.peerCacheMu.Unlock()
	if _, exists := c.usernameCache[username]; !exists {
		c.usernameCacheOrder = append(c.usernameCacheOrder, username)
	}
	c.usernameCache[username] = userID
	limit := c.config().PeerCacheSize
	if limit <= 0 || len(c.usernameCache) <= limit {
		return
	}
	for len(c.usernameCache) > limit && len(c.usernameCacheOrder) > 0 {
		oldest := c.usernameCacheOrder[0]
		delete(c.usernameCache, oldest)
		copy(c.usernameCacheOrder, c.usernameCacheOrder[1:])
		c.usernameCacheOrder[len(c.usernameCacheOrder)-1] = ""
		c.usernameCacheOrder = c.usernameCacheOrder[:len(c.usernameCacheOrder)-1]
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
		hash := user.AccessHash
		if user.Min {
			hash = 0
		}
		c.CachePeer(user.ID, &tg.InputPeerUser{UserID: user.ID, AccessHash: hash})
		username := user.Username
		if username != "" {
			c.cacheUsername(username, user.ID)
		}
		entries = append(entries, &storage.Peer{
			ID:          user.ID,
			Type:        storage.PeerTypeUser,
			AccessHash:  hash,
			Username:    username,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			PhoneNumber: user.Phone,
			IsBot:       user.Bot,
			Language:    user.LangCode,
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
			if v.Min {
				accessHash = 0
			}
			c.CachePeer(v.ID, &tg.InputPeerChannel{ChannelID: v.ID, AccessHash: accessHash})
			username := v.Username
			if username != "" {
				c.cacheUsername(username, v.ID)
			}
			entries = append(entries, &storage.Peer{
				ID:         v.ID,
				Type:       storage.PeerTypeChannel,
				AccessHash: accessHash,
				Username:   username,
			})
		}
	}
	if c.config().SavePeers && len(entries) > 0 {
		if ps := c.peerStore(); ps != nil {
			for _, entry := range entries {
				_ = ps.SavePeer(entry)
			}
		}
	}
}

// backfillMinAccessHashes replaces a min entity's access hash with the known
// full hash from the peer store, so handlers can build a usable InputPeer from
// update entities even for chats delivered as min. No-op when the peer store
// has no full hash (no regression).
func (c *Client) backfillMinAccessHashes(chatMap map[int64]*types.Chat, userMap map[int64]*types.User) {
	ps := c.peerStore()
	if ps == nil {
		return
	}
	for _, ch := range chatMap {
		if !ch.IsMin {
			continue
		}
		raw, ok := ch.Raw.(*tg.Channel)
		if !ok {
			continue
		}
		if p, err := ps.GetPeer(raw.ID); err == nil && p != nil && p.AccessHash != 0 && p.AccessHash != ch.AccessHash {
			ch.AccessHash = p.AccessHash
		}
	}
	for _, u := range userMap {
		if !u.IsMin {
			continue
		}
		if p, err := ps.GetPeer(u.ID); err == nil && p != nil && p.AccessHash != 0 && p.AccessHash != u.AccessHash {
			u.AccessHash = p.AccessHash
		}
	}
}

func (c *Client) loadPeersFromStorage() {
	if !c.config().SavePeers {
		return
	}
	ps := c.peerStore()
	if ps == nil {
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
			c.cacheUsername(p.Username, p.ID)
		}
	}
}
