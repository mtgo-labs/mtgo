package tg

import (
	"bytes"
	"testing"
)

func TestGzipPacked(t *testing.T) {
	inner := &mockTLObject{data: []byte{0xDE, 0xAD, 0xBE, 0xEF}}
	gz := &GzipPacked{Data: inner}

	var buf bytes.Buffer
	if err := gz.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	decoded, err := DecodeGzipPacked(r)
	if err != nil {
		t.Fatal(err)
	}
	innerObj, err := decoded.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if innerObj.ConstructorID() != 0xDEADBEEF {
		t.Fatalf("expected 0xDEADBEEF, got 0x%x", innerObj.ConstructorID())
	}
}

func TestGzipPacked_ConstructorID(t *testing.T) {
	gz := &GzipPacked{}
	if gz.ConstructorID() != 0x3072CFA1 {
		t.Fatalf("expected 0x3072CFA1, got 0x%x", gz.ConstructorID())
	}
}
