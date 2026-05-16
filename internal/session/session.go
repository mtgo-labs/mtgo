package session

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
	"github.com/mtgo-labs/storage"
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

	// msgFactory generates unique message IDs and sequence numbers.
	msgFactory *MsgFactory

	// results maps message IDs to channels that receive RPC response objects.
	results map[int64]chan tg.TLObject
	// resultsMu protects the results map.
	resultsMu sync.Mutex

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
	// onPanic is called (if non-nil) when a dispatchUpdate goroutine panics.
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

	s := &Session{
		dc:           dc,
		storage:      st,
		deviceModel:  deviceModel,
		appVersion:   appVersion,
		systemLang:   systemLang,
		langCode:     langCode,
		authKey:      authKey,
		sessionID:    sid,
		msgFactory:   NewMsgFactory(time.Now()),
		results:      make(map[int64]chan tg.TLObject),
		cancel:       make(chan struct{}),
		pingInterval: 60 * time.Second,
		updateSem:    make(chan struct{}, 64),
	}

	if len(authKey) > 0 {
		s.authKeyID = computeAuthKeyID(authKey)
	}

	return s, nil
}

func (s *Session) registerResult(msgID int64) chan tg.TLObject {
	ch := make(chan tg.TLObject, 1)
	s.resultsMu.Lock()
	s.results[msgID] = ch
	s.resultsMu.Unlock()
	return ch
}

func (s *Session) unregisterResult(msgID int64) {
	s.resultsMu.Lock()
	delete(s.results, msgID)
	s.resultsMu.Unlock()
}

func (s *Session) deliverResult(msgID int64, obj tg.TLObject) {
	s.resultsMu.Lock()
	ch, ok := s.results[msgID]
	s.resultsMu.Unlock()
	if ok {
		select {
		case ch <- obj:
		default:
		}
	}
}

func (s *Session) addAck(msgID int64) {
	s.acksMu.Lock()
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
	return s.authKey
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
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(s.sessionID))
	return buf[:]
}

// Send encrypts and sends a TLObject as a single MTProto message, then waits
// for the server's response. The msgID and seqNo identify the message.
// It returns the raw response bytes or an error if the auth key is missing,
// the transport is unset, the send fails, or the timeout is exceeded.
func (s *Session) Send(msgID int64, seqNo uint32, body tg.TLObject, timeout time.Duration) (tg.TLObject, error) {
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

	job := &sendJob{
		encrypted: encrypted,
		deadline:  time.Now().Add(timeout),
		done:      make(chan error, 1),
	}
	select {
	case s.sendCh <- job:
	case <-time.After(timeout):
		s.unregisterResult(msgID)
		return nil, fmt.Errorf("session: send: write queue full: %w", ErrSendTimeout)
	}
	if err := <-job.done; err != nil {
		s.unregisterResult(msgID)
		if s.onDisconnect != nil {
			s.onDisconnect(err)
		}
		return nil, fmt.Errorf("session: send: %w", err)
	}

	select {
	case obj := <-ch:
		s.unregisterResult(msgID)
		return obj, nil
	case <-time.After(timeout):
		s.unregisterResult(msgID)
		return nil, ErrSendTimeout
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
		s.resultsMu.Lock()
		_, hasResult := s.results[msgID]
		s.resultsMu.Unlock()
		if hasResult {
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

func unpackIncomingMessage(data, sessionID, authKey, authKeyID []byte) (*tg.MTProtoMessage, []byte, error) {
	msg, decrypted, err := crypto.Unpack(data, sessionID, authKey, authKeyID)
	if err != nil {
		return nil, nil, fmt.Errorf("session: unpack incoming message: %w", err)
	}
	return msg, decrypted, nil
}

// Invoke sends an RPC query and decodes the response into a TLObject.
// It retries the request up to retries times with the given per-attempt
// timeout. Returns the decoded response object or the last error encountered.
func (s *Session) Invoke(query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error) {
	if retries < 1 {
		retries = 1
	}

	var lastErr error
	for i := 0; i < retries; i++ {
		msgID := s.msgFactory.AllocateMsgID()
		seqNo := s.msgFactory.AllocateSeqNo(true)

		obj, err := s.Send(msgID, uint32(seqNo), query, timeout)
		if err != nil {
			lastErr = err
			continue
		}

		if bad, ok := obj.(tg.BadMsgNotificationClass); ok {
			switch v := bad.(type) {
			case *tg.BadMsgNotification:
				lastErr = fmt.Errorf("bad message notification: code=%d", v.ErrorCode)
			case *tg.BadServerSalt:
				lastErr = fmt.Errorf("bad server salt: code=%d", v.ErrorCode)
			default:
				lastErr = fmt.Errorf("bad message notification: %T", bad)
			}
			continue
		}

		if rpcErr, ok := obj.(*tg.RPCError); ok {
			if rpcErr.ErrorCode == 303 {
				return obj, nil
			}
			parsed := tgerr.New(int(rpcErr.ErrorCode), rpcErr.ErrorMessage)
			if d, ok := tgerr.AsFloodWait(parsed); ok {
				time.Sleep(d + time.Second)
				retries++
				continue
			}
			// Non-retryable errors (auth, permission, bad request) fail immediately.
			if rpcErr.ErrorCode == 401 || rpcErr.ErrorCode == 400 || rpcErr.ErrorCode == 403 {
				return nil, parsed
			}
			lastErr = parsed
			continue
		}

		return obj, nil
	}
	return nil, fmt.Errorf("session: invoke: retries exhausted: %w", lastErr)
}

// Start launches the receive, writer, and keep-alive ping background workers and
// performs an initial ping to verify connectivity. Returns an error if the
// initial ping fails, in which case the session is stopped automatically.
func (s *Session) Start(timeout time.Duration) error {
	s.cancel = make(chan struct{})
	s.sendCh = make(chan *sendJob, 64)
	s.connected = true

	go s.writer()
	go s.recvWorker()
	go s.pingWorker()

	_, err := s.Invoke(&tg.PingRequest{PingID: time.Now().UnixNano()}, 3, timeout)
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

func (s *Session) writer() {
	for {
		select {
		case <-s.cancel:
			return
		case job := <-s.sendCh:
			s.transport.SetWriteDeadline(job.deadline)
			err := s.transport.Send(job.encrypted)
			s.transport.SetWriteDeadline(time.Time{})
			job.done <- err
		}
	}
}

func (s *Session) recvWorker() {
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
		msg, _, err := unpackIncomingMessage(data, s.sessionIDBytes(), s.authKey, s.authKeyID)
		if err != nil {
			continue
		}
		if msg == nil || msg.Body == nil {
			continue
		}

		if container, ok := msg.Body.(*tg.MsgContainer); ok {
			for _, subMsg := range container.Messages {
				if subMsg.Body != nil {
					s.handlePacket(subMsg.MsgID, subMsg.SeqNo, subMsg.Body)
				}
			}
			continue
		}

		s.handlePacket(msg.MsgID, msg.SeqNo, msg.Body)
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

			job := &sendJob{
				encrypted: encrypted,
				deadline:  time.Now().Add(30 * time.Second),
				done:      make(chan error, 1),
			}
			select {
			case s.sendCh <- job:
			default:
				continue
			}
			if err := <-job.done; err != nil {
				if s.onDisconnect != nil {
					s.onDisconnect(err)
				}
				continue
			}
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
