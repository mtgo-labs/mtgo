package session

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
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

const (
	numFutureSalts       = 4
	initialSaltFetchWait = 15 * time.Second
	saltFetchInterval    = time.Hour
	ackFlushInterval     = 30 * time.Second
	pfsRenewalInterval   = 60 * time.Second
)

// hasDecodedResults returns true if any goroutine is waiting for a decoded TL
// RPC result. Raw result waiters are tracked separately so they do not force
// TL decoding or gzip unpacking.
func (s *Session) hasDecodedResults() bool {
	return s.pending.HasAny()
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
	updateSem chan struct{}
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
// When enabled, Send coalesces concurrent RPCs into msg_container messages.
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

// trackWriteResult updates the write circuit breaker. Consecutive failures open the
// breaker, which blocks new writes via ErrWriteCircuitOpen but does NOT kill the
// session. Successful writes reset both counter and breaker flag.
func (s *Session) trackWriteResult(err error) {
	threshold := int(s.writeBreakerThreshold.Load())
	if threshold <= 0 {
		return
	}
	if err != nil {
		newCount := s.consecWriteFailures.Add(1)
		if int(newCount) >= threshold && !s.writeBreakerOpen.Load() {
			s.writeBreakerOpen.Store(true)
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
	default:
	}
}

func (s *Session) sendQuickAck(msgID int64) {
	s.sendServiceMessage(&tg.MsgsAck{MsgIds: []int64{msgID}})
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

	// When the outbound batcher is enabled, delegate packing+writing to it.
	// The batcher coalesces multiple concurrent RPCs into a msg_container.
	if b := s.outboundBatcher; b != nil {
		priority := RoutePriority(body)
		handle, err := b.Submit(ctx, msgID, seqNo, body, priority, timeout)
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
		return nil, fmt.Errorf("session: send: %w", err)
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
		return obj, err
	case <-ctx.Done():
		s.pending.Cancel(msgID)
		s.sendRPCDrop(msgID)
		return nil, ctx.Err()
	case <-s.done:
		s.pending.Reject(msgID, ErrSessionClosed)
		<-handle.Done()
		obj, _, err := handle.Result()
		return obj, err
	case <-respTimer.C:
		s.pending.Cancel(msgID)
		return nil, ErrSendTimeout
	}
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
	s.addAck(msgID)

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
		s.sendQuickAck(msgID)
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
// bounded by the updateSem semaphore. If 128 dispatches are already in
// flight, the update is dropped.
func (s *Session) dispatchUpdate(obj tg.TLObject) {
	s.mu.RLock()
	handlerFn := s.onUpdate
	panicFn := s.onPanic
	s.mu.RUnlock()
	select {
	case s.updateSem <- struct{}{}:
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
	default:
	}
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

	if err := s.writeEncrypted(ctx, encrypted, timeout); err != nil {
		s.pending.Cancel(msgID)
		return nil, fmt.Errorf("session: send raw: %w", err)
	}

	respTimer := time.NewTimer(timeout)
	defer respTimer.Stop()
	select {
	case <-handle.Done():
		_, raw, err := handle.Result()
		return raw, err
	case <-ctx.Done():
		s.pending.Cancel(msgID)
		s.sendRPCDrop(msgID)
		return nil, ctx.Err()
	case <-s.done:
		s.pending.Reject(msgID, ErrSessionClosed)
		<-handle.Done()
		_, raw, err := handle.Result()
		return raw, err
	case <-respTimer.C:
		s.pending.Cancel(msgID)
		return nil, ErrSendTimeout
	}
}

// InvokeRaw sends a TLObject query and returns the matching rpc_result's raw
// result:Object payload bytes without gzip unpacking or TL decoding. It retries
// the request up to retries times with the given per-attempt timeout.
func (s *Session) InvokeRaw(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) ([]byte, error) {
	var buf bytes.Buffer
	if err := query.Encode(&buf); err != nil {
		return nil, fmt.Errorf("session: invoke raw: encode query: %w", err)
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
			lastErr = fmt.Errorf("invoke raw: send: %w", err)
			continue
		}

		if len(data) < 4 {
			lastErr = fmt.Errorf("invoke raw: rpc result too short: %d", len(data))
			continue
		}

		if err := checkRawRPCError(data); err != nil {
			return nil, err
		}

		return data, nil
	}
	if lastErr == nil {
		return nil, fmt.Errorf("session: invoke raw: retries exhausted (%d)", retries)
	}
	return nil, fmt.Errorf("session: invoke raw: retries exhausted (%d): %w", retries, lastErr)
}

func checkRawRPCError(data []byte) error {
	if len(data) < 4 {
		return nil
	}
	constructorID := binary.LittleEndian.Uint32(data[:4])
	if constructorID != tg.RPCErrorTypeID {
		return nil
	}
	r := tg.NewReader(data[4:])
	defer tg.ReleaseReader(r)
	rpcErr, err := tg.DecodeRPCError(r)
	if err != nil {
		return nil
	}
	parsed := tgerr.New(int(rpcErr.ErrorCode), rpcErr.ErrorMessage)
	return fmt.Errorf("invoke raw: %w", parsed)
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
			lastErr = fmt.Errorf("invoke %s: send: %w", methodName, err)
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
				lastErr = fmt.Errorf("invoke %s: bad message (msg_id=%d, code=%d)", methodName, msgID, v.ErrorCode)
			case *tg.BadServerSalt:
				lastErr = fmt.Errorf("invoke %s: bad server salt (msg_id=%d, code=%d)", methodName, msgID, v.ErrorCode)
				if badSaltRetries == 0 && i+1 >= maxAttempts {
					badSaltRetries++
					maxAttempts++
				}
			default:
				lastErr = fmt.Errorf("invoke %s: bad message notification: %T", methodName, bad)
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
			// G13: MSG_WAIT_FAILED — the referenced msg in an invokeAfterMsg
			// chain failed. Clear the chain dependency and resend unwrapped
			// so the query is not permanently blocked.
			if hasChain && tgerr.Is(parsed, tgerr.ErrMSGWaitFailed) {
				s.ClearChain(chainID)
				if s.log != nil {
					s.log.Warnf("chain msg wait failed method=%s chain_id=%d, retrying unwrapped", methodName, chainID)
				}
				lastErr = fmt.Errorf("invoke %s: chain msg wait failed", methodName)
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
				lastErr = fmt.Errorf("invoke %s: connection not inited", methodName)
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
				lastErr = fmt.Errorf("invoke %s: persistent timestamp outdated", methodName)
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
						return nil, fmt.Errorf("invoke %s: auth key perm empty: rebind failed: %w", methodName, err)
					}
					lastErr = fmt.Errorf("invoke %s: auth key perm empty, temp key rebound", methodName)
					backoff = 0
					maxAttempts++
					continue
				}
				// Not PFS — fall through to the 401 surface below.
			}

			if rpcErr.ErrorCode == 401 || rpcErr.ErrorCode == 400 || rpcErr.ErrorCode == 403 {
				return nil, fmt.Errorf("invoke %s: %w", methodName, parsed)
			}
			lastErr = fmt.Errorf("invoke %s: %w", methodName, parsed)
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
		return nil, fmt.Errorf("invoke %s: retries exhausted (%d)", methodName, retries)
	}
	return nil, fmt.Errorf("invoke %s: retries exhausted (%d): %w", methodName, retries, lastErr)
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
	s.sm.transition(StateIdle, StateConnecting)

	// Start the errgroup so readLoop is running during the initial ping.
	g := newErrGroup(groupCtx)
	g.Go(s.readLoop)
	g.Go(s.ackLoop)

	s.group = g

	initPingCtx, initPingCancel := context.WithTimeout(pingCtx, 60*time.Second)
	_, err := s.Invoke(initPingCtx, &tg.PingRequest{PingID: time.Now().UnixNano()}, 3, timeoutFromContext(pingCtx))
	initPingCancel()
	if err != nil {
		g.Cancel()
		s.sm.transition(StateConnecting, StateClosed)
		close(s.done)
		return fmt.Errorf("session: initial ping: %w", err)
	}

	s.sm.transition(StateConnecting, StateActive)

	return nil
}

