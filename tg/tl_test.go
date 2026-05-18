package tg

import (
	"bytes"
	"testing"
)

type mockTLObject struct {
	data []byte
}

func (m *mockTLObject) Encode(b *bytes.Buffer) error {
	WriteInt(b, m.ConstructorID())
	_, err := b.Write(m.data)
	return err
}

func (m *mockTLObject) ConstructorID() uint32 {
	return 0xDEADBEEF
}

func init() {
	Registry[0xDEADBEEF] = func(r *Reader) (TLObject, error) {
		return &mockTLObject{}, nil
	}
}

func TestTLObject_Interface(t *testing.T) {
	var _ TLObject = &mockTLObject{}
}

func TestEncodeTLObject(t *testing.T) {
	obj := &mockTLObject{data: []byte{0x01, 0x02, 0x03}}
	var buf bytes.Buffer
	err := EncodeTLObject(&buf, obj)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte{0xEF, 0xBE, 0xAD, 0xDE, 0x01, 0x02, 0x03}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Fatalf("expected %x, got %x", expected, buf.Bytes())
	}
}
