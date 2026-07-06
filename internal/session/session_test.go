package session

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/tg"
)

const testUpdateObjectTypeID = 0x1badb002

type testUpdateObject struct {
	Value int32
}

func (v *testUpdateObject) ConstructorID() uint32 { return testUpdateObjectTypeID }

func (v *testUpdateObject) Encode(b *bytes.Buffer) error {
	tg.WriteInt(b, testUpdateObjectTypeID)
	tg.WriteInt(b, uint32(v.Value))
	return nil
}

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
	tg.Registry[tg.PingTypeID] = func(r *tg.Reader) (tg.TLObject, error) {
		v := &tg.PingRequest{}
		pingID, err := r.ReadInt64()
		if err != nil {
			return nil, err
		}
		v.PingID = pingID
		return v, nil
	}
	tg.Registry[testUpdateObjectTypeID] = func(r *tg.Reader) (tg.TLObject, error) {
		value, err := r.ReadInt32()
		if err != nil {
			return nil, err
		}
		return &testUpdateObject{Value: value}, nil
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
	handle, err := s.pending.Register(msgID, false)
	if err != nil {
		t.Fatalf("Register() error: %v", err)
	}

	s.pending.Resolve(msgID, tg.TLBool(true))

	<-handle.Done()
	obj, _, err := handle.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, ok := obj.(tg.TLBool)
	if !ok || !bool(b) {
		t.Errorf("received data = %T %[1]v, want tg.TLBool(true)", obj)
	}

	if s.pending.Has(msgID) {
		t.Error("result still exists after resolve")
	}
}

func TestSessionAckTracking(t *testing.T) {
	dc := DataCenter{ID: 1}
	st := newTestStorage()
	s, err := NewSession(dc, st, "Dev", "1.0", "en", "en")
	if err != nil {
		t.Fatalf("NewSession() error: %v", err)
	}

	// Initialize ackCh so addAck can send to it.
	s.ackCh = make(chan int64, 1024)

	s.addAck(10)
	s.addAck(20)
	s.addAck(30)

	// Read acks back from the channel.
	var acks []int64
	for i := 0; i < 3; i++ {
		select {
		case id := <-s.ackCh:
			acks = append(acks, id)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for ack %d", i)
		}
	}
	if len(acks) != 3 {
		t.Fatalf("got %d acks, want 3", len(acks))
	}
	if acks[0] != 10 || acks[1] != 20 || acks[2] != 30 {
		t.Errorf("acks = %v, want [10 20 30]", acks)
	}

	// Channel should be empty now.
	select {
	case <-s.ackCh:
		t.Error("unexpected extra ack in channel")
	default:
	}
}

func TestSessionRawMsgsAckCleansTrackedContainer(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.TrackContainer(1001, []int64{2001, 2002})

	var body bytes.Buffer
	tg.WriteVectorLong(&body, []int64{1001})
	s.handleRawMsgsAck(body.Bytes())

	if got := s.containerTracker.NackContainer(1001); len(got) != 0 {
		t.Fatalf("NackContainer() after raw ACK = %v, want empty", got)
	}
}

func TestSessionRawMsgResendReqNacksTrackedContainer(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.ackCh = make(chan int64, 1024)
	s.TrackContainer(1001, []int64{2001, 2002})

	var body bytes.Buffer
	tg.WriteVectorLong(&body, []int64{1001})
	s.handleRawMsgResendReq(body.Bytes())

	if got := s.containerTracker.AckContainer(1001); len(got) != 0 {
		t.Fatalf("AckContainer() after raw resend req = %v, want empty", got)
	}
	select {
	case got := <-s.ackCh:
		if got != 1001 {
			t.Fatalf("ackCh got %d, want 1001", got)
		}
	default:
		t.Fatal("ackCh missing resend request ack")
	}
}

type mockTransport struct {
	sendCh    chan []byte
	recvCh    chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		sendCh: make(chan []byte, 100),
		recvCh: make(chan []byte, 100),
		done:   make(chan struct{}),
	}
}

func (m *mockTransport) Send(data []byte) error {
	cp := make([]byte, len(data))
	copy(cp, data)
	m.sendCh <- cp
	return nil
}

func (m *mockTransport) Recv() ([]byte, error) {
	select {
	case data, ok := <-m.recvCh:
		if !ok {
			return nil, fmt.Errorf("transport closed")
		}
		return data, nil
	case <-m.done:
		return nil, fmt.Errorf("transport closed")
	}
}

func (m *mockTransport) Close() error {
	m.closeOnce.Do(func() { close(m.done) })
	return nil
}

func (m *mockTransport) IsConnected() bool {
	select {
	case <-m.done:
		return false
	default:
		return true
	}
}

func (m *mockTransport) SetWriteDeadline(t time.Time) error {
	return nil
}

func (m *mockTransport) SetReadDeadline(t time.Time) error {
	return nil
}

func makeAuthKey() []byte {
	return make([]byte, 256)
}

var serverMsgIDCounter int64

func makeServerMsgID() int64 {
	return (time.Now().Unix()<<32 | 1) + atomic.AddInt64(&serverMsgIDCounter, 1)<<2
}

func makeEncryptedResponse(s *Session, msgID int64, seqNo uint32, body tg.TLObject) []byte {
	message := &tg.MTProtoMessage{
		MsgID: msgID,
		SeqNo: seqNo,
		Body:  body,
	}
	encrypted, err := crypto.Pack(message, s.saltMgr.Load(), s.sessionIDBytes(), s.authKey, s.authKeyID)
	if err != nil {
		panic("makeEncryptedResponse: " + err.Error())
	}
	return encrypted
}

func makeEncryptedRawResponse(s *Session, msgID int64, seqNo uint32, body []byte) []byte {
	encrypted, err := crypto.PackRaw(msgID, seqNo, body, s.saltMgr.Load(), s.sessionIDBytes(), s.authKey, s.authKeyID)
	if err != nil {
		panic("makeEncryptedRawResponse: " + err.Error())
	}
	return encrypted
}

func encodeTLObject(t *testing.T, obj tg.TLObject) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := tg.EncodeTLObject(&buf, obj); err != nil {
		t.Fatalf("encode %T: %v", obj, err)
	}
	return buf.Bytes()
}

