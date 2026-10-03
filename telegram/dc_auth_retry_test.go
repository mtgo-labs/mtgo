package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tgerr"
)

func TestRetryFloodWaitRetriesFloodWait(t *testing.T) {
	attempts := 0
	err := invokeWithFloodPolicy(context.Background(), 24*time.Hour, 0, nil, func() error {
		attempts++
		if attempts == 1 {
			return tgerr.New(420, "FLOOD_WAIT_0")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retry DC authorization: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestRetryFloodWaitHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0
	err := invokeWithFloodPolicy(ctx, 24*time.Hour, 0, nil, func() error {
		attempts++
		return tgerr.New(420, "FLOOD_WAIT_75")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("retry DC authorization error = %v, want context canceled", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestRetryFloodWaitReturnsNonFloodError(t *testing.T) {
	want := errors.New("authorization failed")
	attempts := 0
	err := invokeWithFloodPolicy(context.Background(), 24*time.Hour, 0, nil, func() error {
		attempts++
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("retry DC authorization error = %v, want %v", err, want)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestRetryTransferFloodWaitRetriesUntilSuccess(t *testing.T) {
	attempts := 0
	err := invokeWithFloodPolicy(withTransferRetry(context.Background()), 24*time.Hour, 0, nil, func() error {
		attempts++
		if attempts == 1 {
			return tgerr.New(420, "FLOOD_WAIT_0")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retry transfer flood wait: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestRetryTransferFloodWaitPreservesNonTransferPolicy(t *testing.T) {
	flood := tgerr.New(420, "FLOOD_WAIT_75")
	attempts := 0
	// Without the transfer marker the threshold stays at SleepThreshold (0
	// here), so the flood surfaces after exactly one attempt.
	err := invokeWithFloodPolicy(context.Background(), 0, 0, nil, func() error {
		attempts++
		return flood
	})
	if !errors.Is(err, flood) {
		t.Fatalf("retry transfer flood wait error = %v, want %v", err, flood)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}
