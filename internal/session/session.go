package session

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// Transport abstracts the underlying network transport used by a Session to
// send and receive raw byte payloads. Implementations must be safe for
// concurrent use via Send and Recv from separate goroutines.
type Transport interface {
	// Send writes a raw encrypted payload to the transport.
	Send(data []byte) error
	// Recv blocks until a raw encrypted payload is received.
	Recv() ([]byte, error)
	// Close terminates the transport connection and releases resources.
	Close() error
	// IsConnected reports whether the transport is currently connected.
	IsConnected() bool
	// SetWriteDeadline sets the write deadline on the underlying connection.
	SetWriteDeadline(t time.Time) error
}

// sendJob is a unit of work for the writer goroutine.
type sendJob struct {
	encrypted []byte
	deadline  time.Time
	done      chan error
}

const (
	numFutureSalts       = 4
	initialSaltFetchWait = 15 * time.Second
	saltFetchInterval    = time.Hour
	ackFlushInterval     = 30 * time.Second
	housekeeperTick      = time.Second

	defaultDispatchQueueSize = 256
)

var sendJobPool = sync.Pool{
	New: func() any {
		return &sendJob{}
	},
}

var defaultHousekeeper = newGlobalHousekeeper()

type globalHousekeeper struct {
	mu       sync.Mutex
	sessions map[*Session]struct{}
	once     sync.Once
}

func newGlobalHousekeeper() *globalHousekeeper {
	return &globalHousekeeper{sessions: make(map[*Session]struct{})}
}

func (h *globalHousekeeper) register(s *Session) {
	h.mu.Lock()
	h.sessions[s] = struct{}{}
	h.mu.Unlock()
	h.once.Do(func() {
		go h.run()
	})
}

func (h *globalHousekeeper) unregister(s *Session) {
	h.mu.Lock()
	delete(h.sessions, s)
	h.mu.Unlock()
}

func (h *globalHousekeeper) hasSession(s *Session) bool {
	h.mu.Lock()
	_, ok := h.sessions[s]
	h.mu.Unlock()
	return ok
}

func (h *globalHousekeeper) snapshot() []*Session {
	h.mu.Lock()
	sessions := make([]*Session, 0, len(h.sessions))
	for s := range h.sessions {
		sessions = append(sessions, s)
	}
	h.mu.Unlock()
	return sessions
}

func (h *globalHousekeeper) run() {
	ticker := time.NewTicker(housekeeperTick)
	defer ticker.Stop()

	ackTicker := time.NewTicker(ackFlushInterval)
	defer ackTicker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			for _, s := range h.snapshot() {
				s.runScheduledMaintenance(now)
			}
		case <-ackTicker.C:
			for _, s := range h.snapshot() {
				s.flushAcks()
			}
		}
	}
}

func getSendJob() *sendJob {
	return sendJobPool.Get().(*sendJob)
}

func putSendJob(job *sendJob) {
	job.encrypted = nil
	sendJobPool.Put(job)
}

// hasDecodedResults returns true if any goroutine is waiting for a decoded TL
// RPC result. Raw result waiters are tracked separately so they do not force
// TL decoding or gzip unpacking.
func (s *Session) hasDecodedResults() bool {
	return s.pendingResults.Load() > 0
}

