package telegram

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
)

type gatedFailDialer struct {
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

type gatedSuccessDialer struct {
	addr    string
	entered chan struct{}
	release chan struct{}
}

type disconnectOrderDialer struct {
	oldClosed *atomic.Bool
	observed  chan bool
}

func (d *disconnectOrderDialer) Dial(string, string, time.Duration) (net.Conn, error) {
	d.observed <- d.oldClosed.Load()
	return nil, errors.New("test dial failure")
}

func (d *gatedSuccessDialer) Dial(network, _ string, timeout time.Duration) (net.Conn, error) {
	select {
	case d.entered <- struct{}{}:
	default:
	}
	<-d.release
	return net.DialTimeout(network, d.addr, timeout)
}

func (d *gatedFailDialer) Dial(string, string, time.Duration) (net.Conn, error) {
	d.calls.Add(1)
	select {
	case d.entered <- struct{}{}:
	default:
	}
	<-d.release
	return nil, errors.New("gated dial failure")
}

func TestConnectSerializesNetworkChange(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{InMemory: true, NoUpdates: true, ReconnectEnabled: false})
	client.updateConfig(func(cfg *Config) { cfg.ReconnectEnabled = false })
	st := NewMemoryStorage()
	_ = st.SetAuthKey(make([]byte, 256))
	_ = st.SetDCID(2)
	client.setTestStorage(st)
	dialer := &gatedFailDialer{entered: make(chan struct{}, 1), release: make(chan struct{})}
	client.setTestDialer(dialer)

	connectDone := make(chan error, 1)
	go func() { connectDone <- client.Connect(time.Second) }()
	select {
	case <-dialer.entered:
	case <-time.After(time.Second):
		t.Fatal("Connect did not enter dial")
	}
	if client.autoConnectMu.TryLock() {
		client.autoConnectMu.Unlock()
		t.Fatal("Connect did not hold the main-session lifecycle gate")
	}
	networkDone := make(chan struct{})
	go func() {
		client.NotifyNetworkChange()
		close(networkDone)
	}()
	select {
	case <-networkDone:
		t.Fatal("NotifyNetworkChange raced an in-flight Connect")
	case <-time.After(50 * time.Millisecond):
	}
	close(dialer.release)
	if err := <-connectDone; err == nil {
		t.Fatal("Connect unexpectedly succeeded")
	}
	select {
	case <-networkDone:
	case <-time.After(time.Second):
		t.Fatal("NotifyNetworkChange did not finish")
	}
	if got := dialer.calls.Load(); got != 1 {
		t.Fatalf("dial calls = %d, want 1", got)
	}
}

func TestCloseWinsAgainstInFlightConnectPublication(t *testing.T) {
	client, srv := newTestClient(1, "hash", Config{NoUpdates: true, ReconnectEnabled: true})
	defer srv.Close()
	dialer := &gatedSuccessDialer{
		addr:    srv.Addr(),
		entered: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	client.setTestDialer(dialer)

	connectDone := make(chan error, 1)
	go func() { connectDone <- client.Connect(5 * time.Second) }()
	select {
	case <-dialer.entered:
	case <-time.After(time.Second):
		t.Fatal("Connect did not enter dial")
	}
	closeDone := make(chan struct{})
	go func() {
		client.Close()
		close(closeDone)
	}()
	deadline := time.Now().Add(time.Second)
	for !client.state.IsClosed() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !client.state.IsClosed() {
		t.Fatal("Close did not publish terminal state")
	}
	close(dialer.release)

	if err := <-connectDone; !errors.Is(err, ErrClientClosed) {
		t.Fatalf("Connect() = %v, want ErrClientClosed", err)
	}
	select {
	case <-closeDone:
	case <-time.After(time.Second):
		t.Fatal("Close did not finish")
	}
	if !client.state.IsClosed() {
		t.Fatalf("state = %v, want closed", client.state.State())
	}
	if client.Session() != nil {
		t.Fatal("in-flight Connect published a session after Close")
	}
	if client.reconnectMgr.IsRunning() {
		t.Fatal("Close left reconnect manager running")
	}
}

func TestWaitForConnectRechecksAfterSignalReplacement(t *testing.T) {
	client, _ := NewClient(1, "hash", nil)
	client.mu.Lock()
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		done <- client.waitForConnect(ctx)
	}()
	<-started
	// Let the waiter pass its initial state checks and block while snapshotting
	// connChanged under c.mu.
	time.Sleep(20 * time.Millisecond)
	client.state.SetConnected()
	close(client.connChanged)
	client.connChanged = make(chan struct{})
	client.mu.Unlock()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("waitForConnect() = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("waitForConnect blocked on replacement signal channel")
	}
}

