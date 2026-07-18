package telegram

import (
	"errors"
	"testing"
	"time"
)

func TestConnectionMetricsSnapshot(t *testing.T) {
	m := newConnectionMetrics()
	m.recordDialStart(3)
	m.recordDialFailure("DC2(1.1.1.1:443)", errors.New("timeout"))
	m.recordDialSuccess("DC2(2.2.2.2:443)", 25*time.Millisecond)
	m.recordDCConfigRefresh(nil)
	m.recordDCConfigRefresh(errors.New("config failed"))
	m.recordHTTPRequest("https://dc2.example/api", 30*time.Millisecond, nil)
	m.recordHTTPRequest("https://dc3.example/api", 40*time.Millisecond, errors.New("HTTP 503"))
	m.recordConnected()
	time.Sleep(time.Millisecond)
	m.recordDisconnected(errors.New("session exited"))

	snap := m.Snapshot()
	if snap.Attempts != 1 {
		t.Fatalf("Attempts = %d, want 1", snap.Attempts)
	}
	if snap.RacedAttempts != 1 || snap.RacedCandidates != 3 {
		t.Fatalf("racing = attempts %d candidates %d, want 1/3", snap.RacedAttempts, snap.RacedCandidates)
	}
	if snap.Failures != 1 || snap.Successes != 1 {
		t.Fatalf("success/failure = %d/%d, want 1/1", snap.Successes, snap.Failures)
	}
	if snap.LastEndpoint != "DC2(2.2.2.2:443)" {
		t.Fatalf("LastEndpoint = %q", snap.LastEndpoint)
	}
	if snap.LastConnectLatency != 25*time.Millisecond {
		t.Fatalf("LastConnectLatency = %v", snap.LastConnectLatency)
	}
	if snap.DCConfigRefreshes != 1 || snap.DCConfigFailures != 1 {
		t.Fatalf("config refresh = %d/%d, want 1/1", snap.DCConfigRefreshes, snap.DCConfigFailures)
	}
	if snap.HTTPRequests != 2 || snap.HTTPSuccesses != 1 || snap.HTTPFailures != 1 {
		t.Fatalf("HTTP requests/successes/failures = %d/%d/%d, want 2/1/1", snap.HTTPRequests, snap.HTTPSuccesses, snap.HTTPFailures)
	}
	if snap.LastHTTPEndpoint != "https://dc3.example/api" || snap.LastHTTPLatency != 40*time.Millisecond || snap.LastHTTPFailure != "HTTP 503" {
		t.Fatalf("last HTTP request = %q/%v/%q", snap.LastHTTPEndpoint, snap.LastHTTPLatency, snap.LastHTTPFailure)
	}
	if snap.LastReconnectReason != "session exited" {
		t.Fatalf("LastReconnectReason = %q", snap.LastReconnectReason)
	}
	if snap.LastConnectionLifetime <= 0 {
		t.Fatal("LastConnectionLifetime should be recorded")
	}
}

func TestClientConnectionSnapshotZeroValue(t *testing.T) {
	c, err := NewClient(12345, "hash", &Config{InMemory: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	snap := c.ConnectionSnapshot()
	if snap.Attempts != 0 || snap.Successes != 0 || snap.Failures != 0 {
		t.Fatalf("initial snapshot = %+v, want zero counters", snap)
	}
	if c.LoadSnapshot().Connection != snap {
		t.Fatal("LoadSnapshot should include ConnectionSnapshot")
	}
}
