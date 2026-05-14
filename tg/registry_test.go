package tg

import (
	"bytes"
	"testing"
)

func TestRegistry(t *testing.T) {
	_, ok := Registry[MTProtoMessageID]
	if !ok {
		t.Fatal("MTProtoMessageID not in registry")
	}
	_, ok = Registry[MsgContainerID]
	if !ok {
		t.Fatal("MsgContainerID not in registry")
	}
}

func TestReadTLObject_Unknown(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	_, err := ReadTLObject(buf)
	if err == nil {
		t.Fatal("expected error for unknown constructor")
	}
}
