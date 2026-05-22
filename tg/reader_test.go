package tg

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestVectorTooLargeErrorMessage(t *testing.T) {
	err := CheckVectorCount(maxVectorElements + 1)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "count=") {
		t.Fatalf("error should contain count: %s", msg)
	}
	if !strings.Contains(msg, "max=") {
		t.Fatalf("error should contain max: %s", msg)
	}
	vtle, ok := err.(*vectorTooLargeError)
	if !ok {
		t.Fatalf("expected *vectorTooLargeError, got %T", err)
	}
	if vtle.count != maxVectorElements+1 {
		t.Fatalf("expected count %d, got %d", maxVectorElements+1, vtle.count)
	}
}

func TestReader_ReadVectorInt_BufferBoundsShort(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(vectorBareID))
	binary.Write(&buf, binary.LittleEndian, uint32(5))

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	_, err := r.ReadVectorInt()
	if err == nil {
		t.Fatal("expected error for short buffer")
	}
	if err.Error() != "unexpected EOF" {
		t.Fatalf("expected unexpected EOF, got %v", err)
	}
}

func TestReader_ReadVectorLong_BufferBoundsShort(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(vectorBareID))
	binary.Write(&buf, binary.LittleEndian, uint32(3))

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	_, err := r.ReadVectorLong()
	if err == nil {
		t.Fatal("expected error for short buffer")
	}
	if err.Error() != "unexpected EOF" {
		t.Fatalf("expected unexpected EOF, got %v", err)
	}
}

func TestReader_ReadVectorInt_BufferBoundsExact(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(vectorBareID))
	binary.Write(&buf, binary.LittleEndian, uint32(3))
	for i := 0; i < 3; i++ {
		binary.Write(&buf, binary.LittleEndian, int32(i+10))
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	result, err := r.ReadVectorInt()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result))
	}
	if result[0] != 10 || result[1] != 11 || result[2] != 12 {
		t.Fatalf("expected [10 11 12], got %v", result)
	}
}

func TestReader_ReadVectorLong_BufferBoundsExact(t *testing.T) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(vectorBareID))
	binary.Write(&buf, binary.LittleEndian, uint32(2))
	for i := 0; i < 2; i++ {
		binary.Write(&buf, binary.LittleEndian, int64(i+100))
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	result, err := r.ReadVectorLong()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}
	if result[0] != 100 || result[1] != 101 {
		t.Fatalf("expected [100 101], got %v", result)
	}
}
