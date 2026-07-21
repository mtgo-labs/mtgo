package session

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type sessionLogger interface {
	Debugf(format string, v ...any)
	Warnf(format string, v ...any)
	Errorf(format string, v ...any)
}

// authKeyLength is the required size in bytes of an MTProto authorization key.
const authKeyLength = 256

// Transport abstracts the underlying network transport used by a Session to
// send and receive raw byte payloads. Implementations must be safe for
// concurrent use via Send and Recv from separate goroutines.
type Transport interface {
	// Send writes a raw encrypted payload to the transport.
	Send(data []byte) error
	// Recv blocks until a raw encrypted payload is received.
	Recv() ([]byte, error)
	// Close terminates the transport connection and releases resources.
	// Implementations must guarantee that Close unblocks any in-flight
	// Send or Recv calls, causing them to return an error.
	Close() error
	// IsConnected reports whether the transport is currently connected.
	IsConnected() bool
	// SetWriteDeadline sets the write deadline on the underlying connection.
	SetWriteDeadline(t time.Time) error
	// SetReadDeadline sets the read deadline on the underlying connection.
	SetReadDeadline(t time.Time) error
}

type httpWaitTransport interface {
	HTTPWaitParams() (maxDelay, waitAfter, maxWait int32, enabled bool)
	StartHTTPWait(frame func(context.Context) ([]byte, error))
}

const (
	numFutureSalts       = 4
	initialSaltFetchWait = 15 * time.Second
	saltFetchInterval    = time.Hour
	ackFlushInterval     = 30 * time.Second
	slowWriteThreshold   = 3 * time.Second
)

// hasDecodedResults returns true if any goroutine is waiting for a decoded TL
// RPC result. Raw result waiters are tracked separately so they do not force
// TL decoding or gzip unpacking.
func (s *Session) hasDecodedResults() bool {
	return s.pending.HasAnyDecoded()
}

func (s *Session) checkWrite() error {
	switch s.sm.State() {
	case StateActive, StateConnecting:
		return nil
	case StateClosed:
		return ErrSessionClosed
	case StateDraining:
		return ErrDraining
	default:
		return ErrNotConnected
	}
}

// AuthFunc is a function that performs key generation/authentication against
// the server using the provided transport. It returns an AuthResult containing
// the established auth key, server salt, and server time.
type AuthFunc func(transport Transport) (*AuthResult, error)

// sessionShutdownCause captures the first error that terminated a session
// for diagnostic purposes. Stored atomically via Session.shutdownCause;
// first writer wins so cascading goroutine exits never overwrite the root cause.
type sessionShutdownCause struct {
	err    error
	source string
	at     time.Time
}

// Session manages an encrypted MTProto session with a Telegram data center.
// It handles message ID and sequence number generation, encrypted message
// packing/unpacking, RPC invocation with retries, keep-alive pings, and
// dispatching of server-initiated updates.
type Session struct {
	// dc is the data center this session is connected to.
	dc DataCenter
	// storage persists session data such as the auth key.
	storage storage.Storage
	// deviceModel is the device model string sent during initialization.
	deviceModel string
	// appVersion is the application version string sent during initialization.
	appVersion string
	// systemLang is the system language code sent during initialization.
	systemLang string
	// langCode is the language pack code sent during initialization.
	langCode string

	// authKey is the 256-byte authorization key shared with the server.
	authKey []byte
	// authKeyID is the lower 64 bits of SHA1(authKey), used to identify the
	// auth key in encrypted MTProto packets.
	authKeyID []byte
	// saltMgr manages the current and future server salts, tracking
	// validity windows and scheduling proactive refresh.
	saltMgr *saltManager
	// sessionID is a random identifier for this session, unique per connection.
	sessionID int64
	// sidBytes is the little-endian encoding of sessionID, cached to avoid
	// allocating on every Send call. Populated at construction time.
	sidBytes [8]byte

	// msgFactory generates unique message IDs and sequence numbers.
	msgFactory *MsgFactory

	// msgIDValidator checks incoming server msg_ids for parity, replay,
	// and temporal validity as required by the MTProto security guidelines.
	msgIDValidator *msgIDValidator

	// pending manages all outstanding RPC call lifecycles.
	pending *PendingManager

	// sm is the session connection state machine.
	sm *stateMachine
	// mu protects the mutable config fields below: authKey, authKeyID,
	// transport, pingInterval, onUpdate, onPanic, onNewSession.
	mu sync.RWMutex

	// transport is the underlying network transport for sending/receiving data.
	transport Transport
	// pfs manages the PFS temporary auth key lifecycle. nil when PFS is disabled.
	pfs *TempKeyManager
	// writeMux serializes writes to the transport. Every outbound message
	// (RPC, service, ack, ping) acquires this mutex, writes directly, and
	// releases it. No goroutine hop, no channel, no silent drops.
	writeMux sync.Mutex

	// pingInterval controls how often keep-alive pings are sent.
	pingInterval time.Duration
	// pongTimeout is how long to wait for a pong before considering the
	// connection dead.
	pongTimeout time.Duration
	// pingMux protects pingCbs.
	pingMux sync.Mutex
	// pingCbs maps pingID to a channel that is closed when the matching
	// pong is received.
	pingCbs map[int64]chan struct{}
	// Ping health values are lock-free because they are read by observability
	// while the ping loop updates them.
	rttEWMA     atomic.Int64
	rttVariance atomic.Int64
	lastPong    atomic.Int64
	onRTT       func(time.Duration, time.Duration)
	// ackCh is a channel consumed by ackLoop to batch and send message
	// acknowledgments.
	ackCh chan int64
	// done is closed exactly once in handleClose, signalling Send/SendRaw
	// and service message writes to abort.
	done chan struct{}

	// runCancel cancels the context used by the background errgroup when
	// Start/StartContext is used.
	runCancel context.CancelFunc
	// runDone is closed when the background run exits.
	runDone chan struct{}
	// group holds the errgroup created during runInit. runLoop adds remaining
	// goroutines to it.
	group *errGroup

	// onUpdate is called when the server pushes unsolicited updates.
	onUpdate func(tg.TLObject)
	// updateSem bounds the number of concurrent update dispatch goroutines.
	// dispatchSem bounds the number of concurrent dispatchRaw goroutines to
	// prevent goroutine explosion under message flood (#23).
	updateSem   chan struct{}
	dispatchSem chan struct{}
	// onPanic is called (if non-nil) when a dispatch goroutine panics.
	onPanic func(panicValue any)
	// onNewSession is called when the server sends a new_session_created
	// notification, indicating the previous session was destroyed. The
	// callback fires in the receive goroutine; long work should be
	// dispatched to a new goroutine.
	onNewSession func(firstMsgID int64, uniqueID int64, serverSalt int64)
	// log receives structured log output. When nil, logging is suppressed.
	log sessionLogger

	consecWriteFailures   atomic.Int32
	writeBreakerThreshold atomic.Int32
	writeBreakerOpen      atomic.Bool
	shutdownCause         atomic.Pointer[sessionShutdownCause]

	// Production-hardening fields (end of struct to keep hot-path fields in
	// the first 6 cache lines — only accessed when features are enabled).
	containerTracker *ContainerTracker
	outboundBatcher  *OutboundBatcher
	stateReqMu       sync.Mutex
	stateReqs        map[int64]*pendingStateReq

	// chainMu protects chains, used for G13 invokeAfterMsg ordering.
	chainMu sync.Mutex
	chains  map[int64]int64 // chainID → last sent msg_id
}

// SetOnPanic sets a callback invoked when a dispatchUpdate goroutine panics.
func (s *Session) SetOnPanic(fn func(panicValue any)) {
	s.mu.Lock()
	s.onPanic = fn
	s.mu.Unlock()
}

// SetOnNewSession registers a callback that fires when the server sends a
// new_session_created notification. This indicates that a previous session
// was destroyed and any updates received in between may have been lost.
// The callback fires in the receive goroutine — dispatch long work to a
// new goroutine to avoid blocking the receive loop.
func (s *Session) SetOnNewSession(fn func(firstMsgID int64, uniqueID int64, serverSalt int64)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onNewSession = fn
}

// EnableOutboundBatching creates and starts an OutboundBatcher on this session.
// When enabled, Send coalesces low-priority RPCs into msg_container messages;
// high-priority RPCs continue to write directly.
// Pass maxContainerBytes=0 for the default 1 MiB, and coalesceWindow=0 for the
// default 10ms. Call before sending RPCs. To disable, call CloseOutboundBatching.
func (s *Session) EnableOutboundBatching(maxContainerBytes int, coalesceWindow time.Duration) {
	b := NewOutboundBatcher(s, maxContainerBytes, coalesceWindow)
	b.Start()
	s.mu.Lock()
	s.outboundBatcher = b
	s.mu.Unlock()
}

// CloseOutboundBatching stops the outbound batcher if active.
func (s *Session) CloseOutboundBatching() {
	s.mu.Lock()
	b := s.outboundBatcher
	s.outboundBatcher = nil
	s.mu.Unlock()
	if b != nil {
		b.Close()
	}
}

// BatcherSnapshot returns the outbound batcher's metrics for introspection
// (FR-020). Returns zero values when batching is disabled.
func (s *Session) BatcherSnapshot() OutboundSnapshot {
	s.mu.RLock()
	b := s.outboundBatcher
	s.mu.RUnlock()
	if b == nil {
		return OutboundSnapshot{}
	}
	return b.Snapshot()
}

// SessionDone returns a channel that is closed when the background Run loop
// exits. Callers can use this to detect session termination.
func (s *Session) SessionDone() <-chan struct{} {
	return s.runDone
}

// ShutdownCause returns the first error that terminated the session, the
// goroutine source that detected it, and the time it was recorded. Returns
// nil, "", time.Time{} if the session has not terminated or was stopped
// without a real error (e.g., explicit Stop / context cancellation).
func (s *Session) ShutdownCause() (source string, at time.Time, err error) {
	cause := s.shutdownCause.Load()
	if cause == nil {
		return "", time.Time{}, nil
	}
	return cause.source, cause.at, cause.err
}

