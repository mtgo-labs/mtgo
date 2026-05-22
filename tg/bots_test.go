package tg

import (
	"bytes"
	"testing"
)

func TestInputKeyboardButtonRequestPeerSchema(t *testing.T) {
	button := &InputKeyboardButtonRequestPeer{
		UsernameRequested: true,
		Text:              "Select channel",
		ButtonID:          1,
		PeerType:          &RequestPeerTypeBroadcast{Creator: true},
		MaxQuantity:       1,
	}

	var buf bytes.Buffer
	if err := button.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	r := NewReader(buf.Bytes())
	defer ReleaseReader(r)

	id, err := r.ReadUint32()
	if err != nil {
		t.Fatalf("read constructor: %v", err)
	}
	if id != 0x02b78156 {
		t.Fatalf("constructor id = %#08x, want 0x02b78156", id)
	}

	flags, err := r.ReadUint32()
	if err != nil {
		t.Fatalf("read flags: %v", err)
	}
	if got, want := Fields(flags).Has(1), true; got != want {
		t.Fatalf("username_requested flag = %v, want %v", got, want)
	}
	if got := Fields(flags).Has(10); got {
		t.Fatalf("style flag = %v, want false", got)
	}
}
