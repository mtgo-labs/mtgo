package session

import (
	"testing"
	"time"
)

func TestSessionRTTEstimatorAndAdaptiveTimeout(t *testing.T) {
	s, err := NewSession(DataCenter{ID: 2}, newTestStorage(), "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	s.SetPongTimeout(30 * time.Second)

	var callbackRTT, callbackVariation time.Duration
	s.SetOnRTT(func(rtt, variation time.Duration) {
		callbackRTT = rtt
		callbackVariation = variation
	})
	s.recordPingRTT(100 * time.Millisecond)

	snapshot := s.HealthSnapshot()
	if snapshot.RTT != 100*time.Millisecond || snapshot.Variation != 50*time.Millisecond {
		t.Fatalf("first health snapshot = %+v", snapshot)
	}
	if snapshot.LastPong.IsZero() {
		t.Fatal("last pong was not recorded")
	}
	if callbackRTT != snapshot.RTT || callbackVariation != snapshot.Variation {
		t.Fatalf("callback = %v/%v, snapshot = %v/%v", callbackRTT, callbackVariation, snapshot.RTT, snapshot.Variation)
	}
	if got := s.adaptivePongTimeout(); got != time.Second {
		t.Fatalf("adaptive timeout = %v, want 1s floor", got)
	}

	s.recordPingRTT(200 * time.Millisecond)
	snapshot = s.HealthSnapshot()
	if snapshot.RTT != 112500*time.Microsecond || snapshot.Variation != 62500*time.Microsecond {
		t.Fatalf("updated health snapshot = %+v", snapshot)
	}
}

func TestRemovePingCallback(t *testing.T) {
	s, err := NewSession(DataCenter{ID: 2}, newTestStorage(), "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	s.pingCbs = make(map[int64]chan struct{})
	s.pingCbs[42] = make(chan struct{})
	s.removePingCallback(42)
	if len(s.pingCbs) != 0 {
		t.Fatalf("ping callbacks = %d, want 0", len(s.pingCbs))
	}
}