// SetPingInterval configures the keep-alive ping interval. Must be called
// before Connect/Start. Zero or negative disables pings.
func (s *Session) SetPingInterval(d time.Duration) {
	s.mu.Lock()
	s.pingInterval = d
	s.mu.Unlock()
}

// SetPongTimeout configures how long to wait for a pong before considering the
// connection dead. Must be called before Connect/Start.
func (s *Session) SetPongTimeout(d time.Duration) {
	s.mu.Lock()
	s.pongTimeout = d
	s.mu.Unlock()
}

// SetOnRTT registers a lightweight callback for ping RTT updates. The callback
// runs in the ping goroutine and must not block.
func (s *Session) SetOnRTT(fn func(rtt, variation time.Duration)) {
	s.mu.Lock()
	s.onRTT = fn
	s.mu.Unlock()
}

// SessionHealthSnapshot is a lock-free view of keepalive health.
type SessionHealthSnapshot struct {
	RTT       time.Duration
	Variation time.Duration
	LastPong  time.Time
}

// HealthSnapshot returns the current smoothed ping RTT and last pong time.
func (s *Session) HealthSnapshot() SessionHealthSnapshot {
	lastPong := s.lastPong.Load()
	snapshot := SessionHealthSnapshot{
		RTT:       time.Duration(s.rttEWMA.Load()),
		Variation: time.Duration(s.rttVariance.Load()),
	}
	if lastPong > 0 {
		snapshot.LastPong = time.Unix(0, lastPong)
	}
	return snapshot
}

func (s *Session) SetSaltRefreshRatio(r float64) {
	s.saltMgr.SetRefreshRatio(r)
}

func (s *Session) SetSaltRefreshMin(d time.Duration) {
	s.saltMgr.SetRefreshMin(d)
}

func (s *Session) SetLogger(l sessionLogger) {
	s.mu.Lock()
	s.log = l
	s.mu.Unlock()
	if s.containerTracker != nil {
		s.containerTracker.SetLogger(l)
	}
}

func (s *Session) SetWriteBreakerThreshold(n int) {
	s.writeBreakerThreshold.Store(int32(n))
}

// ResetWriteBreaker clears the write circuit breaker, resetting the failure
// counter and re-enabling writes. Safe to call from any goroutine.
// Call after session reconnection or when the transport is known-good.
func (s *Session) ResetWriteBreaker() {
	s.consecWriteFailures.Store(0)
	s.writeBreakerOpen.Store(false)
}

// recordShutdownCause stores the first terminating error for diagnostic
// access and logs it exactly once. Uses atomic CompareAndSwap to guarantee
// first-error-wins: subsequent calls from cascading goroutine exits are
// silent no-ops.
func (s *Session) recordShutdownCause(err error, source string) {
	if err == nil {
		return
	}
	cause := &sessionShutdownCause{err: err, source: source, at: time.Now()}
	if !s.shutdownCause.CompareAndSwap(nil, cause) {
		return
	}
	if s.log != nil {
		s.log.Errorf("session: shutdown cause: source=%s dc=%d session_id=%d: %v",
			source, s.dc.ID, s.sessionID, err)
	}
}

// wrapGoroutine wraps an errgroup goroutine function so that the first real
// (non-context) error it returns is recorded as the session's shutdown cause.
// context.Canceled and context.DeadlineExceeded are filtered because they are
// reactive exits caused by another goroutine's failure, not the root cause.
func (s *Session) wrapGoroutine(name string, f func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		err := f(ctx)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			s.recordShutdownCause(err, name)
		}
		return err
	}
}

// trackWriteResult updates the write circuit breaker. After threshold
// consecutive failures the breaker opens and the errgroup is cancelled to
// trigger immediate session shutdown + reconnect (fail-fast). Successful
// writes reset both counter and breaker flag.
func (s *Session) trackWriteResult(err error) {
	threshold := int(s.writeBreakerThreshold.Load())
	if threshold <= 0 {
		return
	}
	if err != nil {
		newCount := s.consecWriteFailures.Add(1)
		if int(newCount) >= threshold && !s.writeBreakerOpen.Load() {
			s.writeBreakerOpen.Store(true)
			s.recordShutdownCause(fmt.Errorf("write circuit breaker: %d consecutive write failures", newCount), "writeBreaker")
			if s.log != nil {
				s.log.Errorf("session: write circuit breaker tripped after %d consecutive failures, forcing reconnect", newCount)
			}
			// Fail-fast: cancel the errgroup to trigger immediate session
			// shutdown + reconnect via handleClose, rather than lingering
			// until the read deadline expires (~60-120s).
			if s.group != nil {
				s.group.Cancel()
			}
		}
	} else {
		s.consecWriteFailures.Store(0)
		if s.writeBreakerOpen.Load() {
			s.writeBreakerOpen.Store(false)
		}
	}
}

func computeAuthKeyID(authKey []byte) []byte {
	h := sha1.Sum(authKey)
	id := make([]byte, 8)
	copy(id, h[12:20])
	return id
}

// computeAuthKeyIDInt64 computes the auth key ID as an int64 value.
// Used for auth.bindTempAuthKey which expects int64 perm_auth_key_id.
func computeAuthKeyIDInt64(authKey []byte) int64 {
	h := sha1.Sum(authKey)
	return int64(h[12]) | int64(h[13])<<8 | int64(h[14])<<16 | int64(h[15])<<24 |
		int64(h[16])<<32 | int64(h[17])<<40 | int64(h[18])<<48 | int64(h[19])<<56
}

// NewSession creates a new Session for the given data center. It loads the
// persisted auth key from storage and generates a random session ID.
// Returns an error if the auth key cannot be loaded or random ID generation
// fails.
func NewSession(dc DataCenter, st storage.Storage, deviceModel, appVersion, systemLang, langCode string) (*Session, error) {
	var sidBytes [8]byte
	if _, err := rand.Read(sidBytes[:]); err != nil {
		return nil, err
	}
	sid := int64(binary.LittleEndian.Uint64(sidBytes[:]))

	authKey, err := st.AuthKey()
	if err != nil {
		return nil, err
	}
	// A non-empty auth key must be exactly 256 bytes; a corrupted/truncated
	// persisted key would otherwise panic later in the crypto Pack/Unpack paths
	// (which index authKey[88:120] etc.). Reject it here with a clean error.
	if len(authKey) != 0 && len(authKey) != authKeyLength {
		return nil, fmt.Errorf("session: invalid stored auth key length %d, expected %d", len(authKey), authKeyLength)
	}

	var encodedSidBytes [8]byte
	binary.LittleEndian.PutUint64(encodedSidBytes[:], uint64(sid))

	s := &Session{
		dc:          dc,
		storage:     st,
		deviceModel: deviceModel,
		appVersion:  appVersion,
		systemLang:  systemLang,
		langCode:    langCode,
		authKey:     authKey,
		sessionID:   sid,
		sidBytes:    encodedSidBytes,
		msgFactory:  NewMsgFactory(time.Now()),
		msgIDValidator: newMsgIDValidator(func() int64 {
			return time.Now().Unix()
		}),
		pending:          NewPendingManager(),
		containerTracker: NewContainerTracker(),
		pingInterval:     60 * time.Second,
		pongTimeout:      30 * time.Second,
		updateSem:        make(chan struct{}, 128),
		dispatchSem:      make(chan struct{}, 64),
		saltMgr:          newSaltManager(time.Now),
		sm:               newStateMachine(),
		chains:           make(map[int64]int64),
	}

	s.writeBreakerThreshold.Store(10)

	if len(authKey) > 0 {
		s.authKeyID = computeAuthKeyID(authKey)
	}

	return s, nil
}

// SetDispatchConfig is kept for API compatibility but is a no-op. The session
// now uses goroutine-per-message dispatch instead of a worker pool.
func (s *Session) SetDispatchConfig(_, _ int) {}

func (s *Session) addAck(msgID int64) {
	select {
	case s.ackCh <- msgID:
	case <-s.done:
	}
}

func requiresAck(seqNo uint32) bool {
	return seqNo&1 != 0
}

func (s *Session) TrackContainer(containerMsgID int64, childMsgIDs []int64) {
	s.containerTracker.TrackContainer(containerMsgID, childMsgIDs)
}

// SetOnDisconnect is kept for API compatibility but is a no-op. The session
// now uses errgroup sibling goroutines; Run() returns on failure.
func (s *Session) SetOnDisconnect(func(error)) {}

// SetAuthKey replaces the current authorization key and recomputes its ID.
// Passing an empty slice clears the key and its ID.
func (s *Session) SetAuthKey(key []byte) {
	key = bytes.Clone(key)
	s.mu.Lock()
	s.authKey = key
	if len(key) > 0 {
		s.authKeyID = computeAuthKeyID(key)
	} else {
		s.authKeyID = nil
	}
	s.mu.Unlock()
}

// SwapAuthKey atomically replaces the active auth key. Used by PFS to switch
// from the permanent key to the temporary key after binding succeeds.
// The caller must ensure no in-flight messages depend on the old key.
func (s *Session) SwapAuthKey(newKey []byte) {
	newKey = bytes.Clone(newKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authKey = newKey
	s.authKeyID = computeAuthKeyID(newKey)
}

// PFS returns the PFS temp key manager, or nil if PFS is not enabled.
func (s *Session) PFS() *TempKeyManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pfs
}

// SetPFS attaches a PFS temp key manager to this session.
func (s *Session) SetPFS(mgr *TempKeyManager) {
	s.mu.Lock()
	s.pfs = mgr
	s.mu.Unlock()
}

// SetServerSalt updates the server salt used for encrypting outgoing messages.
func (s *Session) SetServerSalt(salt int64) {
	s.saltMgr.StoreSimple(salt)
}

// ServerSalt returns the current server salt used for encrypted outgoing messages.
func (s *Session) ServerSalt() int64 {
	return s.saltMgr.Load()
}

// SetServerTime updates the internal message ID generator with the server's
// reported time to keep message IDs monotonically increasing.
func (s *Session) SetServerTime(t time.Time) {
	s.msgFactory.UpdateServerTime(t)
}