func (s *Session) hasRawResults() bool {
	return s.rawPendingResults.Load() > 0
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
	// serverSalt is the current salt value used for message encryption.
	serverSalt atomic.Int64
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

	// results maps message IDs to channels that receive RPC response objects.
	results sync.Map

	// rawResults maps request message IDs to channels that receive raw
	// rpc_result payload bytes (the result:Object bytes only).
	rawResults   map[int64]chan []byte
	rawResultsMu sync.Mutex

	// pendingResults counts active decoded TL RPC result listeners.
	pendingResults atomic.Int64
	// rawPendingResults counts active raw rpc_result payload listeners.
	rawPendingResults atomic.Int64

	// acks accumulates message IDs that need to be acknowledged.
	acks []int64
	// acksMu protects the acks slice.
	acksMu sync.Mutex

	// connected indicates whether the session is currently active.
	connected atomic.Bool
	// cancel is closed to signal background workers to stop.
	cancel chan struct{}

	// transport is the underlying network transport for sending/receiving data.
	transport Transport
	// sendCh is the queue for outgoing encrypted payloads, consumed by the
	// dedicated writer goroutine. Using a channel instead of a mutex ensures
	// that a blocked write to a dead connection never blocks RPC senders.
	sendCh chan *sendJob
	// dispatchCh is a bounded queue for raw messages that need TL decoding.
	dispatchCh chan *tg.MTProtoMessageRaw
	// dispatchWorkers is the number of workers that decode dispatchCh items.
	dispatchWorkers int
	// dispatchQueueSize is the capacity used when creating dispatchCh.
	dispatchQueueSize int
	// receiveErr receives the terminal receive loop error, if any.
	receiveErr chan error
	// pingInterval controls how often keep-alive pings are sent.
	pingInterval time.Duration
	// nextPing and nextSaltFetch are maintained by the global housekeeper.
	nextPing      time.Time
	nextSaltFetch time.Time
	// onUpdate is called when the server pushes unsolicited updates.
	onUpdate func(tg.TLObject)
	// onDisconnect is called when the transport connection dies (recv error or write failure).
	onDisconnect func(error)
	// updateSem bounds the number of concurrent update dispatch goroutines.
	updateSem chan struct{}
	// onPanic is called (if non-nil) when a dispatch goroutine panics.
	onPanic func(panicValue any)
}

// SetOnPanic sets a callback invoked when a dispatchUpdate goroutine panics.
func (s *Session) SetOnPanic(fn func(panicValue any)) {
	s.onPanic = fn
}

// SetOnDisconnect sets a callback invoked when the transport connection dies.
func (s *Session) SetOnDisconnect(fn func(error)) {
	s.onDisconnect = fn
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
		results:      sync.Map{},
		rawResults:   make(map[int64]chan []byte),
		cancel:       make(chan struct{}),
		pingInterval: 60 * time.Second,
		updateSem:    make(chan struct{}, 64),
	}
	s.SetDispatchConfig(0, 0)

	if len(authKey) > 0 {
		s.authKeyID = computeAuthKeyID(authKey)
	}

	return s, nil
}

// SetDispatchConfig configures the bounded TL decode worker pool used by the
// receive path. Non-positive values keep the defaults: runtime.GOMAXPROCS(0)
// workers and a queue capacity of 256.
func (s *Session) SetDispatchConfig(workers, queueSize int) {
	s.dispatchWorkers = resolveDispatchWorkers(workers)
	s.dispatchQueueSize = resolveDispatchQueueSize(queueSize)
}

func resolveDispatchWorkers(workers int) int {
	if workers > 0 {
		return workers
	}
	if workers = runtime.GOMAXPROCS(0); workers > 0 {
		return workers
	}
	return 1
}

func resolveDispatchQueueSize(queueSize int) int {
	if queueSize > 0 {
		return queueSize
	}
	return defaultDispatchQueueSize
}

func (s *Session) registerResult(msgID int64) chan tg.TLObject {
	ch := make(chan tg.TLObject, 1)
	s.results.Store(msgID, ch)
	s.pendingResults.Add(1)
	return ch
}

func (s *Session) unregisterResult(msgID int64) {
	s.results.Delete(msgID)
	s.pendingResults.Add(-1)
}

func (s *Session) deliverResult(msgID int64, obj tg.TLObject) {
	val, ok := s.results.Load(msgID)
	if ok {
		ch := val.(chan tg.TLObject)
		select {
		case ch <- obj:
		default:
		}
	}
}

func (s *Session) registerRawResult(msgID int64) chan []byte {
	ch := make(chan []byte, 1)
	s.rawResultsMu.Lock()
	s.rawResults[msgID] = ch
	s.rawResultsMu.Unlock()
	s.rawPendingResults.Add(1)
	return ch
}

func (s *Session) unregisterRawResult(msgID int64) {
	s.rawResultsMu.Lock()
	delete(s.rawResults, msgID)
	s.rawResultsMu.Unlock()
	s.rawPendingResults.Add(-1)
}

