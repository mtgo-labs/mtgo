package transport

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestTCPIntermediateSendRecv(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediate(conn)

	conn.FeedRead([]byte{0xee, 0xee, 0xee, 0xee})
	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	written := conn.Written()
	if !bytes.Equal(written[:4], []byte{0xee, 0xee, 0xee, 0xee}) {
		t.Fatalf("expected 0xEE*4 marker, got %x", written[:4])
	}

	conn2 := NewMockConn()
	tr.conn = conn2

	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if err := tr.Send(bytes.NewBuffer(payload)); err != nil {
		t.Fatal(err)
	}

	written = conn2.Written()
	if len(written) != 4+len(payload) {
		t.Fatalf("expected %d bytes, got %d", 4+len(payload), len(written))
	}
	length := binary.LittleEndian.Uint32(written[:4])
	if int(length) != len(payload) {
		t.Fatalf("length mismatch: got %d, want %d", length, len(payload))
	}
	if !bytes.Equal(written[4:], payload) {
		t.Fatal("payload mismatch")
	}
}

func TestTCPIntermediateRecv(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediate(conn)

	conn.FeedRead([]byte{0xee, 0xee, 0xee, 0xee})
	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	conn2 := NewMockConn()
	tr.conn = conn2

	payload := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(payload)))
	conn2.FeedRead(append(lenBytes, payload...))

	data, err := tr.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("recv mismatch: got %x, want %x", data, payload)
	}
}

func TestTCPIntermediateRecvEOF(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediate(conn)

	conn.FeedRead([]byte{0xee, 0xee, 0xee, 0xee})
	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	conn.Close()
	_, err := tr.Recv()
	if err == nil {
		t.Fatal("expected error on closed conn")
	}
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}
