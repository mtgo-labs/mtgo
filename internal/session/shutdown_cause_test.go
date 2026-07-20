package session

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestShutdownCauseFirstErrorWins verifies that the first call to
// recordShutdownCause stores the cause and subsequent calls are no-ops.
func TestShutdownCauseFirstErrorWins(t *testing.T) {
	s := newSessionWithAuthKey(t)

	first := errors.New("io: EOF")
	second := errors.New("connection reset by peer")

	s.recordShutdownCause(first, "readLoop")
	s.recordShutdownCause(second, "pingLoop")

	source, _, err := s.ShutdownCause()
	if err != first {
		t.Fatalf("ShutdownCause err = %v, want %v", err, first)
	}
	if source != "readLoop" {
		t.Fatalf("ShutdownCause source = %q, want %q", source, "readLoop")
	}
}

// TestShutdownCauseNilWhenNotSet verifies that ShutdownCause returns zero
// values when no cause has been recorded.
func TestShutdownCauseNilWhenNotSet(t *testing.T) {
	s := newSessionWithAuthKey(t)

	source, at, err := s.ShutdownCause()
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if source != "" {
		t.Fatalf("source = %q, want empty", source)
	}
	if !at.IsZero() {
		t.Fatalf("at = %v, want zero time", at)
	}
}

// TestShutdownCauseNilErrorIgnored verifies that passing nil to
// recordShutdownCause is a no-op.
func TestShutdownCauseNilErrorIgnored(t *testing.T) {
	s := newSessionWithAuthKey(t)

	s.recordShutdownCause(nil, "readLoop")

	_, _, err := s.ShutdownCause()
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

// TestWrapGoroutineFiltersContextCanceled verifies that context.Canceled
// and context.DeadlineExceeded are NOT recorded as shutdown causes.
func TestWrapGoroutineFiltersContextCanceled(t *testing.T) {
	s := newSessionWithAuthKey(t)

	wrapped := s.wrapGoroutine("readLoop", func(ctx context.Context) error {
		return context.Canceled
	})

	err := wrapped(context.Background())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("wrapped should return original error, got %v", err)
	}

	_, _, cause := s.ShutdownCause()
	if cause != nil {
		t.Fatalf("context.Canceled should not be recorded as cause, got %v", cause)
	}
}

// TestWrapGoroutineRecordsRealError verifies that a real (non-context)
// error is recorded as the shutdown cause.
func TestWrapGoroutineRecordsRealError(t *testing.T) {
	s := newSessionWithAuthKey(t)

	realErr := errors.New("session: pong timeout")
	wrapped := s.wrapGoroutine("pingLoop", func(ctx context.Context) error {
		return realErr
	})

	err := wrapped(context.Background())
	if !errors.Is(err, realErr) {
		t.Fatalf("wrapped should return original error, got %v", err)
	}

	source, _, cause := s.ShutdownCause()
	if !errors.Is(cause, realErr) {
		t.Fatalf("ShutdownCause = %v, want %v", cause, realErr)
	}
	if source != "pingLoop" {
		t.Fatalf("source = %q, want %q", source, "pingLoop")
	}
}

// TestShutdownCauseTimestamp verifies that the recorded time is recent.
func TestShutdownCauseTimestamp(t *testing.T) {
	s := newSessionWithAuthKey(t)

	before := time.Now()
	s.recordShutdownCause(errors.New("test error"), "test")
	after := time.Now()

	_, at, _ := s.ShutdownCause()
	if at.Before(before) || at.After(after) {
		t.Fatalf("timestamp %v not in range [%v, %v]", at, before, after)
	}
}
