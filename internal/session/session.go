package session

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
	"github.com/mtgo-labs/mtgo/internal/storage"
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

var sendJobPool = sync.Pool{
	New: func() any {
		return &sendJob{}
	},
}

func getSendJob() *sendJob {
	return sendJobPool.Get().(*sendJob)
}

func putSendJob(job *sendJob) {
	job.encrypted = nil
	sendJobPool.Put(job)
}

// hasPendingResults returns true if any goroutine is waiting for an RPC result
// (TL or raw). When true, incoming messages must be fully decoded for dispatch.
func (s *Session) hasPendingResults() bool {
	return s.pendingResults.Load() > 0
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
	serverSalt int64
	// sessionID is a random identifier for this session, unique per connection.
	sessionID int64
	// sidBytes is the little-endian encoding of sessionID, cached to avoid
	// allocating on every Send call. Populated at construction time.
	sidBytes [8]byte

	// msgFactory generates unique message IDs and sequence numbers.
	msgFactory *MsgFactory

	// results maps message IDs to channels that receive RPC response objects.
	results sync.Map

	// rawResults maps message IDs to channels that receive raw decrypted
	// response bytes (body only, no MTProto framing).
	rawResults   map[int64]chan []byte
	rawResultsMu sync.Mutex

	// pendingResults counts active RPC result listeners (both TL and raw).
	// Non-zero means readLoop must decode incoming messages for dispatch.
	pendingResults atomic.Int64

	// acks accumulates message IDs that need to be acknowledged.
	acks []int64
	// acksMu protects the acks slice.
	acksMu sync.Mutex

	// connected indicates whether the session is currently active.
	connected bool
	// cancel is closed to signal background workers to stop.
	cancel chan struct{}

	// transport is the underlying network transport for sending/receiving data.
	transport Transport
	// sendCh is the queue for outgoing encrypted payloads, consumed by the
	// dedicated writer goroutine. Using a channel instead of a mutex ensures
	// that a blocked write to a dead connection never blocks RPC senders.
	sendCh chan *sendJob
	// pingInterval controls how often keep-alive pings are sent.
	pingInterval time.Duration
	// onUpdate is called when the server pushes unsolicited updates.
	onUpdate func(tg.TLObject)
	// onDisconnect is called when the transport connection dies (recv error or write failure).
	onDisconnect func(error)
	// updateSem bounds the number of concurrent update dispatch goroutines.
	updateSem chan struct{}
	// dispatchSem bounds the number of concurrent message dispatch goroutines.
	dispatchSem chan struct{}
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
		dc:           dc,
		storage:      st,
		deviceModel:  deviceModel,
		appVersion:   appVersion,
		systemLang:   systemLang,
		langCode:     langCode,
		authKey:      authKey,
		sessionID:    sid,
		sidBytes:     encodedSidBytes,
		msgFactory:   NewMsgFactory(time.Now()),
		results:      sync.Map{},
		rawResults:   make(map[int64]chan []byte),
		cancel:       make(chan struct{}),
		pingInterval: 60 * time.Second,
		updateSem:    make(chan struct{}, 64),
		dispatchSem: make(chan struct{}, 128),
	}

	if len(authKey) > 0 {
		s.authKeyID = computeAuthKeyID(authKey)
	}

	return s, nil
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
	s.pendingResults.Add(1)
	return ch
}

func (s *Session) unregisterRawResult(msgID int64) {
	s.rawResultsMu.Lock()
	delete(s.rawResults, msgID)
	s.rawResultsMu.Unlock()
	s.pendingResults.Add(-1)
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
	s.serverSalt = salt
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
	return s.connected
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

	encrypted := crypto.Pack(message, s.serverSalt, s.sessionIDBytes(), s.authKey, s.authKeyID)

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
	encrypted := crypto.Pack(message, s.serverSalt, s.sessionIDBytes(), s.authKey, s.authKeyID)
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
			s.serverSalt = bv.NewServerSalt
			s.deliverResult(bv.BadMsgID, bv)
		}
	case *tg.NewSessionCreated:
		s.serverSalt = v.ServerSalt
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
	default:
	}
}



