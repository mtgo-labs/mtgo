package transport

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestIntermediateNoHeaderConnect(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediateNoHeader(conn)

	if err := tr.Connect(); err != nil {
		t.Fatalf("Connect returned unexpected error: %v", err)
	}
}

func TestIntermediateNoHeaderSend(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediateNoHeader(conn)

	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if err := tr.Send(bytes.NewBuffer(payload)); err != nil {
		t.Fatal(err)
	}

	written := conn.Written()
	if len(written) != 4+len(payload) {
		t.Fatalf("expected %d bytes, got %d", 4+len(payload), len(written))
	}

	length := binary.LittleEndian.Uint32(written[:4])
	if int(length) != len(payload) {
		t.Fatalf("length prefix mismatch: got %d, want %d", length, len(payload))
	}
	if !bytes.Equal(written[4:], payload) {
		t.Fatalf("payload mismatch: got %x, want %x", written[4:], payload)
	}
}

func TestIntermediateNoHeaderRecv(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediateNoHeader(conn)

	payload := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(payload)))
	conn.FeedRead(append(lenBytes, payload...))

	data, err := tr.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("recv mismatch: got %x, want %x", data, payload)
	}
}

func TestIntermediateNoHeaderSendRecvRoundtrip(t *testing.T) {
	sendConn := NewMockConn()
	recvConn := NewMockConn()

	sendTr := NewTCPIntermediateNoHeader(sendConn)
	recvTr := NewTCPIntermediateNoHeader(recvConn)

	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE}
	if err := sendTr.Send(bytes.NewBuffer(payload)); err != nil {
		t.Fatal(err)
	}

	recvConn.FeedRead(sendConn.Written())

	data, err := recvTr.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("roundtrip mismatch: got %x, want %x", data, payload)
	}
}

func TestIntermediateNoHeaderRecvPartialReads(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediateNoHeader(conn)

	payload := []byte{0x11, 0x22, 0x33}
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(payload)))

	conn.FeedRead(lenBytes[:2])
	conn.FeedRead(append(lenBytes[2:], payload[:1]...))
	conn.FeedRead(payload[1:])

	data, err := tr.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("partial read mismatch: got %x, want %x", data, payload)
	}
}

func TestIntermediateNoHeaderRecvEOF(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediateNoHeader(conn)

	conn.Close()
	_, err := tr.Recv()
	if err == nil {
		t.Fatal("expected error on closed conn")
	}
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestIntermediateNoHeaderRecvError(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediateNoHeader(conn)

	testErr := errors.New("test read error")
	conn.SetReadError(testErr)

	_, err := tr.Recv()
	if err == nil {
		t.Fatal("expected error")
	}
	if err != testErr {
		t.Fatalf("expected testErr, got %v", err)
	}
}

func TestIntermediateNoHeaderRecvPayloadTooLarge(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediateNoHeader(conn)

	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(MaxPayloadLen+1))
	conn.FeedRead(lenBytes)

	_, err := tr.Recv()
	if err == nil {
		t.Fatal("expected error for oversized payload")
	}
	if err != ErrPayloadTooLarge {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}
}

func TestIntermediateNoHeaderRecvMultipleMessages(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediateNoHeader(conn)

	p1 := []byte{0x01, 0x02}
	p2 := []byte{0x03, 0x04, 0x05}

	var buf []byte
	len1 := make([]byte, 4)
	binary.LittleEndian.PutUint32(len1, uint32(len(p1)))
	buf = append(buf, len1...)
	buf = append(buf, p1...)

	len2 := make([]byte, 4)
	binary.LittleEndian.PutUint32(len2, uint32(len(p2)))
	buf = append(buf, len2...)
	buf = append(buf, p2...)

	conn.FeedRead(buf)

	data1, err := tr.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data1, p1) {
		t.Fatalf("first message mismatch: got %x, want %x", data1, p1)
	}

	data2, err := tr.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data2, p2) {
		t.Fatalf("second message mismatch: got %x, want %x", data2, p2)
	}
}

func TestIntermediateNoHeaderRecvPartialMidPayload(t *testing.T) {
	conn := NewMockConn()
	tr := NewTCPIntermediateNoHeader(conn)

	payload := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(payload)))

	conn.FeedRead(lenBytes)
	conn.FeedRead(payload[:2])

	_, err := tr.Recv()
	if err == nil {
		t.Fatal("expected error: payload incomplete and connection returns EOF")
	}
}
