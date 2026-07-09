package telegram

import (
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestDedupCacheDuplicatePTSSkipped(t *testing.T) {
	d := newDedupCache()
	if !d.checkAndAdd(100) {
		t.Error("first checkAndAdd(100) = false, want true")
	}
	if d.checkAndAdd(100) {
		t.Error("second checkAndAdd(100) = true, want false (duplicate)")
	}
}

func TestDedupCacheDifferentPTSDispatched(t *testing.T) {
	d := newDedupCache()
	if !d.checkAndAdd(100) {
		t.Error("checkAndAdd(100) = false, want true")
	}
	if !d.checkAndAdd(200) {
		t.Error("checkAndAdd(200) = false, want true (different pts)")
	}
	if !d.checkAndAdd(300) {
		t.Error("checkAndAdd(300) = false, want true (different pts)")
	}
}

func TestDedupCacheZeroKeyAlwaysDispatched(t *testing.T) {
	d := newDedupCache()
	for i := range 5 {
		if !d.checkAndAdd(0) {
			t.Errorf("checkAndAdd(0) iteration %d = false, want true", i)
		}
	}
}

func TestDedupCacheTTLExpiry(t *testing.T) {
	d := newDedupCache()
	// Manually age the entry past the TTL.
	d.mu.Lock()
	d.seen[42] = time.Now().Add(-dedupTTL - time.Second)
	d.mu.Unlock()

	if !d.checkAndAdd(42) {
		t.Error("checkAndAdd(42) after TTL = false, want true (expired)")
	}
}

func TestDedupCacheBoundedCleanup(t *testing.T) {
	d := newDedupCache()
	for i := range dedupMaxEntries {
		d.checkAndAdd(int64(i + 1))
	}

	// Force all entries to expire, then trigger cleanup.
	now := time.Now()
	d.mu.Lock()
	for k := range d.seen {
		d.seen[k] = now.Add(-dedupTTL - time.Hour)
	}
	d.mu.Unlock()

	d.checkAndAdd(int64(dedupMaxEntries + 1))

	d.mu.Lock()
	got := len(d.seen)
	d.mu.Unlock()
	// Only the just-added entry should remain.
	if got > 2 {
		t.Errorf("after cleanup: len(seen) = %d, want <= 2", got)
	}
}

// TestUpdateDedupKeyAccountPTS verifies that account-scoped pts updates produce
// a key equal to the pts value.
func TestUpdateDedupKeyAccountPTS(t *testing.T) {
	upd := &tg.UpdateNewMessage{PTS: 100, PTSCount: 1}
	if key := updateDedupKey(upd); key != 100 {
		t.Errorf("account pts key = %d, want 100", key)
	}
}

// TestUpdateDedupKeyAccountQTS verifies that qts-bearing updates produce a
// distinct key that doesn't collide with pts keys.
func TestUpdateDedupKeyAccountQTS(t *testing.T) {
	upd := &tg.UpdateBotStopped{Stopped: true, Qts: 100}
	key := updateDedupKey(upd)
	if key == 0 {
		t.Fatal("qts key = 0, want non-zero")
	}
	// Must not collide with pts=100.
	if key == 100 {
		t.Errorf("qts key collides with account pts key")
	}
}

// TestUpdateDedupKeyChannelPTS verifies that channel-scoped updates compose
// channelID and pts into a distinct key.
func TestUpdateDedupKeyChannelPTS(t *testing.T) {
	upd := &tg.UpdateDeleteChannelMessages{
		ChannelID: 5,
		PTS:       100,
		PTSCount:  1,
	}
	key := updateDedupKey(upd)
	// Should be channelID<<32 | pts.
	want := int64(5)<<32 | 100
	if key != want {
		t.Errorf("channel pts key = %d, want %d", key, want)
	}
	// Must differ from account pts=100.
	if key == 100 {
		t.Error("channel pts key collides with account pts key")
	}
}

// TestUpdateDedupKeyNoSignature verifies that updates without pts/qts
// produce a zero key (always dispatched).
func TestUpdateDedupKeyNoSignature(t *testing.T) {
	upd := &tg.UpdateChannel{ChannelID: 5}
	if key := updateDedupKey(upd); key != 0 {
		t.Errorf("no-signature key = %d, want 0", key)
	}
}

// TestUpdateDedupKeyNil verifies nil input.
func TestUpdateDedupKeyNil(t *testing.T) {
	if key := updateDedupKey(nil); key != 0 {
		t.Errorf("nil key = %d, want 0", key)
	}
}

// TestUpdateDedupKeyDistinctChannels verifies that the same pts on different
// channels produces different keys.
func TestUpdateDedupKeyDistinctChannels(t *testing.T) {
	upd1 := &tg.UpdateDeleteChannelMessages{
		ChannelID: 1,
		Messages:  []int32{1},
		PTS:       50,
		PTSCount:  1,
	}
	upd2 := &tg.UpdateDeleteChannelMessages{
		ChannelID: 2,
		Messages:  []int32{2},
		PTS:       50,
		PTSCount:  1,
	}
	if updateDedupKey(upd1) == updateDedupKey(upd2) {
		t.Error("same pts on different channels produced same key")
	}
}
