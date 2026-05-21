package transport

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	wsOpText   = 0x1
	wsOpBinary = 0x2
	wsOpClose  = 0x8
	wsOpPing   = 0x9
	wsOpPong   = 0xA

	wsFin    = 0x80
	wsMask   = 0x80
	wsLen16  = 126
	wsLen64  = 127
	wsStatusNormalClosure = 1000
)

var wsKeyGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// wsConn wraps a raw TCP connection (post-HTTP-upgrade) and provides a
// net.Conn that reads and writes WebSocket binary frames. It handles
// fragmentation, masking (client→server), and close frame sending.
type wsConn struct {
	conn   net.Conn
	br     *bufio.Reader
	mu     sync.Mutex // serializes writes

	readBuf    []byte // buffered bytes from the current frame
	frameBuf   []byte // reusable buffer for reading frame payloads
	readEOF    bool   // set when a final frame has been drained
	readErr    error  // sticky read error
}

// wsReadWriter implements the io.ReadWriter needed by bufio.ReadWriter
type wsReadWriter struct {
	*wsConn
}

func (w *wsReadWriter) Read(p []byte) (int, error)  { return w.conn.Read(p) }
func (w *wsReadWriter) Write(p []byte) (int, error) { return w.conn.Write(p) }

// newWSConn creates a new wsConn atop an already-upgraded TCP connection.
// If br is nil, a fresh bufio.Reader is created.
func newWSConn(conn net.Conn, br *bufio.Reader) *wsConn {
	if br == nil {
		br = bufio.NewReaderSize(conn, 4096)
	}
	return &wsConn{
		conn: conn,
		br:   br,
	}
}

// Read reads the next chunk of data from a WebSocket binary frame. It
// buffers across frame boundaries transparently: fragmentation, ping,
// pong, and close frames are handled internally.
func (c *wsConn) Read(p []byte) (int, error) {
	if c.readErr != nil && len(c.readBuf) == 0 {
		return 0, c.readErr
	}

	for len(c.readBuf) == 0 && c.readErr == nil {
		op, payload, err := c.readFrame()
		if err != nil {
			c.readErr = err
			return 0, err
		}
		switch op {
		case wsOpBinary, wsOpText:
			c.readBuf = payload
		case wsOpClose:
			c.readErr = io.EOF
			return 0, io.EOF
		case wsOpPing:
			c.writeControl(wsOpPong, payload)
		case wsOpPong:
			// ignore
		}
	}

	n := copy(p, c.readBuf)
	c.readBuf = c.readBuf[n:]
	return n, nil
}