// DC returns the data center this session is associated with.
func (s *Session) DC() DataCenter {
	return s.dc
}

// SessionID returns the random session identifier for this session.
func (s *Session) SessionID() int64 {
	return s.sessionID
}

// AuthKey returns a copy of the current 256-byte authorization key, or nil if
// no key is set.
func (s *Session) AuthKey() []byte {
	s.mu.RLock()
	authKey := s.authKey
	s.mu.RUnlock()
	if len(authKey) == 0 {
		return nil
	}
	cp := make([]byte, len(authKey))
	copy(cp, authKey)
	return cp
}

// IsConnected reports whether the session is currently active and connected.
func (s *Session) IsConnected() bool {
	return s.sm.isActive()
}

// SetTransport replaces the underlying transport used for sending and
// receiving encrypted payloads.
func (s *Session) SetTransport(t Transport) {
	s.mu.Lock()
	s.transport = t
	s.mu.Unlock()
}

func (s *Session) sessionIDBytes() []byte {
	return s.sidBytes[:]
}

// Send encrypts and sends a TLObject as a single MTProto message, then waits
// for the server's response. The msgID and seqNo identify the message.
// ctx is used for cancellation: when cancelled after the message has been sent,
// an RPCDropAnswerRequest is fired to inform the server the request is no
// longer needed. Returns the raw response bytes or an error.
func (s *Session) Send(ctx context.Context, msgID int64, seqNo uint32, body tg.TLObject, timeout time.Duration) (tg.TLObject, error) {
	if err := s.checkWrite(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	authKey := s.authKey
	authKeyID := s.authKeyID
	transport := s.transport
	s.mu.RUnlock()
	if len(authKey) == 0 {
		return nil, ErrAuthKeyNotSet
	}
	if transport == nil {
		return nil, ErrTransportNotSet
	}

	// When the outbound batcher is enabled, delegate low-priority bulk work to
	// it. High-priority interactive RPCs write directly to avoid the coalescing
	// window.
	// Read under s.mu.RLock to avoid racing with EnableOutboundBatching/
	// CloseOutboundBatching which write s.outboundBatcher under s.mu (#6).
	s.mu.RLock()
	b := s.outboundBatcher
	s.mu.RUnlock()
	if b != nil && RoutePriority(body) == PriorityLow {
		handle, err := b.Submit(ctx, msgID, seqNo, body, PriorityLow, timeout)
		if err != nil {
			return nil, fmt.Errorf("session: send (batched): %w", err)
		}
		return s.waitResponse(ctx, handle, msgID, timeout)
	}

	salt := s.ensureFreshSalt(ctx)
	if salt == 0 {
		salt = s.saltMgr.Load()
	}

	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: seqNo,
		Body:  body,
	}

	encrypted, err := crypto.Pack(message, salt, s.sessionIDBytes(), authKey, authKeyID)
	if err != nil {
		return nil, fmt.Errorf("session: pack message: %w", err)
	}

	handle, regErr := s.pending.Register(msgID, false)
	if regErr != nil {
		return nil, fmt.Errorf("session: send: %w", regErr)
	}

	// Store encrypted payload for msg_resend_req re-transmission.
	handle.StorePayload(encrypted)

	if err := s.writeEncrypted(ctx, encrypted, timeout); err != nil {
		s.pending.Cancel(msgID)
		return nil, fmt.Errorf("session: send: %w", deliveryFailure(handle, err))
	}

	return s.waitResponse(ctx, handle, msgID, timeout)
}

// waitResponse blocks until the pending RPC resolves, the context is cancelled,
// the session closes, or the timeout fires. Shared by both the direct Send path
// and the batched path.
func (s *Session) waitResponse(ctx context.Context, handle *CallHandle, msgID int64, timeout time.Duration) (tg.TLObject, error) {
	respTimer := time.NewTimer(timeout)
	defer respTimer.Stop()
	select {
	case <-handle.Done():
		obj, _, err := handle.Result()
		return obj, deliveryFailure(handle, err)
	case <-ctx.Done():
		s.pending.Cancel(msgID)
		s.sendRPCDrop(msgID)
		return nil, ctx.Err()
	case <-s.done:
		s.pending.Reject(msgID, ErrSessionClosed)
		<-handle.Done()
		obj, _, err := handle.Result()
		return obj, deliveryFailure(handle, err)
	case <-respTimer.C:
		s.pending.Cancel(msgID)
		return nil, deliveryFailure(handle, ErrSendTimeout)
	}
}

func deliveryFailure(handle *CallHandle, err error) error {
	if err == nil || errors.Is(err, ErrMsgNotReceived) || errors.Is(err, ErrRPCDropped) {
		return err
	}
	var deliveryErr *DeliveryError
	if errors.As(err, &deliveryErr) {
		return err
	}
	state := DeliveryUnknown
	if handle != nil && handle.IsAcked() {
		state = DeliveryReceived
	}
	return &DeliveryError{State: state, Err: err}
}

func (s *Session) sendRPCDrop(reqMsgID int64) {
	select {
	case <-s.done:
		return
	default:
	}
	s.mu.RLock()
	authKey := s.authKey
	authKeyID := s.authKeyID
	s.mu.RUnlock()
	drop := &tg.RPCDropAnswerRequest{ReqMsgID: reqMsgID}
	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(false)
	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: uint32(seqNo),
		Body:  drop,
	}
	encrypted, err := crypto.Pack(message, s.saltMgr.Load(), s.sessionIDBytes(), authKey, authKeyID)
	if err != nil {
		return
	}
	_ = s.writeEncryptedDirect(encrypted, 5*time.Second)
}

func (s *Session) handlePacket(msgID int64, seqNo uint32, body tg.TLObject) {
	obj := body
	if gz, ok := body.(*tg.GzipPacked); ok {
		decoded, err := gz.Decode()
		if err != nil {
			if s.log != nil {
				s.log.Warnf("gzip decode failed: %v", err)
			}
			return
		}
		obj = decoded
	}

	switch v := obj.(type) {
	case *tg.Pong:
		s.handlePong(v.PingID)
		s.pending.Resolve(v.MsgID, v)
	case tg.BadMsgNotificationClass:
		switch bv := v.(type) {
		case *tg.BadMsgNotification:
			s.pending.Resolve(bv.BadMsgID, bv)
		case *tg.BadServerSalt:
			s.saltMgr.StoreSimple(bv.NewServerSalt)
			s.pending.Resolve(bv.BadMsgID, bv)
		}
	case *tg.NewSessionCreated:
		s.saltMgr.StoreSimple(v.ServerSalt)
		s.fireNewSession(v.FirstMsgID, v.UniqueID, v.ServerSalt)
	case *tg.FutureSalts:
		s.storeFutureSalts(v)
	case *tg.MsgsAck:
		for _, ackedID := range v.MsgIds {
			s.containerTracker.AckContainer(ackedID)
			s.containerTracker.AckChild(ackedID)
		}
	case *tg.RPCResult:
		result := v.Result
		if gz, ok := result.(*tg.GzipPacked); ok {
			decoded, err := gz.Decode()
			if err != nil {
				if s.log != nil {
					s.log.Warnf("gzip decode rpc result failed: %v", err)
				}
				return
			}
			result = decoded
		}
		s.pending.Resolve(v.ReqMsgID, result)
	case tg.UpdatesClass:
		s.mu.RLock()
		fn := s.onUpdate
		s.mu.RUnlock()
		if fn != nil {
			s.dispatchUpdate(obj)
		}
	default:
		if s.pending.Has(msgID) {
			s.pending.Resolve(msgID, obj)
		} else {
			s.mu.RLock()
			fn := s.onUpdate
			s.mu.RUnlock()
			if fn != nil {
				s.dispatchUpdate(obj)
			}
		}
	}
}

// dispatchUpdate spawns a goroutine to deliver an update to the handler,
// bounded by the updateSem semaphore. Saturation applies backpressure to the
// receive loop so an acknowledged update is never silently dropped.
func (s *Session) dispatchUpdate(obj tg.TLObject) {
	s.mu.RLock()
	handlerFn := s.onUpdate
	panicFn := s.onPanic
	s.mu.RUnlock()
	if handlerFn == nil {
		return
	}
	select {
	case s.updateSem <- struct{}{}:
	case <-s.done:
		return
	}
	go func() {
		defer func() { <-s.updateSem }()
		defer func() {
			if r := recover(); r != nil {
				if panicFn != nil {
					panicFn(r)
				} else {
					if s.log != nil {
						s.log.Errorf("dispatchUpdate panic: %v", r)
					}
				}
			}
		}()
		handlerFn(obj)
	}()
}

