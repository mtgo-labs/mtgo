package mtproxy

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
)

type mockConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
	addr     net.Addr
	deadline time.Time
}

func newMockConn(readData []byte) *mockConn {
	return &mockConn{
		readBuf:  bytes.NewBuffer(readData),
		writeBuf: &bytes.Buffer{},
		addr:     &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 443},
	}
}

func (m *mockConn) Read(p []byte) (int, error)  { return m.readBuf.Read(p) }
func (m *mockConn) Write(p []byte) (int, error) { return m.writeBuf.Write(p) }
func (m *mockConn) Close() error                { m.closed = true; return nil }
func (m *mockConn) LocalAddr() net.Addr         { return m.addr }
func (m *mockConn) RemoteAddr() net.Addr        { return m.addr }
func (m *mockConn) SetDeadline(t time.Time) error {
	m.deadline = t
	return nil
}
func (m *mockConn) SetReadDeadline(t time.Time) error {
	m.deadline = t
	return nil
}
func (m *mockConn) SetWriteDeadline(t time.Time) error {
	m.deadline = t
	return nil
}

type errConn struct {
	writeErr error
	readErr  error
}

func (e *errConn) Read(p []byte) (int, error) {
	if e.readErr != nil {
		return 0, e.readErr
	}
	return 0, errors.New("read error")
}
func (e *errConn) Write(p []byte) (int, error) {
	if e.writeErr != nil {
		return 0, e.writeErr
	}
	return 0, errors.New("write error")
}
func (e *errConn) Close() error                       { return nil }
func (e *errConn) LocalAddr() net.Addr                { return nil }
func (e *errConn) RemoteAddr() net.Addr               { return nil }
func (e *errConn) SetDeadline(t time.Time) error       { return nil }
func (e *errConn) SetReadDeadline(t time.Time) error   { return nil }
func (e *errConn) SetWriteDeadline(t time.Time) error  { return nil }

type failAfterN struct {
	net.Conn
	writesLeft int
	writeErr   error
	readErr    error
}

func (f *failAfterN) Write(p []byte) (int, error) {
	if f.writesLeft <= 0 {
		return 0, f.writeErr
	}
	f.writesLeft--
	return f.Conn.Write(p)
}

func (f *failAfterN) Read(p []byte) (int, error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	return f.Conn.Read(p)
}

func buildValidServerHello(secret, clientRandom []byte) []byte {
	header := []byte{0x16, 0x03, 0x03, 0x00, 0x45}
	serverHello := make([]byte, 69)
	serverHello[0] = 0x02
	binary.BigEndian.PutUint16(serverHello[1:3], 0x0303)

	full := make([]byte, 0, len(header)+len(serverHello))
	full = append(full, header...)
	full = append(full, serverHello...)

	mac := hmac.New(sha256.New, secret)
	mac.Write(clientRandom)
	mac.Write(full)
	expected := mac.Sum(nil)

	copy(full[11:43], expected)
	return full
}

func TestVerifyServerHello(t *testing.T) {
	secret := make([]byte, 16)
	for i := range secret {
		secret[i] = byte(i)
	}
	clientRandom := make([]byte, 32)
	rand.Read(clientRandom)

	t.Run("valid", func(t *testing.T) {
		resp := buildValidServerHello(secret, clientRandom)
		if err := verifyServerHello(secret, clientRandom, resp); err != nil {
			t.Errorf("expected success, got %v", err)
		}
	})

	t.Run("too short", func(t *testing.T) {
		err := verifyServerHello(secret, clientRandom, []byte{1, 2, 3})
		if err == nil {
			t.Error("expected error for short response")
		}
	})

	t.Run("bad signature", func(t *testing.T) {
		resp := buildValidServerHello(secret, clientRandom)
		resp[11] ^= 0xFF
		err := verifyServerHello(secret, clientRandom, resp)
		if err != ErrServerHelloFailed {
			t.Errorf("expected ErrServerHelloFailed, got %v", err)
		}
	})
}

