package transport

import (
	"bytes"
	"io"
	"net"
)

// TCPAbridged implements the MTProto "abridged" transport which uses a
// variable-length prefix (1 or 4 bytes) to encode the payload length in
// 4-byte units, making it more compact than the full transport.
type TCPAbridged struct {
	conn net.Conn
}

// NewTCPAbridged returns a new TCPAbridged transport wrapping conn.
func NewTCPAbridged(conn net.Conn) *TCPAbridged {
	return &TCPAbridged{conn: conn}
}

// Connect sends the 0xEF protocol marker byte to the peer to negotiate the
// abridged transport mode.
func (t *TCPAbridged) Connect() error {
	_, err := t.conn.Write([]byte{0xef})
	return err
}

// Send writes buf to the connection with an abridged-style length prefix.
// The length is encoded in 4-byte units; payloads up to 504 bytes use a
// single-byte prefix while larger payloads use a 4-byte prefix starting
// with 0x7F.
func (t *TCPAbridged) Send(buf *bytes.Buffer) error {
	data := buf.Bytes()
	length := len(data) / 4

	if length <= 126 {
		if _, err := t.conn.Write([]byte{byte(length)}); err != nil {
			return err
		}
	} else {
		var header [4]byte
		header[0] = 0x7f
		header[1] = byte(length)
		header[2] = byte(length >> 8)
		header[3] = byte(length >> 16)
		if _, err := t.conn.Write(header[:]); err != nil {
			return err
		}
	}

	_, err := t.conn.Write(data)
	return err
}

// Recv reads the next abridged-transport framed message from the connection.
// It decodes the variable-length prefix and returns the payload bytes.
func (t *TCPAbridged) Recv() ([]byte, error) {
	var lengthByte [1]byte
	if _, err := io.ReadFull(t.conn, lengthByte[:]); err != nil {
		return nil, err
	}

	var length int
	if lengthByte[0] == 0x7f {
		var lenBytes [3]byte
		if _, err := io.ReadFull(t.conn, lenBytes[:]); err != nil {
			return nil, err
		}
		length = int(lenBytes[0]) | int(lenBytes[1])<<8 | int(lenBytes[2])<<16
	} else {
		length = int(lengthByte[0])
	}

	length *= 4
	data := make([]byte, length)
	if _, err := io.ReadFull(t.conn, data); err != nil {
		return nil, err
	}
	return data, nil
}
