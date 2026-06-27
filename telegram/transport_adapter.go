package telegram

import (
	"bytes"
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

func newSessionTransport(t transport.Transport, conn net.Conn) *sessionTransport {
	return &sessionTransport{transport: t, conn: conn}
}

func newTCPTransport(mode string, conn net.Conn) (tcpTransport, error) {
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
		return nil, fmt.Errorf("telegram: unsupported transport mode %q", mode)
	}
}

// obfuscatedMarkerForMode returns the obfuscated2 protocol tag byte for a
// given transport mode, or false if the mode cannot be obfuscated.
func obfuscatedMarkerForMode(mode string) (byte, bool) {
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
			return nil, fmt.Errorf("telegram: AlwaysObfuscate not supported for transport %q", mode)
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
		return s.conn.Close()
	}
	return nil
}

func (s *sessionTransport) IsConnected() bool {
	if s.conn == nil {
		return false
	}
	return s.conn.RemoteAddr() != nil
}

func (s *sessionTransport) SetWriteDeadline(t time.Time) error {
	if s.conn != nil {
		return s.conn.SetWriteDeadline(t)
	}
	return nil
}

func (s *sessionTransport) SetReadDeadline(t time.Time) error {
	if s.conn != nil {
		return s.conn.SetReadDeadline(t)
	}
	return nil
}
