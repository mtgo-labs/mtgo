package transport

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
)

const maxPayloadLen = 1 << 24 // 16 MiB

// TCPIntermediate implements the MTProto "intermediate" transport which
// prefixes each payload with a simple 4-byte little-endian length field.
// It is simpler than full (no CRC) and more straightforward than abridged
// (fixed-size length prefix).
type TCPIntermediate struct {
	conn net.Conn
}

// NewTCPIntermediate returns a new TCPIntermediate transport wrapping conn.
func NewTCPIntermediate(conn net.Conn) *TCPIntermediate {
	return &TCPIntermediate{conn: conn}
}

// Connect sends the 0xEEEEEEEE protocol marker to the peer to negotiate the
// intermediate transport mode.
func (t *TCPIntermediate) Connect() error {
	_, err := t.conn.Write([]byte{0xee, 0xee, 0xee, 0xee})
	return err
}

// Send writes buf to the connection with a 4-byte little-endian length prefix.
func (t *TCPIntermediate) Send(buf *bytes.Buffer) error {
	data := buf.Bytes()

	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], uint32(len(data)))

	if _, err := t.conn.Write(header[:]); err != nil {
		return err
	}
	_, err := t.conn.Write(data)
	return err
}

// Recv reads the next intermediate-transport framed message from the
// connection. It reads a 4-byte length prefix followed by the payload bytes.
func (t *TCPIntermediate) Recv() ([]byte, error) {
	lenBytes := make([]byte, 4)
	if _, err := io.ReadFull(t.conn, lenBytes); err != nil {
		return nil, err
	}

	length := binary.LittleEndian.Uint32(lenBytes)
	if length > maxPayloadLen {
		return nil, ErrPayloadTooLarge
	}

	data := make([]byte, length)
	if length == 0 {
		return data, nil
	}
	if _, err := io.ReadFull(t.conn, data); err != nil {
		return nil, err
	}
	return data, nil
}