func writeRawMTProtoMessage(b *bytes.Buffer, msgID int64, seqNo uint32, body []byte) {
	tg.WriteLong(b, msgID)
	tg.WriteInt(b, seqNo)
	tg.WriteInt(b, uint32(len(body)))
	b.Write(body)
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

// startTestWorkers initializes internal state and starts a readLoop + ackLoop
// for testing Send/SendRaw without going through the full Start/Run lifecycle.
// Returns a cleanup function that must be called to stop the goroutines.
func startTestWorkers(s *Session, mt *mockTransport) func() {
	ctx, cancel := context.WithCancel(context.Background())
	s.ackCh = make(chan int64, 1024)
	s.pingCbs = make(map[int64]chan struct{})
	s.done = make(chan struct{})
	s.sm.forceSetState(StateActive)
	go func() { _ = s.readLoop(ctx) }()
	go func() { _ = s.ackLoop(ctx) }()
	return func() {
		cancel()
		close(s.done)
		mt.Close() // safe via closeOnce; unblocks readLoop's Recv()
	}
}

func TestSessionSetDispatchConfigNoOp(t *testing.T) {
	s := newSessionWithAuthKey(t)
	// Should not panic or modify any state.
	s.SetDispatchConfig(7, 33)
	s.SetDispatchConfig(0, 0)
	s.SetDispatchConfig(-1, -1)
}

func TestSessionSendAndWait(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	pingID := time.Now().UnixNano()

	sendDone := make(chan error, 1)
	go func() {
		_, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: pingID}, 5*time.Second)
		sendDone <- err
	}()

	<-mt.sendCh

	respMsgID := makeServerMsgID()
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
}

func TestSessionSendRawAndWait(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	pingID := time.Now().UnixNano()
	ping := &tg.PingRequest{PingID: pingID}

	var buf bytes.Buffer
	if err := ping.Encode(&buf); err != nil {
		t.Fatalf("encode ping: %v", err)
	}

	sendDone := make(chan struct {
		data []byte
		err  error
	}, 1)
	go func() {
		data, err := s.SendRaw(context.Background(), msgID, uint32(seqNo), buf.Bytes(), 5*time.Second)
		sendDone <- struct {
			data []byte
			err  error
		}{data, err}
	}()

	<-mt.sendCh

	respMsgID := makeServerMsgID()
	respSeqNo := s.msgFactory.AllocateSeqNo(false)
	pong := &tg.Pong{MsgID: msgID, PingID: pingID}
	encrypted := makeEncryptedResponse(s, respMsgID, uint32(respSeqNo), &tg.RPCResult{
		ReqMsgID: msgID,
		Result:   pong,
	})
	mt.recvCh <- encrypted

	select {
	case result := <-sendDone:
		if result.err != nil {
			t.Fatalf("SendRaw() error: %v", result.err)
		}
		want := encodeTLObject(t, pong)
		if !bytes.Equal(result.data, want) {
			t.Fatalf("SendRaw() returned %x, want raw rpc_result payload %x", result.data, want)
		}
		obj, err := tg.ReadTLObject(tg.NewReader(result.data))
		if err != nil {
			t.Fatalf("decode raw response: %v", err)
		}
		p, ok := obj.(*tg.Pong)
		if !ok {
			t.Fatalf("expected *tg.Pong, got %T", obj)
		}
		if p.PingID != pingID {
			t.Errorf("pong.PingID = %d, want %d", p.PingID, pingID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SendRaw() timed out")
	}
}

func TestSessionSendRawReturnsGzipPackedPayloadWithoutDecode(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)

	sendDone := make(chan struct {
		data []byte
		err  error
	}, 1)
	go func() {
		data, err := s.SendRaw(context.Background(), msgID, uint32(seqNo), encodeTLObject(t, &tg.PingRequest{PingID: 1}), 5*time.Second)
		sendDone <- struct {
			data []byte
			err  error
		}{data, err}
	}()

	<-mt.sendCh

	var gzipPayload bytes.Buffer
	gz := &tg.GzipPacked{Data: &tg.Pong{MsgID: msgID, PingID: 1}}
	tg.WriteInt(&gzipPayload, tg.GzipPackedID)
	if err := gz.Encode(&gzipPayload); err != nil {
		t.Fatalf("encode gzip payload: %v", err)
	}
	var rpcResult bytes.Buffer
	tg.WriteInt(&rpcResult, tg.RPCResultTypeID)
	tg.WriteLong(&rpcResult, msgID)
	rpcResult.Write(gzipPayload.Bytes())
	mt.recvCh <- makeEncryptedRawResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), rpcResult.Bytes())

	select {
	case result := <-sendDone:
		if result.err != nil {
			t.Fatalf("SendRaw() error: %v", result.err)
		}
		if len(result.data) < 4 {
			t.Fatalf("SendRaw() returned short payload: %x", result.data)
		}
		if got := tg.ReadInt(bytes.NewReader(result.data[:4])); got != tg.GzipPackedID {
			t.Fatalf("raw payload constructor = %08x, want gzip_packed %08x", got, tg.GzipPackedID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SendRaw() timed out")
	}
}

func TestSessionSendRawReturnsUnknownPayloadWithoutTLDecode(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)

	sendDone := make(chan struct {
		data []byte
		err  error
	}, 1)
	go func() {
		data, err := s.SendRaw(context.Background(), msgID, uint32(seqNo), encodeTLObject(t, &tg.PingRequest{PingID: 1}), 5*time.Second)
		sendDone <- struct {
			data []byte
			err  error
		}{data, err}
	}()

	<-mt.sendCh

	payload := []byte{0xff, 0xff, 0xff, 0xff, 0x01, 0x02, 0x03, 0x04}
	var rpcResult bytes.Buffer
	tg.WriteInt(&rpcResult, tg.RPCResultTypeID)
	tg.WriteLong(&rpcResult, msgID)
	rpcResult.Write(payload)
	mt.recvCh <- makeEncryptedRawResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), rpcResult.Bytes())

	select {
	case result := <-sendDone:
		if result.err != nil {
			t.Fatalf("SendRaw() error: %v", result.err)
		}
		if !bytes.Equal(result.data, payload) {
			t.Fatalf("SendRaw() returned %x, want unknown raw payload %x", result.data, payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SendRaw() timed out")
	}
}

func TestSessionSendRawRoutesTopLevelGzipPackedRPCResult(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	pong := &tg.Pong{MsgID: msgID, PingID: 1}

	sendDone := make(chan struct {
		data []byte
		err  error
	}, 1)
	go func() {
		data, err := s.SendRaw(context.Background(), msgID, uint32(seqNo), encodeTLObject(t, &tg.PingRequest{PingID: 1}), 5*time.Second)
		sendDone <- struct {
			data []byte
			err  error
		}{data, err}
	}()

	<-mt.sendCh

	var gzipBody bytes.Buffer
	tg.WriteInt(&gzipBody, tg.GzipPackedID)
	if err := (&tg.GzipPacked{Data: &tg.RPCResult{
		ReqMsgID: msgID,
		Result:   pong,
	}}).Encode(&gzipBody); err != nil {
		t.Fatalf("encode gzip body: %v", err)
	}
	mt.recvCh <- makeEncryptedRawResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), gzipBody.Bytes())

	select {
	case result := <-sendDone:
		if result.err != nil {
			t.Fatalf("SendRaw() error: %v", result.err)
		}
		want := encodeTLObject(t, pong)
		if !bytes.Equal(result.data, want) {
			t.Fatalf("SendRaw() returned %x, want raw rpc_result payload %x", result.data, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SendRaw() timed out")
	}
}

func TestSessionSendRawContainerSkipsRawPayloadDecodeAndDispatchesUpdate(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	updateCh := make(chan tg.TLObject, 1)
	s.SetUpdateHandler(func(obj tg.TLObject) {
		updateCh <- obj
	})

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)

	sendDone := make(chan struct {
		data []byte
		err  error
	}, 1)
	go func() {
		data, err := s.SendRaw(context.Background(), msgID, uint32(seqNo), encodeTLObject(t, &tg.PingRequest{PingID: 1}), 5*time.Second)
		sendDone <- struct {
			data []byte
			err  error
		}{data, err}
	}()

	<-mt.sendCh

	payload := []byte{0xff, 0xff, 0xff, 0xff, 0x01, 0x02, 0x03, 0x04}
	var rpcResult bytes.Buffer
	tg.WriteInt(&rpcResult, tg.RPCResultTypeID)
	tg.WriteLong(&rpcResult, msgID)
	rpcResult.Write(payload)

	update := encodeTLObject(t, &testUpdateObject{Value: 7})
	var container bytes.Buffer
	tg.WriteInt(&container, tg.MsgContainerID)
	tg.WriteInt(&container, 2)
	writeRawMTProtoMessage(&container, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), rpcResult.Bytes())
	writeRawMTProtoMessage(&container, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), update)

	mt.recvCh <- makeEncryptedRawResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), container.Bytes())

	select {
	case result := <-sendDone:
		if result.err != nil {
			t.Fatalf("SendRaw() error: %v", result.err)
		}
		if !bytes.Equal(result.data, payload) {
			t.Fatalf("SendRaw() returned %x, want unknown raw payload %x", result.data, payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SendRaw() timed out")
	}

	select {
	case obj := <-updateCh:
		got, ok := obj.(*testUpdateObject)
		if !ok {
			t.Fatalf("update = %T, want *testUpdateObject", obj)
		}
		if got.Value != 7 {
			t.Fatalf("update.Value = %d, want 7", got.Value)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("update dispatch timed out")
	}
}

func TestSessionSendDeliversRpcResultByRequestMsgID(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	pingID := time.Now().UnixNano()

	sendDone := make(chan error, 1)
	go func() {
		_, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: pingID}, 5*time.Second)
		sendDone <- err
	}()

	<-mt.sendCh

	respMsgID := makeServerMsgID()
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
}

