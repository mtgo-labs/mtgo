package telegram

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
)

func TestIsSessionDeadErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"session closed", session.ErrSessionClosed, true},
		{"session draining", session.ErrDraining, true},
		{"session not connected", session.ErrNotConnected, true},
		{"client not connected", ErrNotConnected, true},
		{"wrapped session closed", errors.New("wrap: " + session.ErrSessionClosed.Error()), false},
		{"wrapped via fmt", wrapErr("test", session.ErrSessionClosed), true},
		{"nil", nil, false},
		{"other error", errors.New("something else"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSessionDeadErr(tc.err); got != tc.want {
				t.Fatalf("isSessionDeadErr(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestWaitForConnectAlreadyConnected(t *testing.T) {
	c, _ := NewClient(1, "hash", &Config{InMemory: true})
	c.state.SetConnecting(2)
	c.state.SetConnected()

	if err := c.waitForConnect(context.Background()); err != nil {
		t.Fatalf("waitForConnect when already connected: %v", err)
	}
}

func TestWaitForConnectCancelled(t *testing.T) {
	c, _ := NewClient(1, "hash", &Config{InMemory: true})
	c.state.SetConnecting(2)
	c.state.SetReconnecting(errors.New("disconnected"))

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	if err := c.waitForConnect(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waitForConnect: expected DeadlineExceeded, got %v", err)
	}
}

func TestWaitForConnectClientClosed(t *testing.T) {
	c, _ := NewClient(1, "hash", &Config{InMemory: true})
	c.state.SetClosed()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := c.waitForConnect(ctx); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("waitForConnect: expected ErrClientClosed, got %v", err)
	}
}

func TestRetrySessionErrDisabled(t *testing.T) {
	c, _ := NewClient(1, "hash", &Config{InMemory: true})
	c.state.SetConnecting(2)
	c.state.SetConnected()

	calls := 0
	err := c.retrySessionErr(context.Background(), func() error {
		calls++
		return session.ErrSessionClosed
	})
	if !errors.Is(err, session.ErrSessionClosed) {
		t.Fatalf("expected ErrSessionClosed, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("fn called %d times, want 1", calls)
	}
}

func TestRetrySessionErrRetriesAndSucceeds(t *testing.T) {
	cfg := &Config{
		InMemory:            true,
		RetryRPCOnReconnect: true,
	}
	c, _ := NewClient(1, "hash", cfg)
	c.state.SetConnecting(2)
	c.state.SetConnected()

	var calls atomic.Int32
	firstCallDone := make(chan struct{})

	// Simulate reconnection after the first failed call.
	go func() {
		<-firstCallDone
		time.Sleep(30 * time.Millisecond)
		c.state.SetConnecting(2)
		c.state.SetConnected()
		c.signalReconnect()
	}()

	err := c.retrySessionErr(context.Background(), func() error {
		n := calls.Add(1)
		if n == 1 {
			c.state.SetReconnecting(errors.New("session died"))
			close(firstCallDone)
			return session.ErrSessionClosed
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("fn called %d times, want 2", calls.Load())
	}
}

func TestRetrySessionErrNonSessionError(t *testing.T) {
	cfg := &Config{
		InMemory:            true,
		RetryRPCOnReconnect: true,
	}
	c, _ := NewClient(1, "hash", cfg)
	c.state.SetConnecting(2)
	c.state.SetConnected()

	otherErr := errors.New("some rpc error")
	calls := 0
	err := c.retrySessionErr(context.Background(), func() error {
		calls++
		return otherErr
	})
	if !errors.Is(err, otherErr) {
		t.Fatalf("expected otherErr, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("fn called %d times, want 1 (non-session error should not retry)", calls)
	}
}

func TestRetrySessionErrExhaustsRetries(t *testing.T) {
	cfg := &Config{
		InMemory:               true,
		RetryRPCOnReconnect:    true,
		MaxRPCReconnectRetries: 1, // 0 would default to 3
	}
	c, _ := NewClient(1, "hash", cfg)
	c.state.SetConnecting(2)
	c.state.SetConnected()

	var calls atomic.Int32

	// Continuously simulate reconnection so the retry loop doesn't hang.
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if c.state.IsClosed() {
				return
			}
			c.state.SetConnecting(2)
			c.state.SetConnected()
			c.signalReconnect()
		}
	}()

	err := c.retrySessionErr(context.Background(), func() error {
		n := calls.Add(1)
		// Every call fails with session-death error.
		_ = n
		c.state.SetReconnecting(errors.New("session died"))
		return session.ErrSessionClosed
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// With MaxRPCReconnectRetries=1, we get attempt 0 and attempt 1 = 2 calls.
	if calls.Load() != 2 {
		t.Fatalf("fn called %d times, want 2", calls.Load())
	}
}

func TestRetrySessionErrContextCancelled(t *testing.T) {
	cfg := &Config{
		InMemory:            true,
		RetryRPCOnReconnect: true,
	}
	c, _ := NewClient(1, "hash", cfg)
	c.state.SetConnecting(2)
	c.state.SetConnected()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var calls atomic.Int32
	firstCallDone := make(chan struct{})
	go func() {
		<-firstCallDone
		// Don't reconnect — let the context expire.
	}()

	err := c.retrySessionErr(ctx, func() error {
		n := calls.Add(1)
		if n == 1 {
			c.state.SetReconnecting(errors.New("session died"))
			close(firstCallDone)
			return session.ErrSessionClosed
		}
		return nil
	})
	if err == nil {
		t.Fatal("expected error from context cancellation, got nil")
	}
	// Should only have been called once (first call fails, then wait times out).
	if calls.Load() != 1 {
		t.Fatalf("fn called %d times, want 1", calls.Load())
	}
}

// wrapErr simulates fmt.Errorf("prefix: %w", err) for isSessionDeadErr testing.
func wrapErr(prefix string, err error) error {
	return &wrappedErr{msg: prefix + ": " + err.Error(), cause: err}
}

type wrappedErr struct {
	msg   string
	cause error
}

func (e *wrappedErr) Error() string { return e.msg }
func (e *wrappedErr) Unwrap() error { return e.cause }
