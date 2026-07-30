package telegram

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/mtgo-labs/mtgo/internal/transport"
)

type sessionTransport struct {
	transport transport.Transport
	conn      net.Conn
}

type tcpTransport interface {
	transport.Transport
	Connect() error
}

type closableTransport interface {
	Close() error
}

type connectedTransport interface {
	IsConnected() bool
}

type deadlineTransport interface {
	SetWriteDeadline(time.Time) error
	SetReadDeadline(time.Time) error
}

type httpWaitInnerTransport interface {
	HTTPWaitParams() (maxDelay, waitAfter, maxWait int32)
	StartHTTPWait(frame func(context.Context) ([]byte, error))
}

func newSessionTransport(t transport.Transport, conn net.Conn) *sessionTransport {
	if conn != nil {
		_ = transport.SetTCPNoDelay(conn, true)
	}
	return &sessionTransport{transport: t, conn: conn}
}

func newTCPTransport(mode TransportMode, conn net.Conn) (tcpTransport, error) {
	switch mode {
	case TransportModeAbridged:
		return transport.NewTCPAbridged(conn), nil
	case TransportModeIntermediate:
		return transport.NewTCPIntermediate(conn), nil
	case TransportModePaddedIntermediate:
		return transport.NewTCPPaddedIntermediate(conn), nil
	case TransportModeFull:
		return transport.NewTCPFull(conn), nil
	default:
		return nil, fmt.Errorf("telegram: unsupported transport mode %d", mode)
	}
}

// obfuscatedMarkerForMode returns the obfuscated2 protocol tag byte for a
// given transport mode, or false if the mode cannot be obfuscated.
func obfuscatedMarkerForMode(mode TransportMode) (byte, bool) {
	switch mode {
	case TransportModeAbridged:
		return 0xEF, true
	case TransportModeIntermediate, TransportModePaddedIntermediate:
		return 0xEE, true
	default:
		return 0, false
	}
}

// createTransport creates the transport for a connection, wrapping it in
// obfuscated2 when AlwaysObfuscate is enabled. The inner transport's Connect
// is NOT called — the obfuscated handshake embeds the protocol tag in the
// nonce instead.
func (c *Client) createTransport(conn net.Conn) (tcpTransport, error) {
	mode := c.config().TransportMode
	tp, err := newTCPTransport(mode, conn)
	if err != nil {
		return nil, err
	}
	if c.config().AlwaysObfuscate {
		marker, ok := obfuscatedMarkerForMode(mode)
		if !ok {
			return nil, fmt.Errorf("telegram: AlwaysObfuscate not supported for transport %v", mode)
		}
		return transport.NewTCPObfuscated(tp, marker), nil
	}
	return tp, nil
}

func (s *sessionTransport) Send(data []byte) error {
	buf := bytes.NewBuffer(data)
	return s.transport.Send(buf)
}

func (s *sessionTransport) Recv() ([]byte, error) {
	return s.transport.Recv()
}

func (s *sessionTransport) Close() error {
	if s.conn != nil {
		// Force RST via SO_LINGER=0 before closing. Without this the TCP
		// stack does a graceful FIN → TIME_WAIT, and a fast reconnect
		// can cause the server to flag the new connection as
		// AUTH_KEY_DUPLICATED (406).
		_ = transport.SetTCPLinger(s.conn, 0)
		return s.conn.Close()
	}
	if tp, ok := s.transport.(closableTransport); ok {
		return tp.Close()
	}
	return nil
}

func (s *sessionTransport) IsConnected() bool {
	if s.conn == nil {
		if transport, ok := s.transport.(connectedTransport); ok {
			return transport.IsConnected()
		}
		return false
	}
	return s.conn.RemoteAddr() != nil
}

func (s *sessionTransport) SetWriteDeadline(t time.Time) error {
	if s.conn != nil {
		return s.conn.SetWriteDeadline(t)
	}
	if transport, ok := s.transport.(deadlineTransport); ok {
		return transport.SetWriteDeadline(t)
	}
	return nil
}

func (s *sessionTransport) SetReadDeadline(t time.Time) error {
	if s.conn != nil {
		return s.conn.SetReadDeadline(t)
	}
	if transport, ok := s.transport.(deadlineTransport); ok {
		return transport.SetReadDeadline(t)
	}
	return nil
}

func (s *sessionTransport) HTTPWaitParams() (maxDelay, waitAfter, maxWait int32, enabled bool) {
	transport, ok := s.transport.(httpWaitInnerTransport)
	if !ok {
		return 0, 0, 0, false
	}
	maxDelay, waitAfter, maxWait = transport.HTTPWaitParams()
	return maxDelay, waitAfter, maxWait, true
}

func (s *sessionTransport) StartHTTPWait(frame func(context.Context) ([]byte, error)) {
	if transport, ok := s.transport.(httpWaitInnerTransport); ok {
		transport.StartHTTPWait(frame)
	}
}
