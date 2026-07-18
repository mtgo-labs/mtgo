package telegram

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
)

type adapterHTTPTransport struct {
	closed        atomic.Bool
	readDeadline  time.Time
	writeDeadline time.Time
	waitStarted   atomic.Bool
}

func (*adapterHTTPTransport) Send(*bytes.Buffer) error { return nil }
func (*adapterHTTPTransport) Recv() ([]byte, error)    { return nil, nil }
func (t *adapterHTTPTransport) Close() error {
	t.closed.Store(true)
	return nil
}
func (t *adapterHTTPTransport) IsConnected() bool { return !t.closed.Load() }
func (t *adapterHTTPTransport) SetWriteDeadline(deadline time.Time) error {
	t.writeDeadline = deadline
	return nil
}
func (t *adapterHTTPTransport) SetReadDeadline(deadline time.Time) error {
	t.readDeadline = deadline
	return nil
}
func (*adapterHTTPTransport) HTTPWaitParams() (maxDelay, waitAfter, maxWait int32) {
	return 1, 2, 3
}
func (t *adapterHTTPTransport) StartHTTPWait(frame func(context.Context) ([]byte, error)) {
	t.waitStarted.Store(true)
}

func TestSessionTransportDelegatesHTTPCapabilities(t *testing.T) {
	inner := &adapterHTTPTransport{}
	transport := newSessionTransport(inner, nil)
	if !transport.IsConnected() {
		t.Fatal("IsConnected did not delegate to HTTP transport")
	}
	readDeadline := time.Now().Add(time.Second)
	writeDeadline := readDeadline.Add(time.Second)
	if err := transport.SetReadDeadline(readDeadline); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	if err := transport.SetWriteDeadline(writeDeadline); err != nil {
		t.Fatalf("SetWriteDeadline: %v", err)
	}
	if !inner.readDeadline.Equal(readDeadline) || !inner.writeDeadline.Equal(writeDeadline) {
		t.Fatal("deadlines were not delegated")
	}
	maxDelay, waitAfter, maxWait, enabled := transport.HTTPWaitParams()
	if !enabled || maxDelay != 1 || waitAfter != 2 || maxWait != 3 {
		t.Fatalf("HTTPWaitParams = %d/%d/%d/%v", maxDelay, waitAfter, maxWait, enabled)
	}
	transport.StartHTTPWait(func(context.Context) ([]byte, error) { return nil, nil })
	if !inner.waitStarted.Load() {
		t.Fatal("StartHTTPWait was not delegated")
	}
	if err := transport.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if transport.IsConnected() {
		t.Fatal("transport remains connected after Close")
	}
}

func TestClientHTTPTransportCustomEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api" {
			t.Errorf("path = %q, want /api", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(append([]byte("response:"), body...))
	}))
	defer server.Close()

	cfg := &Config{HTTPTransport: &HTTPTransportConfig{URLs: []string{server.URL + "/api"}}}
	client, err := NewClient(1, "hash", cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	transport, err := client.newHTTPTransport(
		session.DataCenter{ID: 2},
		time.Second,
		client.config().HTTPTransport,
		client.dialer,
	)
	if err != nil {
		t.Fatalf("newHTTPTransport: %v", err)
	}
	defer transport.Close()
	if err := transport.Send([]byte("request")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	response, err := transport.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if string(response) != "response:request" {
		t.Fatalf("response = %q, want response:request", response)
	}
	snapshot := client.ConnectionSnapshot()
	if snapshot.HTTPRequests != 1 || snapshot.HTTPSuccesses != 1 || snapshot.HTTPFailures != 0 {
		t.Fatalf("HTTP metrics = %d/%d/%d, want 1/1/0", snapshot.HTTPRequests, snapshot.HTTPSuccesses, snapshot.HTTPFailures)
	}
	if snapshot.LastHTTPEndpoint != server.URL+"/api" || snapshot.LastHTTPLatency <= 0 {
		t.Fatalf("last HTTP endpoint/latency = %q/%v", snapshot.LastHTTPEndpoint, snapshot.LastHTTPLatency)
	}
}

func TestClientRejectsConflictingHTTPTransport(t *testing.T) {
	client, err := NewClient(1, "hash", &Config{
		HTTPTransport: &HTTPTransportConfig{URLs: []string{"http://127.0.0.1/api"}},
		WebSocket:     true,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.dialTransport(session.DataCenter{ID: 2}, time.Second, nil)
	if err == nil {
		t.Fatal("dialTransport accepted HTTPTransport with WebSocket")
	}
}