func TestSessionSendDecodesRPCResultPayloadFastPath(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	pingID := time.Now().UnixNano()

	sendDone := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: pingID}, 5*time.Second)
		sendDone <- struct {
			obj tg.TLObject
			err error
		}{obj, err}
	}()

	<-mt.sendCh

	pong := &tg.Pong{MsgID: msgID, PingID: pingID}
	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.RPCResult{
		ReqMsgID: msgID,
		Result:   pong,
	})

	select {
	case result := <-sendDone:
		if result.err != nil {
			t.Fatalf("Send() error: %v", result.err)
		}
		got, ok := result.obj.(*tg.Pong)
		if !ok {
			t.Fatalf("Send() = %T, want *tg.Pong", result.obj)
		}
		if got.PingID != pingID {
			t.Fatalf("Pong.PingID = %d, want %d", got.PingID, pingID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Send() timed out")
	}
}

func TestSessionInvokeRetriesBadServerSalt(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	pingID := time.Now().UnixNano()
	invokeDone := make(chan struct {
		obj tg.TLObject
		err error
	}, 1)
	go func() {
		obj, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: pingID}, 1, 5*time.Second)
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
	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.BadServerSalt{
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
	if s.saltMgr.Load() != newSalt {
		t.Fatalf("serverSalt = %x, want %x", s.saltMgr.Load(), newSalt)
	}

	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), &tg.Pong{
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
}

func TestSessionInvokeTimeout(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	_, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: 123}, 1, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestSessionInvokeZeroRetriesDoesNotSend(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.done = make(chan struct{})
	s.sm.forceSetState(StateActive)

	_, err := s.Invoke(context.Background(), &tg.PingRequest{PingID: 123}, 0, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected retries exhausted error, got nil")
	}
	select {
	case <-mt.sendCh:
		t.Fatal("Invoke() sent a request with retries=0")
	default:
	}
	close(s.done)
}

func TestSessionInvokeRawZeroRetriesDoesNotSend(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.done = make(chan struct{})
	s.sm.forceSetState(StateActive)

	_, err := s.InvokeRaw(context.Background(), &tg.PingRequest{PingID: 123}, 0, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected retries exhausted error, got nil")
	}
	select {
	case <-mt.sendCh:
		t.Fatal("InvokeRaw() sent a request with retries=0")
	default:
	}
	close(s.done)
}

func TestSessionInvokeRawRetriesShortRPCResult(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	resultCh := make(chan struct {
		data []byte
		err  error
	}, 1)
	go func() {
		data, err := s.InvokeRaw(context.Background(), &tg.PingRequest{PingID: 123}, 2, 5*time.Second)
		resultCh <- struct {
			data []byte
			err  error
		}{data, err}
	}()

	first := <-mt.sendCh
	firstMsg, err := unpackTestOutgoing(s, first)
	if err != nil {
		t.Fatalf("unpack first request: %v", err)
	}
	var emptyRPCResult bytes.Buffer
	if err := binary.Write(&emptyRPCResult, binary.LittleEndian, uint32(tg.RPCResultTypeID)); err != nil {
		t.Fatalf("write rpc_result constructor: %v", err)
	}
	if err := binary.Write(&emptyRPCResult, binary.LittleEndian, uint64(firstMsg.MsgID)); err != nil {
		t.Fatalf("write req_msg_id: %v", err)
	}
	mt.recvCh <- makeEncryptedRawResponse(s, makeServerMsgID(), 0, emptyRPCResult.Bytes())

	second := <-mt.sendCh
	secondMsg, err := unpackTestOutgoing(s, second)
	if err != nil {
		t.Fatalf("unpack second request: %v", err)
	}
	pong := &tg.Pong{MsgID: secondMsg.MsgID, PingID: 123}
	mt.recvCh <- makeEncryptedResponse(s, makeServerMsgID(), 0, &tg.RPCResult{
		ReqMsgID: secondMsg.MsgID,
		Result:   pong,
	})

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("InvokeRaw() error: %v", result.err)
		}
		want := encodeTLObject(t, pong)
		if !bytes.Equal(result.data, want) {
			t.Fatalf("InvokeRaw() = %x, want %x", result.data, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("InvokeRaw() timed out")
	}
}

func TestSessionStartStop(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.pingInterval = 1 * time.Hour

	// Response goroutine must be started BEFORE Start() because the initial
	// ping is now sent synchronously during runInit.
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
		respMsgID := makeServerMsgID()
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
	// Give a brief moment for connected to flip.
	time.Sleep(50 * time.Millisecond)
	if s.IsConnected() {
		t.Error("IsConnected() = true after Stop()")
	}
}

func TestSessionAckLoopFlushesAcks(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.done = make(chan struct{})
	s.ackCh = make(chan int64, 1024)

	// Use a cancellable context so we can trigger the final flush.
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = s.ackLoop(ctx) }()

	s.addAck(1)
	s.addAck(2)

	// Give ackLoop time to consume acks from the channel.
	time.Sleep(50 * time.Millisecond)

	// Cancel the context to trigger ackLoop's best-effort final flush.
	cancel()

	select {
	case data := <-mt.sendCh:
		msg := unpackIncoming(data, s)
		if msg == nil {
			t.Fatal("service message did not decode")
		}
		ack, ok := msg.Body.(*tg.MsgsAck)
		if !ok {
			t.Fatalf("service message = %T, want *tg.MsgsAck", msg.Body)
		}
		if len(ack.MsgIds) != 2 || ack.MsgIds[0] != 1 || ack.MsgIds[1] != 2 {
			t.Fatalf("ack ids = %v, want [1 2]", ack.MsgIds)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ackLoop did not flush on context cancellation")
	}
	close(s.done)
}

func TestSessionStartIgnoresInvalidIncomingFrame(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.pingInterval = 1 * time.Hour

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
		respMsgID := makeServerMsgID()
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

func unpackTestOutgoing(s *Session, data []byte) (*tg.MTProtoMessage, error) {
	msg, decrypted, err := crypto.Unpack(data, s.sessionIDBytes(), s.authKey, s.authKeyID)
	if decrypted != nil {
		crypto.ReleaseAESBuf(decrypted)
	}
	return msg, err
}

func TestQuickAck(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	quickAck := make([]byte, 4)
	binary.LittleEndian.PutUint32(quickAck, uint32(0x80000001))
	mt.recvCh <- quickAck

	msgID := s.msgFactory.AllocateMsgID()
	respMsgID := makeServerMsgID()
	respSeqNo := s.msgFactory.AllocateSeqNo(false)
	pong := &tg.Pong{MsgID: msgID, PingID: 42}

	sendDone := make(chan error, 1)
	go func() {
		_, err := s.Send(context.Background(), msgID, uint32(s.msgFactory.AllocateSeqNo(true)), &tg.PingRequest{PingID: 42}, 5*time.Second)
		sendDone <- err
	}()

	<-mt.sendCh
	time.Sleep(50 * time.Millisecond)

	mt.recvCh <- makeEncryptedResponse(s, respMsgID, uint32(respSeqNo), pong)

	select {
	case err := <-sendDone:
		if err != nil {
			t.Fatalf("Send() error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Send() timed out")
	}
}

func TestTransportErrorCodeKillsReadLoop(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.ackCh = make(chan int64, 1024)
	s.pingCbs = make(map[int64]chan struct{})
	s.done = make(chan struct{})
	s.sm.forceSetState(StateActive)
	go func() { _ = s.ackLoop(ctx) }()

	errCh := make(chan error, 1)
	go func() { errCh <- s.readLoop(ctx) }()
	defer func() {
		cancel()
		close(s.done)
		mt.Close()
	}()

	// Transport errors are 4-byte signed negative int32 values.
	// -404 = 0xFFFFFE6C = auth key not found.
	var errCodeVal int32 = -404
	errCode := make([]byte, 4)
	binary.LittleEndian.PutUint32(errCode, uint32(errCodeVal))
	mt.recvCh <- errCode

	select {
	case err := <-errCh:
		var te *transport.TransportError
		if !errors.As(err, &te) {
			t.Fatalf("expected *TransportError, got %T: %v", err, err)
		}
		if te.Code != -404 {
			t.Errorf("Code = %d, want -404", te.Code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("readLoop did not exit after transport error")
	}
}

type failingTransport struct {
	mu        sync.Mutex
	failErr   error
	sendCh    chan []byte
	recvCh    chan []byte
	closed    bool
	failCount int
}

func newFailingTransport() *failingTransport {
	return &failingTransport{
		sendCh: make(chan []byte, 100),
		recvCh: make(chan []byte, 100),
	}
}

func (f *failingTransport) Send(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failErr != nil {
		f.failCount++
		return f.failErr
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	f.sendCh <- cp
	return nil
}

func (f *failingTransport) Recv() ([]byte, error) {
	data, ok := <-f.recvCh
	if !ok {
		return nil, fmt.Errorf("transport closed")
	}
	return data, nil
}

func (f *failingTransport) Close() error {
	f.closed = true
	close(f.recvCh)
	return nil
}

func (f *failingTransport) IsConnected() bool                { return !f.closed }
func (f *failingTransport) SetWriteDeadline(time.Time) error { return nil }
func (f *failingTransport) SetReadDeadline(time.Time) error  { return nil }

func (f *failingTransport) SetFail(err error) {
	f.mu.Lock()
	f.failErr = err
	f.mu.Unlock()
}

func (f *failingTransport) FailCount() int {
	f.mu.Lock()
	n := f.failCount
	f.mu.Unlock()
	return n
}

func TestWriteCircuitBreakerTrips(t *testing.T) {
	s := newSessionWithAuthKey(t)
	ft := newFailingTransport()
	s.SetTransport(ft)
	s.done = make(chan struct{})
	s.sm.forceSetState(StateActive)
	s.writeBreakerThreshold = 3

	writeErr := fmt.Errorf("write failed")
	ft.SetFail(writeErr)

	for i := 0; i < 3; i++ {
		err := s.writeEncryptedDirect(make([]byte, 10), time.Second)
		if err == nil {
			t.Fatalf("expected error on write %d", i+1)
		}
	}

	if !s.writeBreakerOpen.Load() {
		t.Fatal("expected writeBreakerOpen to be true after 3 failures")
	}

	err := s.writeEncryptedDirect(make([]byte, 10), time.Second)
	if !errors.Is(err, ErrWriteCircuitOpen) {
		t.Fatalf("expected ErrWriteCircuitOpen, got %v", err)
	}

	close(s.done)
}

func TestWriteCircuitBreakerResetsOnSuccess(t *testing.T) {
	s := newSessionWithAuthKey(t)
	ft := newFailingTransport()
	s.SetTransport(ft)
	s.done = make(chan struct{})
	s.sm.forceSetState(StateActive)
	s.writeBreakerThreshold = 3

	ft.SetFail(fmt.Errorf("write failed"))

	for i := 0; i < 2; i++ {
		_ = s.writeEncryptedDirect(make([]byte, 10), time.Second)
	}

	if s.writeBreakerOpen.Load() {
		t.Fatal("breaker should not be open after 2 failures (threshold=3)")
	}

	ft.SetFail(nil)

	err := s.writeEncryptedDirect(make([]byte, 10), time.Second)
	if err != nil {
		t.Fatalf("expected success after clearing fail, got %v", err)
	}

	if s.consecWriteFailures.Load() != 0 {
		t.Fatalf("consecWriteFailures=%d, want 0 after success", s.consecWriteFailures.Load())
	}

	ft.SetFail(fmt.Errorf("write failed"))
	for i := 0; i < 3; i++ {
		_ = s.writeEncryptedDirect(make([]byte, 10), time.Second)
	}
	if !s.writeBreakerOpen.Load() {
		t.Fatal("breaker should be open after 3 more failures")
	}

	close(s.done)
}

func TestWriteCircuitBreakerDisabledWhenZero(t *testing.T) {
	s := newSessionWithAuthKey(t)
	ft := newFailingTransport()
	s.SetTransport(ft)
	s.done = make(chan struct{})
	s.sm.forceSetState(StateActive)
	s.writeBreakerThreshold = 0

	ft.SetFail(fmt.Errorf("write failed"))

	for i := 0; i < 10; i++ {
		_ = s.writeEncryptedDirect(make([]byte, 10), time.Second)
	}

	if s.writeBreakerOpen.Load() {
		t.Fatal("breaker should not open when threshold=0")
	}

	close(s.done)
}

type panicTransport struct {
	closed bool
}

func (p *panicTransport) Send(data []byte) error { return nil }
func (p *panicTransport) Recv() ([]byte, error) {
	panic("simulated transport panic")
}

func (p *panicTransport) Close() error {
	p.closed = true
	return nil
}
func (p *panicTransport) IsConnected() bool                { return !p.closed }
func (p *panicTransport) SetWriteDeadline(time.Time) error { return nil }
func (p *panicTransport) SetReadDeadline(time.Time) error  { return nil }

func TestReadLoopPanicRecoveryRejectsPending(t *testing.T) {
	s := newSessionWithAuthKey(t)
	pt := &panicTransport{}
	s.SetTransport(pt)
	s.pending.SetMaxPending(10)

	h1, _ := s.pending.Register(1, false)
	h2, _ := s.pending.Register(2, true)

	ctx, cancel := context.WithCancel(context.Background())
	s.ackCh = make(chan int64, 1024)
	s.pingCbs = make(map[int64]chan struct{})
	s.done = make(chan struct{})
	s.sm.forceSetState(StateActive)

	doneCh := make(chan error, 1)
	go func() {
		doneCh <- s.readLoop(ctx)
	}()

	select {
	case err := <-doneCh:
		if err == nil {
			t.Fatal("expected error from readLoop panic")
		}
		cancel()
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("readLoop did not exit after panic")
	}

	select {
	case <-h1.Done():
		_, _, e := h1.Result()
		if !errors.Is(e, ErrSessionClosed) {
			t.Fatalf("h1 err=%v, want ErrSessionClosed", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("h1 not unblocked after panic recovery")
	}

	select {
	case <-h2.Done():
		_, _, e := h2.Result()
		if !errors.Is(e, ErrSessionClosed) {
			t.Fatalf("h2 err=%v, want ErrSessionClosed", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("h2 not unblocked after panic recovery")
	}

	close(s.done)
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		err  error
		want ErrorClass
	}{
		{ErrSessionClosed, ClassClosed},
		{ErrDraining, ClassClosed},
		{ErrWriteCircuitOpen, ClassPermanent},
		{fmt.Errorf("transport error: code %d", -404), ClassUnknown},
	}
	for _, tt := range tests {
		got := ClassifyError(tt.err)
		if got != tt.want {
			t.Errorf("ClassifyError(%v)=%s, want %s", tt.err, got, tt.want)
		}
	}
}

func TestClassifyErrorNil(t *testing.T) {
	if ClassifyError(nil) != ClassUnknown {
		t.Fatal("ClassifyError(nil) should be ClassUnknown")
	}
}

func TestErrorClassString(t *testing.T) {
	tests := []struct {
		c    ErrorClass
		want string
	}{
		{ClassTransient, "transient"},
		{ClassPermanent, "permanent"},
		{ClassClosed, "closed"},
		{ClassRateLimited, "rate_limited"},
		{ClassMigrate, "migrate"},
		{ClassUnknown, "unknown"},
	}
	for _, tt := range tests {
		if tt.c.String() != tt.want {
			t.Errorf("ErrorClass(%d).String()=%q, want %q", tt.c, tt.c.String(), tt.want)
		}
	}
}

func TestErrBusyBeforeTransportWrite(t *testing.T) {
	s := newSessionWithAuthKey(t)
	ft := newFailingTransport()
	s.SetTransport(ft)
	s.done = make(chan struct{})
	s.ackCh = make(chan int64, 1024)
	s.pingCbs = make(map[int64]chan struct{})
	s.sm.forceSetState(StateActive)
	s.pending.SetMaxPending(2)

	for i := int64(0); i < 2; i++ {
		_, _ = s.pending.Register(i, false)
	}

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)

	_, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: 1}, time.Second)
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("Send() error=%v, want ErrBusy", err)
	}

	if ft.FailCount() > 0 {
		t.Fatal("transport Send should not have been called when ErrBusy is returned")
	}

	close(s.done)
}

func TestSlowUpdateHandlerDoesNotBlockRPC(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	updateCh := make(chan tg.TLObject, 1)
	s.SetUpdateHandler(func(obj tg.TLObject) {
		time.Sleep(500 * time.Millisecond)
		updateCh <- obj
	})

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	pingID := time.Now().UnixNano()

	sendDone := make(chan error, 1)
	go func() {
		_, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: pingID}, 5*time.Second)
		sendDone <- err
	}()

	<-mt.sendCh

	var container bytes.Buffer
	pong := &tg.Pong{MsgID: msgID, PingID: pingID}
	rpcResult := encodeTLObject(t, pong)
	var rpcBuf bytes.Buffer
	tg.WriteInt(&rpcBuf, tg.RPCResultTypeID)
	tg.WriteLong(&rpcBuf, msgID)
	rpcBuf.Write(rpcResult)

	update := encodeTLObject(t, &testUpdateObject{Value: 42})

	tg.WriteInt(&container, tg.MsgContainerID)
	tg.WriteInt(&container, 2)
	writeRawMTProtoMessage(&container, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), rpcBuf.Bytes())
	writeRawMTProtoMessage(&container, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), update)

	mt.recvCh <- makeEncryptedRawResponse(s, makeServerMsgID(), uint32(s.msgFactory.AllocateSeqNo(false)), container.Bytes())

	select {
	case err := <-sendDone:
		if err != nil {
			t.Fatalf("Send() error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Send() timed out — slow update handler blocked RPC delivery")
	}

	select {
	case <-updateCh:
	case <-time.After(2 * time.Second):
		t.Fatal("update was not delivered")
	}
}

func TestReceiveLoopNoBlockOnCallerTimeout(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := s.Send(ctx, msgID, uint32(seqNo), &tg.PingRequest{PingID: 1}, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	if s.pending.Count() != 0 {
		t.Fatalf("pending Count()=%d, want 0 after timeout", s.pending.Count())
	}
}

func TestWriteFailureCancelsPending(t *testing.T) {
	s := newSessionWithAuthKey(t)
	ft := newFailingTransport()
	s.SetTransport(ft)
	s.done = make(chan struct{})
	s.ackCh = make(chan int64, 1024)
	s.pingCbs = make(map[int64]chan struct{})
	s.sm.forceSetState(StateActive)

	ft.SetFail(fmt.Errorf("write failed"))

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)

	_, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: 1}, time.Second)
	if err == nil {
		t.Fatal("expected error from Send with failing transport")
	}

	if s.pending.Has(msgID) {
		t.Fatal("pending handle should be removed after write failure")
	}

	close(s.done)
}