func (s *Session) deliverRawResult(msgID int64, data []byte) {
	s.rawResultsMu.Lock()
	ch, ok := s.rawResults[msgID]
	s.rawResultsMu.Unlock()
	if ok {
		select {
		case ch <- data:
		default:
		}
	}
}

func (s *Session) hasRawResultWaiter(msgID int64) bool {
	s.rawResultsMu.Lock()
	_, ok := s.rawResults[msgID]
	s.rawResultsMu.Unlock()
	return ok
}

func (s *Session) addAck(msgID int64) {
	s.acksMu.Lock()
	if s.acks == nil {
		s.acks = make([]int64, 0, 64)
	}
	s.acks = append(s.acks, msgID)
	s.acksMu.Unlock()
}

func (s *Session) drainAcks() []int64 {
	s.acksMu.Lock()
	acks := s.acks
	s.acks = nil
	s.acksMu.Unlock()
	return acks
}

// SetAuthKey replaces the current authorization key and recomputes its ID.
// Passing an empty slice clears the key and its ID.
func (s *Session) SetAuthKey(key []byte) {
	s.authKey = key
	if len(key) > 0 {
		s.authKeyID = computeAuthKeyID(key)
	} else {
		s.authKeyID = nil
	}
}

// SetServerSalt updates the server salt used for encrypting outgoing messages.
func (s *Session) SetServerSalt(salt int64) {
	s.serverSalt.Store(salt)
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
	if len(s.authKey) == 0 {
		return nil
	}
	cp := make([]byte, len(s.authKey))
	copy(cp, s.authKey)
	return cp
}

// IsConnected reports whether the session is currently active and connected.
func (s *Session) IsConnected() bool {
	return s.connected.Load()
}

// SetTransport replaces the underlying transport used for sending and
// receiving encrypted payloads.
func (s *Session) SetTransport(t Transport) {
	s.transport = t
}

func (s *Session) sessionIDBytes() []byte {
	return s.sidBytes[:]
}

// Send encrypts and sends a TLObject as a single MTProto message, then waits
// for the server's response. The msgID and seqNo identify the message.
// ctx is used for cancellation: when cancelled after the message has been sent,
// an RPCDropAnswerRequest is fired to inform the server the request is no longer
// needed. Returns the raw response bytes or an error.
func (s *Session) Send(ctx context.Context, msgID int64, seqNo uint32, body tg.TLObject, timeout time.Duration) (tg.TLObject, error) {
	if len(s.authKey) == 0 {
		return nil, ErrAuthKeyNotSet
	}
	if s.transport == nil {
		return nil, ErrTransportNotSet
	}

	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: seqNo,
		Body:  body,
	}

	encrypted := crypto.Pack(message, s.serverSalt.Load(), s.sessionIDBytes(), s.authKey, s.authKeyID)

	ch := s.registerResult(msgID)

	job := getSendJob()
	job.encrypted = encrypted
	job.deadline = time.Now().Add(timeout)
	job.done = make(chan error, 1)

	writeTimer := time.NewTimer(timeout)
	select {
	case s.sendCh <- job:
		writeTimer.Stop()
	case <-ctx.Done():
		writeTimer.Stop()
		putSendJob(job)
		s.unregisterResult(msgID)
		return nil, ctx.Err()
	case <-writeTimer.C:
		putSendJob(job)
		s.unregisterResult(msgID)
		return nil, fmt.Errorf("session: send: write queue full: %w", ErrSendTimeout)
	}
	if err := <-job.done; err != nil {
		putSendJob(job)
		s.unregisterResult(msgID)
		if s.onDisconnect != nil {
			s.onDisconnect(err)
		}
		return nil, fmt.Errorf("session: send: %w", err)
	}
	putSendJob(job)

	respTimer := time.NewTimer(timeout)
	select {
	case obj := <-ch:
		respTimer.Stop()
		s.unregisterResult(msgID)
		return obj, nil
	case <-ctx.Done():
		respTimer.Stop()
		s.unregisterResult(msgID)
		s.sendRPCDrop(msgID)
		return nil, ctx.Err()
	case <-s.cancel:
		respTimer.Stop()
		s.unregisterResult(msgID)
		return nil, ErrSessionClosed
	case <-respTimer.C:
		s.unregisterResult(msgID)
		return nil, ErrSendTimeout
	}
}