func TestFakeTLSHandshake(t *testing.T) {
	secret := make([]byte, 16)
	for i := range secret {
		secret[i] = byte(i)
	}

	t.Run("write error on client hello", func(t *testing.T) {
		conn := &errConn{writeErr: errors.New("write fail")}
		_, err := fakeTLSHandshake(conn, secret, "google.com")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("read error on server hello header", func(t *testing.T) {
		conn := &errConn{readErr: errors.New("read fail")}
		_, err := fakeTLSHandshake(conn, secret, "google.com")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("wrong record type", func(t *testing.T) {
		mc := newMockConn([]byte{0x17, 0x03, 0x03, 0x00, 0x05, 1, 2, 3, 4, 5})
		_, err := fakeTLSHandshake(mc, secret, "google.com")
		if err == nil {
			t.Error("expected error for wrong record type")
		}
	})

	t.Run("read error on server hello body", func(t *testing.T) {
		mc := newMockConn([]byte{0x16, 0x03, 0x03, 0x00, 0x45})
		_, err := fakeTLSHandshake(mc, secret, "google.com")
		if err == nil {
			t.Error("expected error for short read")
		}
	})

	t.Run("bad server hello signature", func(t *testing.T) {
		serverHello := make([]byte, 69)
		rand.Read(serverHello)
		header := []byte{0x16, 0x03, 0x03, 0x00, 0x45}
		data := append(header, serverHello...)
		mc := newMockConn(data)
		_, err := fakeTLSHandshake(mc, secret, "google.com")
		if err == nil {
			t.Error("expected error for bad server hello")
		}
	})
}

func TestTLSConnReadWrite(t *testing.T) {
	secret := make([]byte, 16)
	rand.Read(secret)
	clientRandom := make([]byte, 32)
	rand.Read(clientRandom)

	t.Run("read app data", func(t *testing.T) {
		payload := []byte("hello mtproxy")
		ctr := testCTR()
		encrypted := ctr.Process(payload)

		record := make([]byte, tlsRecordHeaderLen+len(encrypted))
		record[0] = tlsAppDataType
		record[1] = 0x03
		record[2] = 0x03
		binary.BigEndian.PutUint16(record[3:], uint16(len(encrypted)))
		copy(record[tlsRecordHeaderLen:], encrypted)

		mc := newMockConn(record)
		dec := testCTR()
		tc := &tlsConn{conn: mc, enc: nil, dec: dec}

		buf := make([]byte, 64)
		n, err := tc.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(encrypted) {
			t.Errorf("read %d bytes, want %d", n, len(encrypted))
		}
	})

	t.Run("read skips change cipher spec", func(t *testing.T) {
		ccs := []byte{tlsChangeCipherSpec, 0x03, 0x03, 0x00, 0x01, 0x01}
		payload := []byte("after ccs")
		ctr := testCTR()
		encrypted := ctr.Process(payload)

		appRecord := make([]byte, tlsRecordHeaderLen+len(encrypted))
		appRecord[0] = tlsAppDataType
		appRecord[1] = 0x03
		appRecord[2] = 0x03
		binary.BigEndian.PutUint16(appRecord[3:], uint16(len(encrypted)))
		copy(appRecord[tlsRecordHeaderLen:], encrypted)

		mc := newMockConn(append(ccs, appRecord...))
		dec := testCTR()
		tc := &tlsConn{conn: mc, dec: dec}

		buf := make([]byte, 64)
		n, err := tc.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buf[:n], payload) {
			t.Errorf("expected %q, got %q", payload, buf[:n])
		}
	})

	t.Run("read unexpected record type", func(t *testing.T) {
		record := []byte{0x15, 0x03, 0x03, 0x00, 0x02, 0x01, 0x00}
		mc := newMockConn(record)
		tc := &tlsConn{conn: mc, dec: testCTR()}
		_, err := tc.Read(make([]byte, 64))
		if err == nil {
			t.Error("expected error for unexpected record type")
		}
	})

	t.Run("read oversized record", func(t *testing.T) {
		record := make([]byte, tlsRecordHeaderLen)
		record[0] = tlsAppDataType
		record[1] = 0x03
		record[2] = 0x03
		binary.BigEndian.PutUint16(record[3:], maxTLSRecordPayload+1)
		mc := newMockConn(record)
		tc := &tlsConn{conn: mc, dec: testCTR()}
		_, err := tc.Read(make([]byte, 64))
		if err == nil {
			t.Error("expected error for oversized record")
		}
	})

	t.Run("write first write includes ccs prefix", func(t *testing.T) {
		mc := newMockConn(nil)
		tc := &tlsConn{conn: mc, enc: testCTR(), firstWrite: true}

		data := []byte("test data")
		n, err := tc.Write(data)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(data) {
			t.Errorf("write returned %d, want %d", n, len(data))
		}

		written := mc.writeBuf.Bytes()
		expectedPrefix := []byte{0x14, 0x03, 0x03, 0x00, 0x01, 0x01}
		if !bytes.HasPrefix(written, expectedPrefix) {
			t.Errorf("first write missing CCS prefix, got %x", written[:6])
		}
		if tc.firstWrite {
			t.Error("firstWrite should be false after first write")
		}
	})

	t.Run("write subsequent no prefix", func(t *testing.T) {
		mc := newMockConn(nil)
		tc := &tlsConn{conn: mc, enc: testCTR(), firstWrite: false}

		data := []byte("more data")
		n, err := tc.Write(data)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(data) {
			t.Errorf("write returned %d, want %d", n, len(data))
		}

		written := mc.writeBuf.Bytes()
		if written[0] != tlsAppDataType {
			t.Errorf("record type = 0x%02x, want 0x%02x", written[0], tlsAppDataType)
		}
	})

	t.Run("write error", func(t *testing.T) {
		conn := &errConn{writeErr: errors.New("fail")}
		tc := &tlsConn{conn: conn, enc: testCTR(), firstWrite: false}
		_, err := tc.Write([]byte("x"))
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("read error", func(t *testing.T) {
		conn := &errConn{readErr: errors.New("fail")}
		tc := &tlsConn{conn: conn, dec: testCTR()}
		_, err := tc.Read(make([]byte, 10))
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestTLSConnPassthrough(t *testing.T) {
	mc := newMockConn(nil)
	tc := &tlsConn{conn: mc}

	if err := tc.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if !mc.closed {
		t.Error("underlying conn not closed")
	}
	if tc.LocalAddr() == nil {
		t.Error("LocalAddr returned nil")
	}
	if tc.RemoteAddr() == nil {
		t.Error("RemoteAddr returned nil")
	}
	if err := tc.SetDeadline(time.Time{}); err != nil {
		t.Errorf("SetDeadline: %v", err)
	}
	if err := tc.SetReadDeadline(time.Time{}); err != nil {
		t.Errorf("SetReadDeadline: %v", err)
	}
	if err := tc.SetWriteDeadline(time.Time{}); err != nil {
		t.Errorf("SetWriteDeadline: %v", err)
	}
}

func TestReadFull(t *testing.T) {
	t.Run("complete read", func(t *testing.T) {
		mc := newMockConn([]byte{1, 2, 3, 4, 5})
		buf := make([]byte, 5)
		n, err := readFull(mc, buf)
		if err != nil {
			t.Fatal(err)
		}
		if n != 5 {
			t.Errorf("read %d, want 5", n)
		}
	})

	t.Run("short read", func(t *testing.T) {
		mc := newMockConn([]byte{1, 2})
		buf := make([]byte, 5)
		_, err := readFull(mc, buf)
		if err == nil {
			t.Error("expected error on short read")
		}
	})
}

func TestObfuscated2Handshake(t *testing.T) {
	secret := make([]byte, 16)
	for i := range secret {
		secret[i] = byte(i)
	}

	t.Run("success", func(t *testing.T) {
		mc := newMockConn(nil)
		obfs, err := obfuscated2Handshake(mc, secret, 2, 0xdd)
		if err != nil {
			t.Fatal(err)
		}
		if mc.writeBuf.Len() != obfuscatedHeaderLen {
			t.Errorf("wrote %d bytes, want %d", mc.writeBuf.Len(), obfuscatedHeaderLen)
		}
		if obfs == nil {
			t.Error("expected non-nil obfuscatedConn")
		}
	})

	t.Run("write error", func(t *testing.T) {
		conn := &errConn{writeErr: errors.New("fail")}
		_, err := obfuscated2Handshake(conn, secret, 2, 0xdd)
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestObfuscatedConnReadWrite(t *testing.T) {
	secret := make([]byte, 16)
	rand.Read(secret)

	t.Run("read", func(t *testing.T) {
		data := []byte("test payload data")
		mc := newMockConn(data)
		obfs, err := obfuscated2Handshake(mc, secret, 1, 0xee)
		if err != nil {
			t.Fatal(err)
		}

		buf := make([]byte, len(data))
		_, err = obfs.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("write", func(t *testing.T) {
		mc := newMockConn(nil)
		obfs, err := obfuscated2Handshake(mc, secret, 1, 0xee)
		if err != nil {
			t.Fatal(err)
		}

		data := []byte("write test")
		n, err := obfs.Write(data)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(data) {
			t.Errorf("write returned %d, want %d", n, len(data))
		}
	})

	t.Run("write error", func(t *testing.T) {
		mc := newMockConn(nil)
		conn := &failAfterN{Conn: mc, writesLeft: 1, writeErr: errors.New("fail")}
		obfs, err := obfuscated2Handshake(conn, secret, 1, 0xee)
		if err != nil {
			t.Fatal(err)
		}
		_, err = obfs.Write([]byte("x"))
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("read error", func(t *testing.T) {
		mc := newMockConn(nil)
		conn := &failAfterN{Conn: mc, writesLeft: 1, readErr: errors.New("fail")}
		obfs, err := obfuscated2Handshake(conn, secret, 1, 0xee)
		if err != nil {
			t.Fatal(err)
		}
		_, err = obfs.Read(make([]byte, 10))
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestObfuscatedConnPassthrough(t *testing.T) {
	mc := newMockConn(nil)
	obfs, err := obfuscated2Handshake(mc, make([]byte, 16), 1, 0xdd)
	if err != nil {
		t.Fatal(err)
	}

	if err := obfs.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if !mc.closed {
		t.Error("underlying conn not closed")
	}
	if obfs.LocalAddr() == nil {
		t.Error("LocalAddr returned nil")
	}
	if obfs.RemoteAddr() == nil {
		t.Error("RemoteAddr returned nil")
	}
	if err := obfs.SetDeadline(time.Time{}); err != nil {
		t.Errorf("SetDeadline: %v", err)
	}
	if err := obfs.SetReadDeadline(time.Time{}); err != nil {
		t.Errorf("SetReadDeadline: %v", err)
	}
	if err := obfs.SetWriteDeadline(time.Time{}); err != nil {
		t.Errorf("SetWriteDeadline: %v", err)
	}
}

func TestHandshake(t *testing.T) {
	secret := Secret{
		Type:   SecretSecured,
		Secret: make([]byte, 16),
		Tag:    0xdd,
	}

	t.Run("obfuscated2 only", func(t *testing.T) {
		mc := newMockConn(nil)
		_, err := Handshake(mc, secret, 2)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with fake tls", func(t *testing.T) {
		tlsSecret := Secret{
			Type:   SecretTLS,
			Secret: make([]byte, 16),
			Tag:    0xee,
			Domain: "google.com",
		}
		mc := newMockConn(nil)
		_, err := Handshake(mc, tlsSecret, 2)
		if err == nil {
			t.Error("expected error (can't complete TLS handshake over mock)")
		}
	})

	t.Run("obfuscated2 write failure", func(t *testing.T) {
		conn := &errConn{writeErr: errors.New("fail")}
		_, err := Handshake(conn, secret, 2)
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestCurve25519DoubleX(t *testing.T) {
	x := big.NewInt(42)
	result := curve25519DoubleX(x)
	if result == nil {
		t.Error("result should not be nil")
	}
	if result.Sign() < 0 {
		t.Error("result should be non-negative")
	}
}

func testCTR() *crypto.CTRCipher {
	key := make([]byte, 32)
	iv := make([]byte, 16)
	for i := range key {
		key[i] = byte(i)
	}
	for i := range iv {
		iv[i] = byte(i)
	}
	c, err := crypto.NewCTRCipher(key, iv)
	if err != nil {
		panic(err)
	}
	return c
}
