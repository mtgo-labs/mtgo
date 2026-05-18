package tg

import (
	"bytes"
	"testing"
)

func TestInputPeerUser_RoundTrip(t *testing.T) {
	original := &InputPeerUser{UserID: 12345, AccessHash: 67890}

	var buf bytes.Buffer
	if err := original.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	obj, err := ReadTLObject(r)
	if err != nil {
		t.Fatal(err)
	}
	decoded := obj.(*InputPeerUser)
	if decoded.UserID != 12345 || decoded.AccessHash != 67890 {
		t.Fatalf("mismatch: %+v", decoded)
	}
}

func TestInputPeerUser_ConstructorID(t *testing.T) {
	obj := &InputPeerUser{UserID: 1, AccessHash: 2}
	if obj.ConstructorID() != 0xDDE8A54C {
		t.Fatalf("expected 0xDDE8A54C, got 0x%08X", obj.ConstructorID())
	}
}

func TestInputPeerChat_RoundTrip(t *testing.T) {
	original := &InputPeerChat{ChatID: 99999}

	var buf bytes.Buffer
	if err := original.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	obj, err := ReadTLObject(r)
	if err != nil {
		t.Fatal(err)
	}
	decoded := obj.(*InputPeerChat)
	if decoded.ChatID != 99999 {
		t.Fatalf("mismatch: %+v", decoded)
	}
}

func TestInputPeerEmpty_RoundTrip(t *testing.T) {
	original := &InputPeerEmpty{}

	var buf bytes.Buffer
	if err := original.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	decoded, err := DecodeInputPeerEmpty(r)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.ConstructorID() != InputPeerEmptyTypeID {
		t.Fatalf("mismatch: 0x%08X", decoded.ConstructorID())
	}
}

func TestBool_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	WriteBool(&buf, true)
	if !ReadBool(bytes.NewReader(buf.Bytes())) {
		t.Fatal("expected true")
	}

	buf.Reset()
	WriteBool(&buf, false)
	if ReadBool(bytes.NewReader(buf.Bytes())) {
		t.Fatal("expected false")
	}
}

func TestRegistry_ContainsInputPeerUser(t *testing.T) {
	constructor, ok := Registry[0xDDE8A54C]
	if !ok {
		t.Fatal("InputPeerUser not in registry")
	}

	var buf bytes.Buffer
	original := &InputPeerUser{UserID: 111, AccessHash: 222}
	if err := original.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	obj, err := ReadTLObject(r)
	if err != nil {
		t.Fatal(err)
	}

	decoded := obj.(*InputPeerUser)
	if decoded.UserID != 111 || decoded.AccessHash != 222 {
		t.Fatalf("registry decode mismatch: %+v", decoded)
	}
	_ = constructor
}

func TestRegistry_ContainsInputPeerChat(t *testing.T) {
	_, ok := Registry[0x35A95CB9]
	if !ok {
		t.Fatal("InputPeerChat not in registry")
	}

	var buf bytes.Buffer
	original := &InputPeerChat{ChatID: 42}
	if err := original.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	obj, err := ReadTLObject(r)
	if err != nil {
		t.Fatal(err)
	}

	decoded := obj.(*InputPeerChat)
	if decoded.ChatID != 42 {
		t.Fatalf("registry decode mismatch: %+v", decoded)
	}
}

func TestUserEmpty_RoundTrip(t *testing.T) {
	original := &UserEmpty{ID: 777}

	var buf bytes.Buffer
	if err := original.Encode(&buf); err != nil {
		t.Fatal(err)
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)
	obj, err := ReadTLObject(r)
	if err != nil {
		t.Fatal(err)
	}
	decoded := obj.(*UserEmpty)
	if decoded.ID != 777 {
		t.Fatalf("mismatch: got %d", decoded.ID)
	}
}

func TestUser_Interface(t *testing.T) {
	var u UserClass = &UserEmpty{ID: 1}
	if u.ConstructorID() != UserEmptyTypeID {
		t.Fatalf("expected 0x%08X, got 0x%08X", UserEmptyTypeID, u.ConstructorID())
	}

	u2 := &User{ID: 2, Self: true}
	var _ UserClass = u2
	if u2.ID != 2 {
		t.Fatal("expected ID=2")
	}
	if !u2.Self {
		t.Fatal("expected Self=true")
	}
}

func TestStringPrimitives_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	WriteString(&buf, "hello world")
	s := ReadString(bytes.NewReader(buf.Bytes()))
	if s != "hello world" {
		t.Fatalf("mismatch: got %q", s)
	}
}

func TestBytesPrimitives_RoundTrip(t *testing.T) {
	original := []byte{0x01, 0x02, 0x03, 0x04}
	var buf bytes.Buffer
	WriteBytes(&buf, original)
	b := ReadBytes(bytes.NewReader(buf.Bytes()))
	if !bytes.Equal(b, original) {
		t.Fatalf("mismatch: got %x", b)
	}
}
