package telegram

import (
	"reflect"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

const (
	dedupTTL        = 5 * time.Second
	dedupMaxEntries = 1000
)

// dedupCache prevents the same update from being dispatched twice when it
// arrives both from an RPC response (e.g. updatesrecovery's
// dispatchRecovered) and from the server push stream. It is a bounded map of
// update signatures keyed by pts/qts/channelID values, with a short TTL.
type dedupCache struct {
	mu   sync.Mutex
	seen map[int64]time.Time
}

func newDedupCache() *dedupCache {
	return &dedupCache{seen: make(map[int64]time.Time)}
}

// checkAndAdd returns true if key is new (should dispatch), false if it was
// seen within the TTL window (duplicate — skip). A zero key means the update
// has no dedup signature and is always dispatched.
func (d *dedupCache) checkAndAdd(key int64) bool {
	if key == 0 {
		return true
	}
	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()
	if t, ok := d.seen[key]; ok && now.Sub(t) < dedupTTL {
		return false
	}
	if len(d.seen) >= dedupMaxEntries {
		d.cleanupLocked(now)
	}
	d.seen[key] = now
	return true
}

func (d *dedupCache) cleanupLocked(now time.Time) {
	for k, t := range d.seen {
		if now.Sub(t) >= dedupTTL {
			delete(d.seen, k)
		}
	}
}

// updateDedupKey extracts a dedup signature from a single update. The key
// space is partitioned so that account pts, account qts, and channel pts never
// collide:
//   - Account pts (no channelID): key = int64(pts)        [1 .. 2^31]
//   - Account qts (no channelID): key = int64(qts)|1<<62   [> 2^62]
//   - Channel pts:                key = ch<<32 | int64(pts) [2^32 .. ~2^62]
//
// Returns 0 when the update carries no pts/qts signature (always dispatch).
func updateDedupKey(upd tg.UpdateClass) int64 {
	if upd == nil {
		return 0
	}
	v := reflect.ValueOf(upd)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0
	}
	pts := dedupReadInt32(v, "PTS")
	channelID := dedupReadInt64(v, "ChannelID")
	if pts > 0 {
		if channelID > 0 {
			return channelID<<32 | int64(uint32(pts))
		}
		return int64(pts)
	}
	qts := dedupReadInt32(v, "Qts")
	if qts > 0 {
		return int64(1)<<62 | int64(qts)
	}
	return 0
}

func dedupReadInt32(v reflect.Value, name string) int32 {
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.Int32 {
		return 0
	}
	return int32(f.Int())
}

func dedupReadInt64(v reflect.Value, name string) int64 {
	f := v.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.Int64 {
		return 0
	}
	return f.Int()
}
