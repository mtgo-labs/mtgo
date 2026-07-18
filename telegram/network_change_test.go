package telegram

import (
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
)

func TestNotifyNetworkChangeInvalidatesNetworkState(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true})
	client.updateConfig(func(cfg *Config) { cfg.ReconnectEnabled = false })
	client.state.SetConnecting(2)
	client.state.SetConnected()

	option := session.DataCenter{ID: 2}
	client.dcOptionPool.AddOption(option)
	client.dcOptionPool.RecordFailure(option)
	if _, err := client.dcOptionPool.Candidates(1); !errors.Is(err, session.ErrAllEndpointsFailing) {
		t.Fatalf("Candidates before change = %v", err)
	}

	warm := &poolCloseTracker{}
	client.connPool.Put(2, option, warm)
	client.NotifyNetworkChange()

	if client.connPool.Count() != 0 || warm.closed.Load() != 1 {
		t.Fatal("network change did not close warm connections")
	}
	if _, err := client.dcOptionPool.Candidates(1); err != nil {
		t.Fatalf("endpoint health was not reset: %v", err)
	}
	if client.state.IsConnected() {
		t.Fatal("client remained connected after network change")
	}
	snapshot := client.ConnectionSnapshot()
	if snapshot.LastReconnectReason != ErrNetworkChanged.Error() {
		t.Fatalf("reconnect reason = %q", snapshot.LastReconnectReason)
	}
}

func TestReconnectManagerWakeIsNonBlocking(t *testing.T) {
	rm := newReconnectManager(nil, defaultBackoffConfig)
	done := make(chan struct{})
	go func() {
		for range 1000 {
			rm.Wake()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Wake blocked")
	}
}