func TestRegisterBeforeWrite(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	msgID := s.msgFactory.AllocateMsgID()
	seqNo := s.msgFactory.AllocateSeqNo(true)
	pingID := time.Now().UnixNano()

	sendDone := make(chan error, 1)
	go func() {
		_, err := s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: pingID}, 5*time.Second)
		sendDone <- err
	}()

	<-mt.sendCh

	if !s.pending.Has(msgID) {
		t.Fatal("pending handle should exist before response arrives")
	}

	respMsgID := makeServerMsgID()
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
}

func TestSetWriteBreakerThreshold(t *testing.T) {
	s := newSessionWithAuthKey(t)
	s.SetWriteBreakerThreshold(5)
	if s.writeBreakerThreshold != 5 {
		t.Fatalf("threshold=%d, want 5", s.writeBreakerThreshold)
	}
}

func TestLockOrderingStress(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.pingInterval = 1 * time.Hour

	pingReady := make(chan struct{})
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
		respMsgID := makeServerMsgID()
		respSeqNo := s.msgFactory.AllocateSeqNo(false)
		pong := &tg.Pong{MsgID: msg.MsgID, PingID: ping.PingID}
		encrypted := makeEncryptedResponse(s, respMsgID, uint32(respSeqNo), pong)
		mt.recvCh <- encrypted
		close(pingReady)
	}()

	err := s.Start(3 * time.Second)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	<-pingReady

	var wg sync.WaitGroup
	const writers = 20
	const readers = 5
	wg.Add(writers + readers)

	for i := 0; i < writers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				msgID := s.msgFactory.AllocateMsgID()
				seqNo := s.msgFactory.AllocateSeqNo(true)
				_, _ = s.Send(context.Background(), msgID, uint32(seqNo), &tg.PingRequest{PingID: int64(id)*100 + int64(j)}, 50*time.Millisecond)
			}
		}(i)
	}

	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = s.IsConnected()
				_ = s.AuthKey()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	time.Sleep(20 * time.Millisecond)
	s.Stop()
	wg.Wait()
}

