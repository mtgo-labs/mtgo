package telegram

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type gatedConnectionTelemetry struct {
	entered  chan struct{}
	release  chan struct{}
	observed chan string
}

func (*gatedConnectionTelemetry) ObserveRPC(context.Context, RPCObservation) {}

func (t *gatedConnectionTelemetry) ObserveConnection(_ context.Context, observation ConnectionObservation) {
	if observation.Kind == "first" {
		close(t.entered)
		<-t.release
	}
	t.observed <- observation.Kind
}

type reentrantConnectionTelemetry struct {
	client *Client
	action func(*Client)

	armed atomic.Bool
	count atomic.Int32
	once  sync.Once
	seen  chan struct{}
	done  chan struct{}
}

func (*reentrantConnectionTelemetry) ObserveRPC(context.Context, RPCObservation) {}

func (t *reentrantConnectionTelemetry) ObserveConnection(_ context.Context, observation ConnectionObservation) {
	if observation.Kind != "connected" {
		return
	}
	t.count.Add(1)
	armed := t.armed.Load()
	select {
	case t.seen <- struct{}{}:
	default:
	}
	if armed {
		t.once.Do(func() {
			t.action(t.client)
			close(t.done)
		})
	}
}

func waitConnectionObserverIdle(t *testing.T, metrics *connectionMetrics) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		metrics.observerMu.Lock()
		idle := !metrics.observerRunning && len(metrics.observerQueue) == 0
		metrics.observerMu.Unlock()
		if idle {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("connection telemetry dispatcher did not become idle")
}

func TestConnectionTelemetryDispatchesAsynchronouslyInOrder(t *testing.T) {
	observer := &gatedConnectionTelemetry{
		entered:  make(chan struct{}),
		release:  make(chan struct{}),
		observed: make(chan string, 2),
	}
	metrics := newConnectionMetrics(observer)

	observeReturned := make(chan struct{})
	go func() {
		metrics.observe("first", "", time.Now(), nil)
		close(observeReturned)
	}()
	select {
	case <-observer.entered:
	case <-time.After(time.Second):
		t.Fatal("first connection observation was not dispatched")
	}
	select {
	case <-observeReturned:
	case <-time.After(time.Second):
		close(observer.release)
		t.Fatal("connection observation blocked its lifecycle caller")
	}
	metrics.observe("second", "", time.Now(), nil)
	close(observer.release)

	for _, want := range []string{"first", "second"} {
		select {
		case got := <-observer.observed:
			if got != want {
				t.Fatalf("connection observation = %q, want %q", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("connection observation %q was not dispatched", want)
		}
	}
	waitConnectionObserverIdle(t, metrics)
}

func TestConnectionTelemetryMayDisconnectDuringConnect(t *testing.T) {
	observer := &reentrantConnectionTelemetry{
		action: func(client *Client) { _ = client.Disconnect() },
		seen:   make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
	observer.armed.Store(true)
	client, _ := NewClient(1, "hash", &Config{InMemory: true, Telemetry: observer})
	observer.client = client
	client.storage = NewMemoryStorage()
	client.state.SetConnecting(2)
	client.state.SetConnected()

	// activateMainSession records this event while autoConnectMu is held. Keep
	// that lifecycle boundary here without starting a real session so a
	// synchronous observer would deterministically deadlock in Disconnect.
	client.autoConnectMu.Lock()
	recordDone := make(chan struct{})
	go func() {
		client.connMetrics.recordConnected()
		close(recordDone)
	}()
	select {
	case <-recordDone:
	case <-time.After(time.Second):
		client.autoConnectMu.Unlock()
		t.Fatal("recordConnected blocked while telemetry reentered Disconnect")
	}
	client.autoConnectMu.Unlock()
	select {
	case <-observer.done:
	case <-time.After(5 * time.Second):
		t.Fatal("connection telemetry deadlocked while disconnecting during Connect")
	}
	if client.IsConnected() {
		t.Fatal("client remained connected after telemetry called Disconnect")
	}
	client.Close()
	waitConnectionObserverIdle(t, client.connMetrics)
}

func TestConnectionTelemetryMayCloseDuringReconnect(t *testing.T) {
	observer := &reentrantConnectionTelemetry{
		action: func(client *Client) { client.Close() },
		seen:   make(chan struct{}, 2),
		done:   make(chan struct{}),
	}
	client, server := newTestClient(1, "hash", Config{
		NoUpdates:            true,
		ReconnectEnabled:     true,
		ReconnectBaseDelay:   time.Millisecond,
		ReconnectMaxDelay:    time.Millisecond,
		ReconnectMaxAttempts: 3,
		Telemetry:            observer,
	})
	observer.client = client
	defer server.Close()

	if err := client.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	select {
	case <-observer.seen:
	case <-time.After(time.Second):
		t.Fatal("initial connected observation was not dispatched")
	}
	observer.armed.Store(true)
	client.triggerReconnectMode(errors.New("test reconnect"), true)

	select {
	case <-observer.done:
	case <-time.After(5 * time.Second):
		t.Fatal("connection telemetry deadlocked while closing during reconnect")
	}
	if !client.state.IsClosed() {
		t.Fatalf("state = %v, want closed", client.state.State())
	}
	if client.reconnectMgr.IsRunning() {
		t.Fatal("reconnect manager remained running after telemetry closed the client")
	}
	waitConnectionObserverIdle(t, client.connMetrics)
	if got := observer.count.Load(); got != 2 {
		t.Fatalf("connected observations = %d, want one initial and one reconnect event", got)
	}
}