// runLoop adds the remaining errgroup goroutines and blocks until the context
// is cancelled or a goroutine returns a fatal error.
func (s *Session) runLoop(ctx context.Context) error {
	g := s.group
	g.Go(s.stateCheckLoop)
	g.Go(s.pingLoop)
	g.Go(s.saltLoop)
	g.Go(s.handleClose)
	g.Go(s.pfsRenewalLoop)

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
		return err
	}
	go func() {
		defer close(s.runDone)
		s.runLoop(runCtx)
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

func (s *Session) sendServiceMessage(body tg.TLObject) {
	select {
	case <-s.done:
		return
	default:
	}
	s.mu.RLock()
	authKey := s.authKey
	authKeyID := s.authKeyID
	s.mu.RUnlock()
	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(false)
	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: uint32(seqNo),
		Body:  body,
	}
	encrypted, err := crypto.Pack(message, s.saltMgr.Load(), s.sessionIDBytes(), authKey, authKeyID)
	if err != nil {
		if s.log != nil {
			s.log.Errorf("session: pack service message: %v", err)
		}
		return
	}
	_ = s.writeEncryptedDirect(encrypted, 10*time.Second)
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

	s.writeMux.Lock()
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
	err := tp.Send(encrypted)
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

		s.addAck(raw.MsgID)

		rawHandled := s.handleRawPacket(raw.MsgID, raw.BodyRaw)
		needsDecodedResult := s.hasDecodedResults()

		crypto.ReleaseAESBuf(decrypted)

		if rawHandled || (!needsDecodedResult && updateFn == nil) {
			continue
		}

		if needsDecodedResult || updateFn != nil {
			handlers.Add(1)
			go func(raw *tg.MTProtoMessageRaw) {
				defer handlers.Done()
				s.dispatchRaw(raw)
			}(raw)
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

	delay := s.pingInterval + s.pongTimeout
	ticker := time.NewTicker(s.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		pingID := time.Now().UnixNano()
		pongCh := make(chan struct{})

		s.pingMux.Lock()
		s.pingCbs[pingID] = pongCh
		s.pingMux.Unlock()

		s.sendServiceMessage(&tg.PingDelayDisconnectRequest{
			PingID:          pingID,
			DisconnectDelay: int32(delay.Seconds()),
		})

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pongCh:
		case <-time.After(s.pongTimeout):
			return fmt.Errorf("session: pong timeout")
		}
	}
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
	ticker := time.NewTicker(ackFlushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(buf) == 0 {
			return
		}
		batch := make([]int64, len(buf))
		copy(batch, buf)
		buf = buf[:0]
		s.sendServiceMessage(&tg.MsgsAck{MsgIds: batch})
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return ctx.Err()
		case msgID := <-s.ackCh:
			buf = append(buf, msgID)
			if len(buf) >= 8192 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// saltLoop is an errgroup goroutine that periodically requests future salts
// from the server.
func (s *Session) saltLoop(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(initialSaltFetchWait):
	}

	s.fetchSaltsWithRetry(ctx)

	for {
		wait := s.saltMgr.NextRefreshIn()
		if wait <= 0 {
			wait = defaultSaltRefreshMin
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
			s.fetchSaltsWithRetry(ctx)
		}
	}
}

// pfsRenewalLoop is an errgroup goroutine that proactively renews the PFS
// temporary key before it expires. When NeedsRotation returns true the loop
// returns an error, which cancels the errgroup and triggers a reconnect that
// generates a fresh temp key. If PFS is disabled the loop is a no-op that
// blocks until the context is cancelled.
//
// Ported from tdesktop's create_gen_auth_key_actor renewal timer and
// gotd/td's session.rekey logic. See https://core.telegram.org/api/pfs.
func (s *Session) pfsRenewalLoop(ctx context.Context) error {
	pfs := s.PFS()
	if pfs == nil || !pfs.IsEnabled() {
		<-ctx.Done()
		return ctx.Err()
	}

	ticker := time.NewTicker(pfsRenewalInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if pfs.NeedsRotation() {
				return fmt.Errorf("session: pfs temp key needs rotation")
			}
		}
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
		s.addAck(msgID)
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

// TempKeyManager manages PFS temporary auth key lifecycle.
// Ported from td/td/telegram/net/Session.cpp:1488-1498 (auth_loop TmpAuthKey).
type TempKeyManager struct {
	dcID      int
	testMode  bool
	permKey   []byte    // permanent auth key
	tempKey   []byte    // current temp auth key
	tempKeyID int64     // SHA1-based temp key ID
	expiresAt time.Time // when the temp key expires
	bound     bool      // whether auth.bindTempAuthKey succeeded
	enabled   bool      // PFS mode flag
	createdAt time.Time // when this manager (and perm key) was initialized
	needInit  bool      // caller must call initConnection after bind
	storage   storage.Storage
	mu        sync.Mutex
}

// NewTempKeyManager creates a new temp key manager.
func NewTempKeyManager(dcID int, testMode bool, permKey []byte, enabled bool, st storage.Storage) *TempKeyManager {
	return &TempKeyManager{
		dcID:      dcID,
		testMode:  testMode,
		permKey:   permKey,
		enabled:   enabled,
		createdAt: time.Now(),
		storage:   st,
	}
}

// IsEnabled reports whether PFS mode is active.
func (m *TempKeyManager) IsEnabled() bool {
	return m.enabled
}

// IsBound reports whether the temp key has been successfully bound to the
// permanent key via auth.bindTempAuthKey.
func (m *TempKeyManager) IsBound() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bound
}

// PermKey returns the permanent auth key. Used for fallback when bind fails.
func (m *TempKeyManager) PermKey() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.permKey
}

