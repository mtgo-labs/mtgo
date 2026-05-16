package transport

import (
	"bytes"
	"encoding/binary"
	"net"
)

// TCPIntermediateNoHeader implements the MTProto intermediate transport
// framing (4-byte LE length prefix) without sending the 0xEE protocol
// marker during Connect. This is used for WebSocket connections where
// the protocol marker is embedded in the obfuscated2 nonce instead.
type TCPIntermediateNoHeader struct {
	conn net.Conn
	buf  []byte
}

func NewTCPIntermediateNoHeader(conn net.Conn) *TCPIntermediateNoHeader {
	return &TCPIntermediateNoHeader{conn: conn}
}

func (t *TCPIntermediateNoHeader) Connect() error {
	return nil
}

func (t *TCPIntermediateNoHeader) Send(buf *bytes.Buffer) error {
	data := buf.Bytes()

	packet := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(packet[:4], uint32(len(data)))
	copy(packet[4:], data)

	_, err := t.conn.Write(packet)
	return err
}

func (t *TCPIntermediateNoHeader) Recv() ([]byte, error) {
	for len(t.buf) < 4 {
		if err := t.fill(); err != nil {
			return nil, err
		}
	}

	length := binary.LittleEndian.Uint32(t.buf[:4])
	if length > maxPayloadLen {
		return nil, ErrPayloadTooLarge
	}

	needed := 4 + int(length)
	for len(t.buf) < needed {
		if err := t.fill(); err != nil {
			return nil, err
		}
	}

	payload := make([]byte, length)
	copy(payload, t.buf[4:needed])
	t.buf = t.buf[needed:]
	return payload, nil
}

func (t *TCPIntermediateNoHeader) fill() error {
	tmp := make([]byte, 1<<20)
	n, err := t.conn.Read(tmp)
	if err != nil {
		return err
	}
	t.buf = append(t.buf, tmp[:n]...)
	return nil
}
