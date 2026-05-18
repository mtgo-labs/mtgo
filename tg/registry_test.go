package tg

import (
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
	r := NewReader([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	defer ReleaseReader(r)
	_, err := ReadTLObject(r)
	if err == nil {
		t.Fatal("expected error for unknown constructor")
	}
}
