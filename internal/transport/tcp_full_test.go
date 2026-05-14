package transport

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"testing"
)

func TestTCPFullSendRecv(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPFull(conn)

	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if err := tr.Send(bytes.NewBuffer(payload)); err != nil {
		t.Fatal(err)
	}

	written := conn.Written()
	if len(written) != len(payload)+12 {
		t.Fatalf("expected %d bytes, got %d", len(payload)+12, len(written))
	}

	packetLen := binary.LittleEndian.Uint32(written[:4])
	if int(packetLen) != len(payload)+12 {
		t.Fatalf("packet_len mismatch: got %d, want %d", packetLen, len(payload)+12)
	}

	checksum := binary.LittleEndian.Uint32(written[len(written)-4:])
	computed := crc32.ChecksumIEEE(written[:len(written)-4])
	if checksum != computed {
		t.Fatalf("CRC32 mismatch: got %x, computed %x", checksum, computed)
	}

	seqNo := binary.LittleEndian.Uint32(written[4:8])
	if seqNo != 0 {
		t.Fatalf("expected seq_no 0, got %d", seqNo)
	}

	conn2 := NewMockConn()
	tr.conn = conn2
	conn2.FeedRead(written)

	data, err := tr.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("recv mismatch: got %x, want %x", data, payload)
	}
}

func TestTCPFullSeqNoIncrement(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPFull(conn)

	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	payload := []byte{0x01}
	tr.Send(bytes.NewBuffer(payload))

	conn2 := NewMockConn()
	tr.conn = conn2
	tr.Send(bytes.NewBuffer(payload))

	written := conn2.Written()
	seqNo := binary.LittleEndian.Uint32(written[4:8])
	if seqNo != 1 {
		t.Fatalf("expected seq_no 1, got %d", seqNo)
	}
}

func TestTCPFullBadCRC(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPFull(conn)

	packet := []byte{
		0x0C, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x01, 0x02, 0x03, 0x04,
		0xFF, 0xFF, 0xFF, 0xFF,
	}
	conn.FeedRead(packet)

	_, err := tr.Recv()
	if err == nil {
		t.Fatal("expected error for bad CRC")
	}
}

func TestTCPFullRecvEOF(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPFull(conn)

	conn.Close()
	_, err := tr.Recv()
	if err == nil {
		t.Fatal("expected error on closed conn")
	}
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}