func (s *Session) sendRPCDrop(reqMsgID int64) {
	drop := &tg.RPCDropAnswerRequest{ReqMsgID: reqMsgID}
	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(false)
	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: uint32(seqNo),
		Body:  drop,
	}
	encrypted := crypto.Pack(message, s.serverSalt.Load(), s.sessionIDBytes(), s.authKey, s.authKeyID)
	job := getSendJob()
	job.encrypted = encrypted
	job.deadline = time.Now().Add(5 * time.Second)
	select {
	case s.sendCh <- job:
	default:
		putSendJob(job)
	}
}

func (s *Session) handlePacket(msgID int64, seqNo uint32, body tg.TLObject) {
	s.addAck(msgID)

	obj := body
	if gz, ok := body.(*tg.GzipPacked); ok {
		decoded, err := gz.Decode()
		if err != nil {
			return
		}
		obj = decoded
	}

	switch v := obj.(type) {
	case *tg.Pong:
		s.deliverResult(v.MsgID, v)
	case tg.BadMsgNotificationClass:
		switch bv := v.(type) {
		case *tg.BadMsgNotification:
			s.deliverResult(bv.BadMsgID, bv)
		case *tg.BadServerSalt:
			s.serverSalt.Store(bv.NewServerSalt)
			s.deliverResult(bv.BadMsgID, bv)
		}
	case *tg.NewSessionCreated:
		s.serverSalt.Store(v.ServerSalt)
	case *tg.FutureSalts:
		s.storeFutureSalts(v)
	case *tg.MsgsAck:
	case *tg.RPCResult:
		result := v.Result
		if gz, ok := result.(*tg.GzipPacked); ok {
			decoded, err := gz.Decode()
			if err != nil {
				return
			}
			result = decoded
		}
		s.deliverResult(v.ReqMsgID, result)
	case tg.UpdatesClass:
		if s.onUpdate != nil {
			s.dispatchUpdate(obj)
		}
	default:
		if _, hasResult := s.results.Load(msgID); hasResult {
			s.deliverResult(msgID, obj)
		} else if s.onUpdate != nil {
			s.dispatchUpdate(obj)
		}
	}
}

// dispatchUpdate spawns a goroutine to deliver an update to the handler,
// bounded by the updateSem semaphore. If 64 dispatches are already in
// flight, this blocks until one completes, providing backpressure.
func (s *Session) dispatchUpdate(obj tg.TLObject) {
	select {
	case <-s.cancel:
		return
	case s.updateSem <- struct{}{}:
		go func() {
			defer func() { <-s.updateSem }()
			defer func() {
				if r := recover(); r != nil {
					if s.onPanic != nil {
						s.onPanic(r)
					} else {
						log.Printf("session: dispatchUpdate panic: %v", r)
					}
				}
			}()
			s.onUpdate(obj)
		}()
	}
}

