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
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type sessionLogger interface {
	Debugf(format string, v ...any)
	Warnf(format string, v ...any)
	Errorf(format string, v ...any)
}

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
)

// hasDecodedResults returns true if any goroutine is waiting for a decoded TL
// RPC result. Raw result waiters are tracked separately so they do not force
// TL decoding or gzip unpacking.
func (s *Session) hasDecodedResults() bool {
	return s.pending.HasAny()
}

func (s *Session) hasRawResults() bool {
	return s.pending.HasAnyRaw()
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
	// transport, pingInterval, onUpdate, onPanic.
	mu sync.RWMutex

	// transport is the underlying network transport for sending/receiving data.
	transport Transport
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
	// log receives structured log output. When nil, logging is suppressed.
	log sessionLogger

	consecWriteFailures   int
	writeBreakerThreshold int
	writeBreakerOpen      atomic.Bool
}

// SetOnPanic sets a callback invoked when a dispatchUpdate goroutine panics.
func (s *Session) SetOnPanic(fn func(panicValue any)) {
	s.mu.Lock()
	s.onPanic = fn
	s.mu.Unlock()
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
}

func (s *Session) SetWriteBreakerThreshold(n int) {
	s.writeBreakerThreshold = n
}

func (s *Session) trackWriteResult(err error) {
	if s.writeBreakerThreshold <= 0 {
		return
	}
	if err != nil {
		s.consecWriteFailures++
		if s.consecWriteFailures >= s.writeBreakerThreshold && !s.writeBreakerOpen.Load() {
			s.writeBreakerOpen.Store(true)
			if s.group != nil {
				s.group.Cancel()
			}
		}
	} else {
		s.consecWriteFailures = 0
	}
}

func computeAuthKeyID(authKey []byte) []byte {
	h := sha1.Sum(authKey)
	id := make([]byte, 8)
	copy(id, h[12:20])
	return id
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
		pending:      NewPendingManager(),
		pingInterval: 60 * time.Second,
		pongTimeout:  30 * time.Second,
		updateSem:    make(chan struct{}, 128),
		saltMgr:      newSaltManager(time.Now),
		sm:           newStateMachine(),

		writeBreakerThreshold: 3,
	}

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

// SetServerSalt updates the server salt used for encrypting outgoing messages.
func (s *Session) SetServerSalt(salt int64) {
	s.saltMgr.StoreSimple(salt)
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

	if err := s.writeEncrypted(ctx, encrypted, timeout); err != nil {
		s.pending.Cancel(msgID)
		return nil, fmt.Errorf("session: send: %w", err)
	}

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
	case *tg.FutureSalts:
		s.storeFutureSalts(v)
	case *tg.MsgsAck:
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
	var backoff time.Duration
	for i := 0; i < retries; i++ {
		if i > 0 {
			time.Sleep(backoff)
		}
		msgID := s.msgFactory.AllocateMsgID()
		seqNo := s.msgFactory.AllocateSeqNo(true)

		data, err := s.SendRaw(ctx, msgID, uint32(seqNo), bodyBytes, timeout)
		if err != nil {
			lastErr = fmt.Errorf("invoke raw: send: %w", err)
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

	var lastErr error
	var backoff time.Duration
	for i := 0; i < retries; i++ {
		if i > 0 {
			time.Sleep(backoff)
		}
		msgID := s.msgFactory.AllocateMsgID()
		seqNo := s.msgFactory.AllocateSeqNo(true)

		obj, err := s.Send(ctx, msgID, uint32(seqNo), query, timeout)
		if err != nil {
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

		return obj, nil
	}
	if lastErr == nil {
		return nil, fmt.Errorf("invoke %s: retries exhausted (%d)", methodName, retries)
	}
	return nil, fmt.Errorf("invoke %s: retries exhausted (%d): %w", methodName, retries, lastErr)
}

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
	g.Go(s.pingLoop)
	g.Go(s.saltLoop)
	g.Go(s.handleClose)

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

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return 0
		case <-s.done:
			return 0
		case <-time.After(100 * time.Millisecond):
		}
		if !s.saltMgr.IsExpired() {
			return s.saltMgr.Load()
		}
	}
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
		return
	}
	_ = s.writeEncryptedDirect(encrypted, 10*time.Second)
}

// writeEncrypted snapshots transport state, acquires writeMux, sets the write
// deadline from ctx or timeout (whichever is sooner), writes the encrypted
// payload, and releases the mutex. Returns the transport error, if any.
// Lock ordering: mu is always acquired BEFORE writeMux, never inside it.
func (s *Session) writeEncrypted(ctx context.Context, encrypted []byte, timeout time.Duration) error {
	if s.writeBreakerThreshold > 0 && s.writeBreakerOpen.Load() {
		return ErrWriteCircuitOpen
	}

	deadline := time.Now().Add(timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}

	s.mu.RLock()
	tp := s.transport
	s.mu.RUnlock()
	if tp == nil {
		return ErrTransportNotSet
	}

	s.writeMux.Lock()
	defer s.writeMux.Unlock()

	if s.writeBreakerThreshold > 0 && s.writeBreakerOpen.Load() {
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

// writeEncryptedDirect is the context-less variant used by service messages
// and RPCDropAnswer where there is no caller context to respect.
func (s *Session) writeEncryptedDirect(encrypted []byte, timeout time.Duration) error {
	if s.writeBreakerThreshold > 0 && s.writeBreakerOpen.Load() {
		return ErrWriteCircuitOpen
	}

	deadline := time.Now().Add(timeout)

	s.mu.RLock()
	tp := s.transport
	s.mu.RUnlock()
	if tp == nil {
		return ErrTransportNotSet
	}

	s.writeMux.Lock()
	defer s.writeMux.Unlock()

	if s.writeBreakerThreshold > 0 && s.writeBreakerOpen.Load() {
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
		if len(data) == 4 {
			code := int32(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24)
			if code < 0 {
				continue
			}
			return fmt.Errorf("transport error: code %d", -code)
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
		s.pending.Resolve(int64(binary.LittleEndian.Uint64(body[4:12])), &tg.BadMsgNotification{
			BadMsgID:    int64(binary.LittleEndian.Uint64(body[4:12])),
			BadMsgSeqno: int32(binary.LittleEndian.Uint32(body[12:16])),
			ErrorCode:   int32(binary.LittleEndian.Uint32(body[16:20])),
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
		s.saltMgr.StoreSimple(int64(binary.LittleEndian.Uint64(body[20:28])))
	case tg.FutureSaltsTypeID:
		return s.handleRawFutureSalts(body)
	case tg.MsgsAckTypeID:
	case tg.MsgDetailedInfoTypeID:
		if len(body) >= 20 {
			s.addAck(int64(binary.LittleEndian.Uint64(body[12:20])))
		}
	case tg.MsgNewDetailedInfoTypeID:
		if len(body) >= 12 {
			s.addAck(int64(binary.LittleEndian.Uint64(body[4:12])))
		}
	case tg.MsgsStateReqTypeID:
		s.handleRawMsgsStateReq(body[4:])
	case tg.MsgResendReqTypeID:
		s.handleRawMsgResendReq(body[4:])
	case tg.MsgsAllInfoTypeID:
	default:
		return false
	}
	return true
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

func (s *Session) handleRawMsgsStateReq(body []byte) {
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
		ReqMsgID: 0,
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
