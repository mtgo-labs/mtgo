package telegram

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
)

type recordingTelemetry struct {
	mu          sync.Mutex
	rpcs        []RPCObservation
	connections []ConnectionObservation
}

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
	err := c.retrySessionErr(context.Background(), func() error {
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

	observer.mu.Lock()
	defer observer.mu.Unlock()
	if len(observer.connections) != 2 {
		t.Fatalf("connection observations = %d, want 2", len(observer.connections))
	}
	if observer.connections[0].Kind != "dial.success" || observer.connections[1].Kind != "disconnected" {
		t.Fatalf("connection observations = %+v", observer.connections)
	}
}
