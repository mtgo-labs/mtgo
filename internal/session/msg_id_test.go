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
