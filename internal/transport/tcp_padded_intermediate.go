package transport

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
)

// TCPPaddedIntermediate implements the MTProto padded intermediate transport.
// It uses intermediate framing, but each packet carries 0-15 bytes of random
// transport padding and is negotiated with the 0xDDDDDDDD marker.
type TCPPaddedIntermediate struct {
	conn net.Conn
}

func NewTCPPaddedIntermediate(conn net.Conn) *TCPPaddedIntermediate {
	return &TCPPaddedIntermediate{conn: conn}
}

func (t *TCPPaddedIntermediate) Connect() error {
	_, err := t.conn.Write([]byte{0xdd, 0xdd, 0xdd, 0xdd})
	return err
}

func (t *TCPPaddedIntermediate) Send(buf *bytes.Buffer) error {
	data := buf.Bytes()

	var n [1]byte
	if _, err := rand.Read(n[:]); err != nil {
		return err
	}
	padding := make([]byte, int(n[0])%16)
	if len(padding) > 0 {
		if _, err := rand.Read(padding); err != nil {
			return err
		}
	}

	packetLen := len(data) + len(padding)
	packet := make([]byte, 4+packetLen)
	binary.LittleEndian.PutUint32(packet[:4], uint32(packetLen))
	copy(packet[4:], data)
	copy(packet[4+len(data):], padding)

	_, err := t.conn.Write(packet)
	return err
}

func (t *TCPPaddedIntermediate) Recv() ([]byte, error) {
	lenBytes := make([]byte, 4)
	if _, err := io.ReadFull(t.conn, lenBytes); err != nil {
		return nil, err
	}

	length := binary.LittleEndian.Uint32(lenBytes)
	if length > maxPayloadLen+15 {
		return nil, ErrPayloadTooLarge
	}

	data := make([]byte, length)
	if length == 0 {
		return data, nil
	}
	if _, err := io.ReadFull(t.conn, data); err != nil {
		return nil, err
	}
	return trimPaddedIntermediatePayload(data), nil
}

var zeros8 = make([]byte, 8)

func trimPaddedIntermediatePayload(data []byte) []byte {
	if len(data) >= 20 && bytes.Equal(data[:8], zeros8) {
		msgLen := binary.LittleEndian.Uint32(data[16:20])
		end := 20 + int(msgLen)
		if end <= len(data) && len(data)-end <= 15 {
			return data[:end]
		}
	}

	if len(data) >= 24 {
		paddingLen := (len(data) - 24) % 16
		if paddingLen <= 15 {
			return data[:len(data)-paddingLen]
		}
	}

	return data
}