func TestGoleakSessionLifecycle(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)
	s.pingInterval = 1 * time.Hour

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
		respMsgID := makeServerMsgID()
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
		t.Fatal("not connected after Start()")
	}

	s.Stop()
	time.Sleep(100 * time.Millisecond)

	if s.sm.State() != StateClosed {
		t.Fatalf("state=%s, want Closed", s.sm.State())
	}

	if s.pending.Count() != 0 {
		t.Fatalf("pending.Count()=%d, want 0 after Stop", s.pending.Count())
	}
}

func TestRejectAllUnblocksWaiters(t *testing.T) {
	pm := NewPendingManager()

	var handles []*CallHandle
	for i := int64(0); i < 10; i++ {
		h, err := pm.Register(i, i%2 == 0)
		if err != nil {
			t.Fatalf("Register(%d) error: %v", i, err)
		}
		handles = append(handles, h)
	}

	var wg sync.WaitGroup
	for _, h := range handles {
		wg.Add(1)
		go func(h *CallHandle) {
			defer wg.Done()
			<-h.Done()
		}(h)
	}

	pm.RejectAll(ErrSessionClosed)
	wg.Wait()

	for i, h := range handles {
		_, _, err := h.Result()
		if !errors.Is(err, ErrSessionClosed) {
			t.Fatalf("handle[%d] err=%v, want ErrSessionClosed", i, err)
		}
	}

	pm.mu.Lock()
	leaked := len(pm.pending)
	pm.mu.Unlock()
	if leaked != 0 {
		t.Fatalf("pending map has %d entries after RejectAll", leaked)
	}
}

