package transport

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
)

type wsListener struct {
	addr   net.Addr
	ch     chan net.Conn
	closed chan struct{}
	once   sync.Once
}

func WebsocketListener(addr net.Addr) (net.Listener, http.Handler) {
	l := &wsListener{
		addr:   addr,
		ch:     make(chan net.Conn, 4),
		closed: make(chan struct{}),
	}
	return l, l
}

func (l *wsListener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wsConn, err := wsAccept(w, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	obfsConn, err := acceptObfuscated2(wsConn)
	if err != nil {
		wsConn.Close()
		return
	}

	select {
	case <-l.closed:
		obfsConn.Close()
		return
	case <-r.Context().Done():
		obfsConn.Close()
		return
	case l.ch <- obfsConn:
	}
}

func (l *wsListener) Accept() (net.Conn, error) {
	for {
		select {
		case <-l.closed:
			return nil, net.ErrClosed
		case conn := <-l.ch:
			return conn, nil
		}
	}
}

func (l *wsListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *wsListener) Addr() net.Addr { return l.addr }

func DialWebsocket(ctx context.Context, addr string) (net.Conn, error) {
	rawConn, err := dialWebsocketTCP(ctx, addr)
	if err != nil {
		return nil, err
	}

	wsConn, err := wsDial(rawConn, addr)
	if err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("ws: dial: %w", err)
	}

	obfsConn, err := dialObfuscated2(wsConn, 0xEE)
	if err != nil {
		wsConn.Close()
		return nil, err
	}

	return &wsConnCloser{Conn: obfsConn}, nil
}

func dialWebsocketTCP(ctx context.Context, addr string) (net.Conn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("ws: parse addr: %w", err)
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "wss" {
			port = "443"
		} else {
			port = "80"
		}
	}
	addrHost := net.JoinHostPort(host, port)

	if u.Scheme == "wss" {
		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		conn, err := tls.DialWithDialer(dialer, "tcp", addrHost, &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		})
		if err != nil {
			return nil, fmt.Errorf("ws: tls dial: %w", err)
		}
		return conn, nil
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addrHost)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

type wsConnCloser struct {
	net.Conn
}

type obfsConn struct {
	net.Conn
	enc     *crypto.CTRCipher
	dec     *crypto.CTRCipher
	readBuf []byte
}

func (c *obfsConn) Read(p []byte) (int, error) {
	if cap(c.readBuf) < len(p) {
		c.readBuf = make([]byte, len(p))
	}
	buf := c.readBuf[:len(p)]
	n, err := c.Conn.Read(buf)
	if n > 0 {
		c.dec.ProcessTo(buf[:n], p[:n])
	}
	return n, err
}

func (c *obfsConn) Write(p []byte) (int, error) {
	encrypted := c.enc.Process(p)
	if _, err := c.Conn.Write(encrypted); err != nil {
		return 0, err
	}
	return len(p), nil
}

func dialObfuscated2(conn net.Conn, marker byte) (*obfsConn, error) {
	nonce := make([]byte, 64)
	for {
		if _, err := rand.Read(nonce); err != nil {
			return nil, fmt.Errorf("obfuscated2: generate nonce: %w", err)
		}
		if invalidObfuscated2Nonce(nonce) {
			continue
		}
		break
	}

	nonce[56] = marker
	nonce[57] = marker
	nonce[58] = marker
	nonce[59] = marker

	encKey := make([]byte, 32)
	encIV := make([]byte, 16)
	copy(encKey, nonce[8:40])
	copy(encIV, nonce[40:56])

	reversed := make([]byte, 48)
	for i := 0; i < 48; i++ {
		reversed[i] = nonce[55-i]
	}
	decKey := make([]byte, 32)
	decIV := make([]byte, 16)
	copy(decKey, reversed[0:32])
	copy(decIV, reversed[32:48])

	enc, err := crypto.NewCTRCipher(encKey, encIV)
	if err != nil {
		return nil, fmt.Errorf("obfuscated2: create enc cipher: %w", err)
	}
	dec, err := crypto.NewCTRCipher(decKey, decIV)
	if err != nil {
		return nil, fmt.Errorf("obfuscated2: create dec cipher: %w", err)
	}

	encrypted := enc.Process(nonce)
	copy(nonce[56:64], encrypted[56:64])

	if _, err := conn.Write(nonce); err != nil {
		return nil, fmt.Errorf("obfuscated2: write nonce: %w", err)
	}

	return &obfsConn{Conn: conn, enc: enc, dec: dec}, nil
}

func invalidObfuscated2Nonce(nonce []byte) bool {
	if len(nonce) < 8 {
		return true
	}
	if nonce[0] == 0xEF {
		return true
	}

	prefix := binary.LittleEndian.Uint32(nonce[:4])
	switch prefix {
	case 0x02010316, 0xDDDDDDDD, 0xEEEEEEEE,
		0x54534F50, 0x20544547, 0x44414548, 0x4954504F:
		return true
	}

	return nonce[4] == 0 && nonce[5] == 0 && nonce[6] == 0 && nonce[7] == 0
}

func acceptObfuscated2(conn net.Conn) (*obfsConn, error) {
	nonce := make([]byte, 64)
	if _, err := io.ReadFull(conn, nonce); err != nil {
		return nil, fmt.Errorf("obfuscated2: read nonce: %w", err)
	}

	reversed := make([]byte, 48)
	for i := 0; i < 48; i++ {
		reversed[i] = nonce[55-i]
	}
	encKey := make([]byte, 32)
	encIV := make([]byte, 16)
	copy(encKey, reversed[0:32])
	copy(encIV, reversed[32:48])

	decKey := make([]byte, 32)
	decIV := make([]byte, 16)
	copy(decKey, nonce[8:40])
	copy(decIV, nonce[40:56])

	enc, err := crypto.NewCTRCipher(encKey, encIV)
	if err != nil {
		return nil, fmt.Errorf("obfuscated2: create enc cipher: %w", err)
	}
	dec, err := crypto.NewCTRCipher(decKey, decIV)
	if err != nil {
		return nil, fmt.Errorf("obfuscated2: create dec cipher: %w", err)
	}

	dec.Process(make([]byte, 64))

	return &obfsConn{Conn: conn, enc: enc, dec: dec}, nil
}