// NeedsInitConnection reports whether the caller must call initConnection
// (wrapping help.getConfig) after a successful temp key binding. The flag is
// set by Bind and cleared once read.
//
// The PFS spec requires rewriting client info after each binding — see
// https://core.telegram.org/api/pfs.
func (m *TempKeyManager) NeedsInitConnection() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := m.needInit
	m.needInit = false
	return n
}

// GetKey returns the current temp key and key ID. If PFS is disabled or no
// temp key exists, returns the permanent key.
func (m *TempKeyManager) GetKey() ([]byte, int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.enabled || len(m.tempKey) == 0 {
		return m.permKey, computeAuthKeyIDInt64(m.permKey)
	}
	return m.tempKey, m.tempKeyID
}

// NeedsRotation reports whether the temp key is approaching expiry and needs rotation.
func (m *TempKeyManager) NeedsRotation() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.enabled || len(m.tempKey) == 0 {
		return false
	}
	// Rotate when within 30 seconds of expiry.
	return time.Until(m.expiresAt) < 30*time.Second
}

// FallbackToPermKey disables PFS for this session (e.g., after bind failure).
func (m *TempKeyManager) FallbackToPermKey() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
	m.tempKey = nil
	m.tempKeyID = 0
	m.bound = false
}

// Generate performs DH exchange to generate a new temp key for PFS.
// Uses p_q_inner_data_temp_dc so the server treats the key as temporary.
// Ported from td/td/telegram/net/Session.cpp:1488-1498 (create_gen_auth_key_actor).
func (m *TempKeyManager) Generate(transport Transport) error {
	auth := &Auth{
		DC:       m.dcID,
		TestMode: m.testMode,
	}

	// Request a temp key with 24h expiry, matching MadelineProto's PFS_DURATION.
	expiresIn := int32(24 * 60 * 60) // 24 hours
	result, err := auth.CreateTemp(transport, expiresIn)
	if err != nil {
		return fmt.Errorf("temp key DH exchange: %w", err)
	}

	m.mu.Lock()
	m.tempKey = result.AuthKey
	m.tempKeyID = computeAuthKeyIDInt64(result.AuthKey)
	m.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	m.bound = false
	m.mu.Unlock()

	return nil
}