func TestCompletedCallsFreePendingCapacity(t *testing.T) {
	pm := NewPendingManager()
	pm.SetMaxPending(5)

	for cycle := 0; cycle < 5; cycle++ {
		var handles []*CallHandle
		for i := int64(0); i < 5; i++ {
			h, err := pm.Register(int64(cycle)*5+i, false)
			if err != nil {
				t.Fatalf("cycle %d: Register(%d) error: %v", cycle, i, err)
			}
			handles = append(handles, h)
		}

		_, err := pm.Register(999, false)
		if !errors.Is(err, ErrBusy) {
			t.Fatalf("cycle %d: expected ErrBusy, got %v", cycle, err)
		}

		for _, h := range handles {
			pm.Cancel(int64(cycle)*5 + 0)
			_ = h
		}
		_ = handles

		for i := int64(0); i < 5; i++ {
			if i == 0 {
				pm.Cancel(int64(cycle)*5 + i)
			} else {
				pm.Resolve(int64(cycle)*5+i, &tg.Pong{})
			}
		}
	}
}

func TestWriteBreakerBlocksWritesButKeepsSessionAlive(t *testing.T) {
	s := newSessionWithAuthKey(t)
	ft := newFailingTransport()
	s.SetTransport(ft)
	s.done = make(chan struct{})
	s.ackCh = make(chan int64, 1024)
	s.pingCbs = make(map[int64]chan struct{})
	s.sm.forceSetState(StateActive)
	s.writeBreakerThreshold = 2

	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	g := newErrGroup(parentCtx)
	g.Go(s.ackLoop)
	s.group = g

	ft.SetFail(fmt.Errorf("write failed"))

	// Two consecutive failures should open the breaker.
	err := s.writeEncryptedDirect(make([]byte, 10), time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
	err = s.writeEncryptedDirect(make([]byte, 10), time.Second)
	if err == nil {
		t.Fatal("expected error")
	}

	if !s.writeBreakerOpen.Load() {
		t.Fatal("breaker should be open")
	}

	// The group context should NOT be cancelled — the session stays alive.
	select {
	case <-g.ctx.Done():
		t.Fatal("group context should NOT be cancelled when breaker opens")
	case <-time.After(100 * time.Millisecond):
	}

	// Subsequent writes should get ErrWriteCircuitOpen.
	err = s.writeEncryptedDirect(make([]byte, 10), time.Second)
	if !errors.Is(err, ErrWriteCircuitOpen) {
		t.Fatalf("expected ErrWriteCircuitOpen, got %v", err)
	}

	close(s.done)
}

func TestResetWriteBreaker(t *testing.T) {
	s := newSessionWithAuthKey(t)
	ft := newFailingTransport()
	s.SetTransport(ft)
	s.done = make(chan struct{})
	s.sm.forceSetState(StateActive)
	s.writeBreakerThreshold = 2

	// Open the breaker.
	ft.SetFail(fmt.Errorf("write failed"))
	_ = s.writeEncryptedDirect(make([]byte, 10), time.Second)
	_ = s.writeEncryptedDirect(make([]byte, 10), time.Second)
	if !s.writeBreakerOpen.Load() {
		t.Fatal("breaker should be open")
	}

	// Reset should clear it.
	s.ResetWriteBreaker()
	if s.writeBreakerOpen.Load() {
		t.Fatal("breaker should be closed after reset")
	}
	if s.consecWriteFailures.Load() != 0 {
		t.Fatalf("consecWriteFailures=%d after reset", s.consecWriteFailures.Load())
	}

	// Writes should succeed again after reset.
	ft.SetFail(nil)
	err := s.writeEncryptedDirect(make([]byte, 10), time.Second)
	if err != nil {
		t.Fatalf("expected success after reset, got %v", err)
	}

	close(s.done)
}

func TestQuickAckSentOnUpdatesClass(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	updateCh := make(chan tg.TLObject, 1)
	s.SetUpdateHandler(func(obj tg.TLObject) {
		updateCh <- obj
	})

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	update := &tg.Updates{Updates: []tg.UpdateClass{}, Users: []tg.UserClass{}, Chats: []tg.ChatClass{}, Date: 12345, Seq: 1}
	serverMsgID := makeServerMsgID()
	encrypted := makeEncryptedResponse(s, serverMsgID, uint32(s.msgFactory.AllocateSeqNo(false)), update)

	go func() {
		time.Sleep(50 * time.Millisecond)
		mt.recvCh <- encrypted
	}()

	select {
	case data := <-mt.sendCh:
		msg, _, err := crypto.Unpack(data, s.sessionIDBytes(), s.authKey, s.authKeyID)
		if err != nil {
			t.Fatalf("unpack: %v", err)
		}
		if msg == nil || msg.Body == nil {
			t.Fatal("no message body")
		}
		ack, ok := msg.Body.(*tg.MsgsAck)
		if !ok {
			t.Fatalf("expected MsgsAck, got %T", msg.Body)
		}
		if len(ack.MsgIds) != 1 {
			t.Fatalf("expected 1 ack msg ID, got %d", len(ack.MsgIds))
		}
		if ack.MsgIds[0] != serverMsgID {
			t.Errorf("ack msg ID = %d, want %d", ack.MsgIds[0], serverMsgID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for quick ACK")
	}
}

func TestBatchAckSentOnNonUpdates(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	ping := &tg.Pong{MsgID: 1, PingID: 42}
	serverMsgID := makeServerMsgID()
	encrypted := makeEncryptedResponse(s, serverMsgID, uint32(s.msgFactory.AllocateSeqNo(false)), ping)

	go func() {
		time.Sleep(50 * time.Millisecond)
		mt.recvCh <- encrypted
	}()

	for i := 0; i < 10; i++ {
		select {
		case data := <-mt.sendCh:
			msg, _, err := crypto.Unpack(data, s.sessionIDBytes(), s.authKey, s.authKeyID)
			if err != nil {
				continue
			}
			if msg == nil || msg.Body == nil {
				continue
			}
			ack, ok := msg.Body.(*tg.MsgsAck)
			if !ok {
				continue
			}
			for _, id := range ack.MsgIds {
				if id == serverMsgID {
					t.Fatalf("Pong should NOT trigger quick ACK (expected batched ack via ackLoop), but got immediate ack for msgID %d", id)
				}
			}
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func TestPendingManagerMarkAllUnknown(t *testing.T) {
	pm := NewPendingManager()

	// Register 3 pending queries.
	h1, err := pm.Register(1001, false)
	if err != nil {
		t.Fatalf("Register(1001): %v", err)
	}
	_ = h1
	h2, err := pm.Register(1002, false)
	if err != nil {
		t.Fatalf("Register(1002): %v", err)
	}
	_ = h2
	h3, err := pm.Register(1003, true)
	if err != nil {
		t.Fatalf("Register(1003): %v", err)
	}
	_ = h3

	// Mark all as unknown.
	ids := pm.MarkAllUnknown()
	if len(ids) != 3 {
		t.Fatalf("MarkAllUnknown: got %d ids, want 3", len(ids))
	}

	// GetUnknown should return the same IDs.
	unknown := pm.GetUnknown()
	if len(unknown) != 3 {
		t.Fatalf("GetUnknown: got %d, want 3", len(unknown))
	}
}

func TestPendingManagerRejectExcessUnknowns(t *testing.T) {
	pm := NewPendingManager()

	// Register 5 pending queries.
	for i := int64(0); i < 5; i++ {
		_, err := pm.Register(2001+i, false)
		if err != nil {
			t.Fatalf("Register(%d): %v", 2001+i, err)
		}
	}

	// Mark all as unknown.
	pm.MarkAllUnknown()

	// Reject excess with cap of 3 — should reject 2.
	rejected := pm.RejectExcessUnknowns(3)
	if rejected != 2 {
		t.Fatalf("RejectExcessUnknowns(3): rejected %d, want 2", rejected)
	}

	// Should have 3 remaining.
	remaining := pm.GetUnknown()
	if len(remaining) != 3 {
		t.Fatalf("GetUnknown after reject: got %d, want 3", len(remaining))
	}
}

func TestPendingManagerRejectExcessUnknownsUnderCap(t *testing.T) {
	pm := NewPendingManager()

	// Register 2 queries, cap is 5 — no rejection needed.
	for i := int64(0); i < 2; i++ {
		_, _ = pm.Register(3001+i, false)
	}
	pm.MarkAllUnknown()

	rejected := pm.RejectExcessUnknowns(5)
	if rejected != 0 {
		t.Fatalf("RejectExcessUnknowns(5): rejected %d, want 0", rejected)
	}

	remaining := pm.GetUnknown()
	if len(remaining) != 2 {
		t.Fatalf("GetUnknown: got %d, want 2", len(remaining))
	}
}

func TestSessionPrepareForReconnect(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	// Register some pending queries.
	_, _ = s.pending.Register(4001, false)
	_, _ = s.pending.Register(4002, false)
	_, _ = s.pending.Register(4003, true)

	// PrepareForReconnect should return all pending IDs.
	ids := s.PrepareForReconnect()
	if len(ids) != 3 {
		t.Fatalf("PrepareForReconnect: got %d ids, want 3", len(ids))
	}

	// HasUnknownQueries should be true.
	if !s.HasUnknownQueries() {
		t.Fatal("HasUnknownQueries should be true")
	}
}

func TestSessionHasUnknownQueriesEmpty(t *testing.T) {
	s := newSessionWithAuthKey(t)
	mt := newMockTransport()
	s.SetTransport(mt)

	cleanup := startTestWorkers(s, mt)
	defer cleanup()

	// No pending queries.
	if s.HasUnknownQueries() {
		t.Fatal("HasUnknownQueries should be false when no pending queries")
	}
}

func TestDCOptionPoolFindBest(t *testing.T) {
	pool := NewDCOptionPool(2, 16*time.Second)

	// Add 2 endpoints.
	dc1 := DataCenter{ID: 2}
	dc2 := DataCenter{ID: 2, IPv6: true}
	pool.AddOption(dc1)
	pool.AddOption(dc2)

	// Both are untested — should return one of them.
	best, err := pool.FindBest()
	if err != nil {
		t.Fatalf("FindBest: %v", err)
	}
	if best != dc1 && best != dc2 {
		t.Fatalf("FindBest returned unexpected endpoint: %v", best)
	}

	// Record success on dc1.
	pool.RecordSuccess(dc1)

	// Now dc1 is Ok, dc2 is Untested — should prefer dc1.
	best, err = pool.FindBest()
	if err != nil {
		t.Fatalf("FindBest after success: %v", err)
	}
	if best != dc1 {
		t.Fatalf("FindBest should prefer dc1 (Ok), got %v", best)
	}

	// Record failure on dc1.
	pool.RecordFailure(dc1)

	// Now dc1 is Error, dc2 is Untested — should prefer dc2.
	best, err = pool.FindBest()
	if err != nil {
		t.Fatalf("FindBest after failure: %v", err)
	}
	if best != dc2 {
		t.Fatalf("FindBest should prefer dc2 (Untested), got %v", best)
	}
}

func TestDCOptionPoolCoolDown(t *testing.T) {
	pool := NewDCOptionPool(2, 100*time.Millisecond)

	dc := DataCenter{ID: 2}
	pool.AddOption(dc)

	// Record failure.
	pool.RecordFailure(dc)

	// Should fail — all endpoints in cool-down.
	_, err := pool.FindBest()
	if err == nil {
		t.Fatal("FindBest should fail when all endpoints in cool-down")
	}

	// Wait for cool-down to expire.
	time.Sleep(150 * time.Millisecond)

	// Should succeed now.
	best, err := pool.FindBest()
	if err != nil {
		t.Fatalf("FindBest after cool-down: %v", err)
	}
	if best != dc {
		t.Fatalf("FindBest returned unexpected endpoint: %v", best)
	}
}

func TestConnectionPoolGetPut(t *testing.T) {
	pool := NewConnectionPool(10 * time.Second)

	dc := DataCenter{ID: 2}
	conn := &mockTransport{}

	// Cache miss.
	_, ok := pool.Get(2, dc)
	if ok {
		t.Fatal("Get should return false on cache miss")
	}

	// Put.
	pool.Put(2, dc, conn)

	// Cache hit.
	got, ok := pool.Get(2, dc)
	if !ok {
		t.Fatal("Get should return true on cache hit")
	}
	if got != conn {
		t.Fatal("Get returned wrong connection")
	}

	// Second get should miss (consumed).
	_, ok = pool.Get(2, dc)
	if ok {
		t.Fatal("Get should return false after consumption")
	}
}

func TestConnectionPoolExpiry(t *testing.T) {
	pool := NewConnectionPool(100 * time.Millisecond)

	dc := DataCenter{ID: 2}
	conn := &mockTransport{}

	pool.Put(2, dc, conn)

	// Wait for expiry.
	time.Sleep(150 * time.Millisecond)

	// Should miss (expired).
	_, ok := pool.Get(2, dc)
	if ok {
		t.Fatal("Get should return false after expiry")
	}
}

func TestConnectionPoolEvict(t *testing.T) {
	pool := NewConnectionPool(10 * time.Second)

	dc := DataCenter{ID: 2}
	conn := &mockTransport{}

	pool.Put(2, dc, conn)
	pool.Evict(2, dc)

	_, ok := pool.Get(2, dc)
	if ok {
		t.Fatal("Get should return false after Evict")
	}
}

func TestConnectionPoolPurge(t *testing.T) {
	pool := NewConnectionPool(100 * time.Millisecond)

	dc := DataCenter{ID: 2}
	pool.Put(2, dc, &mockTransport{})
	pool.Put(2, DataCenter{ID: 2, IPv6: true}, &mockTransport{})

	if pool.Count() != 2 {
		t.Fatalf("Count before purge: %d, want 2", pool.Count())
	}

	time.Sleep(150 * time.Millisecond)

	purged := pool.Purge()
	if purged != 2 {
		t.Fatalf("Purge: %d, want 2", purged)
	}
	if pool.Count() != 0 {
		t.Fatalf("Count after purge: %d, want 0", pool.Count())
	}
}

func TestRouteQuery(t *testing.T) {
	tests := []struct {
		name  string
		query tg.TLObject
		want  SessionSlotType
	}{
		{"upload.saveFilePart", &tg.UploadSaveFilePartRequest{}, SlotUpload},
		{"upload.saveBigFilePart", &tg.UploadSaveBigFilePartRequest{}, SlotUpload},
		{"upload.getFile", &tg.UploadGetFileRequest{}, SlotDownload},
		{"upload.getWebFile", &tg.UploadGetWebFileRequest{}, SlotDownload},
		{"ping (main)", &tg.PingRequest{}, SlotMain},
		{"sendMessage (main)", &tg.MessagesSendMessageRequest{}, SlotMain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RouteQuery(tt.query)
			if got != tt.want {
				t.Errorf("RouteQuery(%T) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}