// SendRaw encrypts and sends raw body bytes as a single MTProto message,
// then waits for the server's raw response bytes. Unlike [Send], it bypasses
// TL serialization and deserialization entirely. Returns the raw decrypted
// body bytes from the response, or an error.
func (s *Session) SendRaw(ctx context.Context, msgID int64, seqNo uint32, bodyBytes []byte, timeout time.Duration) ([]byte, error) {
	if len(s.authKey) == 0 {
		return nil, ErrAuthKeyNotSet
	}
	if s.transport == nil {
		return nil, ErrTransportNotSet
	}

	encrypted := crypto.PackRaw(msgID, seqNo, bodyBytes, s.serverSalt, s.sessionIDBytes(), s.authKey, s.authKeyID)

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

// InvokeRaw sends a TLObject query and returns the raw response body bytes
// without TL decoding. It retries the request up to retries times with the
// given per-attempt timeout.
func (s *Session) InvokeRaw(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) ([]byte, error) {
	if retries < 1 {
		retries = 1
	}

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
	return nil, fmt.Errorf("session: invoke raw: retries exhausted (%d): %w", retries, lastErr)
}

// Invoke sends an RPC query and decodes the response into a TLObject.
// It retries the request up to retries times with the given per-attempt
// timeout. Returns the decoded response object or the last error encountered.
func (s *Session) Invoke(ctx context.Context, query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error) {
	if retries < 1 {
		retries = 1
	}

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

// Start launches the receive, writer, and keep-alive ping background workers and
// performs an initial ping to verify connectivity. Returns an error if the
// initial ping fails, in which case the session is stopped automatically.
func (s *Session) Start(timeout time.Duration) error {
	s.cancel = make(chan struct{})
	s.sendCh = make(chan *sendJob, 64)
	s.connected = true

	go s.ackLoop()
	go s.saltLoop()
	go s.writer()
	go s.readLoop()
	go s.pingWorker()

	_, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: time.Now().UnixNano()}, 3, timeout)
	if err != nil {
		s.Stop()
		return fmt.Errorf("session: start: initial ping failed: %w", err)
	}
	return nil
}

// Stop signals all background workers to exit, closes the cancel channel,
// and closes the underlying transport.
func (s *Session) Stop() {
	s.connected = false
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
	s.serverSalt = fs.Salts[0].Salt
}

func (s *Session) sendServiceMessage(body tg.TLObject) {
	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(false)
	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: uint32(seqNo),
		Body:  body,
	}
	encrypted := crypto.Pack(message, s.serverSalt, s.sessionIDBytes(), s.authKey, s.authKeyID)
	job := getSendJob()
	job.encrypted = encrypted
	job.deadline = time.Now().Add(10 * time.Second)
	select {
	case s.sendCh <- job:
	default:
		putSendJob(job)
	}
}

func (s *Session) saltLoop() {
	select {
	case <-s.cancel:
		return
	case <-time.After(15 * time.Second):
	}

	const numSalts = 4

	for {
		s.sendServiceMessage(&tg.GetFutureSaltsRequest{Num: numSalts})

		select {
		case <-s.cancel:
			return
		case <-time.After(1 * time.Hour):
		}
	}
}

func (s *Session) ackLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.cancel:
			return
		case <-ticker.C:
			acks := s.drainAcks()
			if len(acks) == 0 {
				continue
			}
			msgID := s.msgFactory.AllocateMsgID()
			seqNo := s.msgFactory.AllocateSeqNo(false)
			message := &tg.MTProtoMessage{
				MsgID: msgID,
				SeqNo: uint32(seqNo),
				Body:  &tg.MsgsAck{MsgIds: acks},
			}
			encrypted := crypto.Pack(message, s.serverSalt, s.sessionIDBytes(), s.authKey, s.authKeyID)
			job := &sendJob{
				encrypted: encrypted,
				deadline:  time.Now().Add(10 * time.Second),
			}
			select {
			case s.sendCh <- job:
			default:
			}
		}
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
			}
			if err != nil && s.onDisconnect != nil {
				s.onDisconnect(err)
			}
		}
	}
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