// SendRaw encrypts and sends raw body bytes as a single MTProto message, then
// waits for the matching rpc_result and returns its raw result:Object payload
// bytes. Unlike [Send], the response path does not gzip-unpack or TL-decode the
// payload.
func (s *Session) SendRaw(ctx context.Context, msgID int64, seqNo uint32, bodyBytes []byte, timeout time.Duration) ([]byte, error) {
	if err := s.checkWrite(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	authKey := s.authKey
	authKeyID := s.authKeyID
	transport := s.transport
	s.mu.RUnlock()
	if len(authKey) == 0 {
		return nil, ErrAuthKeyNotSet
	}
	if transport == nil {
		return nil, ErrTransportNotSet
	}

	salt := s.ensureFreshSalt(ctx)
	if salt == 0 {
		salt = s.saltMgr.Load()
	}

	encrypted, err := crypto.PackRaw(msgID, seqNo, bodyBytes, salt, s.sessionIDBytes(), authKey, authKeyID)
	if err != nil {
		return nil, fmt.Errorf("session: send raw: %w", err)
	}

	handle, regErr := s.pending.Register(msgID, true)
	if regErr != nil {
		return nil, fmt.Errorf("session: send raw: %w", regErr)
	}
	handle.StorePayload(encrypted)

	if err := s.writeEncrypted(ctx, encrypted, timeout); err != nil {
		s.pending.Cancel(msgID)
		return nil, fmt.Errorf("session: send raw: %w", deliveryFailure(handle, err))
	}

	respTimer := time.NewTimer(timeout)
	defer respTimer.Stop()
	select {
	case <-handle.Done():
		_, raw, err := handle.Result()
		return raw, deliveryFailure(handle, err)
	case <-ctx.Done():
		s.pending.Cancel(msgID)
		return nil, ctx.Err()
	case <-s.done:
		s.pending.Reject(msgID, ErrSessionClosed)
		<-handle.Done()
		_, raw, err := handle.Result()
		return raw, deliveryFailure(handle, err)
	case <-respTimer.C:
		s.pending.Cancel(msgID)
		return nil, deliveryFailure(handle, ErrSendTimeout)
	}
}

// InvokeRaw sends a TLObject query and returns the matching rpc_result's raw
// result:Object payload bytes without gzip unpacking or TL decoding. It retries
// the request up to retries times with the given per-attempt timeout.
func (s *Session) InvokeRaw(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) ([]byte, error) {
	var buf bytes.Buffer
	if err := query.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode query: %w", err)
	}
	bodyBytes := buf.Bytes()

	var lastErr error
	for i := 0; i < retries; i++ {
		msgID := s.msgFactory.AllocateMsgID()
		seqNo := s.msgFactory.AllocateSeqNo(true)

		data, err := s.SendRaw(ctx, msgID, uint32(seqNo), bodyBytes, timeout)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("send: %w", err)
			var deliveryErr *DeliveryError
			if errors.As(err, &deliveryErr) {
				return nil, lastErr
			}
			continue
		}

		if len(data) < 4 {
			lastErr = fmt.Errorf("rpc result too short: %d", len(data))
			continue
		}

		if err := checkRawRPCError(data); err != nil {
			return nil, err
		}

		return data, nil
	}
	if lastErr == nil {
		return nil, fmt.Errorf("retries exhausted (%d)", retries)
	}
	return nil, fmt.Errorf("retries exhausted (%d): %w", retries, lastErr)
}

func checkRawRPCError(data []byte) error {
	return checkRawRPCErrorDepth(data, 0)
}

func checkRawRPCErrorDepth(data []byte, depth int) error {
	if len(data) < 4 {
		return nil
	}
	constructorID := binary.LittleEndian.Uint32(data[:4])
	switch constructorID {
	case tg.RPCErrorTypeID:
		r := tg.NewReader(data[4:])
		defer tg.ReleaseReader(r)
		rpcErr, err := tg.DecodeRPCError(r)
		if err != nil {
			return fmt.Errorf("decode raw rpc error: %w", err)
		}
		return tgerr.New(int(rpcErr.ErrorCode), rpcErr.ErrorMessage)
	case tg.GzipPackedID:
		if depth >= 4 {
			return fmt.Errorf("decode raw rpc error: gzip nesting exceeds 4 levels")
		}
		innerConstructor, err := tg.PeekGzipPackedConstructor(data[4:])
		if err != nil {
			return fmt.Errorf("decode raw gzip payload: %w", err)
		}
		if innerConstructor != tg.RPCErrorTypeID && innerConstructor != tg.GzipPackedID {
			return nil
		}

		r := tg.NewReader(data[4:])
		gz, err := tg.DecodeGzipPacked(r)
		tg.ReleaseReader(r)
		if err != nil {
			return fmt.Errorf("decode raw gzip payload: %w", err)
		}
		payload, ok := gz.Data.(*tg.GzipPackedData)
		if !ok {
			return fmt.Errorf("decode raw gzip payload: unexpected payload type %T", gz.Data)
		}
		return checkRawRPCErrorDepth(payload.Raw, depth+1)
	default:
		return nil
	}
}

// Invoke sends an RPC query and decodes the response into a TLObject.
// It retries the request up to retries times with the given per-attempt
// timeout. Returns the decoded response object or the last error encountered.
func (s *Session) Invoke(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error) {
	methodName := typeName(query)
	chainID, hasChain := ChainIDFromContext(ctx) // G13: invokeAfterMsg chain

	var lastErr error
	var backoff time.Duration
	maxAttempts := retries
	badSaltRetries := 0
	g12Retries := 0 // G12: bound additional error-recovery retries
	for i := 0; i < maxAttempts; i++ {
		if i > 0 {
			time.Sleep(backoff)
		}
		msgID := s.msgFactory.AllocateMsgID()
		seqNo := s.msgFactory.AllocateSeqNo(true)

		sendQuery := query
		if hasChain {
			sendQuery = s.wrapChain(chainID, query)
		}
		obj, err := s.Send(ctx, msgID, uint32(seqNo), sendQuery, timeout)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("send: %w", err)
			var deliveryErr *DeliveryError
			if errors.As(err, &deliveryErr) {
				return nil, lastErr
			}
			if backoff == 0 {
				backoff = 100 * time.Millisecond
			} else {
				backoff = backoff * 2
				if backoff > 2*time.Second {
					backoff = 2 * time.Second
				}
			}
			continue
		}

		if bad, ok := obj.(tg.BadMsgNotificationClass); ok {
			switch v := bad.(type) {
			case *tg.BadMsgNotification:
				lastErr = fmt.Errorf("bad message (msg_id=%d, code=%d)", msgID, v.ErrorCode)
			case *tg.BadServerSalt:
				lastErr = fmt.Errorf("bad server salt (msg_id=%d, code=%d)", msgID, v.ErrorCode)
				if badSaltRetries == 0 && i+1 >= maxAttempts {
					badSaltRetries++
					maxAttempts++
				}
			default:
				lastErr = fmt.Errorf("bad message notification: %T", bad)
			}
			if backoff == 0 {
				backoff = 100 * time.Millisecond
			} else {
				backoff = backoff * 2
				if backoff > 2*time.Second {
					backoff = 2 * time.Second
				}
			}
			continue
		}

		if rpcErr, ok := obj.(*tg.RPCError); ok {
			if rpcErr.ErrorCode == 303 {
				return obj, nil
			}
			parsed := tgerr.New(int(rpcErr.ErrorCode), rpcErr.ErrorMessage)
			if rpcErr.ErrorCode == 420 {
				return obj, nil
			}
			// G13: MSG_WAIT_FAILED — the referenced msg in an invokeAfterMsg
			// chain failed. Clear the chain dependency and resend unwrapped
			// so the query is not permanently blocked.
			if hasChain && tgerr.Is(parsed, tgerr.ErrMSGWaitFailed) {
				s.ClearChain(chainID)
				if s.log != nil {
					s.log.Warnf("chain msg wait failed method=%s chain_id=%d, retrying unwrapped", methodName, chainID)
				}
				lastErr = fmt.Errorf("chain msg wait failed")
				backoff = 100 * time.Millisecond
				maxAttempts++
				continue
			}

			// G12: CONNECTION_NOT_INITED (400) — the server lost track of the
			// connection init state. Retry; the caller (client layer) wraps in
			// initConnection which re-establishes server-side state.
			if g12Retries < maxG12Retries && tgerr.Is(parsed, tgerr.ErrConnectionNotInited) {
				g12Retries++
				if s.log != nil {
					s.log.Warnf("connection not inited method=%s attempt=%d", methodName, i+1)
				}
				lastErr = fmt.Errorf("connection not inited")
				backoff = 100 * time.Millisecond
				maxAttempts++
				continue
			}

			// G12: PERSISTENT_TIMESTAMP_OUTDATED (400) — transient server-side
			// state issue. Brief delay then retry.
			if g12Retries < maxG12Retries && tgerr.Is(parsed, tgerr.ErrPersistentTimestampOutdated) {
				g12Retries++
				if s.log != nil {
					s.log.Warnf("persistent timestamp outdated method=%s attempt=%d", methodName, i+1)
				}
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-s.done:
					return nil, ErrSessionClosed
				case <-time.After(1 * time.Second):
				}
				lastErr = fmt.Errorf("persistent timestamp outdated")
				backoff = 0
				maxAttempts++
				continue
			}

			// G12: AUTH_KEY_PERM_EMPTY (401) — in PFS mode the temp key may
			// have been invalidated. Re-bind and retry. Without PFS this error
			// should not occur; surface it to the caller.
			if g12Retries < maxG12Retries && tgerr.Is(parsed, tgerr.ErrAuthKeyPermEmpty) {
				g12Retries++
				if pfs := s.PFS(); pfs != nil && pfs.IsEnabled() {
					if s.log != nil {
						s.log.Warnf("auth key perm empty method=%s, rebinding temp key", methodName)
					}
					if err := pfs.Bind(ctx, s.sessionID, s.Invoke); err != nil {
						return nil, fmt.Errorf("auth key perm empty: rebind failed: %w", err)
					}
					lastErr = fmt.Errorf("auth key perm empty, temp key rebound")
					backoff = 0
					maxAttempts++
					continue
				}
				// Not PFS — fall through to the 401 surface below.
			}

			if rpcErr.ErrorCode == 401 || rpcErr.ErrorCode == 400 || rpcErr.ErrorCode == 403 {
				return nil, parsed
			}
			lastErr = parsed
			if backoff == 0 {
				backoff = 100 * time.Millisecond
			} else {
				backoff = backoff * 2
				if backoff > 2*time.Second {
					backoff = 2 * time.Second
				}
			}
			continue
		}

		if hasChain {
			s.SetChain(msgID, chainID)
		}
		return obj, nil
	}
	if lastErr == nil {
		return nil, fmt.Errorf("retries exhausted (%d)", retries)
	}
	return nil, fmt.Errorf("retries exhausted (%d): %w", retries, lastErr)
}

// DropRPC sends rpc_drop_answer for the given msg_id, asking the server to
// cancel the in-flight RPC. After the server responds, the pending handle for
// msgID is rejected with ErrRPCDropped so the original caller unblocks.
//
// This is a best-effort cancel — the server may have already processed the
// request. The server's response indicates the outcome:
//   - RPCAnswerUnknown: the server has no record of msgID (already gone).
//   - RPCAnswerDroppedRunning: the RPC was running and the result was discarded.
//   - RPCAnswerDropped: the RPC result was discarded.
func (s *Session) DropRPC(ctx context.Context, msgID int64) error {
	result, err := s.Invoke(ctx, &tg.RPCDropAnswerRequest{ReqMsgID: msgID}, 1, 10*time.Second)
	if err != nil {
		return err
	}
	// Reject the pending handle so the original caller unblocks with a typed
	// error. If the handle already completed (server raced ahead), Reject is
	// a no-op.
	s.pending.Reject(msgID, ErrRPCDropped)
	if _, ok := result.(tg.RPCDropAnswerClass); !ok {
		return fmt.Errorf("session: drop rpc: unexpected result type %T", result)
	}
	return nil
}

