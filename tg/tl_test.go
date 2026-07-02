package tg

import (
	"bytes"
	"io"
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
	Registry[0xDEADBEEF] = func(r io.Reader) (TLObject, error) {
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

func TestUnknownConstructorError_Getters(t *testing.T) {
	e := &UnknownConstructorError{ID: 0x12345678}
	if e.ErrorCode() != 0 {
		t.Errorf("ErrorCode() = %d, want 0", e.ErrorCode())
	}
	if e.ErrorType() != "UNKNOWN_CONSTRUCTOR" {
		t.Errorf("ErrorType() = %q, want %q", e.ErrorType(), "UNKNOWN_CONSTRUCTOR")
	}
	if e.ErrorArg() != 0x12345678 {
		t.Errorf("ErrorArg() = %d, want %d", e.ErrorArg(), 0x12345678)
	}
}
