package transport

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestTCPPaddedIntermediateSend(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPPaddedIntermediate(conn)

	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	payload := bytes.Repeat([]byte{0x11}, 40)
	if err := tr.Send(bytes.NewBuffer(payload)); err != nil {
		t.Fatal(err)
	}

	written := conn.Written()
	if !bytes.Equal(written[:4], []byte{0xdd, 0xdd, 0xdd, 0xdd}) {
		t.Fatalf("marker = %x, want dddddddd", written[:4])
	}

	packet := written[4:]
	packetLen := int(binary.LittleEndian.Uint32(packet[:4]))
	if packetLen < len(payload) || packetLen > len(payload)+15 {
		t.Fatalf("packet length = %d, want payload length plus 0-15 padding", packetLen)
	}
	if len(packet[4:]) != packetLen {
		t.Fatalf("written packet payload = %d, want %d", len(packet[4:]), packetLen)
	}
	if !bytes.Equal(packet[4:4+len(payload)], payload) {
		t.Fatal("payload prefix mismatch")
	}
}

func TestTCPPaddedIntermediateRecvTrimsUnencryptedPadding(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPPaddedIntermediate(conn)

	payload := make([]byte, 24)
	binary.LittleEndian.PutUint32(payload[16:20], 4)
	copy(payload[20:], []byte{1, 2, 3, 4})
	padded := append(append([]byte(nil), payload...), bytes.Repeat([]byte{0xaa}, 7)...)

	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(padded)))
	conn.FeedRead(append(lenBytes, padded...))

	got, err := tr.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("recv = %x, want %x", got, payload)
	}
}

func TestTCPPaddedIntermediateRecvTrimsEncryptedPadding(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPPaddedIntermediate(conn)

	payload := bytes.Repeat([]byte{0x22}, 24+32)
	padded := append(append([]byte(nil), payload...), bytes.Repeat([]byte{0xbb}, 9)...)

	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(padded)))
	conn.FeedRead(append(lenBytes, padded...))

	got, err := tr.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("recv length = %d, want %d", len(got), len(payload))
	}
}
