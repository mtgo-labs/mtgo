package transport

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
)

type TCPPaddedIntermediate struct {
	conn    net.Conn
	readBuf []byte
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
	padLen := int(n[0]) % 16
	var padding [15]byte
	if padLen > 0 {
		if _, err := rand.Read(padding[:padLen]); err != nil {
			return err
		}
	}

	packetLen := len(data) + padLen
	var lenBytes [4]byte
	binary.LittleEndian.PutUint32(lenBytes[:], uint32(packetLen))

	if _, err := t.conn.Write(lenBytes[:]); err != nil {
		return err
	}
	if _, err := t.conn.Write(data); err != nil {
		return err
	}
	if padLen > 0 {
		if _, err := t.conn.Write(padding[:padLen]); err != nil {
			return err
		}
	}
	return nil
}

func (t *TCPPaddedIntermediate) Recv() ([]byte, error) {
	var lenBytes [4]byte
	if _, err := io.ReadFull(t.conn, lenBytes[:]); err != nil {
		return nil, err
	}

	length := binary.LittleEndian.Uint32(lenBytes[:])
	if length > MaxPayloadLen+15 {
		return nil, ErrPayloadTooLarge
	}

	if length == 0 {
		return nil, nil
	}

	if cap(t.readBuf) < int(length) {
		t.readBuf = make([]byte, length)
	}
	data := t.readBuf[:length]
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
