package transport

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/coder/websocket"
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
	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"binary"},
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	netConn := websocket.NetConn(context.Background(), wsConn, websocket.MessageBinary)

	obfsConn, err := acceptObfuscated2(netConn)
	if err != nil {
		netConn.Close()
		wsConn.Close(websocket.StatusNormalClosure, "close")
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
	wsConn, _, err := websocket.Dial(ctx, addr, &websocket.DialOptions{
		Subprotocols: []string{"binary"},
	})
	if err != nil {
		return nil, fmt.Errorf("ws: dial: %w", err)
	}

	netConn := websocket.NetConn(context.Background(), wsConn, websocket.MessageBinary)

	obfsConn, err := dialObfuscated2(netConn, 0xEE)
	if err != nil {
		netConn.Close()
		wsConn.Close(websocket.StatusNormalClosure, "close")
		return nil, err
	}

	return &wsConnCloser{Conn: obfsConn, ws: wsConn}, nil
}

type wsConnCloser struct {
	net.Conn
	ws *websocket.Conn
}

func (c *wsConnCloser) Close() error {
	_ = c.ws.Close(websocket.StatusNormalClosure, "close")
	return c.Conn.Close()
}

type obfsConn struct {
	net.Conn
	enc *crypto.CTRCipher
	dec *crypto.CTRCipher
}

func (c *obfsConn) Read(p []byte) (int, error) {
	buf := make([]byte, len(p))
	n, err := c.Conn.Read(buf)
	if n > 0 {
		plain := c.dec.Process(buf[:n])
		copy(p, plain)
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

	enc := crypto.NewCTRCipher(encKey, encIV)
	dec := crypto.NewCTRCipher(decKey, decIV)

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

	switch string(nonce[:4]) {
	case "\x16\x03\x01\x02", "\xdd\xdd\xdd\xdd", "\xee\xee\xee\xee", "POST", "GET ", "HEAD", "OPTI":
		return true
	}

	return nonce[4] == 0 && nonce[5] == 0 && nonce[6] == 0 && nonce[7] == 0
}

func acceptObfuscated2(conn net.Conn) (*obfsConn, error) {
	nonce := make([]byte, 64)
	if _, err := readFull(conn, nonce); err != nil {
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

	enc := crypto.NewCTRCipher(encKey, encIV)
	dec := crypto.NewCTRCipher(decKey, decIV)

	dec.Process(make([]byte, 64))

	return &obfsConn{Conn: conn, enc: enc, dec: dec}, nil
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	read := 0
	for read < len(buf) {
		n, err := conn.Read(buf[read:])
		read += n
		if err != nil {
			return read, err
		}
	}
	return read, nil
}