func TestConnectionReadinessBlocksApplicationRPCs(t *testing.T) {
	client, _ := NewClient(1, "hash", nil)
	client.state.SetConnecting(2)
	client.state.SetConnected()
	sess := new(session.Session)
	client.beginMainReadiness(sess, true)

	readyDone := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() { readyDone <- client.ensureConnectedContext(ctx) }()
	select {
	case err := <-readyDone:
		t.Fatalf("application readiness returned early: %v", err)
	case <-time.After(30 * time.Millisecond):
	}

	authCtx := context.WithValue(context.Background(), authConnectContextKey{}, true)
	if err := client.ensureConnectedContext(authCtx); err != nil {
		t.Fatalf("internal auth readiness bypass = %v", err)
	}
	client.finishMainReadiness(sess, nil)
	select {
	case err := <-readyDone:
		if err != nil {
			t.Fatalf("application readiness = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("application readiness did not unblock")
	}
}

func TestContextAwareAPIStopsWaitingForConnectionReadiness(t *testing.T) {
	client, _ := NewClient(1, "hash", nil)
	client.state.SetConnecting(2)
	client.state.SetConnected()
	client.beginMainReadiness(new(session.Session), true)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.GetQRCodeLoginToken(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetQRCodeLoginToken() = %v, want context.Canceled", err)
	}
}

func TestConnectionReadinessSurvivesReconnectDuringAuthentication(t *testing.T) {
	client, _ := NewClient(1, "hash", nil)
	oldSession := new(session.Session)
	replacement := new(session.Session)
	client.session = oldSession
	client.beginMainReadiness(oldSession, true)
	_, token := client.mainSessionReadiness()
	if token == nil {
		t.Fatal("missing initial readiness token")
	}
	client.detachMainReadinessForReconnect(oldSession)
	client.session = replacement
	client.beginMainReadiness(replacement, false)
	if !client.mainReadinessRequiresAuth(replacement) {
		t.Fatal("replacement lost pending startup authentication")
	}
	current, currentToken := client.mainSessionReadiness()
	if current != replacement || currentToken != token {
		t.Fatal("startup authentication did not hand readiness ownership to replacement")
	}
	client.finishMainReadinessToken(token, nil)
	if err := client.waitMainReadiness(context.Background()); err != nil {
		t.Fatalf("replacement readiness = %v", err)
	}
}

func TestNetworkChangePreservesStartupAuthReadiness(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{
		ReconnectEnabled:     true,
		ReconnectBaseDelay:   time.Hour,
		ReconnectMaxDelay:    time.Hour,
		ReconnectMaxAttempts: 1,
	})
	st := NewMemoryStorage()
	if err := st.SetAuthKey(make([]byte, 256)); err != nil {
		t.Fatal(err)
	}
	if err := st.SetDCID(2); err != nil {
		t.Fatal(err)
	}
	client.storage = st
	oldSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	client.session = oldSession
	client.state.SetConnecting(2)
	client.state.SetConnected()
	client.beginMainReadiness(oldSession, true)
	_, token := client.mainSessionReadiness()
	if token == nil {
		t.Fatal("missing startup-auth readiness token")
	}
	dialer := &gatedFailDialer{entered: make(chan struct{}, 1), release: make(chan struct{})}
	client.setTestDialer(dialer)

	client.NotifyNetworkChange()
	select {
	case <-dialer.entered:
	case <-time.After(time.Second):
		t.Fatal("network-change reconnect did not enter dial")
	}
	client.readyMu.Lock()
	current := client.ready
	closed := current == nil || current.closed
	requiresAuth := current != nil && current.requiresAuth
	client.readyMu.Unlock()
	if current != token || closed || !requiresAuth {
		t.Fatalf("startup readiness after network change = %#v, want open auth token %#v", current, token)
	}

	close(dialer.release)
	client.reconnectMgr.Stop()
}

func TestReconnectStopsAndInvalidatesCachedAuxSessions(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{ReconnectEnabled: false})
	st := NewMemoryStorage()
	aux, err := session.NewSession(session.DataCenter{ID: 4}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	tp := &adapterHTTPTransport{}
	aux.SetTransport(newSessionTransport(tp, nil))
	client.sessions[sessionKey{dcID: 4}] = aux
	client.sessionsGeneration = 7
	client.state.SetConnecting(2)
	client.state.SetConnected()

	client.triggerReconnectMode(errors.New("test reconnect"), true)
	if !tp.closed.Load() {
		t.Fatal("cached auxiliary transport remained open during reconnect")
	}
	client.sessionsMu.Lock()
	defer client.sessionsMu.Unlock()
	if len(client.sessions) != 0 {
		t.Fatal("cached auxiliary session remained published during reconnect")
	}
	if client.sessionsGeneration != 8 {
		t.Fatalf("auxiliary generation = %d, want 8", client.sessionsGeneration)
	}
}

func TestReconnectClosesOldMainBeforeReplacementDial(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{
		ReconnectEnabled:     true,
		ReconnectMaxAttempts: 1,
	})
	st := NewMemoryStorage()
	if err := st.SetAuthKey(make([]byte, 256)); err != nil {
		t.Fatal(err)
	}
	if err := st.SetDCID(2); err != nil {
		t.Fatal(err)
	}
	client.storage = st
	oldSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	tp := &adapterHTTPTransport{}
	oldSession.SetTransport(newSessionTransport(tp, nil))
	client.session = oldSession
	client.state.SetConnecting(2)
	client.state.SetConnected()
	observed := make(chan bool, 1)
	client.setTestDialer(&disconnectOrderDialer{oldClosed: &tp.closed, observed: observed})

	client.triggerReconnectMode(errors.New("test reconnect"), true)
	select {
	case closed := <-observed:
		if !closed {
			t.Fatal("replacement dial began before the old main transport closed")
		}
	case <-time.After(time.Second):
		t.Fatal("replacement dial did not start")
	}
	client.reconnectMgr.Stop()
}