// SendRaw encrypts and sends raw body bytes as a single MTProto message, then
// waits for the matching rpc_result and returns its raw result:Object payload
// bytes. Unlike [Send], the response path does not gzip-unpack or TL-decode the
// payload.
func (s *Session) SendRaw(ctx context.Context, msgID int64, seqNo uint32, bodyBytes []byte, timeout time.Duration) ([]byte, error) {
	if len(s.authKey) == 0 {
		return nil, ErrAuthKeyNotSet
	}
	if s.transport == nil {
		return nil, ErrTransportNotSet
	}

	encrypted := crypto.PackRaw(msgID, seqNo, bodyBytes, s.serverSalt.Load(), s.sessionIDBytes(), s.authKey, s.authKeyID)

	ch := s.registerRawResult(msgID)

	job := getSendJob()
	job.encrypted = encrypted
	job.deadline = time.Now().Add(timeout)
	job.done = make(chan error, 1)

	writeTimer := time.NewTimer(timeout)
	select {
	case s.sendCh <- job:
		writeTimer.Stop()
	case <-writeTimer.C:
		putSendJob(job)
		s.unregisterRawResult(msgID)
		return nil, fmt.Errorf("session: send raw: write queue full: %w", ErrSendTimeout)
	}
	if err := <-job.done; err != nil {
		putSendJob(job)
		s.unregisterRawResult(msgID)
		if s.onDisconnect != nil {
			s.onDisconnect(err)
		}
		return nil, fmt.Errorf("session: send raw: %w", err)
	}
	putSendJob(job)

	respTimer := time.NewTimer(timeout)
	select {
	case data := <-ch:
		respTimer.Stop()
		s.unregisterRawResult(msgID)
		return data, nil
	case <-ctx.Done():
		respTimer.Stop()
		s.unregisterRawResult(msgID)
		s.sendRPCDrop(msgID)
		return nil, ctx.Err()
	case <-s.cancel:
		respTimer.Stop()
		s.unregisterRawResult(msgID)
		return nil, ErrSessionClosed
	case <-respTimer.C:
		s.unregisterRawResult(msgID)
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
			lastErr = fmt.Errorf("invoke raw: send: %w", err)
			continue
		}
		return data, nil
	}
	if lastErr == nil {
		return nil, fmt.Errorf("session: invoke raw: retries exhausted (%d)", retries)
	}
	return nil, fmt.Errorf("session: invoke raw: retries exhausted (%d): %w", retries, lastErr)
}