// deriveMsgAESKeyIV computes the MTProto v1 AES key and IV from an auth key
// and message key. x is the offset into auth_key (0 for client→server,
// 8 for server→client). This is the same algorithm used in session/tdesktop.
func deriveMsgAESKeyIV(authKey []byte, msgKey [16]byte, x int) (key [32]byte, iv [32]byte) {
	sha1A := sha1.Sum(append(msgKey[:], authKey[x:x+32]...))
	sha1B := sha1.Sum(append(append(authKey[x+32:x+48], msgKey[:]...), authKey[x+48:x+64]...))
	sha1C := sha1.Sum(append(authKey[x+64:x+96], msgKey[:]...))
	sha1D := sha1.Sum(append(msgKey[:], authKey[x+96:x+128]...))

	copy(key[0:8], sha1A[0:8])
	copy(key[8:20], sha1B[8:20])
	copy(key[20:32], sha1C[4:16])

	copy(iv[0:12], sha1A[8:20])
	copy(iv[12:20], sha1B[0:8])
	copy(iv[20:24], sha1C[16:20])
	copy(iv[24:32], sha1D[0:8])
	return key, iv
}

// buildEncryptedBindMessage constructs the encrypted_message for
// auth.bindTempAuthKey, following the format described at
// https://core.telegram.org/method/auth.bindTempAuthKey#binding-message-contents
//
// The message contains a bind_auth_key_inner payload, wrapped in a standard
// MTProto message structure, encrypted with AES-IGE using a key derived from
// the permanent auth key.
func (m *TempKeyManager) buildEncryptedBindMessage(permKey, tempKey []byte, permKeyID, nonce, sessionID int64, expiresAt int32) ([]byte, error) {
	// 1. Serialize bind_auth_key_inner.
	inner := &tg.BindAuthKeyInner{
		Nonce:         nonce,
		TempAuthKeyID: computeAuthKeyIDInt64(tempKey),
		PermAuthKeyID: permKeyID,
		TempSessionID: sessionID,
		ExpiresAt:     expiresAt,
	}
	var innerBuf bytes.Buffer
	if err := inner.Encode(&innerBuf); err != nil {
		return nil, fmt.Errorf("encode bind_auth_key_inner: %w", err)
	}
	innerBytes := innerBuf.Bytes()

	// 2. Build MTProto message: random(16) + msg_id(8) + seq_no(4) + length(4) + data
	var randPrefix [16]byte
	if _, err := rand.Read(randPrefix[:]); err != nil {
		return nil, fmt.Errorf("generate random prefix: %w", err)
	}
	now := time.Now()
	msgID := (now.Unix() << 32) | int64(now.Nanosecond()&^3)

	msg := make([]byte, 0, 32+len(innerBytes))
	msg = append(msg, randPrefix[:]...)
	var buf8 [8]byte
	binary.LittleEndian.PutUint64(buf8[:], uint64(msgID))
	msg = append(msg, buf8[:]...)
	var buf4 [4]byte
	binary.LittleEndian.PutUint32(buf4[:], 0) // seq_no = 0
	msg = append(msg, buf4[:]...)
	binary.LittleEndian.PutUint32(buf4[:], uint32(len(innerBytes)))
	msg = append(msg, buf4[:]...)
	msg = append(msg, innerBytes...)

	// 3. msg_key = last 16 bytes of SHA1(message)
	msgHash := sha1.Sum(msg)
	var msgKey [16]byte
	copy(msgKey[:], msgHash[4:20])

	// 4. Pad to 16-byte multiple with random bytes.
	padLen := (16 - len(msg)%16) % 16
	if padLen > 0 {
		pad := make([]byte, padLen)
		if _, err := rand.Read(pad); err != nil {
			return nil, fmt.Errorf("generate padding: %w", err)
		}
		msg = append(msg, pad...)
	}

	// 5. Derive AES key/IV from permanent auth key + msg_key (x=0 client→server).
	aesKey, aesIV := deriveMsgAESKeyIV(permKey, msgKey, 0)

	// 6. AES-IGE encrypt.
	encrypted, err := crypto.IGEEncrypt(msg, aesKey[:], aesIV[:])
	if err != nil {
		return nil, fmt.Errorf("encrypt binding message: %w", err)
	}
	defer crypto.ReleaseAESBuf(encrypted)

	// 7. Final: perm_auth_key_id(8) + msg_key(16) + encrypted_data.
	result := make([]byte, 0, 8+16+len(encrypted))
	binary.LittleEndian.PutUint64(buf8[:], uint64(permKeyID))
	result = append(result, buf8[:]...)
	result = append(result, msgKey[:]...)
	result = append(result, encrypted...)
	return result, nil
}