// Write sends p as a single WebSocket binary frame (FIN + opcode 0x2).
func (c *wsConn) Write(p []byte) (int, error) {
	if err := c.writeFrame(wsOpBinary|wsFin, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close sends a WebSocket close frame and closes the underlying TCP
// connection. It writes the frame synchronously; any error is silently
// swallowed because the caller wants to close regardless.
func (c *wsConn) Close() error {
	_ = c.writeCloseFrame(wsStatusNormalClosure, "client close")
	return c.conn.Close()
}

// LocalAddr implements net.Conn.
func (c *wsConn) LocalAddr() net.Addr                { return c.conn.LocalAddr() }

// RemoteAddr implements net.Conn.
func (c *wsConn) RemoteAddr() net.Addr               { return c.conn.RemoteAddr() }

// SetDeadline implements net.Conn.
func (c *wsConn) SetDeadline(t time.Time) error       { return c.conn.SetDeadline(t) }

// SetReadDeadline implements net.Conn.
func (c *wsConn) SetReadDeadline(t time.Time) error   { return c.conn.SetReadDeadline(t) }

// SetWriteDeadline implements net.Conn.
func (c *wsConn) SetWriteDeadline(t time.Time) error  { return c.conn.SetWriteDeadline(t) }

// --- frame I/O ---

func (c *wsConn) readFrame() (op byte, payload []byte, err error) {
	var header [2]byte
	if _, err = io.ReadFull(c.br, header[:]); err != nil {
		return 0, nil, err
	}

	op = header[0] & 0x0F
	masked := header[1]&wsMask != 0

	length := int64(header[1] & 0x7F)
	switch {
	case length == wsLen16:
		var b [2]byte
		if _, err = io.ReadFull(c.br, b[:]); err != nil {
			return 0, nil, err
		}
		length = int64(binary.BigEndian.Uint16(b[:]))
	case length == wsLen64:
		var b [8]byte
		if _, err = io.ReadFull(c.br, b[:]); err != nil {
			return 0, nil, err
		}
		length = int64(binary.BigEndian.Uint64(b[:]))
	}

	if length < 0 || length > int64(MaxPayloadLen) {
		return 0, nil, fmt.Errorf("ws: frame too large: %d", length)
	}

	var maskKey [4]byte
	if masked {
		if _, err = io.ReadFull(c.br, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}

	if cap(c.frameBuf) < int(length) {
		c.frameBuf = make([]byte, length)
	}
	payload = c.frameBuf[:length]
	if _, err = io.ReadFull(c.br, payload); err != nil {
		return 0, nil, err
	}

	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	return op, payload, nil
}

func (c *wsConn) writeFrame(opcode byte, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	length := len(payload)

	buf := make([]byte, 0, 14+length)
	buf = append(buf, opcode|wsFin)

	switch {
	case length <= 125:
		buf = append(buf, byte(length)|wsMask)
	case length <= 65535:
		buf = append(buf, wsLen16|wsMask)
		buf = append(buf, byte(length>>8), byte(length))
	default:
		buf = append(buf, wsLen64|wsMask)
		buf = binary.BigEndian.AppendUint64(buf, uint64(length))
	}

	var maskKey [4]byte
	if _, err := rand.Read(maskKey[:]); err != nil {
		return err
	}
	buf = append(buf, maskKey[:]...)

	start := len(buf)
	buf = append(buf, payload...)
	for i := range payload {
		buf[start+i] ^= maskKey[i%4]
	}

	_, err := c.conn.Write(buf)
	return err
}

func (c *wsConn) writeControl(opcode byte, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var maskKey [4]byte
	if _, err := rand.Read(maskKey[:]); err != nil {
		return err
	}

	buf := make([]byte, 0, 2+4+len(payload))
	buf = append(buf, opcode|wsFin, byte(len(payload))|wsMask)
	buf = append(buf, maskKey[:]...)
	start := len(buf)
	buf = append(buf, payload...)
	for i := range payload {
		buf[start+i] ^= maskKey[i%4]
	}

	_, err := c.conn.Write(buf)
	return err
}

func (c *wsConn) writeCloseFrame(code int, reason string) error {
	payload := make([]byte, 2+len(reason))
	binary.BigEndian.PutUint16(payload, uint16(code))
	copy(payload[2:], reason)
	return c.writeControl(wsOpClose, payload)
}

// --- dial ---

// wsDial performs an HTTP WebSocket upgrade over the given TCP connection
// and returns a wsConn ready for binary frame I/O.
func wsDial(conn net.Conn, addr string) (*wsConn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("ws: parse addr: %w", err)
	}

	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("ws: generate key: %w", err)
	}
	keyB64 := base64.StdEncoding.EncodeToString(key)

	host := u.Host
	path := u.Path
	if path == "" {
		path = "/"
	}

	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + keyB64 + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Protocol: binary\r\n" +
		"\r\n"

	if _, err := conn.Write([]byte(req)); err != nil {
		return nil, fmt.Errorf("ws: write handshake: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		return nil, fmt.Errorf("ws: read handshake response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("ws: unexpected status: %d", resp.StatusCode)
	}

	expectedAccept := computeAcceptKey(keyB64)
	if resp.Header.Get("Sec-WebSocket-Accept") != expectedAccept {
		return nil, errors.New("ws: invalid Sec-WebSocket-Accept")
	}

	return newWSConn(conn, br), nil
}

// wsAccept performs a server-side WebSocket upgrade handshake.
func wsAccept(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	if r.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "not a websocket request", http.StatusBadRequest)
		return nil, errors.New("ws: not a websocket request")
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing Sec-WebSocket-Key", http.StatusBadRequest)
		return nil, errors.New("ws: missing key")
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("ws: server does not support hijacking")
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, fmt.Errorf("ws: hijack: %w", err)
	}

	acceptKey := computeAcceptKey(key)
	_, _ = bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n" +
		"Sec-WebSocket-Protocol: binary\r\n\r\n")
	_ = bufrw.Flush()

	return newWSConn(conn, bufrw.Reader), nil
}

func computeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte(wsKeyGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
