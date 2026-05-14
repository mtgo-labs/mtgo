package transport

import (
	"bytes"
	"testing"
)

func TestTCPAbridgedSmallPayload(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPAbridged(conn)

	conn.FeedRead([]byte{0xef})
	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	written := conn.Written()
	if len(written) < 1 || written[0] != 0xEF {
		t.Fatalf("expected 0xEF marker, got %x", written)
	}

	payload := make([]byte, 16)
	for i := range payload {
		payload[i] = byte(i + 1)
	}
	buf := bytes.NewBuffer(payload)

	conn2 := NewMockConn()
	tr.conn = conn2

	if err := tr.Send(buf); err != nil {
		t.Fatal(err)
	}

	written = conn2.Written()
	if len(written) != 1+len(payload) {
		t.Fatalf("expected %d bytes, got %d", 1+len(payload), len(written))
	}
	if written[0] != 4 {
		t.Fatalf("expected length byte 4, got %d", written[0])
	}
	if !bytes.Equal(written[1:], payload) {
		t.Fatal("payload mismatch")
	}
}

func TestTCPAbridgedRecvSmall(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPAbridged(conn)

	conn.FeedRead([]byte{0xef})
	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	conn2 := NewMockConn()
	tr.conn = conn2

	payload := make([]byte, 16)
	for i := range payload {
		payload[i] = byte(i)
	}
	feed := append([]byte{4}, payload...)
	conn2.FeedRead(feed)

	data, err := tr.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("recv mismatch: got %x, want %x", data, payload)
	}
}

func TestTCPAbridgedLargePayload(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPAbridged(conn)

	conn.FeedRead([]byte{0xef})
	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	payload := make([]byte, 512)
	for i := range payload {
		payload[i] = byte(i)
	}
	buf := bytes.NewBuffer(payload)

	conn2 := NewMockConn()
	tr.conn = conn2

	if err := tr.Send(buf); err != nil {
		t.Fatal(err)
	}

	written := conn2.Written()
	if written[0] != 0x7f {
		t.Fatalf("expected 0x7f prefix for large payload, got %x", written[0])
	}
	length := int(written[1]) | int(written[2])<<8 | int(written[3])<<16
	if length != 128 {
		t.Fatalf("expected length 128, got %d", length)
	}
	if !bytes.Equal(written[4:], payload) {
		t.Fatal("payload mismatch for large payload")
	}
}

func TestTCPAbridgedRecvEOF(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPAbridged(conn)

	conn.FeedRead([]byte{0xef})
	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	conn.Close()
	_, err := tr.Recv()
	if err == nil {
		t.Fatal("expected error on closed conn")
	}
}