// ErrBindRequiresKeyRotation signals that auth.bindTempAuthKey returned
// ENCRYPTED_MESSAGE_INVALID and the permanent auth key is older than 60 seconds.
// Both the permanent and temporary keys must be dropped and recreated.
var ErrBindRequiresKeyRotation = fmt.Errorf("session: ENCRYPTED_MESSAGE_INVALID with stale perm key; both keys must be recreated")

// Bind calls auth.bindTempAuthKey to bind the temp key to the permanent key.
// The encrypted_message is constructed per the MTProto PFS spec.
// Ported from td/td/telegram/net/Session.cpp:1556-1579 (need_send_bind_key).
//
// If the server returns ENCRYPTED_MESSAGE_INVALID and the permanent key was
// created more than 60 seconds ago, Bind returns ErrBindRequiresKeyRotation.
// The caller must then drop both keys, recreate them, and retry.
// See https://core.telegram.org/api/pfs for the full recovery procedure.
func (m *TempKeyManager) Bind(ctx context.Context, sessionID int64, invoke func(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error)) error {
	m.mu.Lock()
	tempKey := m.tempKey
	permKey := m.permKey
	expiresAt := m.expiresAt
	createdAt := m.createdAt
	m.mu.Unlock()

	if len(tempKey) == 0 {
		return fmt.Errorf("temp key not generated")
	}
	if len(permKey) < 256 {
		return fmt.Errorf("permanent key too short: %d bytes", len(permKey))
	}

	permKeyID := computeAuthKeyIDInt64(permKey)

	// Generate random nonce.
	var nonceBytes [8]byte
	if _, err := rand.Read(nonceBytes[:]); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}
	nonce := int64(binary.LittleEndian.Uint64(nonceBytes[:]))

	// Build the encrypted binding message.
	encMsg, err := m.buildEncryptedBindMessage(permKey, tempKey, permKeyID, nonce, sessionID, int32(expiresAt.Unix()))
	if err != nil {
		return fmt.Errorf("build bind message: %w", err)
	}

	bindReq := &tg.AuthBindTempAuthKeyRequest{
		PermAuthKeyID:    permKeyID,
		Nonce:            nonce,
		ExpiresAt:        int32(expiresAt.Unix()),
		EncryptedMessage: encMsg,
	}

	_, err = invoke(ctx, bindReq, 1, 10*time.Second)
	if err != nil {
		m.mu.Lock()
		m.bound = false
		m.mu.Unlock()

		// Handle ENCRYPTED_MESSAGE_INVALID per the PFS spec.
		if tgerr.Is(err, tgerr.ErrEncryptedMessageInvalid) {
			if time.Since(createdAt) > 60*time.Second {
				return ErrBindRequiresKeyRotation
			}
			return fmt.Errorf("auth.bindTempAuthKey (ENCRYPTED_MESSAGE_INVALID, key <60s old, will retry): %w", err)
		}
		return fmt.Errorf("auth.bindTempAuthKey: %w", err)
	}

	m.mu.Lock()
	m.bound = true
	m.needInit = true // caller must initConnection after bind per PFS spec
	m.mu.Unlock()
	return nil
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
