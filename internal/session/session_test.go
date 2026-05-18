package session

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
)

type testStorage struct {
	dcID    int
	authKey []byte
	apiID   int32
}

func newTestStorage() *testStorage {
	return &testStorage{}
}

func (t *testStorage) DCID() (int, error)                   { return t.dcID, nil }
func (t *testStorage) SetDCID(v int) error                  { t.dcID = v; return nil }
func (t *testStorage) APIID() (int32, error)                { return t.apiID, nil }
func (t *testStorage) SetAPIID(v int32) error               { t.apiID = v; return nil }
func (t *testStorage) APIHash() (string, error)             { return "", nil }
func (t *testStorage) SetAPIHash(string) error              { return nil }
func (t *testStorage) TestMode() (bool, error)              { return false, nil }
func (t *testStorage) SetTestMode(bool) error               { return nil }
func (t *testStorage) AuthKey() ([]byte, error)             { return t.authKey, nil }
func (t *testStorage) SetAuthKey(v []byte) error            { t.authKey = v; return nil }
func (t *testStorage) UserID() (int64, error)               { return 0, nil }
func (t *testStorage) SetUserID(int64) error                { return nil }
func (t *testStorage) IsBot() (bool, error)                 { return false, nil }
func (t *testStorage) SetIsBot(bool) error                  { return nil }
func (t *testStorage) FirstName() (string, error)           { return "", nil }
func (t *testStorage) SetFirstName(string) error            { return nil }
func (t *testStorage) LastName() (string, error)            { return "", nil }
func (t *testStorage) SetLastName(string) error             { return nil }
func (t *testStorage) Username() (string, error)            { return "", nil }
func (t *testStorage) SetUsername(string) error             { return nil }
func (t *testStorage) Date() (int, error)                   { return 0, nil }
func (t *testStorage) SetDate(int) error                    { return nil }
func (t *testStorage) ServerAddress() (string, error)       { return "", nil }
func (t *testStorage) SetServerAddress(string) error        { return nil }
func (t *testStorage) Port() (int, error)                   { return 443, nil }
func (t *testStorage) SetPort(int) error                    { return nil }
func (t *testStorage) State() ([]byte, error)               { return nil, nil }
func (t *testStorage) SetState([]byte) error                { return nil }
func (t *testStorage) ExportSessionString() (string, error) { return "", nil }
func (t *testStorage) Close() error                         { return nil }
func (t *testStorage) SessionID() (string, error)           { return "test", nil }
func (t *testStorage) SetSessionID(string) error            { return nil }

func init() {
	tg.Registry[tg.PingTypeID] = func(r io.Reader) (tg.TLObject, error) {
		v := &tg.PingRequest{}
		var buf [8]byte
		r.Read(buf[:])
		v.PingID = int64(binary.LittleEndian.Uint64(buf[:]))
		return v, nil
	}
}

func TestNewSession(t *testing.T) {
	dc := DataCenter{ID: 2, TestMode: false}
	st := newTestStorage()
	s, err := NewSession(dc, st, "TestDevice", "1.0", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() error: %v", err)
	}
	if s.DC().ID != 2 {
		t.Errorf("DC().ID = %d, want 2", s.DC().ID)
	}
	if s.SessionID() == 0 {
		t.Error("SessionID() is zero, want non-zero")
	}
	if s.IsConnected() {
		t.Error("IsConnected() = true, want false")
	}
	if s.AuthKey() != nil {
		t.Errorf("AuthKey() = %v, want nil (empty storage)", s.AuthKey())
	}
}

func TestComputeAuthKeyID(t *testing.T) {
	authKey := make([]byte, 256)
	for i := range authKey {
		authKey[i] = byte(i)
	}

	got := computeAuthKeyID(authKey)
	want := []byte{50, 209, 88, 110, 164, 87, 223, 200}
	if !bytes.Equal(got, want) {
		t.Fatalf("computeAuthKeyID() = %v, want %v", got, want)
	}
}

func TestSessionResultChannel(t *testing.T) {
	dc := DataCenter{ID: 1}
	st := newTestStorage()
	s, err := NewSession(dc, st, "Dev", "1.0", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() error: %v", err)
	}

	msgID := int64(100)
	ch := s.registerResult(msgID)

	s.deliverResult(msgID, tg.TLBool(true))

	select {
	case obj := <-ch:
		b, ok := obj.(tg.TLBool)
		if !ok || !bool(b) {
			t.Errorf("received data = %T %[1]v, want tg.TLBool(true)", obj)
		}
	default:
		t.Error("expected data on channel but got none")
	}

	s.unregisterResult(msgID)

	s.resultsMu.Lock()
	_, exists := s.results[msgID]
	s.resultsMu.Unlock()
	if exists {
		t.Error("result still exists after unregister")
	}
}

