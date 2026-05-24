package session

import (
	"bytes"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestKnownFingerprints(t *testing.T) {
	fps := knownFingerprints()
	if len(fps) == 0 {
		t.Fatal("knownFingerprints() returned empty slice, expected at least one fingerprint")
	}

	seen := make(map[int64]bool, len(fps))
	for _, fp := range fps {
		if seen[fp] {
			t.Errorf("duplicate fingerprint %d", fp)
		}
		seen[fp] = true

		if _, ok := crypto.GetServerKey(fp); !ok {
			t.Errorf("fingerprint %d not found in crypto.GetServerKey()", fp)
		}
	}

	if len(fps) != len(crypto.ServerPublicKeys) {
		t.Errorf("knownFingerprints() returned %d entries, ServerPublicKeys has %d",
			len(fps), len(crypto.ServerPublicKeys))
	}
}

func TestFindKeyFingerprint(t *testing.T) {
	a := &Auth{}

	fps := knownFingerprints()
	if len(fps) == 0 {
		t.Skip("no known fingerprints to test with")
	}

	fp, ok := a.findKeyFingerprint(fps)
	if !ok {
		t.Fatal("findKeyFingerprint(known) returned false")
	}
	if _, keyOK := crypto.GetServerKey(fp); !keyOK {
		t.Errorf("returned fingerprint %d not in ServerPublicKeys", fp)
	}

	_, ok = a.findKeyFingerprint([]int64{12345, -99999})
	if ok {
		t.Error("findKeyFingerprint with unknown fingerprints should return false")
	}

	_, ok = a.findKeyFingerprint(nil)
	if ok {
		t.Error("findKeyFingerprint(nil) should return false")
	}

	_, ok = a.findKeyFingerprint([]int64{})
	if ok {
		t.Error("findKeyFingerprint(empty) should return false")
	}

	mixed := []int64{12345, fps[0], -99999}
	fp, ok = a.findKeyFingerprint(mixed)
	if !ok {
		t.Fatal("findKeyFingerprint with mixed list should find the valid one")
	}
	if fp != fps[0] {
		t.Errorf("expected fingerprint %d, got %d", fps[0], fp)
	}
}

func TestGenerateRandomBN(t *testing.T) {
	a := &Auth{}

	n, err := a.generateRandomBN(256)
	if err != nil {
		t.Fatalf("generateRandomBN(256) error: %v", err)
	}
	if n == nil {
		t.Fatal("generateRandomBN returned nil")
	}
	if n.Sign() <= 0 {
		t.Error("generateRandomBN returned non-positive number")
	}
	if n.BitLen() != 256 {
		t.Errorf("expected 256 bits, got %d", n.BitLen())
	}

	n2, err := a.generateRandomBN(256)
	if err != nil {
		t.Fatalf("second generateRandomBN(256) error: %v", err)
	}
	if n.Cmp(n2) == 0 {
		t.Error("two consecutive random values should not be equal")
	}

	bigN, err := a.generateRandomBN(2048)
	if err != nil {
		t.Fatalf("generateRandomBN(2048) error: %v", err)
	}
	if bigN.BitLen() != 2048 {
		t.Errorf("expected 2048 bits, got %d", bigN.BitLen())
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for 0 bits")
			}
		}()
		a.generateRandomBN(0)
	}()
}

func TestSerializeDeserializeTL(t *testing.T) {
	a := &Auth{}

	obj := &tg.PingRequest{PingID: 42}

	encoded, err := a.serializeTL(obj)
	if err != nil {
		t.Fatalf("serializeTL error: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("serializeTL returned empty bytes")
	}

	constructor := tg.ReadInt(bytes.NewReader(encoded[:4]))
	if constructor != tg.PingTypeID {
		t.Errorf("constructor ID = 0x%x, want 0x%x", constructor, tg.PingTypeID)
	}

	decoded, err := a.deserializeTL(encoded)
	if err != nil {
		t.Fatalf("deserializeTL error: %v", err)
	}

	decodedReq, ok := decoded.(*tg.PingRequest)
	if !ok {
		t.Fatalf("expected *tg.PingRequest, got %T", decoded)
	}
	if decodedReq.PingID != 42 {
		t.Errorf("PingID = %d, want 42", decodedReq.PingID)
	}
}

func TestDeserializeTLInvalid(t *testing.T) {
	a := &Auth{}

	_, err := a.deserializeTL([]byte{})
	if err == nil {
		t.Error("expected error for empty input")
	}

	_, err = a.deserializeTL([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	if err == nil {
		t.Error("expected error for unknown constructor")
	}
}

func TestSerializeTLEmptyObject(t *testing.T) {
	a := &Auth{}

	obj := &tg.PingRequest{PingID: 42}
	encoded, err := a.serializeTL(obj)
	if err != nil {
		t.Fatalf("serializeTL error: %v", err)
	}
	if len(encoded) < 12 {
		t.Errorf("encoded too short: %d bytes", len(encoded))
	}
}

func TestSaltEntryValidUntil(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	sm := newSaltManager(func() time.Time { return now })

	sm.StoreSimple(12345)

	got := sm.ValidUntil()
	want := now.Add(2 * time.Hour)
	if !got.Equal(want) {
		t.Errorf("ValidUntil() = %v, want %v", got, want)
	}
}

func TestSaltEntryValidUntilZero(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	sm := newSaltManager(func() time.Time { return now })

	got := sm.ValidUntil()
	if !got.IsZero() {
		t.Errorf("ValidUntil() = %v, want zero time before any salt is stored", got)
	}
}

func TestSaltEntryValidUntilFromFuture(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	sm := newSaltManager(func() time.Time { return now })

	entries := saltEntriesFromFuture([]*tg.FutureSalt{
		{
			ValidSince: 1736899200,
			ValidUntil: 1736906400,
			Salt:       99,
		},
	})
	sm.StoreFromFutureSalts(entries)

	got := sm.ValidUntil()
	want := time.Unix(1736906400, 0)
	if !got.Equal(want) {
		t.Errorf("ValidUntil() = %v, want %v", got, want)
	}
}


