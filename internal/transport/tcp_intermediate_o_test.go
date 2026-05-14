package transport

import (
	"testing"
)

func TestTCPIntermediateOInterface(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediateO(conn)

	var _ Transport = tr

	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	written := conn.Written()
	if len(written) != 64 {
		t.Fatalf("expected 64-byte nonce, got %d", len(written))
	}
}
