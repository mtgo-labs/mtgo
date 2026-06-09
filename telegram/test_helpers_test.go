package telegram

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/tg"
)

type testServer struct {
	listener net.Listener
	authKey  []byte
	done     chan struct{}
}

func newTestServer(authKey []byte) (*testServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &testServer{
		listener: ln,
		authKey:  authKey,
		done:     make(chan struct{}),
	}
	go s.serve()
	return s, nil
}

func (s *testServer) Addr() string {
	return s.listener.Addr().String()
}

func (s *testServer) Close() {
	s.listener.Close()
	close(s.done)
}

func (s *testServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *testServer) handleConn(conn net.Conn) {
	defer conn.Close()

	handshake := make([]byte, 4)
	if _, err := io.ReadFull(conn, handshake); err != nil {
		return
	}

	authKeyID := computeAuthKeyID(s.authKey)

	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return
		}
		length := binary.LittleEndian.Uint32(lenBuf)
		if length > 1<<24 {
			return
		}
		data := make([]byte, length)
		if length > 0 {
			if _, err := io.ReadFull(conn, data); err != nil {
				return
			}
		}

		constructorID, pingID, decrypted, err := unpackSafeNoSessionCheck(data, s.authKey, authKeyID)
		if err != nil {
			return
		}

		if constructorID == tg.PingTypeID {
			salt := int64(binary.LittleEndian.Uint64(decrypted[0:8]))
			sessionID := decrypted[8:16]

			respMsgID := (time.Now().Unix() << 32) | 1
			respSeqNo := uint32(0)

			origMsgID := int64(binary.LittleEndian.Uint64(decrypted[16:24]))
			pong := &tg.Pong{MsgID: origMsgID, PingID: pingID}
			respMsg := &tg.MTProtoMessage{MsgID: respMsgID, SeqNo: respSeqNo, Body: pong}

			encrypted, err := crypto.Pack(respMsg, salt, sessionID, s.authKey, authKeyID)
			if err != nil {
				continue
			}

			var resp bytes.Buffer
			binary.Write(&resp, binary.LittleEndian, uint32(len(encrypted)))
			resp.Write(encrypted)

			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			conn.Write(resp.Bytes())
		}
	}
}

func unpackSafeNoSessionCheck(data, authKey, authKeyID []byte) (constructorID uint32, pingID int64, decrypted []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unpack error: %v", r)
		}
	}()
	if len(data) < 24 {
		return 0, 0, nil, fmt.Errorf("data too short")
	}
	if !bytes.Equal(data[:8], authKeyID) {
		return 0, 0, nil, fmt.Errorf("auth_key_id mismatch")
	}
	msgKey := data[8:24]
	encrypted := data[24:]
	keyArr, ivArr := crypto.KDF(authKey, msgKey, false)
	decrypted, err = crypto.IGEDecrypt(encrypted, keyArr[:], ivArr[:])
	if err != nil {
		return 0, 0, nil, err
	}
	tmpChk := make([]byte, 32+len(decrypted))
	copy(tmpChk, authKey[96:128])
	copy(tmpChk[32:], decrypted)
	msgKeyCheck := sha256.Sum256(tmpChk)
	if !bytes.Equal(msgKey, msgKeyCheck[8:24]) {
		return 0, 0, nil, fmt.Errorf("msg_key mismatch")
	}
	r := bytes.NewReader(decrypted[16:])
	_ = tg.ReadLong(r)
	_ = tg.ReadInt(r)
	length := tg.ReadInt(r)
	if length > 1<<20 {
		return 0, 0, nil, fmt.Errorf("message body too large")
	}
	bodyBytes := make([]byte, length)
	if _, err2 := io.ReadFull(r, bodyBytes); err2 != nil {
		return 0, 0, nil, err2
	}
	constructorID = binary.LittleEndian.Uint32(bodyBytes[0:4])
	if constructorID == tg.PingTypeID && len(bodyBytes) >= 12 {
		pingID = int64(binary.LittleEndian.Uint64(bodyBytes[4:12]))
	}
	return constructorID, pingID, decrypted, nil
}

func computeAuthKeyID(key []byte) []byte {
	h := sha1.Sum(key)
	id := make([]byte, 8)
	copy(id, h[12:20])
	return id
}

type testServerDialer struct {
	addr string
}

func (d *testServerDialer) Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", d.addr, timeout)
}

func newTestClient(appID int, appHash string, opts ...Config) (*Client, *testServer) {
	st := NewMemoryStorage()
	authKey := make([]byte, 256)
	_ = st.SetAuthKey(authKey)
	_ = st.SetDCID(2)
	var cfg *Config
	if len(opts) > 0 {
		cfg = &opts[0]
	}
	c, _ := NewClient(int32(appID), appHash, cfg)
	c.cfg.TransportMode = TransportModeIntermediate
	c.setTestStorage(st)

	sess, err := session.NewSession(session.DataCenter{ID: 2}, st, "Test", "0.1", "en", "en")
	if err != nil {
		panic(err)
	}
	c.setTestSession(sess)

	srv, err := newTestServer(authKey)
	if err != nil {
		panic(err)
	}

	dialer := &testServerDialer{addr: srv.Addr()}
	c.setTestDialer(transport.Dialer(dialer))

	return c, srv
}
