package transport

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"net"
)

type TCPIntermediateNoHeader struct {
	conn    net.Conn
	br      *bufio.Reader
	buf     []byte
	readBuf []byte
}

func NewTCPIntermediateNoHeader(conn net.Conn) *TCPIntermediateNoHeader {
	return &TCPIntermediateNoHeader{
		conn: conn,
		br:   bufio.NewReaderSize(conn, 1<<20),
	}
}

func (t *TCPIntermediateNoHeader) Connect() error {
	return nil
}

// Conn returns the underlying net.Conn.
func (t *TCPIntermediateNoHeader) Conn() net.Conn { return t.conn }

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
	if length > MaxPayloadLen {
		return nil, ErrPayloadTooLarge
	}

	needed := 4 + int(length)
	for len(t.buf) < needed {
		if err := t.fill(); err != nil {
			return nil, err
		}
	}

	if cap(t.readBuf) < int(length) {
		t.readBuf = make([]byte, length)
	}
	payload := t.readBuf[:length]
	copy(payload, t.buf[4:needed])

	remaining := copy(t.buf, t.buf[needed:])
	t.buf = t.buf[:remaining]

	return payload, nil
}

func (t *TCPIntermediateNoHeader) fill() error {
	if cap(t.readBuf) < 1<<20 {
		t.readBuf = make([]byte, 1<<20)
	}
	tmp := t.readBuf[:1<<20]
	n, err := t.br.Read(tmp)
	if err != nil {
		return err
	}
	t.buf = append(t.buf, tmp[:n]...)
	return nil
}