// maxG12Retries bounds the number of G12 error-recovery retries (for
// CONNECTION_NOT_INITED, PERSISTENT_TIMESTAMP_OUTDATED, AUTH_KEY_PERM_EMPTY)
// before surfacing the error to the caller.
const maxG12Retries = 2

func typeName(v tg.TLObject) string {
	if v == nil {
		return "unknown"
	}
	switch t := v.(type) {
	case interface{ ConstructorID() uint32 }:
		return fmt.Sprintf("%T(cid=%08x)", v, t.ConstructorID())
	default:
		return fmt.Sprintf("%T", v)
	}
}

// runInit validates state, initializes internal channels, sets connected, and
// performs the initial ping to verify connectivity. It starts the errgroup
// goroutines before the ping so that readLoop can receive the pong.
func (s *Session) runInit(groupCtx context.Context) error {
	return s.runInitWithCtx(groupCtx, groupCtx)
}

// runInitWithCtx is like runInit but separates the ping timeout context from
// the errgroup context. pingCtx bounds the initial ping; groupCtx drives the
// errgroup goroutines.
func (s *Session) runInitWithCtx(pingCtx, groupCtx context.Context) error {
	s.mu.RLock()
	authKey := s.authKey
	tp := s.transport
	s.mu.RUnlock()
	if len(authKey) == 0 {
		return ErrAuthKeyNotSet
	}
	if tp == nil {
		return ErrTransportNotSet
	}

	s.ackCh = make(chan int64, 1024)
	s.pingCbs = make(map[int64]chan struct{})
	s.done = make(chan struct{})
	s.stateReqs = make(map[int64]*pendingStateReq)
	s.chains = make(map[int64]int64)
	s.consecWriteFailures.Store(0)
	s.writeBreakerOpen.Store(false)
	s.shutdownCause.Store(nil)
	s.sm.transition(StateIdle, StateConnecting)

	// Start the errgroup so readLoop is running during the initial ping.
	g := newErrGroup(groupCtx)
	g.Go(s.wrapGoroutine("readLoop", s.readLoop))
	g.Go(s.wrapGoroutine("ackLoop", s.ackLoop))

	s.group = g

	initPingCtx, initPingCancel := context.WithTimeout(pingCtx, 60*time.Second)
	_, err := s.Invoke(initPingCtx, &tg.PingRequest{PingID: time.Now().UnixNano()}, 3, timeoutFromContext(pingCtx))
	initPingCancel()
	if err != nil {
		s.recordShutdownCause(fmt.Errorf("initial ping: %w", err), "runInit")
		g.Cancel()
		close(s.done)
		_ = tp.Close()
		_ = g.Wait()
		s.sm.transition(StateConnecting, StateClosed)
		return fmt.Errorf("session: initial ping: %w", err)
	}
	if httpTransport, ok := tp.(httpWaitTransport); ok {
		maxDelay, waitAfter, maxWait, enabled := httpTransport.HTTPWaitParams()
		if enabled {
			httpTransport.StartHTTPWait(func(ctx context.Context) ([]byte, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				default:
				}
				return s.buildServiceMessage(&tg.HTTPWait{
					MaxDelay:  maxDelay,
					WaitAfter: waitAfter,
					MaxWait:   maxWait,
				})
			})
		}
	}

	s.sm.transition(StateConnecting, StateActive)

	return nil
}

// runLoop adds the remaining errgroup goroutines and blocks until the context
// is cancelled or a goroutine returns a fatal error.
func (s *Session) runLoop(ctx context.Context) error {
	g := s.group
	g.Go(s.wrapGoroutine("stateCheckLoop", s.stateCheckLoop))
	g.Go(s.wrapGoroutine("pingLoop", s.pingLoop))
	g.Go(s.wrapGoroutine("saltLoop", s.saltLoop))
	g.Go(s.handleClose)
	if pfs := s.PFS(); pfs != nil && pfs.IsEnabled() {
		g.Go(s.wrapGoroutine("pfsRenewalLoop", s.pfsRenewalLoop))
	}

	err := g.Wait()
	s.sm.transitionTo(StateClosed)
	return err
}

// Run is the main blocking entry point. It performs the initial ping
// synchronously, then starts the remaining errgroup sibling goroutines
// (pingLoop, saltLoop, handleClose) and blocks until the context is cancelled
// or a fatal error occurs.
func (s *Session) Run(ctx context.Context) error {
	if !s.sm.canConnect() {
		state := s.sm.State()
		if state == StateClosed {
			return ErrSessionClosed
		}
		return fmt.Errorf("session: run: already connected (state=%s)", state)
	}
	if err := s.runInit(ctx); err != nil {
		return err
	}
	return s.runLoop(ctx)
}

// Start launches the session in a background goroutine and returns after the
// initial ping succeeds. The timeout bounds the startup phase. Use Stop to
// terminate the background session.
func (s *Session) Start(timeout time.Duration) error {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return s.StartContext(ctx)
}

// StartContext launches the session in a background goroutine and returns
// after the initial ping succeeds. The ctx bounds the startup ping. Use Stop
// to terminate the background session.
func (s *Session) StartContext(ctx context.Context) error {
	if !s.sm.canConnect() {
		state := s.sm.State()
		if state == StateClosed {
			return ErrSessionClosed
		}
		return fmt.Errorf("session: start: already connected (state=%s)", state)
	}
	runCtx, runCancel := context.WithCancel(context.Background())
	s.runCancel = runCancel
	s.runDone = make(chan struct{})

	// Use the caller's context for ping timeout but runCtx for the errgroup
	// so the goroutines survive after the caller's context expires.
	if err := s.runInitWithCtx(ctx, runCtx); err != nil {
		runCancel()
		close(s.runDone)
		return err
	}
	go func() {
		defer close(s.runDone)
		if err := s.runLoop(runCtx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			s.recordShutdownCause(err, "runLoop")
		}
	}()
	return nil
}

// Stop cancels the background context and waits for the session to exit.
func (s *Session) Stop() {
	if s.runCancel != nil {
		s.runCancel()
	}
	if s.runDone != nil {
		<-s.runDone
	}
}

func timeoutFromContext(ctx context.Context) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		if timeout > 0 {
			return timeout
		}
	}
	return 60 * time.Second
}

func (s *Session) ensureFreshSalt(ctx context.Context) int64 {
	if !s.saltMgr.IsExpired() {
		return s.saltMgr.Load()
	}
	if s.log != nil {
		s.log.Warnf("salt pre-fetch: salt expired before send, triggering immediate refresh")
	}
	s.sendServiceMessage(&tg.GetFutureSaltsRequest{Num: numFutureSalts})

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = s.saltMgr.WaitForValid(waitCtx)
	return s.saltMgr.Load()
}

func (s *Session) storeFutureSalts(fs *tg.FutureSalts) {
	if len(fs.Salts) == 0 {
		return
	}
	s.saltMgr.StoreFromFutureSalts(saltEntriesFromFuture(fs.Salts))
}

func (s *Session) sendServiceMessage(body tg.TLObject) error {
	select {
	case <-s.done:
		return ErrSessionClosed
	default:
	}
	encrypted, err := s.buildServiceMessage(body)
	if err != nil {
		if s.log != nil {
			s.log.Errorf("session: pack service message: %v", err)
		}
		return fmt.Errorf("session: pack service message: %w", err)
	}
	if err := s.writeEncryptedDirect(encrypted, 10*time.Second); err != nil {
		if s.log != nil {
			s.log.Warnf("session: service message write failed (%T): %v", body, err)
		}
		return err
	}
	return nil
}

func (s *Session) buildServiceMessage(body tg.TLObject) ([]byte, error) {
	s.mu.RLock()
	authKey := s.authKey
	authKeyID := s.authKeyID
	s.mu.RUnlock()
	message := &tg.MTProtoMessage{
		MsgID: s.msgFactory.AllocateMsgID(),
		SeqNo: uint32(s.msgFactory.AllocateSeqNo(false)),
		Body:  body,
	}
	return crypto.Pack(message, s.saltMgr.Load(), s.sessionIDBytes(), authKey, authKeyID)
}

// writeEncryptedImpl is the shared implementation for writeEncrypted and
// writeEncryptedDirect. It snapshots transport state, acquires writeMux, sets
// the write deadline, writes the encrypted payload, and releases the mutex.
func (s *Session) writeEncryptedImpl(deadline time.Time, encrypted []byte) error {
	if s.writeBreakerThreshold.Load() > 0 && s.writeBreakerOpen.Load() {
		return ErrWriteCircuitOpen
	}

	s.mu.RLock()
	tp := s.transport
	s.mu.RUnlock()
	if tp == nil {
		return ErrTransportNotSet
	}

	lockStart := time.Now()
	s.writeMux.Lock()
	if wait := time.Since(lockStart); wait > slowWriteThreshold && s.log != nil {
		s.log.Warnf("session: slow write lock wait: %v", wait)
	}
	defer s.writeMux.Unlock()

	if s.writeBreakerThreshold.Load() > 0 && s.writeBreakerOpen.Load() {
		return ErrWriteCircuitOpen
	}

	select {
	case <-s.done:
		return ErrSessionClosed
	default:
	}

	tp.SetWriteDeadline(deadline)
	writeStart := time.Now()
	err := tp.Send(encrypted)
	if elapsed := time.Since(writeStart); elapsed > slowWriteThreshold && s.log != nil {
		s.log.Warnf("session: slow transport write: %v", elapsed)
	}
	tp.SetWriteDeadline(time.Time{})
	s.trackWriteResult(err)
	return err
}

