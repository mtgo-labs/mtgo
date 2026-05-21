package transport

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"sync/atomic"
)

type TCPFull struct {
	conn    net.Conn
	seqNo   uint32
	readBuf []byte
}

// NewTCPFull returns a new TCPFull transport wrapping conn.
func NewTCPFull(conn net.Conn) *TCPFull {
	return &TCPFull{conn: conn}
}

// Connect resets the internal sequence counter. It does not perform any
// network I/O because the underlying connection is already established.
func (t *TCPFull) Connect() error {
	atomic.StoreUint32(&t.seqNo, 0)
	return nil
}

// Send writes buf to the connection with a full transport header consisting of
// a 4-byte length, 4-byte sequence number, the payload, and a 4-byte CRC32
// checksum. It increments the sequence number after each successful write.
func (t *TCPFull) Send(buf *bytes.Buffer) error {
	data := buf.Bytes()

	packet := make([]byte, 4+4+len(data)+4)
	binary.LittleEndian.PutUint32(packet[0:4], uint32(len(data)+12))
	binary.LittleEndian.PutUint32(packet[4:8], atomic.LoadUint32(&t.seqNo))
	copy(packet[8:8+len(data)], data)
	binary.LittleEndian.PutUint32(packet[8+len(data):], crc32.ChecksumIEEE(packet[:8+len(data)]))

	atomic.AddUint32(&t.seqNo, 1)

	_, err := t.conn.Write(packet)
	return err
}

// Recv reads the next full-transport framed message from the connection. It
// verifies the CRC32 checksum and returns the payload bytes without the
// header and checksum. Returns [ErrCRC32Mismatch] on checksum failure.
func (t *TCPFull) Recv() ([]byte, error) {
	var lenBytes [4]byte
	if _, err := io.ReadFull(t.conn, lenBytes[:]); err != nil {
		return nil, err
	}

	packetLen := binary.LittleEndian.Uint32(lenBytes[:])
	if packetLen < 12 {
		return nil, fmt.Errorf("tcp_full: packet too short: %d", packetLen)
	}
	if packetLen-4 > uint32(MaxPayloadLen) {
		return nil, ErrPayloadTooLarge
	}

	// packetLen-4 is guaranteed <= MaxPayloadLen (16 MiB), which fits in int
	// on all supported platforms.
	restLen := int(packetLen - 4)
	if cap(t.readBuf) < restLen {
		t.readBuf = make([]byte, restLen)
	}
	rest := t.readBuf[:restLen]
	if _, err := io.ReadFull(t.conn, rest); err != nil {
		return nil, err
	}

	h := crc32.NewIEEE()
	h.Write(lenBytes[:])
	h.Write(rest[:len(rest)-4])
	computed := h.Sum32()

	checksum := binary.LittleEndian.Uint32(rest[len(rest)-4:])
	if checksum != computed {
		return nil, ErrCRC32Mismatch
	}

	return rest[4 : len(rest)-4], nil
}
