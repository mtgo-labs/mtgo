package tg

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestInt(t *testing.T) {
	var buf bytes.Buffer
	WriteInt(&buf, 42)
	WriteInt(&buf, uint32(math.MaxInt32)+1)
	WriteInt(&buf, math.MaxInt32)

	r := bytes.NewReader(buf.Bytes())
	v1 := ReadInt(r)
	v2 := ReadInt(r)
	v3 := ReadInt(r)

	if v1 != 42 {
		t.Fatalf("expected 42, got %d", v1)
	}
	if v2 != uint32(math.MaxInt32)+1 {
		t.Fatalf("expected -1, got %d", v2)
	}
	if v3 != math.MaxInt32 {
		t.Fatalf("expected %d, got %d", math.MaxInt32, v3)
	}
}

func TestLong(t *testing.T) {
	var buf bytes.Buffer
	WriteLong(&buf, 123456789012345)
	WriteLong(&buf, -987654321098765)

	r := bytes.NewReader(buf.Bytes())
	v1 := ReadLong(r)
	v2 := ReadLong(r)

	if v1 != 123456789012345 {
		t.Fatalf("expected 123456789012345, got %d", v1)
	}
	if v2 != -987654321098765 {
		t.Fatalf("expected -987654321098765, got %d", v2)
	}
}

func TestInt128(t *testing.T) {
	var buf bytes.Buffer
	val := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}
	WriteInt128(&buf, val)

	got := ReadInt128(bytes.NewReader(buf.Bytes()))
	if got != val {
		t.Fatalf("expected %x, got %x", val, got)
	}
}

func TestInt256(t *testing.T) {
	var buf bytes.Buffer
	val := [32]byte{}
	for i := range val {
		val[i] = byte(i)
	}
	WriteInt256(&buf, val)

	got := ReadInt256(bytes.NewReader(buf.Bytes()))
	if got != val {
		t.Fatalf("expected %x, got %x", val, got)
	}
}

func TestDouble(t *testing.T) {
	var buf bytes.Buffer
	WriteDouble(&buf, 3.14159)

	got := ReadDouble(bytes.NewReader(buf.Bytes()))
	if math.Abs(got-3.14159) > 1e-10 {
		t.Fatalf("expected 3.14159, got %f", got)
	}
}

func TestString(t *testing.T) {
	var buf bytes.Buffer
	WriteString(&buf, "hello world")
	WriteString(&buf, "")
	WriteString(&buf, "こんにちは")

	r := bytes.NewReader(buf.Bytes())
	s1 := ReadString(r)
	s2 := ReadString(r)
	s3 := ReadString(r)

	if s1 != "hello world" {
		t.Fatalf("expected 'hello world', got %q", s1)
	}
	if s2 != "" {
		t.Fatalf("expected empty, got %q", s2)
	}
	if s3 != "こんにちは" {
		t.Fatalf("expected 'こんにちは', got %q", s3)
	}
}

func TestBytes(t *testing.T) {
	var buf bytes.Buffer
	WriteBytes(&buf, []byte{0x00, 0x01, 0x02})
	WriteBytes(&buf, nil)
	WriteBytes(&buf, []byte{0xFF})

	r := bytes.NewReader(buf.Bytes())
	b1 := ReadBytes(r)
	b2 := ReadBytes(r)
	b3 := ReadBytes(r)

	if !bytes.Equal(b1, []byte{0x00, 0x01, 0x02}) {
		t.Fatalf("expected [0,1,2], got %x", b1)
	}
	if b2 != nil {
		t.Fatalf("expected nil, got %x", b2)
	}
	if !bytes.Equal(b3, []byte{0xFF}) {
		t.Fatalf("expected [FF], got %x", b3)
	}
}

func TestBytesLongFormPadding(t *testing.T) {
	long := bytes.Repeat([]byte{0xAB}, 256)

	var buf bytes.Buffer
	WriteBytes(&buf, long)
	WriteInt(&buf, 0x11223344)

	data := buf.Bytes()
	if len(data) != 264 {
		t.Fatalf("expected 264 bytes, got %d", len(data))
	}
	if !bytes.Equal(data[:4], []byte{254, 0, 1, 0}) {
		t.Fatalf("unexpected long-form bytes header: %x", data[:4])
	}

	r := bytes.NewReader(data)
	got := ReadBytes(r)
	if !bytes.Equal(got, long) {
		t.Fatalf("expected long bytes payload, got %x", got)
	}
	if got := ReadInt(r); got != 0x11223344 {
		t.Fatalf("expected following int 0x11223344, got %#x", got)
	}
}

func TestBool(t *testing.T) {
	var buf bytes.Buffer
	WriteBool(&buf, true)
	WriteBool(&buf, false)

	r := bytes.NewReader(buf.Bytes())
	b1 := ReadBool(r)
	b2 := ReadBool(r)

	if !b1 {
		t.Fatal("expected true")
	}
	if b2 {
		t.Fatal("expected false")
	}
}

func TestBoolConstructorIDs(t *testing.T) {
	if BoolTrueID != 0x997275B5 {
		t.Fatalf("BoolTrueID wrong: %x", BoolTrueID)
	}
	if BoolFalseID != 0xBC799737 {
		t.Fatalf("BoolFalseID wrong: %x", BoolFalseID)
	}
}

func TestVectorInt(t *testing.T) {
	var buf bytes.Buffer
	WriteVectorInt(&buf, []int32{1, 2, 3})

	got := ReadVectorInt(bytes.NewReader(buf.Bytes()))
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("expected [1,2,3], got %v", got)
	}
}

func TestVectorLong(t *testing.T) {
	var buf bytes.Buffer
	WriteVectorLong(&buf, []int64{10, 20})

	got := ReadVectorLong(bytes.NewReader(buf.Bytes()))
	if len(got) != 2 || got[0] != 10 || got[1] != 20 {
		t.Fatalf("expected [10,20], got %v", got)
	}
}

func TestVectorLongWireFormat(t *testing.T) {
	var buf bytes.Buffer
	WriteVectorLong(&buf, []int64{10, 20})
	data := buf.Bytes()

	if len(data) < 8 {
		t.Fatal("too short")
	}
	id := binary.LittleEndian.Uint32(data[0:4])
	if id != 0x1cb5c415 {
		t.Fatalf("expected vector bare ID 0x1cb5c415, got %x", id)
	}
	count := binary.LittleEndian.Uint32(data[4:8])
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}

func TestVectorString(t *testing.T) {
	var buf bytes.Buffer
	WriteVectorString(&buf, []string{"a", "bc"})

	got := ReadVectorString(bytes.NewReader(buf.Bytes()))
	if len(got) != 2 || got[0] != "a" || got[1] != "bc" {
		t.Fatalf("expected [a bc], got %v", got)
	}
}
