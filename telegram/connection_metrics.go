package telegram

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type connectionMetrics struct {
	observer Telemetry

	attempts          atomic.Int64
	successes         atomic.Int64
	failures          atomic.Int64
	racedAttempts     atomic.Int64
	racedCandidates   atomic.Int64
	dcConfigRefreshes atomic.Int64
	dcConfigFailures  atomic.Int64
	httpRequests      atomic.Int64
	httpSuccesses     atomic.Int64
	httpFailures      atomic.Int64

	mu                     sync.RWMutex
	lastEndpoint           string
	lastFailureEndpoint    string
	lastFailure            string
	lastReconnectReason    string
	lastConnectLatency     time.Duration
	lastSuccessfulDial     time.Time
	currentConnectedSince  time.Time
	lastConnectionLifetime time.Duration
	lastHTTPEndpoint       string
	lastHTTPLatency        time.Duration
	lastHTTPFailure        string
	pingRTT                time.Duration
	pingRTTVariation       time.Duration
	lastPong               time.Time
}

// ConnectionSnapshot is a read-only snapshot of connection and transport
// decisions useful for production diagnostics.
type ConnectionSnapshot struct {
	Attempts               int64
	Successes              int64
	Failures               int64
	RacedAttempts          int64
	RacedCandidates        int64
	DCConfigRefreshes      int64
	DCConfigFailures       int64
	HTTPRequests           int64
	HTTPSuccesses          int64
	HTTPFailures           int64
	LastEndpoint           string
	LastFailureEndpoint    string
	LastFailure            string
	LastReconnectReason    string
	LastConnectLatency     time.Duration
	LastSuccessfulDial     time.Time
	CurrentConnectedSince  time.Time
	LastConnectionLifetime time.Duration
	LastHTTPEndpoint       string
	LastHTTPLatency        time.Duration
	LastHTTPFailure        string
	PingRTT                time.Duration
	PingRTTVariation       time.Duration
	LastPong               time.Time
}

func newConnectionMetrics(observer ...Telemetry) *connectionMetrics {
	m := &connectionMetrics{}
	if len(observer) > 0 {
		m.observer = observer[0]
	}
	return m
}

func (m *connectionMetrics) observe(kind, endpoint string, started time.Time, err error) {
	if m == nil || m.observer == nil {
		return
	}
	m.observer.ObserveConnection(context.Background(), ConnectionObservation{
		Kind: kind, Endpoint: endpoint, StartedAt: started, EndedAt: time.Now(), Error: err,
	})
}

func (m *connectionMetrics) recordDialStart(endpointCount int) {
	if m == nil {
		return
	}
	m.attempts.Add(1)
	if endpointCount > 1 {
		m.racedAttempts.Add(1)
		m.racedCandidates.Add(int64(endpointCount))
	}
}

func (m *connectionMetrics) recordDialSuccess(endpoint string, latency time.Duration) {
	if m == nil {
		return
	}
	m.successes.Add(1)
	m.mu.Lock()
	m.lastEndpoint = endpoint
	m.lastConnectLatency = latency
	m.lastSuccessfulDial = time.Now()
	m.lastFailure = ""
	m.lastFailureEndpoint = ""
	m.mu.Unlock()
	m.observe("dial.success", endpoint, time.Now().Add(-latency), nil)
}

func (m *connectionMetrics) recordDialFailure(endpoint string, err error) {
	if m == nil {
		return
	}
	m.failures.Add(1)
	m.mu.Lock()
	m.lastFailureEndpoint = endpoint
	if err != nil {
		m.lastFailure = err.Error()
	} else {
		m.lastFailure = ""
	}
	m.mu.Unlock()
	m.observe("dial.failure", endpoint, time.Now(), err)
}

func (m *connectionMetrics) recordConnected() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.currentConnectedSince = time.Now()
	m.mu.Unlock()
	m.observe("connected", "", time.Now(), nil)
}

func (m *connectionMetrics) recordDisconnected(reason error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if !m.currentConnectedSince.IsZero() {
		m.lastConnectionLifetime = time.Since(m.currentConnectedSince)
		m.currentConnectedSince = time.Time{}
	}
	if reason != nil {
		m.lastReconnectReason = reason.Error()
	}
	m.mu.Unlock()
	m.observe("disconnected", "", time.Now(), reason)
}

func (m *connectionMetrics) recordDCConfigRefresh(err error) {
	if m == nil {
		return
	}
	if err != nil {
		m.dcConfigFailures.Add(1)
		m.mu.Lock()
		m.lastFailure = err.Error()
		m.mu.Unlock()
		return
	}
	m.dcConfigRefreshes.Add(1)
}

func (m *connectionMetrics) recordHTTPRequest(endpoint string, latency time.Duration, err error) {
	if m == nil {
		return
	}
	m.httpRequests.Add(1)
	if err == nil {
		m.httpSuccesses.Add(1)
	} else {
		m.httpFailures.Add(1)
	}
	m.mu.Lock()
	m.lastHTTPEndpoint = endpoint
	m.lastHTTPLatency = latency
	if err != nil {
		m.lastHTTPFailure = err.Error()
	} else {
		m.lastHTTPFailure = ""
	}
	m.mu.Unlock()
	m.observe("http.request", endpoint, time.Now().Add(-latency), err)
}

func (m *connectionMetrics) recordPingRTT(rtt, variation time.Duration) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.pingRTT = rtt
	m.pingRTTVariation = variation
	m.lastPong = time.Now()
	m.mu.Unlock()
}

func (m *connectionMetrics) Snapshot() ConnectionSnapshot {
	if m == nil {
		return ConnectionSnapshot{}
	}
	snap := ConnectionSnapshot{
		Attempts:          m.attempts.Load(),
		Successes:         m.successes.Load(),
		Failures:          m.failures.Load(),
		RacedAttempts:     m.racedAttempts.Load(),
		RacedCandidates:   m.racedCandidates.Load(),
		DCConfigRefreshes: m.dcConfigRefreshes.Load(),
		DCConfigFailures:  m.dcConfigFailures.Load(),
		HTTPRequests:      m.httpRequests.Load(),
		HTTPSuccesses:     m.httpSuccesses.Load(),
		HTTPFailures:      m.httpFailures.Load(),
	}
	m.mu.RLock()
	snap.LastEndpoint = m.lastEndpoint
	snap.LastFailureEndpoint = m.lastFailureEndpoint
	snap.LastFailure = m.lastFailure
	snap.LastReconnectReason = m.lastReconnectReason
	snap.LastConnectLatency = m.lastConnectLatency
	snap.LastSuccessfulDial = m.lastSuccessfulDial
	snap.CurrentConnectedSince = m.currentConnectedSince
	snap.LastConnectionLifetime = m.lastConnectionLifetime
	snap.LastHTTPEndpoint = m.lastHTTPEndpoint
	snap.LastHTTPLatency = m.lastHTTPLatency
	snap.LastHTTPFailure = m.lastHTTPFailure
	snap.PingRTT = m.pingRTT
	snap.PingRTTVariation = m.pingRTTVariation
	snap.LastPong = m.lastPong
	m.mu.RUnlock()
	return snap
}
