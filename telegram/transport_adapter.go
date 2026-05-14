package telegram

import (
	"bytes"
	"fmt"
	"net"

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
