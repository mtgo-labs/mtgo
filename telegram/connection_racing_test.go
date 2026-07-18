package telegram

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
)

type racingDialer struct {
	delays map[string]time.Duration

	mu      sync.Mutex
	dialled []string
}

func (d *racingDialer) Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address, timeout)
}

func (d *racingDialer) DialContext(ctx context.Context, _, address string, _ time.Duration) (net.Conn, error) {
	d.mu.Lock()
	d.dialled = append(d.dialled, address)
	d.mu.Unlock()

	timer := time.NewTimer(d.delays[address])
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}

	client, server := net.Pipe()
	go func() {
		_, _ = io.Copy(io.Discard, server)
		_ = server.Close()
	}()
	return client, nil
}

func (d *racingDialer) wasDialled(address string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, candidate := range d.dialled {
		if candidate == address {
			return true
		}
	}
	return false
}

func (d *racingDialer) dialCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.dialled)
}

func TestDialRacedTCPTransportScopesCandidatesAndReturnsFirstSuccess(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	fast := session.DataCenter{ID: 2, IPAddress: "127.0.0.2", PortValue: 443}
	slow := session.DataCenter{ID: 2, IPAddress: "127.0.0.3", PortValue: 443}
	foreign := session.DataCenter{ID: 3, IPAddress: "127.0.0.4", PortValue: 443}
	dialer := &racingDialer{delays: map[string]time.Duration{
		"127.0.0.2:443": 0,
		"127.0.0.3:443": time.Second,
		"127.0.0.4:443": 0,
	}}
	client.dialer = dialer
	client.dcOptionPool.AddOption(fast)
	client.dcOptionPool.AddOption(slow)
	client.dcOptionPool.AddOption(foreign)

	started := time.Now()
	tp, err := client.dialRacedTCPTransport(fast, 2*time.Second)
	if err != nil {
		t.Fatalf("dialRacedTCPTransport: %v", err)
	}
	defer tp.Close()
	if elapsed := time.Since(started); elapsed >= 100*time.Millisecond {
		t.Fatalf("first successful dial returned after %v, want under 100ms", elapsed)
	}
	if dialer.wasDialled("127.0.0.4:443") {
		t.Fatal("racer dialled an endpoint belonging to another DC")
	}
}

func TestDialRacedTCPTransportConcurrentSessions(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	fast := session.DataCenter{ID: 2, IPAddress: "127.0.0.2", PortValue: 443}
	slow := session.DataCenter{ID: 2, IPAddress: "127.0.0.3", PortValue: 443}
	client.dialer = &racingDialer{delays: map[string]time.Duration{
		"127.0.0.2:443": 0,
		"127.0.0.3:443": time.Second,
	}}
	client.dcOptionPool.AddOption(fast)
	client.dcOptionPool.AddOption(slow)

	const sessionCount = 128
	errCh := make(chan error, sessionCount)
	var wg sync.WaitGroup
	wg.Add(sessionCount)
	for range sessionCount {
		go func() {
			defer wg.Done()
			tp, err := client.dialRacedTCPTransport(fast, 2*time.Second)
			if err != nil {
				errCh <- err
				return
			}
			if err := tp.Close(); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent dial: %v", err)
	}
}

func TestDialRacedTCPTransportReusesWarmCandidate(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	endpoint := session.DataCenter{ID: 2, IPAddress: "127.0.0.2", PortValue: 443}
	dialer := &racingDialer{delays: map[string]time.Duration{"127.0.0.2:443": 0}}
	client.dialer = dialer
	client.dcOptionPool.AddOption(endpoint)

	conn, peer := net.Pipe()
	defer peer.Close()
	warm := newSessionTransport(nil, conn)
	client.connPool.Put(endpoint.ID, endpoint, warm)

	got, err := client.dialRacedTCPTransport(endpoint, time.Second)
	if err != nil {
		t.Fatalf("dialRacedTCPTransport: %v", err)
	}
	defer got.Close()
	if got != warm {
		t.Fatal("dialRacedTCPTransport did not return the cached transport")
	}
	if dialer.dialCount() != 0 {
		t.Fatalf("warm cache hit performed %d network dials, want 0", dialer.dialCount())
	}
}

func TestDrainRacingLosersCachesSuccessfulTransport(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	endpoint := session.DataCenter{ID: 2, IPAddress: "127.0.0.2", PortValue: 443}
	conn, peer := net.Pipe()
	defer peer.Close()
	warm := newSessionTransport(nil, conn)
	results := make(chan dialResult, 1)
	results <- dialResult{endpoint: endpoint, st: warm}

	client.drainRacingLosers(results, 1)
	cached, ok := client.connPool.Get(endpoint.ID, endpoint)
	if !ok {
		t.Fatal("successful racing loser was not cached")
	}
	defer cached.Close()
	if cached != warm {
		t.Fatal("cached racing loser transport does not match result")
	}
}

type blockingHandshakeDialer struct{}

func (blockingHandshakeDialer) Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	return blockingHandshakeDialer{}.DialContext(context.Background(), network, address, timeout)
}

func (blockingHandshakeDialer) DialContext(context.Context, string, string, time.Duration) (net.Conn, error) {
	client, _ := net.Pipe()
	return client, nil
}

func TestDialTCPTransportContextCancelsBlockedHandshake(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.dialer = blockingHandshakeDialer{}

	_, err = client.dialTCPTransportContext(
		context.Background(),
		session.DataCenter{ID: 2, IPAddress: "127.0.0.2", PortValue: 443},
		20*time.Millisecond,
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("dialTCPTransportContext error = %v, want context deadline exceeded", err)
	}
}
