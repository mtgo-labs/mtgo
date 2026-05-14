package transport

import (
	"testing"
)

func TestTCPAbridgedOSendRecv(t *testing.T) {
	conn := NewMockConn()
	inner := NewTCPAbridged(conn)
	tr := NewTCPObfuscated(inner, 0xEF)

	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	written := conn.Written()
	if len(written) != 64 {
		t.Fatalf("expected 64-byte nonce, got %d", len(written))
	}

	var _ Transport = tr
}
