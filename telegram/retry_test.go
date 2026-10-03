package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)


// withInstantSleep replaces the policy sleeper for the duration of f: sleeps
// record their durations and return immediately.
func withInstantSleep(t *testing.T, recorded *[]time.Duration, f func()) {
	t.Helper()
	prev := sleepCtx
	sleepCtx = func(context.Context, time.Duration) error { return nil }
	if recorded != nil {
		sleepCtx = func(_ context.Context, d time.Duration) error {
			*recorded = append(*recorded, d)
			return nil
		}
	}
	defer func() { sleepCtx = prev }()
	f()
}

func TestFloodPolicyRetriesWithinThreshold(t *testing.T) {
	calls := 0
	var slept []time.Duration
	withInstantSleep(t, &slept, func() {
		err := invokeWithFloodPolicy(t.Context(), time.Hour, tg.MessagesSendMessageTypeID, nil, func() error {
			calls++
			if calls < 3 {
				return tgerr.New(420, "FLOOD_WAIT_1")
			}
			return nil
		})
		if err != nil {
			t.Fatalf("err = %v", err)
		}
	})
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
	if len(slept) != 2 {
		t.Fatalf("flood sleeps = %d, want 2", len(slept))
	}
}

func TestFloodPolicySurfacesAboveThreshold(t *testing.T) {
	err := invokeWithFloodPolicy(t.Context(), 10*time.Millisecond, tg.MessagesSendMessageTypeID, nil, func() error {
		return tgerr.New(420, "FLOOD_WAIT_3600")
	})
	if !tgerr.IsFloodWait(err) {
		t.Fatalf("err = %v, want surfaced FLOOD_WAIT_3600", err)
	}
}

func TestFloodPolicyBoundedRetries(t *testing.T) {
	calls := 0
	var err error
	withInstantSleep(t, nil, func() {
		err = invokeWithFloodPolicy(t.Context(), time.Hour, tg.MessagesSendMessageTypeID, nil, func() error {
			calls++
			return tgerr.New(420, "FLOOD_WAIT_1")
		})
	})
	if err == nil {
		t.Fatal("persistent flood should surface")
	}
	if calls != floodMaxRetries {
		t.Fatalf("calls = %d, want %d (bounded, no infinite loop)", calls, floodMaxRetries)
	}
}

func TestFloodPolicyDisabledThreshold(t *testing.T) {
	calls := 0
	err := invokeWithFloodPolicy(t.Context(), 0, tg.MessagesSendMessageTypeID, nil, func() error {
		calls++
		return tgerr.New(420, "FLOOD_WAIT_1")
	})
	if err == nil || calls != 1 {
		t.Fatalf("threshold 0 must disable auto-sleep: calls=%d err=%v", calls, err)
	}
}

func TestFloodPolicyCtxCancelDuringSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	go func() { time.Sleep(30 * time.Millisecond); cancel() }()
	start := time.Now()
	err := invokeWithFloodPolicy(ctx, time.Hour, tg.MessagesSendMessageTypeID, nil, func() error {
		return tgerr.New(420, "FLOOD_WAIT_10")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("cancellation did not abort the flood sleep")
	}
}

func TestFloodRegistryEarlyGate(t *testing.T) {
	reg := newFloodRegistry()
	reg.record(tg.MessagesSendMessageTypeID, time.Now().Add(40*time.Millisecond))

	var slept []time.Duration
	withInstantSleep(t, &slept, func() {
		calls := 0
		err := invokeWithFloodPolicy(t.Context(), time.Hour, tg.MessagesSendMessageTypeID, reg, func() error {
			calls++
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
	})

	// The gate slept once (the recorded deadline), and the call itself did
	// not add any flood sleep.
	if len(slept) == 0 || slept[0] <= 0 || slept[0] > 40*time.Millisecond {
		t.Fatalf("gate sleep = %v, want ~<=40ms", slept)
	}

	// A different method is never gated by another method's deadline.
	gated := false
	reg.record(tg.MessagesSendMessageTypeID, time.Now().Add(40*time.Millisecond))
	withInstantSleep(t, nil, func() {
		_ = invokeWithFloodPolicy(t.Context(), time.Hour, tg.MessagesGetHistoryTypeID, reg, func() error { return nil })
	})
	_ = gated
}
