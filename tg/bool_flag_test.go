package tg

import (
	"bytes"
	"testing"
)

// TestPhoneEditGroupCallParticipant_BoolFlagFalse verifies that SetVideoPaused(false)
// correctly encodes boolFalse on the wire — the core bug that was fixed.
func TestPhoneEditGroupCallParticipant_BoolFlagFalse(t *testing.T) {
	call := &InputGroupCall{ID: 1, AccessHash: 2}
	participant := &InputPeerUser{UserID: 100, AccessHash: 200}

	t.Run("SetVideoPaused_false_sends_boolFalse", func(t *testing.T) {
		req := &PhoneEditGroupCallParticipantRequest{
			Call:        call,
			Participant: participant,
		}
		req.SetVideoPaused(false)

		var buf bytes.Buffer
		if err := req.Encode(&buf); err != nil {
			t.Fatal(err)
		}

		// Flag bit 4 must be set.
		if !req.Flags.Has(4) {
			t.Fatal("flag bit 4 (video_paused) should be set after SetVideoPaused(false)")
		}

		// The encoded data must contain boolFalse (0xbc799737) for video_paused.
		// Structure: constructorID(4) + flags(4) + call + participant + [muted?] + [volume?] + [raise_hand?] + [video_stopped?] + video_paused(4)
		// Since only video_paused is set (bit 4), the boolFalse should appear after call+participant.
		data := buf.Bytes()
		boolFalseBytes := []byte{0x37, 0x97, 0x79, 0xbc} // 0xbc799737 LE
		if !bytes.Contains(data, boolFalseBytes) {
			t.Fatalf("encoded data does not contain boolFalse; got %x", data)
		}

		// Must NOT contain boolTrue (0x997275b5).
		boolTrueBytes := []byte{0xb5, 0x75, 0x72, 0x99}
		if bytes.Contains(data, boolTrueBytes) {
			t.Fatal("encoded data should not contain boolTrue")
		}
	})

	t.Run("SetVideoPaused_true_sends_boolTrue", func(t *testing.T) {
		req := &PhoneEditGroupCallParticipantRequest{
			Call:        call,
			Participant: participant,
		}
		req.SetVideoPaused(true)

		var buf bytes.Buffer
		if err := req.Encode(&buf); err != nil {
			t.Fatal(err)
		}

		if !req.Flags.Has(4) {
			t.Fatal("flag bit 4 should be set")
		}

		data := buf.Bytes()
		boolTrueBytes := []byte{0xb5, 0x75, 0x72, 0x99}
		if !bytes.Contains(data, boolTrueBytes) {
			t.Fatalf("encoded data does not contain boolTrue; got %x", data)
		}
	})

	t.Run("no_Set_field_absent", func(t *testing.T) {
		req := &PhoneEditGroupCallParticipantRequest{
			Call:        call,
			Participant: participant,
			VideoPaused: false, // zero value, no Set called
		}

		var buf bytes.Buffer
		if err := req.Encode(&buf); err != nil {
			t.Fatal(err)
		}

		// Flag bit 4 must NOT be set — field is absent.
		if req.Flags.Has(4) {
			t.Fatal("flag bit 4 should not be set when SetVideoPaused was not called")
		}

		// Encoded data should be shorter — no boolFalse for video_paused.
		data := buf.Bytes()
		boolFalseBytes := []byte{0x37, 0x97, 0x79, 0xbc}
		if bytes.Contains(data, boolFalseBytes) {
			t.Fatal("encoded data should not contain any bool values when no ?Bool field is set")
		}
	})
}

// TestPhoneEditGroupCallParticipant_GetHelpers verifies Get* methods return (value, ok) correctly.
func TestPhoneEditGroupCallParticipant_GetHelpers(t *testing.T) {
	req := &PhoneEditGroupCallParticipantRequest{}

	// Not set → ok=false
	val, ok := req.GetVideoPaused()
	if ok {
		t.Fatalf("GetVideoPaused() should return ok=false when not set, got val=%v ok=%v", val, ok)
	}

	// Set to false → ok=true, val=false
	req.SetVideoPaused(false)
	val, ok = req.GetVideoPaused()
	if !ok {
		t.Fatal("GetVideoPaused() should return ok=true after SetVideoPaused(false)")
	}
	if val {
		t.Fatal("GetVideoPaused() should return val=false")
	}

	// Set to true → ok=true, val=true
	req.SetVideoPaused(true)
	val, ok = req.GetVideoPaused()
	if !ok || !val {
		t.Fatalf("GetVideoPaused() should return val=true ok=true, got val=%v ok=%v", val, ok)
	}

	// Muted
	req.SetMuted(false)
	val, ok = req.GetMuted()
	if !ok || val {
		t.Fatalf("GetMuted() should return val=false ok=true, got val=%v ok=%v", val, ok)
	}
}

// TestInputPeerNotifySettings_BoolFlagFalse verifies the fix works for types too.
func TestInputPeerNotifySettings_BoolFlagFalse(t *testing.T) {
	t.Run("SetSilent_false_sends_boolFalse", func(t *testing.T) {
		settings := &InputPeerNotifySettings{}
		settings.SetSilent(false)

		var buf bytes.Buffer
		if err := settings.Encode(&buf); err != nil {
			t.Fatal(err)
		}

		if !settings.Flags.Has(1) {
			t.Fatal("flag bit 1 (silent) should be set after SetSilent(false)")
		}

		data := buf.Bytes()
		boolFalseBytes := []byte{0x37, 0x97, 0x79, 0xbc}
		if !bytes.Contains(data, boolFalseBytes) {
			t.Fatalf("encoded data does not contain boolFalse; got %x", data)
		}
	})

	t.Run("SetShowPreviews_false_sends_boolFalse", func(t *testing.T) {
		settings := &InputPeerNotifySettings{}
		settings.SetShowPreviews(false)

		if !settings.Flags.Has(0) {
			t.Fatal("flag bit 0 (show_previews) should be set")
		}

		val, ok := settings.GetShowPreviews()
		if !ok || val {
			t.Fatalf("GetShowPreviews() = (%v, %v), want (false, true)", val, ok)
		}
	})
}

// TestBoolFlag_SetUnsetRoundTrip verifies that calling Set then Unset works correctly.
func TestBoolFlag_SetUnsetRoundTrip(t *testing.T) {
	req := &PhoneEditGroupCallParticipantRequest{}

	req.SetVideoPaused(false)
	if _, ok := req.GetVideoPaused(); !ok {
		t.Fatal("should be set after SetVideoPaused")
	}

	// Manually unset the flag
	req.Flags.Unset(4)
	if _, ok := req.GetVideoPaused(); ok {
		t.Fatal("should not be set after Unset(4)")
	}
}
