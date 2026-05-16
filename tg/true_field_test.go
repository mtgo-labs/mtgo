package tg

import (
	"bytes"
	"testing"
)

// TestTrueFieldEncode verifies that true-typed fields (bare flags) are NOT
// serialized as WriteBool in Encode — only flag bits in SetFlags carry the value.
func TestTrueFieldEncode(t *testing.T) {
	// AutoSaveSettings: photos:flags.0?true videos:flags.1?true video_max_size:flags.2?long
	t.Run("AutoSaveSettings", func(t *testing.T) {
		v := &AutoSaveSettings{
			Photos:       true,
			Videos:       true,
			VideoMaxSize: 1024,
		}

		var buf bytes.Buffer
		if err := v.Encode(&buf); err != nil {
			t.Fatal(err)
		}
		data := buf.Bytes()

		// Expected wire format:
		//   4 bytes: constructor ID (0xc84834ce)
		//   4 bytes: flags (bit 0|1|2 = 0x07)
		//   8 bytes: VideoMaxSize as long
		// Total: 16 bytes
		//
		// If true-typed fields were wrongly encoded as WriteBool,
		// we'd see extra 4+4 bytes (BoolTrue) = 24 bytes total.
		if len(data) != 16 {
			t.Fatalf("AutoSaveSettings Encode len=%d, want 16. Extra bytes mean true-typed fields are leaking WriteBool.\nData: %x", len(data), data)
		}

		flags := Fields(data[4]) | Fields(data[5])<<8 | Fields(data[6])<<16 | Fields(data[7])<<24
		if !flags.Has(0) {
			t.Error("flag bit 0 (photos) not set")
		}
		if !flags.Has(1) {
			t.Error("flag bit 1 (videos) not set")
		}
		if !flags.Has(2) {
			t.Error("flag bit 2 (video_max_size) not set")
		}
	})

	// AccountSaveAutoSaveSettings: users:flags.0?true chats:flags.1?true broadcasts:flags.2?true peer:flags.3?InputPeer settings:AutoSaveSettings
	t.Run("AccountSaveAutoSaveSettingsRequest", func(t *testing.T) {
		settings := &AutoSaveSettings{
			Photos:       true,
			VideoMaxSize: 2048,
		}
		v := &AccountSaveAutoSaveSettingsRequest{
			Users:      true,
			Chats:      true,
			Broadcasts: true,
			Peer:       &InputPeerSelf{},
			Settings:   settings,
		}

		var buf bytes.Buffer
		if err := v.Encode(&buf); err != nil {
			t.Fatal(err)
		}
		data := buf.Bytes()

		// Expected wire format:
		//   4 bytes: constructor ID (0xd69b8361)
		//   4 bytes: flags (bits 0|1|2|3 = 0x0F)
		//   4 bytes: InputPeerSelf constructor (0x7da07ec9)
		//   nested AutoSaveSettings (16 bytes)
		// Total: 4 + 4 + 4 + 16 = 28 bytes
		if len(data) != 28 {
			t.Fatalf("AccountSaveAutoSaveSettingsRequest Encode len=%d, want 28. Extra bytes mean true-typed fields are leaking WriteBool.\nData: %x", len(data), data)
		}

		flags := Fields(data[4]) | Fields(data[5])<<8 | Fields(data[6])<<16 | Fields(data[7])<<24
		if !flags.Has(0) {
			t.Error("flag bit 0 (users) not set")
		}
		if !flags.Has(1) {
			t.Error("flag bit 1 (chats) not set")
		}
		if !flags.Has(2) {
			t.Error("flag bit 2 (broadcasts) not set")
		}
		if !flags.Has(3) {
			t.Error("flag bit 3 (peer) not set")
		}
	})

	// AccountGetNotifyExceptionsRequest: compare_sound:flags.1?true compare_stories:flags.2?true peer:flags.0?InputNotifyPeer
	t.Run("AccountGetNotifyExceptionsRequest", func(t *testing.T) {
		v := &AccountGetNotifyExceptionsRequest{
			CompareSound:   true,
			CompareStories: true,
		}

		var buf bytes.Buffer
		if err := v.Encode(&buf); err != nil {
			t.Fatal(err)
		}
		data := buf.Bytes()

		// Expected wire format:
		//   4 bytes: constructor ID (0x53577479)
		//   4 bytes: flags (bits 1|2 = 0x06)
		// No more fields (peer not set)
		// Total: 8 bytes
		if len(data) != 8 {
			t.Fatalf("AccountGetNotifyExceptionsRequest Encode len=%d, want 8. Extra bytes mean true-typed fields are leaking WriteBool.\nData: %x", len(data), data)
		}

		flags := Fields(data[4]) | Fields(data[5])<<8 | Fields(data[6])<<16 | Fields(data[7])<<24
		if !flags.Has(1) {
			t.Error("flag bit 1 (compare_sound) not set")
		}
		if !flags.Has(2) {
			t.Error("flag bit 2 (compare_stories) not set")
		}
	})
}
