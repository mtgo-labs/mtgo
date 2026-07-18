package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPSendRecvCopiesPayload(t *testing.T) {
	received := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
			return
		}
		received <- body
		_, _ = w.Write([]byte("response"))
	}))
	defer server.Close()

	transport := newTestHTTP(t, HTTPConfig{URLs: []string{server.URL}})
	defer transport.Close()
	payload := []byte("request")
	if err := transport.Send(bytes.NewBuffer(payload)); err != nil {
		t.Fatalf("Send: %v", err)
	}
	clear(payload)
	if got := <-received; !bytes.Equal(got, []byte("request")) {
		t.Fatalf("request = %q, want request", got)
	}
	got, err := transport.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if !bytes.Equal(got, []byte("response")) {
		t.Fatalf("response = %q, want response", got)
	}
}

func TestHTTPEmptyResponseAndReadDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	transport := newTestHTTP(t, HTTPConfig{URLs: []string{server.URL}})
	defer transport.Close()

	if err := transport.Send(bytes.NewBufferString("request")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if err := transport.SetReadDeadline(time.Now().Add(30 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	if _, err := transport.Recv(); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Recv error = %v, want deadline exceeded", err)
	}
}

func TestHTTPTransportErrorAndResponseLimit(t *testing.T) {
	t.Run("transport error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			var code [4]byte
			errorCode := ErrCodeFlood
			binary.LittleEndian.PutUint32(code[:], uint32(errorCode))
			_, _ = w.Write(code[:])
		}))
		defer server.Close()
		transport := newTestHTTP(t, HTTPConfig{URLs: []string{server.URL}})
		defer transport.Close()
		if err := transport.Send(bytes.NewBufferString("request")); err != nil {
			t.Fatalf("Send: %v", err)
		}
		_, err := transport.Recv()
		if !IsTransportError(err, ErrCodeFlood) {
			t.Fatalf("Recv error = %v, want flood transport error", err)
		}
	})

	t.Run("response limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("12345"))
		}))
		defer server.Close()
		transport := newTestHTTP(t, HTTPConfig{URLs: []string{server.URL}, MaxResponseBytes: 4})
		defer transport.Close()
		if err := transport.Send(bytes.NewBufferString("request")); err != nil {
			t.Fatalf("Send: %v", err)
		}
		if _, err := transport.Recv(); !errors.Is(err, ErrPayloadTooLarge) {
			t.Fatalf("Recv error = %v, want ErrPayloadTooLarge", err)
		}
	})
}

func TestHTTPRejectsExcessConcurrencyAndRedirects(t *testing.T) {
	if _, err := NewHTTP(HTTPConfig{
		URLs:        []string{"http://127.0.0.1/api"},
		MaxInFlight: maxHTTPInFlight + 1,
	}); err == nil {
		t.Fatal("NewHTTP accepted excessive MaxInFlight")
	}

	var redirected atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/target" {
			redirected.Store(true)
			_, _ = w.Write([]byte("unexpected"))
			return
		}
		http.Redirect(w, r, "/target", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	transport := newTestHTTP(t, HTTPConfig{URLs: []string{server.URL}})
	defer transport.Close()
	if err := transport.Send(bytes.NewBufferString("request")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, err := transport.Recv(); err == nil {
		t.Fatal("Recv followed redirect, want status error")
	}
	if redirected.Load() {
		t.Fatal("encrypted POST was forwarded through redirect")
	}
}

func TestHTTPRotatesEndpointAfterFailure(t *testing.T) {
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer failed.Close()
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer healthy.Close()
	transport := newTestHTTP(t, HTTPConfig{URLs: []string{failed.URL, healthy.URL}})
	defer transport.Close()

	if err := transport.Send(bytes.NewBufferString("first")); err != nil {
		t.Fatalf("first Send: %v", err)
	}
	if _, err := transport.Recv(); err == nil {
		t.Fatal("first Recv succeeded, want status error")
	}
	if err := transport.Send(bytes.NewBufferString("second")); err != nil {
		t.Fatalf("second Send: %v", err)
	}
	got, err := transport.Recv()
	if err != nil {
		t.Fatalf("second Recv: %v", err)
	}
	if string(got) != "ok" {
		t.Fatalf("second response = %q, want ok", got)
	}
}

func TestHTTPBoundsConcurrentRequests(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 3)
	var active atomic.Int32
	var maximum atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		current := active.Add(1)
		for old := maximum.Load(); current > old && !maximum.CompareAndSwap(old, current); old = maximum.Load() {
		}
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		active.Add(-1)
	}))
	defer server.Close()
	transport := newTestHTTP(t, HTTPConfig{URLs: []string{server.URL}, MaxInFlight: 2})
	defer transport.Close()

	if err := transport.Send(bytes.NewBufferString("one")); err != nil {
		t.Fatalf("Send one: %v", err)
	}
	if err := transport.Send(bytes.NewBufferString("two")); err != nil {
		t.Fatalf("Send two: %v", err)
	}
	waitSignal(t, started)
	waitSignal(t, started)
	thirdDone := make(chan error, 1)
	go func() { thirdDone <- transport.Send(bytes.NewBufferString("three")) }()
	select {
	case err := <-thirdDone:
		t.Fatalf("third Send returned before a slot was available: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	if err := <-thirdDone; err != nil {
		t.Fatalf("third Send: %v", err)
	}
	if got := maximum.Load(); got != 2 {
		t.Fatalf("maximum concurrent requests = %d, want 2", got)
	}
}