// writeEncrypted snapshots transport state, acquires writeMux, sets the write
// deadline from ctx or timeout (whichever is sooner), writes the encrypted
// payload, and releases the mutex. Returns the transport error, if any.
// Lock ordering: mu is always acquired BEFORE writeMux, never inside it.
func (s *Session) writeEncrypted(ctx context.Context, encrypted []byte, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	return s.writeEncryptedImpl(deadline, encrypted)
}

// writeEncryptedDirect is the context-less variant used by service messages
// and RPCDropAnswer where there is no caller context to respect.
func (s *Session) writeEncryptedDirect(encrypted []byte, timeout time.Duration) error {
	return s.writeEncryptedImpl(time.Now().Add(timeout), encrypted)
}

// readLoop is an errgroup goroutine that reads from the transport and handles
// or dispatches incoming messages.
func (s *Session) readLoop(ctx context.Context) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			if s.log != nil {
				s.log.Errorf("readLoop panic: %v", r)
			}
			s.pending.RejectAll(ErrSessionClosed)
			retErr = fmt.Errorf("session: readLoop panic: %v", r)
		}
	}()

	readTimeout := s.pingInterval * 2
	if readTimeout < 30*time.Second {
		readTimeout = 30 * time.Second
	}

	var handlers sync.WaitGroup
	defer handlers.Wait()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		s.mu.RLock()
		tp := s.transport
		authKey := s.authKey
		authKeyID := s.authKeyID
		updateFn := s.onUpdate
		s.mu.RUnlock()

		tp.SetReadDeadline(time.Now().Add(readTimeout))
		data, err := tp.Recv()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if !s.sm.isActive() {
				return nil
			}
			if isTimeoutError(err) {
				return fmt.Errorf("session: read deadline exceeded: %w", err)
			}
			return err
		}
		if te := transport.DetectTransportError(data); te != nil {
			// Transport error from server (auth key not found, flood,
			// invalid DC, etc.). Propagate to close the session so the
			// reconnect logic can inspect the error code.
			// See https://core.telegram.org/mtproto/mtproto-transports#transport-errors
			s.pending.RejectAll(te)
			return te
		}
		if transport.IsQuickAckToken(data) {
			// Quick ACK token from server (4 bytes, bit 31 set).
			// Indicates the server received and accepted a previously sent
			// payload for processing. Does not indicate RPC completion — the
			// server will still send msgs_ack and the RPC result separately.
			// See https://core.telegram.org/mtproto/mtproto-transports#quick-ack
			continue
		}

		raw, decrypted, err := unpackIncomingMessageEnvelope(data, s.sessionIDBytes(), authKey, authKeyID)
		if err != nil {
			if _, ok := err.(*tgerr.SecurityCheckMismatch); ok {
				return err
			}
			continue
		}

		if !s.msgIDValidator.Check(raw.MsgID) {
			crypto.ReleaseAESBuf(decrypted)
			continue
		}

		// Continuous server-time recalibration: the high 32 bits of every
		// server-originated msg_id encode the server's unix time at send.
		// Monotonically nudging the offset from each inbound message keeps it
		// accurate for the session lifetime without waiting for a 16/17
		// correction (tdlib's check_packet pattern). Cheap lock-free CAS; only
		// ever moves the offset forward, so it cannot perturb msg_id ordering.
		s.msgFactory.AdvanceServerTime(time.Unix(raw.MsgID>>32, 0))

		if requiresAck(raw.SeqNo) {
			s.addAck(raw.MsgID)
		}

		rawHandled := s.handleRawPacket(raw.MsgID, raw.BodyRaw)
		needsDecodedResult := s.hasDecodedResults()

		crypto.ReleaseAESBuf(decrypted)

		if rawHandled || (!needsDecodedResult && updateFn == nil) {
			continue
		}

		if needsDecodedResult || updateFn != nil {
			select {
			case s.dispatchSem <- struct{}{}:
				handlers.Add(1)
				go func(raw *tg.MTProtoMessageRaw) {
					defer handlers.Done()
					defer func() { <-s.dispatchSem }()
					s.dispatchRaw(raw)
				}(raw)
			default:
				// At capacity: process in-line (backpressure on sender).
				s.dispatchRaw(raw)
			}
		}
	}
}

// pingLoop is an errgroup goroutine that sends periodic keep-alive pings and
// detects dead connections via pong timeouts.
func (s *Session) pingLoop(ctx context.Context) error {
	if s.pingInterval <= 0 {
		<-ctx.Done()
		return ctx.Err()
	}

	if jitter := time.Duration(uint64(s.sessionID) % uint64(s.pingInterval)); jitter > 0 {
		timer := time.NewTimer(jitter)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		case <-timer.C:
		}
	}

	ticker := time.NewTicker(s.pingInterval)
	defer ticker.Stop()
	pongTimer := time.NewTimer(time.Hour)
	if !pongTimer.Stop() {
		<-pongTimer.C
	}
	defer pongTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		pingID := time.Now().UnixNano()
		pongCh := make(chan struct{})
		started := time.Now()
		pongTimeout := s.adaptivePongTimeout()
		delay := s.pingInterval + pongTimeout
		disconnectDelay := int32((delay + time.Second - 1) / time.Second)

		s.pingMux.Lock()
		s.pingCbs[pingID] = pongCh
		s.pingMux.Unlock()

		if err := s.sendServiceMessage(&tg.PingDelayDisconnectRequest{
			PingID:          pingID,
			DisconnectDelay: max(disconnectDelay, 1),
		}); err != nil {
			s.removePingCallback(pingID)
			return fmt.Errorf("session: ping write failed: %w", err)
		}

		pongTimer.Reset(pongTimeout)
		select {
		case <-ctx.Done():
			s.removePingCallback(pingID)
			if !pongTimer.Stop() {
				select {
				case <-pongTimer.C:
				default:
				}
			}
			return ctx.Err()
		case <-pongCh:
			if !pongTimer.Stop() {
				select {
				case <-pongTimer.C:
				default:
				}
			}
			s.recordPingRTT(time.Since(started))
		case <-pongTimer.C:
			s.removePingCallback(pingID)
			return fmt.Errorf("session: pong timeout")
		}
	}
}

func (s *Session) removePingCallback(pingID int64) {
	s.pingMux.Lock()
	delete(s.pingCbs, pingID)
	s.pingMux.Unlock()
}

func (s *Session) recordPingRTT(sample time.Duration) {
	oldRTT := time.Duration(s.rttEWMA.Load())
	oldVariation := time.Duration(s.rttVariance.Load())
	newRTT := sample
	newVariation := sample / 2
	if oldRTT > 0 {
		delta := oldRTT - sample
		if delta < 0 {
			delta = -delta
		}
		newVariation = (3*oldVariation + delta) / 4
		newRTT = (7*oldRTT + sample) / 8
	}
	s.rttEWMA.Store(int64(newRTT))
	s.rttVariance.Store(int64(newVariation))
	s.lastPong.Store(time.Now().UnixNano())

	s.mu.RLock()
	callback := s.onRTT
	s.mu.RUnlock()
	if callback != nil {
		callback(newRTT, newVariation)
	}
}

func (s *Session) adaptivePongTimeout() time.Duration {
	configured := s.pongTimeout
	if configured <= 0 {
		configured = 30 * time.Second
	}
	rtt := time.Duration(s.rttEWMA.Load())
	if rtt <= 0 {
		return configured
	}
	variation := time.Duration(s.rttVariance.Load())
	adaptive := rtt + 4*variation
	adaptive = max(adaptive, time.Second)
	return min(adaptive, configured)
}

// handlePong signals the pong channel for the given pingID.
func (s *Session) handlePong(pingID int64) {
	s.pingMux.Lock()
	ch, ok := s.pingCbs[pingID]
	if ok {
		close(ch)
		delete(s.pingCbs, pingID)
	}
	s.pingMux.Unlock()
}

// ackLoop is an errgroup goroutine that batches message acknowledgments and
// sends them periodically or when the batch is full.
func (s *Session) ackLoop(ctx context.Context) error {
	var buf []int64
	seen := make(map[int64]struct{})
	ticker := time.NewTicker(ackFlushInterval)
	defer ticker.Stop()

	flush := func() error {
		if len(buf) == 0 {
			return nil
		}
		batch := make([]int64, len(buf))
		copy(batch, buf)
		start := time.Now()
		if err := s.sendServiceMessage(&tg.MsgsAck{MsgIds: batch}); err != nil {
			return fmt.Errorf("session: flush acknowledgements: %w", err)
		}
		buf = buf[:0]
		clear(seen)
		if elapsed := time.Since(start); elapsed > slowWriteThreshold && s.log != nil {
			s.log.Warnf("session: slow ack flush: count=%d elapsed=%v", len(batch), elapsed)
		}
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			_ = flush()
			return ctx.Err()
		case msgID := <-s.ackCh:
			if _, ok := seen[msgID]; ok {
				continue
			}
			seen[msgID] = struct{}{}
			buf = append(buf, msgID)
			if len(buf) >= 8192 {
				if err := flush(); err != nil {
					return err
				}
			}
		case <-ticker.C:
			if err := flush(); err != nil {
				return err
			}
		}
	}
}

// saltLoop is an errgroup goroutine that periodically requests future salts
// from the server.
func (s *Session) saltLoop(ctx context.Context) error {
	timer := time.NewTimer(initialSaltFetchWait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	s.fetchSaltsWithRetry(ctx)

	for {
		wait := s.saltMgr.NextRefreshIn()
		if wait <= 0 {
			wait = defaultSaltRefreshMin
		}

		timer.Reset(wait)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			s.fetchSaltsWithRetry(ctx)
		}
	}
}