func TestSessionAckTracking(t *testing.T) {
	dc := DataCenter{ID: 1}
	st := newTestStorage()
	s, err := NewSession(dc, st, "Dev", "1.0", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() error: %v", err)
	}

	s.addAck(10)
	s.addAck(20)
	s.addAck(30)

	acks := s.drainAcks()
	if len(acks) != 3 {
		t.Fatalf("drainAcks() returned %d acks, want 3", len(acks))
	}
	if acks[0] != 10 || acks[1] != 20 || acks[2] != 30 {
		t.Errorf("acks = %v, want [10 20 30]", acks)
	}

	acks2 := s.drainAcks()
	if len(acks2) != 0 {
		t.Errorf("second drainAcks() returned %d acks, want 0", len(acks2))
	}
}

type mockTransport struct {
	sendCh chan []byte
	recvCh chan []byte
	closed bool
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		sendCh: make(chan []byte, 100),
		recvCh: make(chan []byte, 100),
	}
}

func (m *mockTransport) Send(data []byte) error {
	cp := make([]byte, len(data))
	copy(cp, data)
	m.sendCh <- cp
	return nil
}

func (m *mockTransport) Recv() ([]byte, error) {
	data := <-m.recvCh
	return data, nil
}

func (m *mockTransport) Close() error {
	m.closed = true
	return nil
}

func (m *mockTransport) IsConnected() bool {
	return !m.closed
}

func (m *mockTransport) SetWriteDeadline(t time.Time) error {
	return nil
}

func makeAuthKey() []byte {
	return make([]byte, 256)
}

func makeEncryptedResponse(s *Session, msgID int64, seqNo uint32, body tg.TLObject) []byte {
	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: seqNo,
		Body:  body,
	}
	return crypto.Pack(message, s.serverSalt, s.sessionIDBytes(), s.authKey, s.authKeyID)
}

func newSessionWithAuthKey(t *testing.T) *Session {
	t.Helper()
	dc := DataCenter{ID: 2}
	st := newTestStorage()
	authKey := makeAuthKey()
	st.SetAuthKey(authKey)
	s, err := NewSession(dc, st, "TestDevice", "1.0", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() error: %v", err)
	}
	return s
}

func startTestWorkers(s *Session) {
	s.cancel = make(chan struct{})
	s.sendCh = make(chan *sendJob, 64)
	s.connected = true
	go s.writer()
	go s.readLoop()
}

func TestSessionSendAndWait(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	startTestWorkers(s)

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	pingID := time.Now().UnixNano()

	sendDone := make(chan error, 1)
	go func() {
		_, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: pingID}, 5*time.Second)
		sendDone <- err
	}()

	<-mt.sendCh

	respMsgID := s.msgFactory.AllocateMsgID()
	respSeqNo := s.msgFactory.AllocateSeqNo(false)
	pong := &tg.Pong{MsgID: msgID, PingID: pingID}
	encrypted := makeEncryptedResponse(s, respMsgID, uint32(respSeqNo), pong)
	mt.recvCh <- encrypted

	select {
	case err := <-sendDone:
		if err != nil {
			t.Fatalf("Send() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Send() timed out")
	}

	close(s.cancel)
}

func TestSessionSendDeliversRpcResultByRequestMsgID(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	startTestWorkers(s)

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	pingID := time.Now().UnixNano()

	sendDone := make(chan error, 1)
	go func() {
		_, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: pingID}, 5*time.Second)
		sendDone <- err
	}()

	<-mt.sendCh

	respMsgID := s.msgFactory.AllocateMsgID()
	respSeqNo := s.msgFactory.AllocateSeqNo(false)
	pong := &tg.Pong{MsgID: msgID, PingID: pingID}
	encrypted := makeEncryptedResponse(s, respMsgID, uint32(respSeqNo), pong)
	mt.recvCh <- encrypted

	select {
	case err := <-sendDone:
		if err != nil {
			t.Fatalf("Send() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Send() timed out")
	}

	close(s.cancel)
}