func TestAuthRPCReconnectPolicyDoesNotRequireApplicationRetry(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{
		ReconnectEnabled:    true,
		RetryRPCOnReconnect: false,
	})
	if client.retryRPCOnReconnect(context.Background()) {
		t.Fatal("application RPC unexpectedly retries on reconnect")
	}
	authCtx := context.WithValue(context.Background(), authConnectContextKey{}, true)
	if !client.retryRPCOnReconnect(authCtx) {
		t.Fatal("startup auth RPC did not inherit reconnect retry")
	}
}

func TestStaleSessionExitDoesNotDetachCurrent(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{ReconnectEnabled: false})
	st := NewMemoryStorage()
	_ = st.SetAuthKey(make([]byte, 256))
	oldSession, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatalf("old session: %v", err)
	}
	current, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatalf("current session: %v", err)
	}
	client.session = current
	client.state.SetConnecting(2)
	client.state.SetConnected()

	client.handleMainSessionExit(oldSession)
	if client.Session() != current {
		t.Fatal("stale session exit detached the current session")
	}
	if !client.state.IsConnected() {
		t.Fatal("stale session exit changed the connection state")
	}
}

func TestCleanupSessionsDoesNotRestartReconnect(t *testing.T) {
	client, srv := newTestClient(1, "hash", Config{
		NoUpdates:            true,
		ReconnectEnabled:     true,
		ReconnectBaseDelay:   time.Hour,
		ReconnectMaxDelay:    time.Hour,
		ReconnectMaxAttempts: 1,
	})
	defer srv.Close()
	if err := client.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	client.cleanupSessions(false)
	if client.reconnectMgr.IsRunning() {
		t.Fatal("intentional cleanup restarted reconnect manager")
	}
	if client.state.IsConnected() {
		t.Fatal("client remained connected after cleanup")
	}
	if client.Storage() == nil {
		t.Fatal("cleanupSessions(false) released storage")
	}
}