// Invoke sends an RPC query and decodes the response into a TLObject.
// It retries the request up to retries times with the given per-attempt
// timeout. Returns the decoded response object or the last error encountered.
func (s *Session) Invoke(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error) {
	methodName := typeName(query)

	var lastErr error
	for i := 0; i < retries; i++ {
		msgID := s.msgFactory.AllocateMsgID()
		seqNo := s.msgFactory.AllocateSeqNo(true)

		obj, err := s.Send(ctx, msgID, uint32(seqNo), query, timeout)
		if err != nil {
			lastErr = fmt.Errorf("invoke %s: send: %w", methodName, err)
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

// Start launches the receive and writer background workers, registers the
// session with the global housekeeper, and performs an initial ping to verify
// connectivity. Returns an error if the initial ping fails, in which case the
// session is stopped automatically.
func (s *Session) Start(timeout time.Duration) error {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return s.start(context.Background(), ctx)
}

// StartContext launches the receive and writer background workers, registers
// the session with the global housekeeper, and performs an initial ping to
// verify connectivity. It returns after the session is ready.
func (s *Session) StartContext(ctx context.Context) error {
	return s.start(context.Background(), ctx)
}

func (s *Session) start(loopCtx, pingCtx context.Context) error {
	s.cancel = make(chan struct{})
	s.sendCh = make(chan *sendJob, 64)
	s.dispatchCh = make(chan *tg.MTProtoMessageRaw, s.dispatchQueueSize)
	s.receiveErr = make(chan error, 1)
	s.connected.Store(true)

	go s.writer()
	s.startDispatchWorkers(loopCtx, s.dispatchWorkers)
	go func() {
		s.receiveErr <- s.receiveLoop(loopCtx)
	}()

	timeout := timeoutFromContext(pingCtx)
	_, err := s.Invoke(pingCtx, &tg.PingRequest{PingID: time.Now().UnixNano()}, 3, timeout)
	if err != nil {
		s.Stop()
		return fmt.Errorf("session: start: initial ping failed: %w", err)
	}
	now := time.Now()
	s.nextPing = now.Add(s.pingInterval)
	s.nextSaltFetch = now.Add(initialSaltFetchWait)
	defaultHousekeeper.register(s)
	return nil
}

// Run starts the session and blocks until ctx is canceled.
func (s *Session) Run(ctx context.Context) error {
	if err := s.start(ctx, ctx); err != nil {
		return err
	}
	select {
	case err := <-s.receiveErr:
		s.Stop()
		return err
	case <-ctx.Done():
		s.Stop()
		return ctx.Err()
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

// Stop signals all background workers to exit, closes the cancel channel,
// and closes the underlying transport.
func (s *Session) Stop() {
	s.connected.Store(false)
	defaultHousekeeper.unregister(s)
	if s.cancel != nil {
		select {
		case <-s.cancel:
		default:
			close(s.cancel)
		}
	}
	if s.transport != nil {
		s.transport.Close()
	}
}

func (s *Session) storeFutureSalts(fs *tg.FutureSalts) {
	if len(fs.Salts) == 0 {
		return
	}
	s.serverSalt.Store(fs.Salts[0].Salt)
}

func (s *Session) sendServiceMessage(body tg.TLObject) {
	select {
	case <-s.cancel:
		return
	default:
	}
	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(false)
	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: uint32(seqNo),
		Body:  body,
	}
	encrypted := crypto.Pack(message, s.serverSalt.Load(), s.sessionIDBytes(), s.authKey, s.authKeyID)
	job := getSendJob()
	job.encrypted = encrypted
	job.deadline = time.Now().Add(10 * time.Second)
	select {
	case s.sendCh <- job:
	case <-s.cancel:
		putSendJob(job)
	default:
		putSendJob(job)
	}
}

func (s *Session) runScheduledMaintenance(now time.Time) {
	select {
	case <-s.cancel:
		return
	default:
	}
	if !s.nextSaltFetch.IsZero() && !now.Before(s.nextSaltFetch) {
		s.sendServiceMessage(&tg.GetFutureSaltsRequest{Num: numFutureSalts})
		s.nextSaltFetch = now.Add(saltFetchInterval)
	}
	pingInterval := s.pingInterval
	if pingInterval <= 0 {
		pingInterval = 60 * time.Second
	}
	if !s.nextPing.IsZero() && !now.Before(s.nextPing) {
		s.sendPing()
		s.nextPing = now.Add(pingInterval)
	}
}

func (s *Session) flushAcks() {
	select {
	case <-s.cancel:
		return
	default:
	}
	acks := s.drainAcks()
	const maxAckBatch = 8192
	for len(acks) > 0 {
		batch := acks
		if len(batch) > maxAckBatch {
			batch = batch[:maxAckBatch]
			acks = acks[maxAckBatch:]
		} else {
			acks = nil
		}
		s.sendServiceMessage(&tg.MsgsAck{MsgIds: batch})
	}
}

func (s *Session) writer() {
	for {
		select {
		case <-s.cancel:
			return
		case job := <-s.sendCh:
			s.transport.SetWriteDeadline(job.deadline)
			err := s.transport.Send(job.encrypted)
			s.transport.SetWriteDeadline(time.Time{})
			if job.done != nil {
				job.done <- err
			} else {
				putSendJob(job)
			}
			if err != nil && s.onDisconnect != nil {
				s.onDisconnect(err)
			}
		}
	}
}

func (s *Session) startDispatchWorkers(ctx context.Context, workers int) {
	if workers < 1 {
		workers = 1
	}
	for range workers {
		go s.dispatchWorker(ctx)
	}
}

func (s *Session) dispatchWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.cancel:
			return
		case raw := <-s.dispatchCh:
			s.dispatchRaw(raw)
		}
	}
}

func (s *Session) dispatchRaw(raw *tg.MTProtoMessageRaw) {
	defer func() {
		if r := recover(); r != nil {
			if s.onPanic != nil {
				s.onPanic(r)
			} else {
				log.Printf("session: dispatchRaw panic: %v", r)
			}
		}
	}()
	bodyReader := tg.NewReader(raw.BodyRaw)
	defer tg.ReleaseReader(bodyReader)
	body, err := tg.ReadTLObject(bodyReader)
	if err != nil {
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
		s.deliverResult(int64(binary.LittleEndian.Uint64(body[4:12])), &tg.Pong{
			MsgID:  int64(binary.LittleEndian.Uint64(body[4:12])),
			PingID: int64(binary.LittleEndian.Uint64(body[12:20])),
		})
	case tg.BadMsgNotificationTypeID:
		if len(body) < 20 {
			return false
		}
		s.deliverResult(int64(binary.LittleEndian.Uint64(body[4:12])), &tg.BadMsgNotification{
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
		s.serverSalt.Store(newSalt)
		s.deliverResult(badMsgID, &tg.BadServerSalt{
			BadMsgID:      badMsgID,
			BadMsgSeqno:   int32(binary.LittleEndian.Uint32(body[12:16])),
			ErrorCode:     int32(binary.LittleEndian.Uint32(body[16:20])),
			NewServerSalt: newSalt,
		})
	case tg.NewSessionCreatedTypeID:
		if len(body) < 28 {
			return false
		}
		s.serverSalt.Store(int64(binary.LittleEndian.Uint64(body[20:28])))
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
	if s.hasRawResultWaiter(reqMsgID) {
		result := make([]byte, len(payload))
		copy(result, payload)
		s.deliverRawResult(reqMsgID, result)
	}
	if _, ok := s.results.Load(reqMsgID); !ok {
		return
	}
	result, err := decodeRawRPCResultPayload(payload)
	if err != nil {
		return
	}
	s.deliverResult(reqMsgID, result)
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
	if !s.hasDecodedResults() && s.onUpdate == nil {
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
	const firstSaltOffset = 24
	if len(body) < firstSaltOffset {
		return false
	}
	if binary.LittleEndian.Uint32(body[16:20]) != tg.VectorTypeID {
		return false
	}
	count := binary.LittleEndian.Uint32(body[20:24])
	if count == 0 {
		return true
	}
	if len(body) < firstSaltOffset+20 {
		return false
	}
	if binary.LittleEndian.Uint32(body[firstSaltOffset:firstSaltOffset+4]) != tg.FutureSaltTypeID {
		return false
	}
	s.serverSalt.Store(int64(binary.LittleEndian.Uint64(body[firstSaltOffset+12 : firstSaltOffset+20])))
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
		if _, ok := s.results.Load(msgIDs[i]); ok {
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

func (s *Session) receiveLoop(ctx context.Context) error {
	var lastDisconnect time.Time
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.cancel:
			return nil
		default:
		}

		data, err := s.transport.Recv()
		if err != nil {
			if !s.connected.Load() {
				return nil
			}
			if s.onDisconnect != nil && time.Since(lastDisconnect) > time.Second {
				s.onDisconnect(err)
				lastDisconnect = time.Now()
			}
			timer := time.NewTimer(100 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-s.cancel:
				timer.Stop()
				return nil
			case <-timer.C:
			}
			continue
		}
		if len(data) == 4 {
			code := int32(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24)
			if code < 0 {
				continue
			}
			if s.onDisconnect != nil {
				s.onDisconnect(fmt.Errorf("transport error: code %d", -code))
			}
			return fmt.Errorf("transport error: code %d", -code)
		}

		raw, decrypted, err := unpackIncomingMessageEnvelope(data, s.sessionIDBytes(), s.authKey, s.authKeyID)
		if err != nil {
			if _, ok := err.(*tgerr.SecurityCheckMismatch); ok {
				if s.onDisconnect != nil {
					s.onDisconnect(err)
				}
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

		if rawHandled || (!needsDecodedResult && s.onUpdate == nil) {
			continue
		}

		// Only decode the TL body if decoded result listeners or update handlers
		// need the object tree.
		if needsDecodedResult || s.onUpdate != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-s.cancel:
				return nil
			case s.dispatchCh <- raw:
			}
		}
	}
}

func (s *Session) sendPing() {
	s.sendServiceMessage(&tg.PingDelayDisconnectRequest{
		PingID:          time.Now().UnixNano(),
		DisconnectDelay: 65,
	})
}

func (s *Session) setPingInterval(d time.Duration) {
	s.pingInterval = d
}

// SetUpdateHandler registers fn as the callback for unsolicited server
// updates (e.g., new messages, status changes). Pass nil to remove the
// handler.
func (s *Session) SetUpdateHandler(fn func(tg.TLObject)) {
	s.onUpdate = fn
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
	if transport != nil {
		s.transport = transport
	}
	if len(s.authKey) == 0 {
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
	if transport != nil {
		s.transport = transport
	}
	if len(s.authKey) == 0 && authFunc != nil {
		result, err := authFunc(s.transport)
		if err != nil {
			return fmt.Errorf("session: connect with auth: %w", err)
		}
		s.authKey = result.AuthKey
		s.authKeyID = computeAuthKeyID(result.AuthKey)
		s.serverSalt.Store(result.ServerSalt)
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
