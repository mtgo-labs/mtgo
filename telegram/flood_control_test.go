package telegram

import (
	"context"
	"testing"
	"time"
)

func TestFloodControlEmptyAllows(t *testing.T) {
	fc := NewFloodControl([]FloodLimit{{Count: 5, WindowMs: 10_000}})
	if w := fc.wakeupAt(0); w != 0 {
		t.Fatalf("empty limiter wakeup = %d, want 0", w)
	}
}

func TestFloodControlBlocksAtLimit(t *testing.T) {
	fc := NewFloodControl([]FloodLimit{{Count: 3, WindowMs: 1_000}})
	fc.addEvent(0)
	fc.addEvent(100)
	fc.addEvent(200)
	if w := fc.wakeupAt(300); w != 1000 {
		t.Fatalf("saturated wakeup = %d, want 1000", w)
	}
}

func TestFloodControlAgesOut(t *testing.T) {
	fc := NewFloodControl([]FloodLimit{{Count: 1, WindowMs: 1_000}})
	fc.addEvent(0)
	if w := fc.wakeupAt(1000); w > 1000 {
		t.Fatalf("aged-out wakeup = %d, want <= 1000", w)
	}
	if w := fc.wakeupAt(1001); w != 0 {
		t.Fatalf("fully aged-out wakeup = %d, want 0", w)
	}
}

func TestFloodControlMultipleRules(t *testing.T) {
	fc := NewFloodControl(defaultFloodLimits)
	for i := int64(0); i < 8; i++ {
		fc.addEvent(i)
	}
	if w := fc.wakeupAt(100); w != 3000 {
		t.Fatalf("burst wakeup = %d, want 3000 (8@3s rule)", w)
	}
}

func TestFloodControlPrunesOldEvents(t *testing.T) {
	fc := NewFloodControl([]FloodLimit{
		{Count: 1, WindowMs: 1_000},
		{Count: 5, WindowMs: 5_000},
	})
	for ts := int64(0); ts <= 10_000; ts += 100 {
		fc.addEvent(ts)
	}
	if len(fc.events) > 52 {
		t.Fatalf("events not pruned: len=%d", len(fc.events))
	}
}

func TestFloodControlWaitImmediateOnEmpty(t *testing.T) {
	fc := NewFloodControl(defaultFloodLimits)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := fc.wait(ctx); err != nil {
		t.Fatalf("first wait: %v", err)
	}
}

func TestConnectionFloodGateAdmitsFirst(t *testing.T) {
	g := newConnectionFloodGate()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	start := time.Now()
	if err := g.wait(ctx); err != nil {
		t.Fatalf("first wait: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("first wait took %v, want immediate", elapsed)
	}
}

func TestConnectionFloodGateThrottles(t *testing.T) {
	g := newConnectionFloodGate()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First connection admitted immediately
	_ = g.wait(ctx)

	// Second should be delayed by the 1/1s flood rule
	start := time.Now()
	if err := g.wait(ctx); err != nil {
		t.Fatalf("second wait: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 900*time.Millisecond {
		t.Fatalf("second wait admitted after %v, want >= 900ms", elapsed)
	}
}

func TestConnectionFloodGateMtprotoErrorGates(t *testing.T) {
	g := newConnectionFloodGate()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Soak the mtproto error limiter with 8 events
	for i := 0; i < 8; i++ {
		g.recordMtprotoError()
	}

	// Next connect attempt should be gated by mtproto-error 8@3s
	start := time.Now()
	if err := g.wait(ctx); err != nil {
		t.Fatalf("wait after mtproto errors: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < 2900*time.Millisecond {
		t.Fatalf("admitted after %v, want >= 2900ms", elapsed)
	}
}

func TestConnectionFloodGateNotifyNetworkUpResetsSanity(t *testing.T) {
	g := &connectionFloodGate{
		sanity:       NewFloodControl(defaultSanityLimits),
		flood:        NewFloodControl([]FloodLimit{{Count: 100, WindowMs: 100}}),
		mtprotoError: NewFloodControl([]FloodLimit{{Count: 100, WindowMs: 100}}),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Exhaust sanity limit (5/10s) — flood limits are permissive
	for range 5 {
		_ = g.wait(ctx)
	}

	// Reset sanity
	g.notifyNetworkUp()

	// Should be admitted immediately
	start := time.Now()
	if err := g.wait(ctx); err != nil {
		t.Fatalf("wait after network up: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("wait after network up took %v, want immediate", elapsed)
	}
}
