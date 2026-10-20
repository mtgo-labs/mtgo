package telegram

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestKeyedMutexesSerializesPerKey verifies the per-DC auth export/import
// serialization primitive: concurrent Locks of the same key must not overlap
// (Telegram returns identical export bytes for concurrent exports toward one
// DC; only the first import succeeds — AUTH_BYTES_INVALID, field report #2),
// while different keys proceed independently.
func TestKeyedMutexesSerializesPerKey(t *testing.T) {
	var k keyedMutexes

	var inCritical atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k.Lock(1)
			defer k.Unlock(1)
			cur := inCritical.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			inCritical.Add(-1)
		}()
	}
	wg.Wait()
	if got := maxConcurrent.Load(); got != 1 {
		t.Fatalf("critical sections for the same key overlapped: max concurrency %d", got)
	}
}

func TestKeyedMutexesIndependentAcrossKeys(t *testing.T) {
	var k keyedMutexes

	// Hold key 1; key 2 must remain lockable (parallel pool creation on
	// different DCs must not serialize against each other).
	k.Lock(1)
	defer k.Unlock(1)

	acquired := make(chan struct{})
	go func() {
		k.Lock(2)
		defer k.Unlock(2)
		close(acquired)
	}()
	select {
	case <-acquired:
	case <-time.After(2 * time.Second):
		t.Fatal("Lock(2) blocked while Lock(1) was held")
	}
}
