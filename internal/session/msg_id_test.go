package session

import (
	"sync"
	"testing"
	"time"
)

func TestMsgIDGeneratorMonotonic(t *testing.T) {
	gen := NewMsgIDGenerator(time.Unix(1700000000, 0))
	first := gen.Next()
	second := gen.Next()
	if second <= first {
		t.Errorf("msg IDs not monotonic: first=%d, second=%d", first, second)
	}
}

func TestMsgIDGeneratorConcurrent(t *testing.T) {
	gen := NewMsgIDGenerator(time.Unix(1700000000, 0))
	var wg sync.WaitGroup
	ids := make([]int64, 100)
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := gen.Next()
			mu.Lock()
			ids[idx] = id
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	seen := make(map[int64]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate msg_id: %d", id)
		}
		seen[id] = true
	}
}

func TestMsgIDGeneratorCorrectness(t *testing.T) {
	ts := time.Unix(1700000000, 0)
	gen := NewMsgIDGenerator(ts)
	id := gen.Next()
	expectedBase := int64(ts.Unix()) << 32
	if id < expectedBase {
		t.Errorf("msg_id=%d less than expected base=%d", id, expectedBase)
	}
	if id%4 != 0 {
		t.Errorf("msg_id=%d not divisible by 4 (must be in 'client' range)", id)
	}
}

func TestMsgIDGeneratorUpdateServerTime(t *testing.T) {
	gen := NewMsgIDGenerator(time.Unix(1700000000, 0))
	_ = gen.Next()
	newTime := time.Unix(1800000000, 0)
	gen.UpdateServerTime(newTime)
	id := gen.Next()
	expectedBase := int64(newTime.Unix()) << 32
	if id < expectedBase {
		t.Errorf("msg_id=%d less than expected base after update=%d", id, expectedBase)
	}
}

func TestMsgIDGeneratorUpdateServerTimeIgnoresOlder(t *testing.T) {
	gen := NewMsgIDGenerator(time.Unix(1800000000, 0))
	// An older time must be ignored to preserve monotonicity.
	gen.UpdateServerTime(time.Unix(1700000000, 0))
	id := gen.Next()
	if id>>32 != 1800000000 {
		t.Errorf("older time not ignored: msg_id base=%d, want 1800000000", id>>32)
	}
}

func TestMsgIDGeneratorForceResetServerTimeBackward(t *testing.T) {
	gen := NewMsgIDGenerator(time.Unix(1800000000, 0))
	_ = gen.Next()
	// Force-reset to an older time (code-17 correction); counter resets so the
	// next id is allocated in the new (lower) epoch.
	gen.ForceResetServerTime(time.Unix(1700000000, 0))
	id := gen.Next()
	if id>>32 != 1700000000 {
		t.Errorf("force reset did not move clock backward: msg_id base=%d, want 1700000000", id>>32)
	}
}