func TestHTTPCloseCancelsRequestsAndRecv(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	transport := newTestHTTP(t, HTTPConfig{URLs: []string{server.URL}})
	if err := transport.Send(bytes.NewBufferString("request")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitSignal(t, started)
	recvDone := make(chan error, 1)
	go func() {
		_, err := transport.Recv()
		recvDone <- err
	}()
	if err := transport.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	close(release)
	if err := <-recvDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("Recv error = %v, want context canceled", err)
	}
	if transport.IsConnected() {
		t.Fatal("transport remains connected after Close")
	}
}

func TestHTTPWaitUsesSinglePollAndDeliversResponse(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{}, 2)
	var active atomic.Int32
	var maximum atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "http_wait" {
			t.Errorf("poll body = %q, want http_wait", body)
		}
		current := active.Add(1)
		for old := maximum.Load(); current > old && !maximum.CompareAndSwap(old, current); old = maximum.Load() {
		}
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		active.Add(-1)
		_, _ = w.Write([]byte("update"))
	}))
	defer server.Close()
	transport := newTestHTTP(t, HTTPConfig{URLs: []string{server.URL}, MaxWait: 1000})
	defer transport.Close()
	frame := func(context.Context) ([]byte, error) { return []byte("http_wait"), nil }
	transport.StartHTTPWait(frame)
	transport.StartHTTPWait(frame)
	waitSignal(t, started)
	select {
	case <-started:
		t.Fatal("multiple http_wait polls started concurrently")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	got, err := transport.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if string(got) != "update" {
		t.Fatalf("response = %q, want update", got)
	}
	if maximum.Load() != 1 {
		t.Fatalf("maximum concurrent polls = %d, want 1", maximum.Load())
	}
}

func TestHTTPWaitDeliversTransportErrorWithoutRetry(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		var code [4]byte
		errorCode := ErrCodeAuthKeyNotFound
		binary.LittleEndian.PutUint32(code[:], uint32(errorCode))
		_, _ = w.Write(code[:])
	}))
	defer server.Close()
	transport := newTestHTTP(t, HTTPConfig{
		URLs:          []string{server.URL},
		MaxWait:       1000,
		RetryInterval: 10 * time.Millisecond,
	})
	defer transport.Close()
	transport.StartHTTPWait(func(context.Context) ([]byte, error) {
		return []byte("http_wait"), nil
	})
	if _, err := transport.Recv(); !IsTransportError(err, ErrCodeAuthKeyNotFound) {
		t.Fatalf("Recv error = %v, want auth-key-not-found transport error", err)
	}
	time.Sleep(30 * time.Millisecond)
	if got := requests.Load(); got != 1 {
		t.Fatalf("poll requests = %d, want 1 after protocol error", got)
	}
}

func TestHTTPScalesAcross1000Sessions(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("pong"))
	}))
	defer server.Close()

	const sessionCount = 1_000
	var wg sync.WaitGroup
	errorsCh := make(chan error, sessionCount)
	for range sessionCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			transport, err := NewHTTP(HTTPConfig{URLs: []string{server.URL}, MaxInFlight: 2})
			if err != nil {
				errorsCh <- err
				return
			}
			defer transport.Close()
			if err := transport.Send(bytes.NewBufferString("ping")); err != nil {
				errorsCh <- err
				return
			}
			response, err := transport.Recv()
			if err != nil {
				errorsCh <- err
				return
			}
			if string(response) != "pong" {
				errorsCh <- errors.New("unexpected response")
			}
		}()
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Errorf("session failed: %v", err)
	}
	if got := requests.Load(); got != sessionCount {
		t.Fatalf("requests = %d, want %d", got, sessionCount)
	}
}

func newTestHTTP(t *testing.T, cfg HTTPConfig) *HTTP {
	t.Helper()
	transport, err := NewHTTP(cfg)
	if err != nil {
		t.Fatalf("NewHTTP: %v", err)
	}
	return transport
}

func waitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for signal")
	}
}
