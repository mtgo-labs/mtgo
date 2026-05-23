package session

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestRegisterAndResolve(t *testing.T) {
	pm := NewPendingManager()
	h := pm.Register(1, false)

	if !pm.Has(1) {
		t.Fatal("expected Has(1) to be true")
	}
	if pm.HasRaw(1) {
		t.Fatal("expected HasRaw(1) to be false for decoded handle")
	}
	if !pm.HasAny() {
		t.Fatal("expected HasAny() to be true")
	}

	obj := &tg.Pong{MsgID: 42}
	if !pm.Resolve(1, obj) {
		t.Fatal("expected Resolve to return true")
	}

	if pm.Has(1) {
		t.Fatal("expected Has(1) to be false after resolve")
	}
	if pm.HasAny() {
		t.Fatal("expected HasAny() to be false after resolve")
	}

	<-h.Done()
	got, _, err := h.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != obj {
		t.Fatal("result mismatch")
	}
}

func TestRegisterAndResolveRaw(t *testing.T) {
	pm := NewPendingManager()
	h := pm.Register(2, true)

	if !pm.HasRaw(2) {
		t.Fatal("expected HasRaw(2) to be true")
	}
	if !pm.HasAnyRaw() {
		t.Fatal("expected HasAnyRaw() to be true")
	}

	data := []byte{1, 2, 3}
	if !pm.ResolveRaw(2, data) {
		t.Fatal("expected ResolveRaw to return true")
	}

	<-h.Done()
	_, raw, err := h.Result()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(raw) != string(data) {
		t.Fatal("raw result mismatch")
	}
}

func TestReject(t *testing.T) {
	pm := NewPendingManager()
	h := pm.Register(3, false)

	rejectErr := errors.New("test error")
	if !pm.Reject(3, rejectErr) {
		t.Fatal("expected Reject to return true")
	}

	<-h.Done()
	_, _, err := h.Result()
	if err != rejectErr {
		t.Fatalf("expected rejectErr, got %v", err)
	}
}

func TestCancel(t *testing.T) {
	pm := NewPendingManager()
	h := pm.Register(4, false)

	if !pm.Cancel(4) {
		t.Fatal("expected Cancel to return true")
	}

	if pm.Has(4) {
		t.Fatal("expected Has(4) to be false after cancel")
	}
	if pm.HasAny() {
		t.Fatal("expected HasAny() to be false after cancel")
	}

	// done should NOT be closed — nobody is waiting
	select {
	case <-h.Done():
		t.Fatal("done should not be closed after cancel")
	default:
	}
}

func TestLateResolveIgnored(t *testing.T) {
	pm := NewPendingManager()
	pm.Register(5, false)

	pm.Cancel(5)

	if pm.Resolve(5, &tg.Pong{}) {
		t.Fatal("expected Resolve on cancelled handle to return false")
	}
}

func TestRejectAll(t *testing.T) {
	pm := NewPendingManager()
	h1 := pm.Register(10, false)
	h2 := pm.Register(11, true)
	h3 := pm.Register(12, false)

	rejectErr := errors.New("shutdown")
	pm.RejectAll(rejectErr)

	<-h1.Done()
	<-h2.Done()
	<-h3.Done()

	for _, h := range []*CallHandle{h1, h2, h3} {
		_, _, err := h.Result()
		if err != rejectErr {
			t.Fatalf("expected rejectErr, got %v", err)
		}
	}

	if pm.HasAny() {
		t.Fatal("expected no pending after RejectAll")
	}
	if pm.Count() != 0 {
		t.Fatalf("expected Count()=0, got %d", pm.Count())
	}
}

func TestOneShotCompletion(t *testing.T) {
	pm := NewPendingManager()
	pm.Register(20, false)

	obj1 := &tg.Pong{MsgID: 1}
	obj2 := &tg.Pong{MsgID: 2}

	if !pm.Resolve(20, obj1) {
		t.Fatal("first resolve should succeed")
	}
	if pm.Resolve(20, obj2) {
		t.Fatal("second resolve should fail")
	}
	if pm.Reject(20, errors.New("x")) {
		t.Fatal("reject after resolve should fail")
	}
}

func TestHasHasRawHasAny(t *testing.T) {
	pm := NewPendingManager()

	if pm.HasAny() || pm.HasAnyRaw() {
		t.Fatal("empty manager should report false")
	}

	pm.Register(1, false)
	pm.Register(2, true)

	if !pm.Has(1) || !pm.Has(2) {
		t.Fatal("both should be present")
	}
	if pm.HasRaw(1) {
		t.Fatal("msgID 1 is decoded, not raw")
	}
	if !pm.HasRaw(2) {
		t.Fatal("msgID 2 is raw")
	}

	pm.Resolve(1, nil)
	if pm.Has(1) {
		t.Fatal("msgID 1 should be gone after resolve")
	}
	if !pm.Has(2) {
		t.Fatal("msgID 2 should still be present")
	}
}

func TestConcurrentResolve(t *testing.T) {
	pm := NewPendingManager()
	pm.Register(100, false)

	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if pm.Resolve(100, &tg.Pong{}) {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()

	if wins.Load() != 1 {
		t.Fatalf("expected exactly 1 winner, got %d", wins.Load())
	}
}

func TestReceiveLoopNoBlock(t *testing.T) {
	pm := NewPendingManager()

	// Simulate 100 pending calls. Cancel half of them (simulating timeouts),
	// then resolve the rest. Verify no blocking.
	for i := int64(0); i < 100; i++ {
		pm.Register(i, i%2 == 0)
	}

	// Cancel first 50 (caller timed out)
	for i := int64(0); i < 50; i++ {
		pm.Cancel(i)
	}

	// Resolve remaining 50 (receive loop delivers)
	for i := int64(50); i < 100; i++ {
		if i%2 == 0 {
			pm.ResolveRaw(i, []byte("data"))
		} else {
			pm.Resolve(i, &tg.Pong{})
		}
	}

	if pm.Count() != 0 {
		t.Fatalf("expected 0 pending, got %d", pm.Count())
	}
}

func TestRejectAllNoMapLeak(t *testing.T) {
	pm := NewPendingManager()
	for i := int64(0); i < 1000; i++ {
		pm.Register(i, i%2 == 0)
	}

	pm.RejectAll(errors.New("done"))

	if pm.Count() != 0 {
		t.Fatalf("expected 0 pending after RejectAll, got %d", pm.Count())
	}
	pm.mu.Lock()
	leaked := len(pm.pending)
	pm.mu.Unlock()
	if leaked != 0 {
		t.Fatalf("expected empty map after RejectAll, got %d entries", leaked)
	}
}

func TestHighConcurrency(t *testing.T) {
	pm := NewPendingManager()
	const n = 500

	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := int64(0); i < n; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			h := pm.Register(id, id%2 == 0)

			// Simulate receive loop resolving quickly
			go func() {
				time.Sleep(time.Microsecond * time.Duration(1+id%10))
				if id%2 == 0 {
					pm.ResolveRaw(id, []byte("r"))
				} else {
					pm.Resolve(id, &tg.Pong{MsgID: id})
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			select {
			case <-h.Done():
				// ok
			case <-ctx.Done():
				errs <- ctx.Err()
				pm.Cancel(id)
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("timeout: %v", err)
	}

	if pm.Count() != 0 {
		t.Fatalf("expected 0 pending, got %d", pm.Count())
	}
}