func TestDisconnectStopsActiveReconnect(t *testing.T) {
	client, _ := NewClient(1, "hash", &Config{
		ReconnectEnabled:   true,
		ReconnectBaseDelay: time.Hour,
		ReconnectMaxDelay:  time.Hour,
	})
	client.storage = NewMemoryStorage()
	client.state.SetConnecting(2)
	client.state.SetReconnecting(errors.New("test reconnect"))
	client.reconnectMgr.Start(context.Background())
	if !client.reconnectMgr.IsRunning() {
		t.Fatal("reconnect manager did not start")
	}

	if err := client.Disconnect(); err != nil {
		t.Fatalf("Disconnect() while reconnecting = %v", err)
	}
	if client.reconnectMgr.IsRunning() {
		t.Fatal("Disconnect left reconnect manager running")
	}
	if client.Storage() != nil {
		t.Fatal("Disconnect left reconnect storage open")
	}
}

func TestReconnectHookCanCloseClient(t *testing.T) {
	client, srv := newTestClient(1, "hash", Config{
		NoUpdates:            true,
		ReconnectEnabled:     true,
		ReconnectBaseDelay:   time.Millisecond,
		ReconnectMaxDelay:    time.Millisecond,
		ReconnectMaxAttempts: 3,
	})
	defer srv.Close()
	if err := client.Connect(5 * time.Second); err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	hookDone := make(chan struct{})
	client.OnReconnect(func(c *Client) {
		c.Close()
		close(hookDone)
	})
	client.triggerReconnectMode(errors.New("test reconnect"), true)
	select {
	case <-hookDone:
	case <-time.After(5 * time.Second):
		t.Fatal("OnReconnect hook deadlocked while closing client")
	}
	if !client.state.IsClosed() {
		t.Fatalf("state = %v, want closed", client.state.State())
	}
	if client.reconnectMgr.IsRunning() {
		t.Fatal("reconnect manager remained running after hook closed client")
	}
}

func TestReconnectManagerConcurrentStartStop(t *testing.T) {
	client, _ := NewClient(1, "hash", nil)
	rm := newReconnectManager(client, backoffConfig{BaseDelay: time.Hour, MaxDelay: time.Hour, Multiplier: 1})
	for range 100 {
		rm.Start(t.Context())
		done := make(chan struct{})
		go func() {
			rm.Stop()
			close(done)
		}()
		rm.Start(t.Context())
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("reconnect manager Stop blocked")
		}
		if rm.IsRunning() {
			t.Fatal("reconnect manager remained running after Stop")
		}
	}
}

func TestReconnectManagerStopWaitsForRetiringGeneration(t *testing.T) {
	rm := newReconnectManager(nil, defaultBackoffConfig)
	done := make(chan struct{})
	rm.mu.Lock()
	rm.done = done
	rm.running.Store(false)
	rm.mu.Unlock()

	stopDone := make(chan struct{})
	go func() {
		rm.Stop()
		close(stopDone)
	}()
	select {
	case <-stopDone:
		t.Fatal("Stop returned before retiring generation finished")
	case <-time.After(30 * time.Millisecond):
	}
	close(done)
	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("Stop did not finish after retiring generation")
	}
}
