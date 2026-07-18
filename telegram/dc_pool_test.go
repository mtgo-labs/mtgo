package telegram

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
)

type poolCloseTracker struct {
	closed atomic.Int32
}

func (c *poolCloseTracker) Close() error {
	c.closed.Add(1)
	return nil
}

func TestDCSessionStatsTracksHealthAndLatency(t *testing.T) {
	var stats dcSessionStats
	stats.observe(8*time.Millisecond, nil, time.Second)
	if got := time.Duration(stats.latencyEWMA.Load()); got != 8*time.Millisecond {
		t.Fatalf("initial EWMA = %v, want 8ms", got)
	}

	stats.observe(time.Millisecond, session.ErrSessionClosed, time.Second)
	if stats.failures.Load() != 1 {
		t.Fatalf("failures = %d, want 1", stats.failures.Load())
	}
	if stats.unhealthyUntil.Load() <= time.Now().UnixNano() {
		t.Fatal("connection failure should start cooldown")
	}

	stats.observe(16*time.Millisecond, errors.New("rpc application error"), time.Second)
	if stats.failures.Load() != 0 || stats.unhealthyUntil.Load() != 0 {
		t.Fatal("application error should prove transport health")
	}
	if got := time.Duration(stats.latencyEWMA.Load()); got != 9*time.Millisecond {
		t.Fatalf("updated EWMA = %v, want 9ms", got)
	}
}

func TestDCSessionPoolSelectsLeastLoadedThenLowestLatency(t *testing.T) {
	first := &dcSessionEntry{}
	second := &dcSessionEntry{}
	first.stats.inFlight.Store(2)
	second.stats.inFlight.Store(1)
	pool := &dcSessionPool{entries: []*dcSessionEntry{first, second}}

	selected, _ := pool.selectEntry()
	if selected != second {
		t.Fatal("pool did not select least-loaded connection")
	}

	first.stats.inFlight.Store(0)
	second.stats.inFlight.Store(0)
	first.stats.latencyEWMA.Store(int64(20 * time.Millisecond))
	second.stats.latencyEWMA.Store(int64(5 * time.Millisecond))
	selected, _ = pool.selectEntry()
	if selected != second {
		t.Fatal("pool did not select lower-latency connection")
	}
}

func TestDCSessionEntryRetiresAfterActiveLease(t *testing.T) {
	closer := &poolCloseTracker{}
	entry := &dcSessionEntry{closer: closer}
	started := entry.beginRequest()
	entry.retire()
	if closer.closed.Load() != 0 {
		t.Fatal("retired entry closed while request was active")
	}

	entry.endRequest(started, nil, time.Second)
	if closer.closed.Load() != 1 {
		t.Fatalf("close calls = %d, want 1", closer.closed.Load())
	}
	entry.close()
	if closer.closed.Load() != 1 {
		t.Fatalf("close calls after duplicate close = %d, want 1", closer.closed.Load())
	}
}
