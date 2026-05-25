package telegram

import (
	"time"

	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// Proxy holds connection details for routing Telegram traffic through an
// intermediate server. Set this when the client must operate behind a firewall
// or when direct access to Telegram servers is unavailable.
type Proxy struct {
	// Addr is the proxy address in "host:port" format. Required when using a proxy.
	Addr string
	// Username is the optional authentication username for the proxy server.
	Username string
	// Password is the optional authentication password for the proxy server.
	Password string
	// Protocol is the proxy protocol: "socks5", "socks4", "http", "https".
	// When empty, defaults to "socks5".
	Protocol string
}

// MTProxyConfig holds connection details for routing Telegram traffic through
// an MTProxy server. The secret string must be a hex-encoded MTProxy secret:
//
//   - dd-prefixed (17 bytes): "dd05fb7..."+14 hex chars — obfuscated2 with PaddedIntermediate
//   - ee-prefixed (18+ bytes): "ee8523..."+domain hex — fake TLS + obfuscated2
//   - simple (16 bytes): raw 16-byte secret — obfuscated2 with Intermediate
//
// See https://core.telegram.org/mtproto/mtproxy for the protocol specification.
type MTProxyConfig struct {
	// Addr is the MTProxy server address in "host:port" format.
	Addr string
	// Secret is the hex-encoded MTProxy secret string.
	Secret string
}

// LogConfig controls logging behaviour for the MTProto client. Use it to
// capture protocol-level events for debugging connection or authentication
// issues that are otherwise invisible.
type LogConfig struct {
	// Level sets the minimum severity that will be emitted. Use LogLevelDebug
	// during development and LogLevelError or LogLevelNone in production.
	Level LogLevel
	// File is the path where log output is written. When empty, logs are
	// discarded unless a custom Logger is provided.
	File string
	// MaxSize is the maximum size in bytes a log file may reach before being
	// rotated. A value of 0 disables rotation.
	MaxSize int64
	// Logger allows injecting a custom logging implementation. When set, it
	// takes precedence over File and Level.
	Logger *Logger
}

// DeviceConfig holds device identity reported to Telegram during init.
type DeviceConfig struct {
	// DeviceModel is the hardware model (e.g. "iPhone 15", "Samsung Galaxy S24").
	DeviceModel string
	// SystemVersion is the OS version (e.g. "iOS 17", "Android 14").
	SystemVersion string
	// AppVersion is the client app version (e.g. "1.0.0").
	AppVersion string
	// LangCode is the two-letter ISO 639-1 UI language code (e.g. "en").
	LangCode string
	// SystemLangCode is the device-level language code (e.g. "en-US").
	SystemLangCode string
	// LangPack names the translation pack (e.g. "tdesktop").
	LangPack string
	// TZOffset is the timezone offset in seconds from UTC.
	TZOffset int
	// ClientPlatform identifies the simulated platform.
	ClientPlatform types.ClientPlatform
}

const (
	// TransportModeAbridged selects the compact MTProto abridged TCP transport.
	TransportModeAbridged = "Abridged"
	// TransportModeIntermediate selects the fixed 4-byte length-prefix TCP transport.
	TransportModeIntermediate = "Intermediate"
	// TransportModePaddedIntermediate selects intermediate framing with 0-15 bytes of transport padding.
	TransportModePaddedIntermediate = "PaddedIntermediate"
	// TransportModeFull selects full TCP framing with sequence numbers and CRC32.
	TransportModeFull = "Full"

	defaultDispatchQueueSize = 256
)

// Config contains every tunable parameter for a Telegram MTProto client.
// Fields that are left at their zero value fall back to the sensible defaults
// defined in DefaultConfig.
//
// Example:
//
//	cfg := telegram.Config{
//	    SessionName:    "my_bot",
//	    BotToken:       "123456:ABC-DEF",
//	    InMemory:       true,
//	    DeviceModel:    "MyApp",
//	    SystemVersion:  "1.0",
//	}
//	client, err := telegram.NewClient(apiID, apiHash, &cfg)
type Config struct {
	// APIID is the application identifier obtained from my.telegram.org.
	// Every client must supply a valid ID to authenticate with Telegram.
	APIID int32
	// APIHash is the secret corresponding to APIID, also from my.telegram.org.
	APIHash string
	// DC specifies the datacenter number to connect to initially. When zero,
	// the client resolves the nearest DC automatically during authorization.
	DC int
	// SessionName is a unique label identifying this session. It is stored
	// via Storage.SetSessionID and used by backends to scope queries.
	// When InMemory is true and Storage is nil, the name is only kept
	// in memory.
	SessionName string
	// BotToken is the Telegram Bot API token. Set this instead of
	// PhoneNumber when authenticating as a bot.
	BotToken string
	// SessionString is a string-encoded session (Telethon, Pyrogram, GramJS,
	// mtcute, or auto-detected format). The client decodes it internally
	// during initialization; errors are returned from Connect/Start.
	//
	//   session.StringSession("auto_detect")
	//   session.TelethonSession("1abc...")
	//   session.PyrogramSession("base64...")
	//
	SessionString string
	// PhoneNumber is the phone number of the Telegram account to authorize,
	// in international format (e.g. "+1234567890").
	PhoneNumber string
	// PhoneCode is the one-time verification code received from Telegram
	// during interactive sign-in. Set programmatically only in automated
	// flows that intercept the code out-of-band.
	PhoneCode string
	// Password is the two-factor authentication password required when the
	// account has 2FA enabled.
	Password string
	// CodeFunc returns the verification code for phone login. When PhoneNumber
	// is set and the session is not yet authorized, Connect calls this function
	// to obtain the OTP. If nil, TerminalCodeFunc (stdin prompt) is used.
	//
	// Example — custom provider:
	//
	//	cfg.CodeFunc = func(ctx context.Context, phone string) (string, error) {
	//	    return readCodeFromWebhook(ctx)
	//	}
	CodeFunc CodeFunc
	// PasswordFunc returns the 2FA password during phone login. Called when
	// the account has two-factor authentication enabled. If nil,
	// TerminalPasswordFunc (stdin prompt) is used.
	PasswordFunc PasswordFunc
	// WorkDir is the filesystem directory where session files are stored.
	// Defaults to the current working directory when empty.
	WorkDir string
	// InMemory keeps all session data in memory instead of writing it to
	// disk. Useful for short-lived clients or environments with no writable
	// filesystem.
	InMemory bool
	// Proxy routes all MTProto traffic through the specified proxy. Leave nil
	// for a direct connection.
	Proxy *Proxy
	// MTProxy configures an MTProxy connection. When set, the client connects
	// to the given proxy address using the provided dd/ee secret and
	// establishes an obfuscated (and optionally fake-TLS) tunnel to the
	// Telegram data center.
	//
	// Example:
	//
	//	cfg.MTProxy = &telegram.MTProxyConfig{
	//	    Addr:   "proxy.example.com:443",
	//	    Secret: "dd05fb7acb549be047a7c585116581418",
	//	}
	MTProxy *MTProxyConfig
	// TestMode connects to Telegram's test datacenters instead of production.
	// Only useful during library development or integration testing.
	TestMode bool
	// IPv6 forces the client to resolve server addresses to IPv6. Enable when
	// the network only provides IPv6 connectivity.
	IPv6 bool
	// NoUpdates disables the long-poll loop that receives real-time updates.
	// Set to true when the client only needs to send requests, not listen for
	// incoming events.
	NoUpdates bool
	// AutoConnect enables automatic connection on first RPC call or update
	// handler registration, without requiring an explicit Connect() call.
	// When true, the client lazily connects when needed and automatically
	// reconnects on disconnection. Defaults to false.
	AutoConnect bool
	// SkipUpdates discards all updates that arrived while the client was
	// offline. Prevents a flood of stale messages on reconnection.
	SkipUpdates bool
	// SleepThreshold is the duration the client waits in flood-wait
	// situations before resuming requests. Telegram signals this when rate
	// limits are approached.
	SleepThreshold time.Duration
	// HandlerTimeout is the maximum time an update handler may run before the
	// client cancels its context. Prevents a slow handler from blocking the
	// update pipeline.
	HandlerTimeout time.Duration
	// Timeout is the TCP connection timeout used when dialing Telegram servers.
	// Defaults to 60 seconds.
	Timeout time.Duration
	// ReqTimeout is the default timeout applied to RPC requests when no deadline
	// is set on the context. Defaults to 60 seconds. Enforced minimum of 1 second.
	ReqTimeout time.Duration
	// Retries is the number of retries for RPC calls on transient errors
	// (timeouts, connection resets, 500s). Non-retryable errors (401, 400, 403)
	// fail immediately regardless of this setting. Defaults to 1.
	// The send timeout per attempt is controlled by ReqTimeout.
	Retries int
	// MaxConcurrentTrans limits how many file transfers may run in parallel.
	// Keep low on bandwidth-constrained networks to avoid throttling.
	MaxConcurrentTrans int
	// DispatchWorkers sets the number of session workers used to TL-decode
	// incoming messages before result/update dispatch. Values <= 0 use
	// runtime.GOMAXPROCS(0). Increase for I/O-heavy update handling; keep near
	// CPU count for CPU-heavy decoding.
	DispatchWorkers int
	// DispatchQueueSize sets the bounded queue capacity for incoming messages
	// waiting for TL decode. Values <= 0 use the default 256. Larger values
	// absorb bursts at the cost of memory; smaller values apply backpressure
	// sooner under high traffic.
	DispatchQueueSize int
	// MaxMessageCacheSize caps the number of messages retained in the
	// internal cache. Older entries are evicted when the limit is exceeded.
	MaxMessageCacheSize int
	// MaxTopicCacheSize caps the number of forum topics retained in the
	// internal cache. Older entries are evicted when the limit is exceeded.
	// Defaults to 1000.
	MaxTopicCacheSize int
	// PeerCacheSize caps the number of peer and username entries cached in memory.
	// When the limit is exceeded, the oldest entries are evicted (FIFO).
	// Setting to 0 (default) disables eviction — the cache grows without bound.
	// Recommended: 5000.
	PeerCacheSize int
	// ParseMode selects the default formatting mode for message text.
	// Use params.ParseModeMarkdown, params.ParseModeHTML, or a raw string
	// like "MarkdownV2". Zero value means no parsing.
	ParseMode params.ParseMode
	// HidePassword masks the 2FA password in logs and error messages. Enable
	// in production to prevent accidental credential leakage.
	HidePassword bool
	// LinkPreviewOptions sets global defaults for link previews on outgoing
	// messages. Individual methods can override these per-call.
	LinkPreviewOptions *types.LinkPreviewOptions
	// Takeout enables a takeout session for exporting Telegram data. When true,
	// the client uses account.initTakeoutSession instead of a normal session
	// and methods like get_chat_history are less prone to FloodWait. Only
	// available for user accounts; bots ignore this setting. Implies
	// NoUpdates=true.
	Takeout bool
	// FetchReplies resolves reply-to references so that quoted messages are
	// included in the incoming Message object.
	FetchReplies bool
	// FetchTopics loads forum topic metadata alongside messages from
	// supergroups that have topics enabled.
	FetchTopics bool
	// FetchStories retrieves user stories in addition to regular updates.
	// Only relevant when the client needs to present story content.
	FetchStories bool
	// FetchStickers downloads sticker metadata so that sticker messages
	// include the full Sticker object rather than just a document reference.
	FetchStickers bool
	// ClientPlatform identifies the simulated device platform sent to
	// Telegram during initialization. Affects which features Telegram exposes.
	ClientPlatform types.ClientPlatform
	// Device configures the device identity reported to Telegram.
	// When set, its fields override the top-level AppVersion, DeviceModel,
	// SystemVersion, LangCode, LangPack, SystemLangCode, TZOffset, and
	// ClientPlatform fields for backwards compatibility.
	Device DeviceConfig
	// AppVersion is the version string reported to Telegram. Used by
	// Telegram's infrastructure for client identification.
	// Deprecated: use Device.AppVersion instead.
	AppVersion string
	// DeviceModel is the hardware model reported to Telegram (e.g.
	// "Samsung Galaxy S24"). Affects session display in active sessions.
	// Deprecated: use Device.DeviceModel instead.
	DeviceModel string
	// SystemVersion is the operating system version reported to Telegram
	// (e.g. "Android 14").
	// Deprecated: use Device.SystemVersion instead.
	SystemVersion string
	// LangCode is the two-letter ISO 639-1 language code for the client's
	// UI language (e.g. "en", "ru").
	// Deprecated: use Device.LangCode instead.
	LangCode string
	// LangPack names the translation pack to use (e.g. "tdesktop" for the
	// desktop client pack). Affects server-side localisation of prompts.
	// Deprecated: use Device.LangPack instead.
	LangPack string
	// SystemLangCode is the device-level language code reported to Telegram.
	// Used for localisation of security notifications.
	// Deprecated: use Device.SystemLangCode instead.
	SystemLangCode string
	// TZOffset is the timezone offset in seconds from UTC. Telegram uses this
	// to display timestamps in the correct local time.
	// Deprecated: use Device.TZOffset instead.
	TZOffset int
	// TransportMode selects the MTProto TCP framing mode for direct TCP
	// connections. Valid values are Abridged, Intermediate,
	// PaddedIntermediate, and Full. When empty, NewClient uses the default
	// Abridged transport mode.
	TransportMode string
	// SavePeers persists encountered peer identifiers to the session file so
	// that they survive restarts without re-fetching.
	SavePeers bool
	// Storage is an optional storage backend for persisting session data.
	// When set, it takes precedence over InMemory and file-based session
	// storage. Use the helper constructors from sub-packages:
	//
	//   sqlite.New("bot.db")
	//   postgres.New(postgres.Config{...})
	//   mongodb.New(ctx, mongodb.Config{...})
	//   storage.NewMemory()
	//
	// If Storage is nil, SessionName is set, and InMemory is false,
	// the client auto-creates an in-memory storage.
	// Or wrap a custom adapter with storage.NewAdapter(adapter).
	Storage storage.Storage
	// WebSocket routes MTProto traffic over a WebSocket connection instead of
	// plain TCP. Useful behind restrictive firewalls that block raw TCP but
	// allow HTTP upgrades.
	WebSocket bool
	// WebSocketTLS enables TLS encryption on the WebSocket transport. Should
	// be true in production to prevent MITM attacks on the WS connection.
	WebSocketTLS bool
	// ServerAddr is an optional override for the DC address to connect to.
	// When set, the client dials this address directly instead of resolving
	// the DC address from the built-in datacenter map. Format: "host:port".
	//
	// Example:
	//
	//	cfg.ServerAddr = "149.154.167.50:443"
	ServerAddr string
	// LocalAddr is the local network address to bind when dialing the server.
	// Useful on multi-homed hosts that need to pin outbound connections to a
	// specific interface. Format: "host:port" (use :0 for auto port).
	//
	// Example:
	//
	//	cfg.LocalAddr = "192.168.1.100:0"
	LocalAddr string
	// Log configures MTProto-level logging for the client. Use it to capture
	// protocol events for debugging connection or authentication issues.
	Log LogConfig

	// ReconnectEnabled enables automatic reconnection when the underlying
	// transport is interrupted. When true, the client retries with exponential
	// backoff up to ReconnectMaxAttempts. Defaults to true.
	ReconnectEnabled bool
	// ReconnectBaseDelay is the initial delay before the first reconnection
	// attempt. Subsequent attempts double the delay until ReconnectMaxDelay
	// is reached. Defaults to 1 second.
	ReconnectBaseDelay time.Duration
	// ReconnectMaxDelay caps the exponential backoff delay between reconnection
	// attempts. Defaults to 60 seconds.
	ReconnectMaxDelay time.Duration
	// ReconnectMaxAttempts is the maximum number of reconnection tries before
	// giving up and reporting a permanent failure. A value of 0 means unlimited
	// retries.
	ReconnectMaxAttempts int
	// HealthEnabled activates periodic health-check pings to the server to
	// detect stale connections early. When false, disconnections are only
	// discovered on the next RPC call. Defaults to true.
	HealthEnabled bool
	// HealthPingInterval is the time between successive health-check pings.
	// Shorter intervals detect failures faster but consume more bandwidth.
	// Defaults to 60 seconds.
	HealthPingInterval time.Duration
	// HealthPongTimeout is the maximum time to wait for a pong response
	// before treating the connection as dead and triggering a reconnect.
	// Defaults to 30 seconds.
	HealthPongTimeout time.Duration

	// UpdateQueueSize is the buffered channel capacity for incoming updates.
	// Larger values absorb bursts but increase memory usage. Defaults to 1024.
	UpdateQueueSize int
	// DurableUpdateQueue persists undelivered updates across reconnects so
	// that no update is lost during brief network outages. Defaults to true.
	DurableUpdateQueue bool
	// MaxUpdateHandlerRetry is the number of times the client will retry
	// calling an update handler that returned an error. After exhausting
	// retries the update is dropped. Defaults to 3.
	MaxUpdateHandlerRetry int
	// UpdateRecoveryEnabled restores updates that may have been lost during
	// a reconnection by fetching missed events from the server. Defaults to true.
	UpdateRecoveryEnabled bool
}

// DefaultConfig provides production-ready defaults for a new client. Override
// individual fields as needed—any zero-value field will fall back to the value
// specified here.
//
// Example:
//
//	cfg := telegram.DefaultConfig
//	cfg.SessionName = "my_session"
//	cfg.InMemory = true
//	client, err := telegram.NewClient(apiID, apiHash, &cfg)
var DefaultConfig = Config{
	SleepThreshold:      10 * time.Second,
	Timeout:             60 * time.Second,
	ReqTimeout:          60 * time.Second,
	MaxConcurrentTrans:  1,
	DispatchQueueSize:   defaultDispatchQueueSize,
	MaxMessageCacheSize: 1000,
	MaxTopicCacheSize:   1000,
	PeerCacheSize:       5000,
	Device: DeviceConfig{
		DeviceModel:    "MTGo",
		SystemVersion:  "1.0.0",
		AppVersion:     "1.0.0",
		LangPack:       "tdesktop",
		LangCode:       "en",
		SystemLangCode: "en",
		ClientPlatform: types.ClientPlatformAndroid,
	},
	SkipUpdates:           true,
	TransportMode:         TransportModeAbridged,
	SavePeers:             true,
	WebSocketTLS:          true,
	FetchReplies:          true,
	FetchTopics:           true,
	FetchStories:          true,
	FetchStickers:         true,
	ReconnectEnabled:      true,
	ReconnectBaseDelay:    1 * time.Second,
	ReconnectMaxDelay:     60 * time.Second,
	HealthEnabled:         true,
	HealthPingInterval:    60 * time.Second,
	HealthPongTimeout:     30 * time.Second,
	UpdateQueueSize:       1024,
	DurableUpdateQueue:    true,
	MaxUpdateHandlerRetry: 3,
	UpdateRecoveryEnabled: true,
}