// pfsRenewalLoop is an errgroup goroutine that proactively renews the PFS
// temporary key after 75% of its lifetime. It uses one deadline timer; expiry
// cancels the errgroup and triggers a reconnect that generates a fresh key. If
// PFS is disabled the loop blocks until the context is cancelled.
//
// Ported from tdesktop's create_gen_auth_key_actor renewal timer and
// gotd/td's session.rekey logic. See https://core.telegram.org/api/pfs.
func (s *Session) pfsRenewalLoop(ctx context.Context) error {
	pfs := s.PFS()
	if pfs == nil || !pfs.IsEnabled() {
		<-ctx.Done()
		return ctx.Err()
	}

	wait := pfs.rotationDueIn()
	if wait <= 0 {
		return fmt.Errorf("session: pfs temp key needs rotation")
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("session: pfs temp key needs rotation")
	}
}

func (s *Session) fetchSaltsWithRetry(ctx context.Context) {
	for attempt := 0; attempt < saltPrefetchMaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if !s.sm.isActive() {
			return
		}

		s.sendServiceMessage(&tg.GetFutureSaltsRequest{Num: numFutureSalts})

		select {
		case <-ctx.Done():
			return
		case <-time.After(saltPrefetchRetryBase * time.Duration(1<<attempt)):
		}

		if !s.saltMgr.IsExpired() {
			return
		}

		backoff := saltPrefetchRetryBase * time.Duration(1<<attempt)
		if backoff > saltPrefetchRetryMax {
			backoff = saltPrefetchRetryMax
		}
		if s.log != nil {
			s.log.Warnf("salt pre-fetch: salt still expired after attempt %d, retrying in %v", attempt+1, backoff)
		}
	}
	if s.log != nil {
		s.log.Warnf("salt pre-fetch: failed to refresh salt after %d attempts", saltPrefetchMaxRetries)
	}
}

// handleClose is an errgroup goroutine that waits for context cancellation
// and performs cleanup: marks disconnected, rejects pending RPCs, closes the
// done channel, and closes the transport.
func (s *Session) handleClose(ctx context.Context) error {
	<-ctx.Done()
	s.sm.transitionTo(StateDraining)
	s.pending.RejectAll(ErrSessionClosed)
	s.containerTracker.Cleanup()
	close(s.done)
	s.mu.RLock()
	tp := s.transport
	s.mu.RUnlock()
	if tp != nil {
		tp.Close()
	}
	return nil
}

func (s *Session) dispatchRaw(raw *tg.MTProtoMessageRaw) {
	s.mu.RLock()
	panicFn := s.onPanic
	s.mu.RUnlock()
	defer func() {
		if r := recover(); r != nil {
			if panicFn != nil {
				panicFn(r)
			} else {
				if s.log != nil {
					s.log.Errorf("dispatchRaw panic: %v", r)
				}
			}
		}
	}()
	bodyReader := tg.NewReader(raw.BodyRaw)
	defer tg.ReleaseReader(bodyReader)
	body, err := tg.ReadTLObject(bodyReader)
	if err != nil {
		if _, ok := err.(*tg.UnknownConstructorError); ok {
			if s.log != nil {
				s.log.Debugf("skipping unknown constructor in msg_id=%d buf_len=%d", raw.MsgID, len(raw.BodyRaw))
			}
			return
		}
		if s.log != nil {
			s.log.Warnf("TL decode failed: msg_id=%d buf_len=%d err=%v", raw.MsgID, len(raw.BodyRaw), err)
		}
		return
	}
	s.processIncoming(&tg.MTProtoMessage{MsgID: raw.MsgID, SeqNo: raw.SeqNo, Body: body})
}

func (s *Session) processIncoming(msg *tg.MTProtoMessage) {
	if msg == nil || msg.Body == nil {
		return
	}
	if container, ok := msg.Body.(*tg.MsgContainer); ok {
		for _, subMsg := range container.Messages {
			if subMsg.Body != nil {
				s.handlePacket(subMsg.MsgID, subMsg.SeqNo, subMsg.Body)
			}
		}
		return
	}
	s.handlePacket(msg.MsgID, msg.SeqNo, msg.Body)
}

func (s *Session) handleRawPacket(msgID int64, body []byte) bool {
	if len(body) < 4 {
		return false
	}
	constructorID := binary.LittleEndian.Uint32(body[:4])
	switch constructorID {
	case tg.RPCResultTypeID:
		s.handleRawRPCResult(body)
	case tg.GzipPackedID:
		return s.handleRawGzipPacked(body[4:])
	case tg.MsgContainerID:
		return s.handleRawContainer(body[4:])
	case tg.PongTypeID:
		if len(body) < 20 {
			return false
		}
		pingID := int64(binary.LittleEndian.Uint64(body[12:20]))
		s.handlePong(pingID)
		s.pending.Resolve(int64(binary.LittleEndian.Uint64(body[4:12])), &tg.Pong{
			MsgID:  int64(binary.LittleEndian.Uint64(body[4:12])),
			PingID: pingID,
		})
	case tg.BadMsgNotificationTypeID:
		if len(body) < 20 {
			return false
		}
		badMsgID := int64(binary.LittleEndian.Uint64(body[4:12]))
		errorCode := int32(binary.LittleEndian.Uint32(body[16:20]))
		if errorCode == 16 || errorCode == 17 {
			// Codes 16 (msg_id too low) and 17 (msg_id too high) indicate
			// client/server clock drift. The server's current time is encoded
			// in the upper 32 bits of the notification's msg_id (msgID parameter).
			serverTime := msgID >> 32
			s.msgFactory.UpdateServerTime(time.Unix(serverTime, 0))
			if s.log != nil {
				s.log.Warnf("session: time corrected (code=%d) server_time=%d", errorCode, serverTime)
			}
			if errorCode == 17 && s.log != nil {
				s.log.Warnf("session: msg_id too high (code=17), reconnect recommended to reset session")
			}
			// Resolve with BadMsgNotification so Invoke's retry loop
			// re-sends the query with the corrected time (via a fresh
			// msg_id from the updated MsgIDGenerator).
			s.pending.Resolve(badMsgID, &tg.BadMsgNotification{
				BadMsgID:    badMsgID,
				BadMsgSeqno: int32(binary.LittleEndian.Uint32(body[12:16])),
				ErrorCode:   errorCode,
			})
			return true
		}
		if errorCode == 20 {
			// Code 20: msg_id too old, server cannot determine whether it
			// received the message. Log and drop — no resolution possible.
			if s.log != nil {
				s.log.Warnf("session: bad_msg code 20 (msg_id too old) for msg_id=%d", badMsgID)
			}
			return true
		}
		if errorCode == 64 {
			// Code 64: invalid container. Mark all children as failed.
			if s.log != nil {
				s.log.Warnf("session: bad_msg code 64 (invalid container) for msg_id=%d", badMsgID)
			}
			for _, childID := range s.containerTracker.NackContainer(badMsgID) {
				s.pending.Reject(childID, fmt.Errorf("session: bad_msg code 64: invalid container (container msg_id=%d)", badMsgID))
			}
			return true
		}
		s.pending.Resolve(badMsgID, &tg.BadMsgNotification{
			BadMsgID:    badMsgID,
			BadMsgSeqno: int32(binary.LittleEndian.Uint32(body[12:16])),
			ErrorCode:   errorCode,
		})
	case tg.BadServerSaltTypeID:
		if len(body) < 28 {
			return false
		}
		badMsgID := int64(binary.LittleEndian.Uint64(body[4:12]))
		newSalt := int64(binary.LittleEndian.Uint64(body[20:28]))
		s.saltMgr.StoreSimple(newSalt)
		s.pending.Resolve(badMsgID, &tg.BadServerSalt{
			BadMsgID:      badMsgID,
			BadMsgSeqno:   int32(binary.LittleEndian.Uint32(body[12:16])),
			ErrorCode:     int32(binary.LittleEndian.Uint32(body[16:20])),
			NewServerSalt: newSalt,
		})
	case tg.NewSessionCreatedTypeID:
		if len(body) < 28 {
			return false
		}
		serverSalt := int64(binary.LittleEndian.Uint64(body[20:28]))
		s.saltMgr.StoreSimple(serverSalt)
		s.fireNewSession(
			int64(binary.LittleEndian.Uint64(body[4:12])),
			int64(binary.LittleEndian.Uint64(body[12:20])),
			serverSalt,
		)
	case tg.FutureSaltsTypeID:
		return s.handleRawFutureSalts(body)
	case tg.MsgsAckTypeID:
		s.handleRawMsgsAck(body[4:])
	case tg.MsgDetailedInfoTypeID:
		if len(body) >= 20 {
			s.addAck(int64(binary.LittleEndian.Uint64(body[12:20])))
		}
	case tg.MsgNewDetailedInfoTypeID:
		if len(body) >= 12 {
			s.addAck(int64(binary.LittleEndian.Uint64(body[4:12])))
		}
	case tg.MsgsStateReqTypeID:
		s.handleRawMsgsStateReq(msgID, body[4:])
	case tg.MsgsStateInfoTypeID:
		s.handleRawMsgsStateInfo(body[4:])
	case tg.MsgResendReqTypeID:
		s.handleRawMsgResendReq(body[4:])
	case tg.MsgsAllInfoTypeID:
		s.handleRawMsgsAllInfo(body[4:])
	default:
		return false
	}
	return true
}

// fireNewSession notifies the registered callback that the server sent a
// new_session_created notification. Safe to call from any goroutine.
func (s *Session) fireNewSession(firstMsgID, uniqueID, serverSalt int64) {
	s.mu.RLock()
	fn := s.onNewSession
	s.mu.RUnlock()
	if fn != nil {
		fn(firstMsgID, uniqueID, serverSalt)
	}
}

func (s *Session) handleRawMsgsAck(body []byte) {
	r := tg.NewReader(body)
	defer tg.ReleaseReader(r)
	msgIDs, err := r.ReadVectorLong()
	if err != nil {
		return
	}
	for _, ackedID := range msgIDs {
		s.pending.MarkAcked(ackedID)
		s.containerTracker.AckContainer(ackedID)
		s.containerTracker.AckChild(ackedID)
	}
}

