package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

func TestFloodPolicyRetriesWithinThreshold(t *testing.T) {
	calls := 0
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
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
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
	err := invokeWithFloodPolicy(t.Context(), time.Hour, tg.MessagesSendMessageTypeID, nil, func() error {
		calls++
		return tgerr.New(420, "FLOOD_WAIT_1")
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

	calls := 0
	start := time.Now()
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
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("early gate did not sleep: %v", elapsed)
	}

	// After the deadline passes, no gating.
	start = time.Now()
	_ = invokeWithFloodPolicy(t.Context(), time.Hour, tg.MessagesSendMessageTypeID, reg, func() error { return nil })
	if time.Since(start) > 10*time.Millisecond {
		t.Fatal("gate should be clear after deadline")
	}

	// A different method is never gated by another method's deadline.
	start = time.Now()
	_ = invokeWithFloodPolicy(t.Context(), time.Hour, tg.MessagesGetHistoryTypeID, reg, func() error { return nil })
	if time.Since(start) > 10*time.Millisecond {
		t.Fatal("registry must key per method")
	}
}