func (s *Session) readLoop() {
	var lastDisconnect time.Time
	for {
		select {
		case <-s.cancel:
			return
		default:
		}

		data, err := s.transport.Recv()
		if err != nil {
			if !s.connected {
				return
			}
			if s.onDisconnect != nil && time.Since(lastDisconnect) > time.Second {
				s.onDisconnect(err)
				lastDisconnect = time.Now()
			}
			select {
			case <-s.cancel:
				return
			case <-time.After(100 * time.Millisecond):
			}
			continue
		}
		raw, decrypted, err := unpackIncomingMessageEnvelope(data, s.sessionIDBytes(), s.authKey, s.authKeyID)
		if err != nil {
			continue
		}

		s.addAck(raw.MsgID)

		// Deliver raw body bytes to registered raw result channels.
		if len(raw.BodyRaw) > 0 {
			bodyCopy := make([]byte, len(raw.BodyRaw))
			copy(bodyCopy, raw.BodyRaw)
			s.deliverRawResult(raw.MsgID, bodyCopy)
			if len(bodyCopy) >= 12 {
				reqMsgID := int64(binary.LittleEndian.Uint64(bodyCopy[4:12]))
				s.deliverRawResult(reqMsgID, bodyCopy)
			}
		}

		crypto.ReleaseAESBuf(decrypted)

		// Only decode the TL body if there are result listeners or update handlers.
		if s.hasPendingResults() || s.onUpdate != nil {
			select {
			case s.dispatchSem <- struct{}{}:
				go func(rawMsg *tg.MTProtoMessageRaw) {
					defer func() { <-s.dispatchSem }()
					defer func() {
						if r := recover(); r != nil {
							if s.onPanic != nil {
								s.onPanic(r)
							} else {
								log.Printf("session: readLoop dispatch panic: %v", r)
							}
						}
					}()
					bodyReader := tg.NewReader(rawMsg.BodyRaw)
					defer tg.ReleaseReader(bodyReader)
					body, err := tg.ReadTLObject(bodyReader)
					if err != nil {
						return
					}
					s.processIncoming(&tg.MTProtoMessage{MsgID: rawMsg.MsgID, SeqNo: rawMsg.SeqNo, Body: body})
				}(raw)
			default:
			}
		}
	}
}

func (s *Session) pingWorker() {
	ticker := time.NewTicker(s.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.cancel:
			return
		case <-ticker.C:
			msgID := s.msgFactory.AllocateMsgID()
			seqNo := s.msgFactory.AllocateSeqNo(false)
			ping := &tg.PingDelayDisconnectRequest{
				PingID:          time.Now().UnixNano(),
				DisconnectDelay: 65,
			}

			message := &tg.MTProtoMessage{
				MsgID: msgID,
				SeqNo: uint32(seqNo),
				Body:  ping,
			}

			encrypted := crypto.Pack(message, s.serverSalt, s.sessionIDBytes(), s.authKey, s.authKeyID)

			job := getSendJob()
			job.encrypted = encrypted
			job.deadline = time.Now().Add(30 * time.Second)
			job.done = make(chan error, 1)
			select {
			case s.sendCh <- job:
			default:
				putSendJob(job)
				continue
			}
			if err := <-job.done; err != nil {
				putSendJob(job)
				if s.onDisconnect != nil {
					s.onDisconnect(err)
				}
				continue
			}
			putSendJob(job)
		}
	}
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
	if transport != nil {
		s.transport = transport
	}
	if len(s.authKey) == 0 {
		return ErrConnectNoAuthKey
	}
	return s.Start(timeout)
}

// ConnectWithAuth sets the transport and performs key generation via authFunc
// if no auth key is currently set. The resulting auth key and server salt are
// persisted to storage. After authentication, the session is started.
func (s *Session) ConnectWithAuth(transport Transport, authFunc AuthFunc, timeout time.Duration) error {
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
		s.serverSalt = result.ServerSalt
		if s.storage != nil {
			if err := s.storage.SetAuthKey(result.AuthKey); err != nil {
				return fmt.Errorf("session: save auth key: %w", err)
			}
		}
		s.msgFactory.UpdateServerTime(time.Unix(int64(result.ServerTime), 0))
	}
	return s.Start(timeout)
}

func unpackIncomingMessageEnvelope(data, sessionID, authKey, authKeyID []byte) (*tg.MTProtoMessageRaw, []byte, error) {
	raw, decrypted, err := crypto.UnpackEnvelope(data, sessionID, authKey, authKeyID)
	if err != nil {
		return nil, nil, fmt.Errorf("session: unpack envelope: %w", err)
	}
	return raw, decrypted, nil
}