// handleRawMsgsAllInfo parses the body of msgs_all_info (after constructor ID)
// which is vector<long> msg_ids + string info. Unlike msgs_state_info, this
// has no req_msg_id field.
func (s *Session) handleRawMsgsAllInfo(body []byte) {
	r := tg.NewReader(body)
	defer tg.ReleaseReader(r)
	msgIDs, err := r.ReadVectorLong()
	if err != nil {
		return
	}
	info, err := r.ReadString()
	if err != nil {
		return
	}
	for i, msgID := range msgIDs {
		if i >= len(info) {
			break
		}
		s.interpretStateByte(msgID, info[i])
	}
}

func (s *Session) handleRawRPCResult(body []byte) {
	if len(body) < 12 {
		return
	}
	reqMsgID := int64(binary.LittleEndian.Uint64(body[4:12]))
	payload := body[12:]
	if !s.pending.ResolveRPCResult(reqMsgID, payload) {
		return
	}
	result, err := decodeRawRPCResultPayload(payload)
	if err != nil {
		s.pending.Reject(reqMsgID, err)
		return
	}
	s.pending.Resolve(reqMsgID, result)
}

func decodeRawRPCResultPayload(payload []byte) (tg.TLObject, error) {
	r := tg.NewReader(payload)
	defer tg.ReleaseReader(r)
	result, err := tg.ReadTLObject(r)
	if err != nil {
		return nil, err
	}
	if gz, ok := result.(*tg.GzipPacked); ok {
		return gz.Decode()
	}
	return result, nil
}

func (s *Session) handleRawGzipPacked(body []byte) bool {
	r := tg.NewReader(body)
	defer tg.ReleaseReader(r)
	gz, err := tg.DecodeGzipPacked(r)
	if err != nil {
		return false
	}
	data, ok := gz.Data.(*tg.GzipPackedData)
	if !ok {
		return false
	}
	return s.handleRawPacket(0, data.Raw)
}

func (s *Session) handleRawContainer(body []byte) bool {
	if len(body) < 4 {
		return false
	}
	count := binary.LittleEndian.Uint32(body[:4])
	if count > 1024 {
		return false
	}
	off := 4
	allHandled := true
	for i := uint32(0); i < count; i++ {
		if off+16 > len(body) {
			return false
		}
		msgID := int64(binary.LittleEndian.Uint64(body[off:]))
		off += 8
		seqNo := binary.LittleEndian.Uint32(body[off:])
		off += 4
		length := int(binary.LittleEndian.Uint32(body[off:]))
		off += 4
		if length < 0 || off+length > len(body) {
			return false
		}
		// Validate each child msgID against replay/parity rules (#21).
		if !s.msgIDValidator.Check(msgID) {
			off += length
			continue
		}
		if requiresAck(seqNo) {
			s.addAck(msgID)
		}
		if !s.handleRawPacket(msgID, body[off:off+length]) && !s.decodeRawPacketIfNeeded(msgID, seqNo, body[off:off+length]) {
			allHandled = false
		}
		off += length
	}
	return allHandled
}

func (s *Session) decodeRawPacketIfNeeded(msgID int64, seqNo uint32, body []byte) bool {
	s.mu.RLock()
	updateFn := s.onUpdate
	s.mu.RUnlock()
	if !s.hasDecodedResults() && updateFn == nil {
		return false
	}
	r := tg.NewReader(body)
	defer tg.ReleaseReader(r)
	obj, err := tg.ReadTLObject(r)
	if err != nil {
		return false
	}
	s.processIncoming(&tg.MTProtoMessage{MsgID: msgID, SeqNo: seqNo, Body: obj})
	return true
}

func (s *Session) handleRawFutureSalts(body []byte) bool {
	const headerSize = 24
	if len(body) < headerSize {
		return false
	}
	if binary.LittleEndian.Uint32(body[16:20]) != tg.VectorTypeID {
		return false
	}
	count := int(binary.LittleEndian.Uint32(body[20:24]))
	if count == 0 {
		return true
	}
	saltSize := 4 + 4 + 4 + 8
	expected := headerSize + count*saltSize
	if len(body) < expected {
		count = (len(body) - headerSize) / saltSize
		if count <= 0 {
			return false
		}
	}
	var entries []saltEntry
	offset := headerSize
	for i := 0; i < count; i++ {
		if binary.LittleEndian.Uint32(body[offset:offset+4]) != tg.FutureSaltTypeID {
			offset += saltSize
			continue
		}
		validSince := int64(binary.LittleEndian.Uint32(body[offset+4 : offset+8]))
		validUntil := int64(binary.LittleEndian.Uint32(body[offset+8 : offset+12]))
		salt := int64(binary.LittleEndian.Uint64(body[offset+12 : offset+20]))
		entries = append(entries, saltEntry{
			validSince: validSince,
			validUntil: validUntil,
			salt:       salt,
		})
		offset += saltSize
	}
	if len(entries) > 0 {
		s.saltMgr.StoreFromFutureSalts(entries)
	}
	return true
}

func (s *Session) handleRawMsgsStateReq(msgID int64, body []byte) {
	if len(body) < 8 {
		return
	}
	r := tg.NewReader(body)
	msgIDs, err := r.ReadVectorLong()
	if err != nil || len(msgIDs) == 0 {
		return
	}
	info := make([]byte, len(msgIDs))
	for i := range msgIDs {
		if s.pending.Has(msgIDs[i]) {
			info[i] = 0x80 | 0x04
		} else {
			info[i] = 0x01
		}
	}
	s.sendServiceMessage(&tg.MsgsStateInfo{
		ReqMsgID: msgID,
		Info:     string(info),
	})
}

func (s *Session) handleRawMsgResendReq(body []byte) {
	if len(body) < 8 {
		return
	}
	r := tg.NewReader(body)
	msgIDs, err := r.ReadVectorLong()
	if err != nil {
		return
	}
	for _, id := range msgIDs {
		// Try to re-send the original encrypted payload.
		if payload := s.pending.GetPayload(id); payload != nil {
			_ = s.writeEncryptedDirect(payload, 10*time.Second)
			continue
		}
		if seqNo, bodyRaw, ok := s.pending.GetResendMessage(id); ok {
			s.mu.RLock()
			authKey := s.authKey
			authKeyID := s.authKeyID
			s.mu.RUnlock()
			payload, err := crypto.PackRaw(id, seqNo, bodyRaw, s.saltMgr.Load(), s.sessionIDBytes(), authKey, authKeyID)
			if err == nil {
				err = s.writeEncryptedDirect(payload, 10*time.Second)
			}
			if err == nil {
				continue
			}
			if s.log != nil {
				s.log.Warnf("msg_resend_req: failed to resend msg_id=%d: %v", id, err)
			}
		}
		// Payload not available — nack and ack as fallback.
		s.containerTracker.NackContainer(id)
		s.addAck(id)
	}
}

func isTimeoutError(err error) bool {
	if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
		return true
	}
	return false
}

// SetUpdateHandler registers fn as the callback for unsolicited server
// updates (e.g., new messages, status changes). Pass nil to remove the
// handler.
func (s *Session) SetUpdateHandler(fn func(tg.TLObject)) {
	s.mu.Lock()
	s.onUpdate = fn
	s.mu.Unlock()
}

// Connect sets the transport and starts the session. It requires that an auth
// key has already been established. Returns an error if no auth key is set.
func (s *Session) Connect(transport Transport, timeout time.Duration) error {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return s.ConnectContext(ctx, transport)
}

// ConnectContext sets the transport and starts the session. It requires that an
// auth key has already been established. The context bounds the startup ping.
func (s *Session) ConnectContext(ctx context.Context, transport Transport) error {
	if !s.sm.canConnect() {
		state := s.sm.State()
		if state == StateClosed {
			return ErrSessionClosed
		}
		return fmt.Errorf("session: connect: already connected (state=%s)", state)
	}
	if transport != nil {
		s.mu.Lock()
		s.transport = transport
		s.mu.Unlock()
	}
	s.mu.RLock()
	authKey := s.authKey
	s.mu.RUnlock()
	if len(authKey) == 0 {
		return ErrConnectNoAuthKey
	}
	return s.StartContext(ctx)
}

// ConnectWithAuth sets the transport and performs key generation via authFunc
// if no auth key is currently set. The resulting auth key and server salt are
// persisted to storage. After authentication, the session is started.
func (s *Session) ConnectWithAuth(transport Transport, authFunc AuthFunc, timeout time.Duration) error {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return s.ConnectWithAuthContext(ctx, transport, authFunc)
}

// ConnectWithAuthContext sets the transport and performs key generation via
// authFunc if no auth key is currently set. The context bounds authentication
// and the startup ping.
func (s *Session) ConnectWithAuthContext(ctx context.Context, transport Transport, authFunc AuthFunc) error {
	if !s.sm.canConnect() {
		state := s.sm.State()
		if state == StateClosed {
			return ErrSessionClosed
		}
		return fmt.Errorf("session: connect: already connected (state=%s)", state)
	}
	if transport != nil {
		s.mu.Lock()
		s.transport = transport
		s.mu.Unlock()
	}
	s.mu.RLock()
	authKey := s.authKey
	s.mu.RUnlock()
	if len(authKey) == 0 && authFunc != nil {
		s.mu.RLock()
		tp := s.transport
		s.mu.RUnlock()
		result, err := authFunc(tp)
		if err != nil {
			return fmt.Errorf("session: connect with auth: %w", err)
		}
		s.mu.Lock()
		s.authKey = result.AuthKey
		s.authKeyID = computeAuthKeyID(result.AuthKey)
		s.mu.Unlock()
		s.saltMgr.StoreSimple(result.ServerSalt)
		if s.storage != nil {
			if err := s.storage.SetAuthKey(result.AuthKey); err != nil {
				return fmt.Errorf("session: save auth key: %w", err)
			}
		}
		s.msgFactory.UpdateServerTime(time.Unix(int64(result.ServerTime), 0))
	}
	return s.StartContext(ctx)
}

func unpackIncomingMessageEnvelope(data, sessionID, authKey, authKeyID []byte) (*tg.MTProtoMessageRaw, []byte, error) {
	raw, decrypted, err := crypto.UnpackEnvelope(data, sessionID, authKey, authKeyID)
	if err != nil {
		return nil, nil, fmt.Errorf("session: unpack envelope: %w", err)
	}
	return raw, decrypted, nil
}
