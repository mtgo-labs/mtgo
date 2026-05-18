package telegram

import (
	"sync"
	"time"
)

// ConnState represents the current connection state of a Telegram client.
//
// Possible states:
//   - ConnStateDisconnected – client is not connected.
//   - ConnStateConnecting – a connection attempt is in progress.
//   - ConnStateConnected – client is connected and ready to send RPC calls.
//   - ConnStateReconnecting – the connection was lost and automatic reconnect is underway.
//   - ConnStateClosed – the client was explicitly closed and cannot reconnect.
//
// Example:
//
//	state := client.ConnState()
//	switch state {
//	case telegram.ConnStateConnected:
//		fmt.Println("client is online")
//	case telegram.ConnStateDisconnected:
//		fmt.Println("client is offline")
//	default:
//		fmt.Println("client state:", state)
//	}
type ConnState int

const (
	ConnStateDisconnected ConnState = iota
	ConnStateConnecting
	ConnStateConnected
	ConnStateReconnecting
	ConnStateClosed
)

func (s ConnState) String() string {
	switch s {
	case ConnStateDisconnected:
		return "disconnected"
	case ConnStateConnecting:
		return "connecting"
	case ConnStateConnected:
		return "connected"
	case ConnStateReconnecting:
		return "reconnecting"
	case ConnStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// HealthStatus holds a snapshot of the client's connection health metrics.
//
// Example:
//
//	health := client.Health()
//	fmt.Printf("state=%s dc=%d reconnects=%d uptime=%s\n",
//		health.State, health.CurrentDC, health.ReconnectCount,
//		time.Since(health.ConnectedSince).Round(time.Second),
//	)
//	if health.LastError != nil {
//		log.Println("last error:", health.LastError)
//	}
type HealthStatus struct {
	State          ConnState
	CurrentDC      int
	LastReadTime   time.Time
	LastWriteTime  time.Time
	LastPingTime   time.Time
	LastPongTime   time.Time
	ReconnectCount int
	LastError      error
	ConnectedSince time.Time
}

type connStateManager struct {
	mu             sync.RWMutex
	state          ConnState
	currentDC      int
	lastRead       time.Time
	lastWrite      time.Time
	lastPing       time.Time
	lastPong       time.Time
	reconnectCount int
	lastErr        error
	connectedSince time.Time

	// tsMu protects the four timestamp fields from contention with the
	// main mu used for state transitions.
	tsMu sync.Mutex
}

func newConnStateManager() *connStateManager {
	return &connStateManager{
		state: ConnStateDisconnected,
	}
}

func (cs *connStateManager) State() ConnState {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.state
}

func (cs *connStateManager) Health() HealthStatus {
	cs.mu.RLock()
	cs.tsMu.Lock()
	defer cs.tsMu.Unlock()
	defer cs.mu.RUnlock()
	return HealthStatus{
		State:          cs.state,
		CurrentDC:      cs.currentDC,
		LastReadTime:   cs.lastRead,
		LastWriteTime:  cs.lastWrite,
		LastPingTime:   cs.lastPing,
		LastPongTime:   cs.lastPong,
		ReconnectCount: cs.reconnectCount,
		LastError:      cs.lastErr,
		ConnectedSince: cs.connectedSince,
	}
}

func (cs *connStateManager) SetConnecting(dcID int) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.state == ConnStateClosed {
		return ErrClientClosed
	}
	if cs.state == ConnStateConnected || cs.state == ConnStateConnecting {
		return ErrAlreadyConnected
	}
	cs.state = ConnStateConnecting
	cs.currentDC = dcID
	return nil
}

func (cs *connStateManager) SetConnected() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.state = ConnStateConnected
	cs.connectedSince = time.Now()
	cs.lastErr = nil
}

func (cs *connStateManager) SetReconnecting(err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.state == ConnStateClosed {
		return
	}
	cs.state = ConnStateReconnecting
	cs.reconnectCount++
	cs.lastErr = err
}

func (cs *connStateManager) SetDisconnected(err error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.state == ConnStateClosed {
		return
	}
	cs.state = ConnStateDisconnected
	cs.lastErr = err
	cs.connectedSince = time.Time{}
}

func (cs *connStateManager) SetClosed() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.state = ConnStateClosed
	cs.connectedSince = time.Time{}
}

func (cs *connStateManager) RecordRead() {
	cs.tsMu.Lock()
	cs.lastRead = time.Now()
	cs.tsMu.Unlock()
}

func (cs *connStateManager) RecordWrite() {
	cs.tsMu.Lock()
	cs.lastWrite = time.Now()
	cs.tsMu.Unlock()
}

func (cs *connStateManager) RecordPing() {
	cs.tsMu.Lock()
	cs.lastPing = time.Now()
	cs.tsMu.Unlock()
}

func (cs *connStateManager) RecordPong() {
	cs.tsMu.Lock()
	cs.lastPong = time.Now()
	cs.tsMu.Unlock()
}

func (cs *connStateManager) SetDC(dcID int) {
	cs.mu.Lock()
	cs.currentDC = dcID
	cs.mu.Unlock()
}

func (cs *connStateManager) DC() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.currentDC
}

func (cs *connStateManager) IsConnected() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.state == ConnStateConnected
}

func (cs *connStateManager) IsClosed() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.state == ConnStateClosed
}

func (cs *connStateManager) CanReconnect() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.state == ConnStateReconnecting || cs.state == ConnStateDisconnected
}

func (cs *connStateManager) RequireConnected() error {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	switch cs.state {
	case ConnStateConnected:
		return nil
	case ConnStateClosed:
		return ErrClientClosed
	case ConnStateReconnecting:
		return ErrReconnectFailed
	default:
		return ErrNotConnected
	}
}

func (cs *connStateManager) ResetReconnectCount() {
	cs.mu.Lock()
	cs.reconnectCount = 0
	cs.mu.Unlock()
}

// ConnectionState is an alias for connStateManager, exported for use by external consumers.
// It provides thread-safe access to connection state transitions and health reporting.
//
// Example:
//
//	state := client.connState.Health()
//	if state.State == telegram.ConnStateConnected {
//		fmt.Println("connected to DC", state.CurrentDC)
//	}
type ConnectionState = connStateManager

func newConnectionState() *ConnectionState {
	return newConnStateManager()
}

func (c *ConnectionState) isConnected() bool {
	return c.IsConnected()
}

func (c *ConnectionState) setConnected(v bool) {
	if v {
		c.SetConnected()
	} else {
		c.SetDisconnected(nil)
	}
}

func (c *ConnectionState) requireConnected() error {
	return c.RequireConnected()
}
