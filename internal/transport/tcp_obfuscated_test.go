package transport

import (
	"bytes"
	"testing"
)

func TestObfuscatedNonceGeneration(t *testing.T) {
	conn := NewMockConn()
	inner := NewTCPIntermediate(conn)
	tr := NewTCPObfuscated(inner, 0xEE)

	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	written := conn.Written()
	if len(written) != 64 {
		t.Fatalf("expected 64-byte nonce, got %d", len(written))
	}

	if written[0] == 0xEF {
		t.Fatal("nonce[0] must not be 0xEF")
	}

	if bytes.Equal(written[4:8], []byte{0, 0, 0, 0}) {
		t.Fatal("nonce[4:8] must not be all zeros")
	}

	// After encryption, the marker at [56:60] will be encrypted
	// so we can't check it directly
}

func TestObfuscatedSendRecvRoundTrip(t *testing.T) {
	connA := NewMockConn()
	innerA := NewTCPIntermediate(connA)
	transportA := NewTCPObfuscated(innerA, 0xEE)

	if err := transportA.Connect(); err != nil {
		t.Fatal(err)
	}

	// Get the nonce that was sent (64 bytes)
	nonce := make([]byte, 64)
	writtenA := connA.Written()
	copy(nonce, writtenA[:64])

	// Create the reverse side using the same nonce
	connB := NewMockConn()
	innerB := NewTCPIntermediate(connB)
	transportB := NewTCPObfuscatedWithNonce(innerB, 0xEE, nonce, true)

	if err := transportB.Connect(); err != nil {
		t.Fatal(err)
	}

	payload := []byte("hello obfuscated world!")
	buf := bytes.NewBuffer(payload)

	if err := transportA.Send(buf); err != nil {
		t.Fatal(err)
	}

	// Get encrypted data after nonce — must re-read after Send()
	// because bytes.Buffer.Bytes() may be invalidated by growth
	allWritten := connA.Written()
	encryptedData := allWritten[64:]
	connB.FeedRead(encryptedData)

	data, err := transportB.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("round-trip failed:\n  want %s\n  got  %s", payload, data)
	}
}

func TestObfuscatedDeterministicNonce(t *testing.T) {
	nonce := make([]byte, 64)
	for i := range nonce {
		nonce[i] = byte(i + 1)
	}
	nonce[0] = 0x01
	nonce[56] = 0xEE
	nonce[57] = 0xEE
	nonce[58] = 0xEE
	nonce[59] = 0xEE

	conn := NewMockConn()
	inner := NewTCPIntermediate(conn)
	tr := NewTCPObfuscatedWithNonce(inner, 0xEE, nonce, false)

	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	payload := []byte("deterministic test")
	buf := bytes.NewBuffer(payload)

	if err := tr.Send(buf); err != nil {
		t.Fatal(err)
	}
	first := conn.Written()

	conn2 := NewMockConn()
	inner2 := NewTCPIntermediate(conn2)
	tr2 := NewTCPObfuscatedWithNonce(inner2, 0xEE, nonce, false)

	if err := tr2.Connect(); err != nil {
		t.Fatal(err)
	}

	buf = bytes.NewBuffer(payload)
	if err := tr2.Send(buf); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first, conn2.Written()) {
		t.Fatal("deterministic nonce should produce identical output")
	}
}