func TestSessionInvokeRetriesBadServerSalt(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	startTestWorkers(s)

	pingID := time.Now().UnixNano()
	invokeDone := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 2, 5*time.Second)
		invokeDone <- struct {
			obj tg.TLObject
			err error
		}{obj: obj, err: err}
	}()

	firstSent := <-mt.sendCh
	firstMsg := unpackIncoming(firstSent, s)
	if firstMsg == nil {
		t.Fatal("first sent message did not decode")
	}

	newSalt := int64(0x0102030405060708)
	mt.recvCh <- makeEncryptedResponse(s, s.msgFactory.AllocateMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.BadServerSalt{
		BadMsgID:      firstMsg.MsgID,
		BadMsgSeqno:   int32(firstMsg.SeqNo),
		ErrorCode:     48,
		NewServerSalt: newSalt,
	})

	secondSent := <-mt.sendCh
	secondMsg := unpackIncoming(secondSent, s)
	if secondMsg == nil {
		t.Fatal("second sent message did not decode")
	}
	if secondMsg.MsgID == firstMsg.MsgID {
		t.Fatal("retry reused msg_id")
	}
	if s.serverSalt != newSalt {
		t.Fatalf("serverSalt = %x, want %x", s.serverSalt, newSalt)
	}

	mt.recvCh <- makeEncryptedResponse(s, s.msgFactory.AllocateMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.Pong{
		MsgID:  secondMsg.MsgID,
		PingID: pingID,
	})

	select {
	case got := <-invokeDone:
		if got.err != nil {
			t.Fatalf("Invoke() error: %v", got.err)
		}
		if _, ok := got.obj.(*tg.Pong); !ok {
			t.Fatalf("Invoke() = %T, want *tg.Pong", got.obj)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Invoke() timed out")
	}

	close(s.cancel)
}

func TestSessionInvokeTimeout(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	startTestWorkers(s)

	_, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: 123}, 1, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestSessionStartStop(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.setPingInterval(1 * time.Hour)

	go func() {
		sentData, ok := <-mt.sendCh
		if !ok {
			return
		}
		msg := unpackIncoming(sentData, s)
		if msg == nil {
			return
		}
		ping, ok := msg.Body.(*tg.PingRequest)
		if !ok {
			return
		}
		respMsgID := s.msgFactory.AllocateMsgID()
		respSeqNo := s.msgFactory.AllocateSeqNo(false)
		pong := &tg.Pong{MsgID: msg.MsgID, PingID: ping.PingID}
		encrypted := makeEncryptedResponse(s, respMsgID, uint32(respSeqNo), pong)
		mt.recvCh <- encrypted
	}()

	err := s.Start(3 * time.Second)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if !s.IsConnected() {
		t.Error("IsConnected() = false after Start()")
	}

	s.Stop()
	if s.IsConnected() {
		t.Error("IsConnected() = true after Stop()")
	}
}

func TestSessionStartIgnoresInvalidIncomingFrame(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.setPingInterval(1 * time.Hour)

	go func() {
		sentData, ok := <-mt.sendCh
		if !ok {
			return
		}
		msg := unpackIncoming(sentData, s)
		if msg == nil {
			return
		}
		ping, ok := msg.Body.(*tg.PingRequest)
		if !ok {
			return
		}
		mt.recvCh <- []byte{}
		respMsgID := s.msgFactory.AllocateMsgID()
		respSeqNo := s.msgFactory.AllocateSeqNo(false)
		pong := &tg.Pong{MsgID: msg.MsgID, PingID: ping.PingID}
		mt.recvCh <- makeEncryptedResponse(s, respMsgID, uint32(respSeqNo), pong)
	}()

	if err := s.Start(3 * time.Second); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	s.Stop()
}

func TestSessionConnectNoAuthKey(t *testing.T) {
	dc := DataCenter{ID: 2}
	st := newTestStorage()
	s, err := NewSession(dc, st, "TestDevice", "1.0", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() error: %v", err)
	}
	mt := newMockTransport()

	err = s.Connect(mt, 1*time.Second)
	if err == nil {
		t.Fatal("expected error when connecting without auth key")
	}
}

func unpackIncoming(data []byte, s *Session) *tg.MTProtoMessage {
	msg, _, _ := crypto.Unpack(data, s.sessionIDBytes(), s.authKey, s.authKeyID)
	return msg
}
