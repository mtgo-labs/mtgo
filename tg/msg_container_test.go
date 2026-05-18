package tg

import (
	"bytes"
	"testing"
)

func TestMsgContainer(t *testing.T) {
	inner := &mockTLObject{data: []byte{0x01}}
	msg1 := &MTProtoMessage{MsgID: 1, SeqNo: 0, Body: inner}
	msg2 := &MTProtoMessage{MsgID: 2, SeqNo: 0, Body: inner}

	container := &MsgContainer{Messages: []*MTProtoMessage{msg1, msg2}}

	var buf bytes.Buffer
	if err := container.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	decoded, err := DecodeMsgContainer(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(decoded.Messages))
	}
	if decoded.Messages[0].MsgID != 1 || decoded.Messages[1].MsgID != 2 {
		t.Fatal("wrong message IDs")
	}
}

func TestMsgContainer_ConstructorID(t *testing.T) {
	c := &MsgContainer{}
	if c.ConstructorID() != 0x73F1F8DC {
		t.Fatalf("expected 0x73F1F8DC, got 0x%x", c.ConstructorID())
	}
}
