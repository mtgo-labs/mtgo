package telegram

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type recordingTelemetry struct {
	mu          sync.Mutex
	rpcs        []RPCObservation
	connections []ConnectionObservation
}

type closingTelemetry struct {
	client *Client
	done   chan struct{}
	once   sync.Once
}

type closingConnectionTelemetry struct {
	client *Client
	done   chan struct{}
	once   sync.Once
}

func (*closingConnectionTelemetry) ObserveRPC(context.Context, RPCObservation) {}

func (t *closingConnectionTelemetry) ObserveConnection(context.Context, ConnectionObservation) {
	t.once.Do(func() {
		t.client.Close()
		close(t.done)
	})
}

func (t *closingTelemetry) ObserveRPC(context.Context, RPCObservation) {
	t.client.Close()
	t.once.Do(func() { close(t.done) })
}

func (*closingTelemetry) ObserveConnection(context.Context, ConnectionObservation) {}

func (r *recordingTelemetry) ObserveRPC(_ context.Context, observation RPCObservation) {
	r.mu.Lock()
	r.rpcs = append(r.rpcs, observation)
	r.mu.Unlock()
}

func (r *recordingTelemetry) ObserveConnection(_ context.Context, observation ConnectionObservation) {
	r.mu.Lock()
	r.connections = append(r.connections, observation)
	r.mu.Unlock()
}

func TestTelemetryRecordsRPCAttemptsAndDelivery(t *testing.T) {
	observer := &recordingTelemetry{}
	c, _ := NewClient(1, "hash", &Config{
		InMemory:            true,
		RetryRPCOnReconnect: true,
		Telemetry:           observer,
	})
	c.state.SetConnecting(2)
	c.state.SetConnected()

	calls := 0
	err := c.retrySessionErr(context.Background(), func(*session.Session) error {
		calls++
		return &session.DeliveryError{State: session.DeliveryReceived, Err: session.ErrSessionClosed}
	}, &tg.MessagesSendMessageRequest{})
	var deliveryErr *RPCDeliveryError
	if !errors.As(err, &deliveryErr) {
		t.Fatalf("error = %v, want delivery error", err)
	}
	observer.mu.Lock()
	defer observer.mu.Unlock()
	if len(observer.rpcs) != 1 {
		t.Fatalf("RPC observations = %d, want 1", len(observer.rpcs))
	}
	got := observer.rpcs[0]
	if got.Method != "MessagesSendMessage" || got.Attempt != 1 || got.ErrorClass != "closed" || got.DeliveryState != RPCDeliveryReceived {
		t.Fatalf("RPC observation = %+v", got)
	}
	if got.EndedAt.Before(got.StartedAt) {
		t.Fatal("RPC observation has negative duration")
	}
}

func TestTelemetryRecordsConnectionEvents(t *testing.T) {
	observer := &recordingTelemetry{}
	metrics := newConnectionMetrics(observer)
	metrics.recordDialSuccess("dc2:443", time.Millisecond)
	metrics.recordDisconnected(errors.New("closed"))

	deadline := time.Now().Add(time.Second)
	var observations []ConnectionObservation
	for time.Now().Before(deadline) {
		observer.mu.Lock()
		observations = append(observations[:0], observer.connections...)
		observer.mu.Unlock()
		if len(observations) == 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if len(observations) != 2 {
		t.Fatalf("connection observations = %d, want 2", len(observations))
	}
	if observations[0].Kind != "dial.success" || observations[1].Kind != "disconnected" {
		t.Fatalf("connection observations = %+v", observations)
	}
}

func TestTelemetryMayCloseClientWithoutDeadlock(t *testing.T) {
	observer := &closingTelemetry{done: make(chan struct{})}
	c, _ := NewClient(1, "hash", &Config{Telemetry: observer})
	observer.client = c
	c.state.SetConnecting(2)
	c.state.SetConnected()

	retryDone := make(chan error, 1)
	go func() {
		retryDone <- c.retrySessionErr(context.Background(), func(*session.Session) error {
			return errors.New("rpc failed")
		})
	}()
	select {
	case <-observer.done:
	case <-time.After(time.Second):
		t.Fatal("telemetry callback deadlocked in Close")
	}
	select {
	case err := <-retryDone:
		if err == nil {
			t.Fatal("retrySessionErr unexpectedly succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("retrySessionErr did not return after telemetry Close")
	}
}

func TestConnectionTelemetryRunsAfterAuthCleanupLock(t *testing.T) {
	observer := &closingConnectionTelemetry{done: make(chan struct{})}
	c, _ := NewClient(1, "hash", &Config{Telemetry: observer})
	observer.client = c
	st := populatedAuthStorage(t)
	c.storage = st

	done := make(chan error, 1)
	go func() {
		done <- c.invalidateMainAuth(tgerr.New(406, "AUTH_KEY_DUPLICATED"))
	}()
	select {
	case <-observer.done:
	case <-time.After(time.Second):
		t.Fatal("connection telemetry callback deadlocked in Close")
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrAuthKeyInvalidated) {
			t.Fatalf("invalidateMainAuth() = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("auth invalidation did not return after telemetry Close")
	}
	if key, _ := st.AuthKey(); len(key) != 0 {
		t.Fatalf("auth key length = %d, want 0 before callback", len(key))
	}
}
