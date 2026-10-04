package tg

import (
	"bytes"
	"errors"
	"strings"
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

func TestEncodeTLObjectNil(t *testing.T) {
	var buf bytes.Buffer
	err := EncodeTLObject(&buf, nil)
	if !errors.Is(err, ErrNilTLObject) {
		t.Fatalf("want ErrNilTLObject, got %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("nil object should not write bytes, wrote %d", buf.Len())
	}
}

// TestEncodeNilRequiredObjectField is a regression test for the reported
// SIGSEGV: encoding a struct with a required object field left nil
// (PageCaption.Credit) must return a typed error naming the field, not
// panic with a nil-pointer dereference.
func TestEncodeNilRequiredObjectField(t *testing.T) {
	v := &PageCaption{Text: &TextPlain{Text: "captured"}} // Credit nil
	var buf bytes.Buffer
	err := v.Encode(&buf)
	if !errors.Is(err, ErrNilTLObject) {
		t.Fatalf("want ErrNilTLObject, got %v", err)
	}
	if !strings.Contains(err.Error(), "credit") {
		t.Fatalf("error should name the offending field, got %v", err)
	}
}

// TestEncodeNilObjectFieldPointer covers nil struct-pointer fields, e.g. a
// PageBlockPhoto without a caption.
func TestEncodeNilObjectFieldPointer(t *testing.T) {
	v := &PageBlockPhoto{PhotoID: 1} // Caption nil
	var buf bytes.Buffer
	err := v.Encode(&buf)
	if !errors.Is(err, ErrNilTLObject) {
		t.Fatalf("want ErrNilTLObject, got %v", err)
	}
	if !strings.Contains(err.Error(), "caption") {
		t.Fatalf("error should name the offending field, got %v", err)
	}
}

// TestEncodeTypedNilReceiver covers a typed nil pointer receiver, which
// cannot be caught by EncodeTLObject's nil-interface check.
func TestEncodeTypedNilReceiver(t *testing.T) {
	var v *PageCaption
	var buf bytes.Buffer
	if err := v.Encode(&buf); !errors.Is(err, ErrNilTLObject) {
		t.Fatalf("want ErrNilTLObject, got %v", err)
	}
}

// TestEncodeNilVectorElement covers nil elements inside object vectors.
func TestEncodeNilVectorElement(t *testing.T) {
	v := &TextConcat{Texts: []RichTextClass{&TextPlain{Text: "a"}, nil}}
	var buf bytes.Buffer
	err := v.Encode(&buf)
	if !errors.Is(err, ErrNilTLObject) {
		t.Fatalf("want ErrNilTLObject, got %v", err)
	}
	if !strings.Contains(err.Error(), "texts") {
		t.Fatalf("error should name the offending field, got %v", err)
	}
}

// TestPageCaptionWireFormat pins the on-wire encoding of pageCaption
// (text followed by a required credit object) so the schema cannot drift
// silently.
func TestPageCaptionWireFormat(t *testing.T) {
	v := &PageCaption{Text: &TextPlain{Text: "t"}, Credit: &TextEmpty{}}
	var buf bytes.Buffer
	if err := v.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	want := []byte{
		0x57, 0x76, 0x74, 0x6f, // pageCaption#6f747657 (LE)
		0xe0, 0x94, 0x46, 0x74, // textPlain#744694e0 (LE)
		0x01, 0x74, 0x00, 0x00, // string "t": len 1, 't', pad to 4
		0x4f, 0x82, 0x3d, 0xdc, // textEmpty#dc3d824f (LE)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("wire format mismatch:\n got %x\nwant %x", buf.Bytes(), want)
	}

	got, err := ReadTLObject(NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	pc, ok := got.(*PageCaption)
	if !ok {
		t.Fatalf("round-trip: got %T, want *PageCaption", got)
	}
	if _, ok := pc.Credit.(*TextEmpty); !ok {
		t.Fatalf("decode round-trip: credit = %T, want *TextEmpty", pc.Credit)
	}
}
