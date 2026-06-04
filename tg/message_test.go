package tg

import (
	"bytes"
	"testing"
)

func TestMessage_EncodeDecode(t *testing.T) {
	body := &mockTLObject{data: []byte{0xAA, 0xBB}}
	msg := &MTProtoMessage{MsgID: 12345, SeqNo: 1, Body: body}

	var buf bytes.Buffer
	if err := msg.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	msg2, err := DecodeMTProtoMessage(r)
	if err != nil {
		t.Fatal(err)
	}
	if msg2.MsgID != 12345 {
		t.Fatalf("expected MsgID 12345, got %d", msg2.MsgID)
	}
	if msg2.SeqNo != 1 {
		t.Fatalf("expected SeqNo 1, got %d", msg2.SeqNo)
	}
}

func TestMessage_ConstructorID(t *testing.T) {
	msg := &MTProtoMessage{}
	if msg.ConstructorID() != 0x5BB8E511 {
		t.Fatalf("expected 0x5BB8E511, got 0x%x", msg.ConstructorID())
	}
}

func TestDecodeFutureSaltsBareVector(t *testing.T) {
	var buf bytes.Buffer
	WriteLong(&buf, 1)
	WriteInt(&buf, 2)
	WriteInt(&buf, 1)
	WriteInt(&buf, 1780476673)
	WriteInt(&buf, 1780480273)
	WriteLong(&buf, 0x1122334455667788)

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	got, err := DecodeFutureSalts(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Salts) != 1 {
		t.Fatalf("len(got.Salts) = %d, want 1", len(got.Salts))
	}
	if got.Salts[0].ValidSince != 1780476673 || got.Salts[0].ValidUntil != 1780480273 || got.Salts[0].Salt != 0x1122334455667788 {
		t.Fatalf("unexpected salt: %+v", got.Salts[0])
	}
}

func TestDecodeFutureSaltsRejectsHugeVector(t *testing.T) {
	var buf bytes.Buffer
	WriteLong(&buf, 1)
	WriteInt(&buf, 2)
	WriteInt(&buf, maxVectorElements+1)

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	_, err := DecodeFutureSalts(r)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*vectorTooLargeError); !ok {
		t.Fatalf("expected vectorTooLargeError, got %T: %v", err, err)
	}
}
