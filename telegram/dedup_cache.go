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
// dedupFields caches struct field indices per type to avoid repeated
// FieldByName lookups on the update dispatch hot path (#29).
type dedupFieldCache struct {
	ptsIndex     []int // nil if no PTS field
	channelIndex []int // nil if no ChannelID field
	qtsIndex     []int // nil if no Qts field
}

var dedupTypeCache sync.Map // reflect.Type → dedupFieldCache

func getDedupFields(t reflect.Type) dedupFieldCache {
	if cached, ok := dedupTypeCache.Load(t); ok {
		return cached.(dedupFieldCache)
	}
	fields := dedupFieldCache{}
	if f, ok := t.FieldByName("PTS"); ok && f.Type.Kind() == reflect.Int32 {
		fields.ptsIndex = f.Index
	}
	if f, ok := t.FieldByName("ChannelID"); ok && f.Type.Kind() == reflect.Int64 {
		fields.channelIndex = f.Index
	}
	if f, ok := t.FieldByName("Qts"); ok && f.Type.Kind() == reflect.Int32 {
		fields.qtsIndex = f.Index
	}
	dedupTypeCache.Store(t, fields)
	return fields
}

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
	fc := getDedupFields(v.Type())
	var pts int32
	if fc.ptsIndex != nil {
		pts = int32(v.FieldByIndex(fc.ptsIndex).Int())
	}
	var channelID int64
	if fc.channelIndex != nil {
		channelID = v.FieldByIndex(fc.channelIndex).Int()
	}
	if pts > 0 {
		if channelID > 0 {
			return channelID<<32 | int64(uint32(pts))
		}
		return int64(pts)
	}
	var qts int32
	if fc.qtsIndex != nil {
		qts = int32(v.FieldByIndex(fc.qtsIndex).Int())
	}
	if qts > 0 {
		return int64(1)<<62 | int64(qts)
	}
	return 0
}
