package telegram

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type poolCloseTracker struct {
	closed atomic.Int32
}

type poolRepairDialer struct {
	calls             atomic.Int32
	plainCalls        atomic.Int32
	sawCanceled       atomic.Bool
	sawDeadline       atomic.Bool
	deadlineRemaining atomic.Int64
}

type gatedPoolRepairDialer struct {
	calls      atomic.Int32
	entered    chan struct{}
	release    chan struct{}
	serverDone chan struct{}
}

func (d *poolRepairDialer) Dial(string, string, time.Duration) (net.Conn, error) {
	d.calls.Add(1)
	d.plainCalls.Add(1)
	return nil, errors.New("pool repair dial failed")
}

func (d *poolRepairDialer) DialContext(ctx context.Context, _, _ string, _ time.Duration) (net.Conn, error) {
	d.calls.Add(1)
	if ctx.Err() != nil {
		d.sawCanceled.Store(true)
	}
	if deadline, ok := ctx.Deadline(); ok {
		d.sawDeadline.Store(true)
		d.deadlineRemaining.Store(time.Until(deadline).Nanoseconds())
	}
	return nil, errors.New("pool repair dial failed")
}

func (d *gatedPoolRepairDialer) Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address, timeout)
}

func (d *gatedPoolRepairDialer) DialContext(ctx context.Context, _, _ string, _ time.Duration) (net.Conn, error) {
	d.calls.Add(1)
	close(d.entered)
	select {
	case <-d.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	clientConn, serverConn := net.Pipe()
	go func() {
		defer close(d.serverDone)
		defer serverConn.Close()
		_, _ = io.CopyN(io.Discard, serverConn, 1)
	}()
	return clientConn, nil
}

func setDCPoolClientConnected(t *testing.T, client *Client, dcID int) {
	t.Helper()
	st := NewMemoryStorage()
	if err := st.SetDCID(dcID); err != nil {
		t.Fatal(err)
	}
	sess, err := session.NewSession(session.DataCenter{ID: dcID}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	client.mu.Lock()
	client.storage = st
	client.session = sess
	client.mu.Unlock()
	client.state.SetDC(dcID)
	client.state.SetConnected()
	t.Cleanup(func() { _ = client.Disconnect() })
}

func (c *poolCloseTracker) Close() error {
	c.closed.Add(1)
	return nil
}

func TestDCSessionStatsTracksHealthAndLatency(t *testing.T) {
	var stats dcSessionStats
	stats.observe(8*time.Millisecond, nil, time.Second)
	if got := time.Duration(stats.latencyEWMA.Load()); got != 8*time.Millisecond {
		t.Fatalf("initial EWMA = %v, want 8ms", got)
	}

	stats.observe(time.Millisecond, session.ErrSessionClosed, time.Second)
	if stats.failures.Load() != 1 {
		t.Fatalf("failures = %d, want 1", stats.failures.Load())
	}
	if stats.unhealthyUntil.Load() <= time.Now().UnixNano() {
		t.Fatal("connection failure should start cooldown")
	}

	stats.observe(16*time.Millisecond, errors.New("rpc application error"), time.Second)
	if stats.failures.Load() != 0 || stats.unhealthyUntil.Load() != 0 {
		t.Fatal("application error should prove transport health")
	}
	if got := time.Duration(stats.latencyEWMA.Load()); got != 9*time.Millisecond {
		t.Fatalf("updated EWMA = %v, want 9ms", got)
	}
}

func TestDCSessionPoolSelectsLeastLoadedThenLowestLatency(t *testing.T) {
	first := &dcSessionEntry{}
	second := &dcSessionEntry{}
	first.stats.inFlight.Store(2)
	second.stats.inFlight.Store(1)
	pool := &dcSessionPool{entries: []*dcSessionEntry{first, second}}

	selected, _ := pool.selectEntry()
	if selected != second {
		t.Fatal("pool did not select least-loaded connection")
	}

	first.stats.inFlight.Store(0)
	second.stats.inFlight.Store(0)
	first.stats.latencyEWMA.Store(int64(20 * time.Millisecond))
	second.stats.latencyEWMA.Store(int64(5 * time.Millisecond))
	selected, _ = pool.selectEntry()
	if selected != second {
		t.Fatal("pool did not select lower-latency connection")
	}
}

func TestDCSessionEntryRetiresAfterActiveLease(t *testing.T) {
	closer := &poolCloseTracker{}
	entry := &dcSessionEntry{closer: closer}
	started := entry.beginRequest()
	entry.retire()
	if closer.closed.Load() != 0 {
		t.Fatal("retired entry closed while request was active")
	}

	entry.endRequest(started, nil, time.Second)
	if closer.closed.Load() != 1 {
		t.Fatalf("close calls = %d, want 1", closer.closed.Load())
	}
	entry.close()
	if closer.closed.Load() != 1 {
		t.Fatalf("close calls after duplicate close = %d, want 1", closer.closed.Load())
	}
}

func TestDCPoolRepairPreservesRPCErrorAndIgnoresCallerCancellation(t *testing.T) {
	cfg := DefaultConfig
	cfg.Timeout = 250 * time.Millisecond
	client, err := NewClient(1, "hash", &cfg)
	if err != nil {
		t.Fatal(err)
	}
	setDCPoolClientConnected(t, client, 2)
	dialer := &poolRepairDialer{}
	client.setTestDialer(dialer)

	originalErr := errors.New("application RPC error")
	active := &dcSessionEntry{
		rpc: tg.NewRPCClient(tg.InvokerFunc(func(
			context.Context,
			tg.TLObject,
			func(*tg.Reader) (tg.TLObject, error),
		) (tg.TLObject, error) {
			return nil, originalErr
		})),
	}
	retired := &dcSessionEntry{}
	retired.retired.Store(true)
	pool := &dcSessionPool{entries: []*dcSessionEntry{active, retired}}
	if !client.dcSessions.putPoolIfGeneration(4, pool, 0) {
		t.Fatal("failed to seed DC pool")
	}
	invoker := &dcPoolInvoker{pool: pool, client: client, dcID: 4}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, gotErr := invoker.RPCInvoke(ctx, nil, nil)
	if !errors.Is(gotErr, originalErr) {
		t.Fatalf("RPC error = %v, want original %v", gotErr, originalErr)
	}
	deadline := time.Now().Add(time.Second)
	for dialer.calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := dialer.calls.Load(); got != 1 {
		t.Fatalf("repair dial calls = %d, want 1", got)
	}
	if dialer.plainCalls.Load() != 0 {
		t.Fatal("repair did not use the context-aware dial path")
	}
	if dialer.sawCanceled.Load() {
		t.Fatal("repair inherited cancellation from the completed RPC")
	}
	if !dialer.sawDeadline.Load() {
		t.Fatal("repair context had no deadline")
	}
	remaining := time.Duration(dialer.deadlineRemaining.Load())
	if remaining <= 0 || remaining > cfg.Timeout {
		t.Fatalf("repair deadline remaining = %v, want (0, %v]", remaining, cfg.Timeout)
	}
}

func TestDCPoolRepairDoesNotDelayRPCResult(t *testing.T) {
	cfg := DefaultConfig
	cfg.Timeout = time.Second
	client, err := NewClient(1, "hash", &cfg)
	if err != nil {
		t.Fatal(err)
	}
	setDCPoolClientConnected(t, client, 2)
	dialer := &gatedPoolRepairDialer{
		entered:    make(chan struct{}),
		release:    make(chan struct{}),
		serverDone: make(chan struct{}),
	}
	client.setTestDialer(dialer)

	originalErr := errors.New("application RPC error")
	active := &dcSessionEntry{
		rpc: tg.NewRPCClient(tg.InvokerFunc(func(
			context.Context,
			tg.TLObject,
			func(*tg.Reader) (tg.TLObject, error),
		) (tg.TLObject, error) {
			return nil, originalErr
		})),
	}
	retired := &dcSessionEntry{}
	retired.retired.Store(true)
	pool := &dcSessionPool{entries: []*dcSessionEntry{active, retired}}
	if !client.dcSessions.putPoolIfGeneration(4, pool, 0) {
		t.Fatal("failed to seed DC pool")
	}
	invoker := &dcPoolInvoker{pool: pool, client: client, dcID: 4}

	done := make(chan error, 1)
	go func() {
		_, invokeErr := invoker.RPCInvoke(context.Background(), nil, nil)
		done <- invokeErr
	}()
	select {
	case gotErr := <-done:
		if !errors.Is(gotErr, originalErr) {
			t.Fatalf("RPC error = %v, want original %v", gotErr, originalErr)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("RPC result waited for background pool repair")
	}
	select {
	case <-dialer.entered:
	case <-time.After(time.Second):
		t.Fatal("background pool repair did not start")
	}
	close(dialer.release)
}

func TestDCPoolRetriesRepairWhenAllEntriesAreRetired(t *testing.T) {
	cfg := DefaultConfig
	cfg.Timeout = 250 * time.Millisecond
	client, err := NewClient(1, "hash", &cfg)
	if err != nil {
		t.Fatal(err)
	}
	setDCPoolClientConnected(t, client, 2)
	dialer := &poolRepairDialer{}
	client.setTestDialer(dialer)

	first := &dcSessionEntry{}
	first.retired.Store(true)
	second := &dcSessionEntry{}
	second.retired.Store(true)
	pool := &dcSessionPool{entries: []*dcSessionEntry{first, second}}
	if !client.dcSessions.putPoolIfGeneration(4, pool, 0) {
		t.Fatal("failed to seed DC pool")
	}
	invoker := &dcPoolInvoker{pool: pool, client: client, dcID: 4}

	for wantCalls := int32(1); wantCalls <= 2; wantCalls++ {
		_, gotErr := invoker.RPCInvoke(context.Background(), nil, nil)
		if !errors.Is(gotErr, ErrNotConnected) {
			t.Fatalf("RPC error = %v, want %v", gotErr, ErrNotConnected)
		}
		if got := dialer.calls.Load(); got != wantCalls {
			t.Fatalf("repair dial calls = %d, want %d", got, wantCalls)
		}
	}
}

func TestDCPoolRepairDoesNotDialAfterDisconnect(t *testing.T) {
	cfg := DefaultConfig
	cfg.AutoConnect = true
	cfg.Timeout = 250 * time.Millisecond
	client, err := NewClient(1, "hash", &cfg)
	if err != nil {
		t.Fatal(err)
	}
	setDCPoolClientConnected(t, client, 2)
	dialer := &poolRepairDialer{}
	client.setTestDialer(dialer)

	invoked := make(chan struct{})
	release := make(chan struct{})
	active := &dcSessionEntry{
		rpc: tg.NewRPCClient(tg.InvokerFunc(func(
			context.Context,
			tg.TLObject,
			func(*tg.Reader) (tg.TLObject, error),
		) (tg.TLObject, error) {
			close(invoked)
			<-release
			return nil, session.ErrSessionClosed
		})),
	}
	retired := &dcSessionEntry{}
	retired.retired.Store(true)
	pool := &dcSessionPool{entries: []*dcSessionEntry{active, retired}}
	if !client.dcSessions.putPoolIfGeneration(4, pool, 0) {
		t.Fatal("failed to seed DC pool")
	}
	invoker := &dcPoolInvoker{pool: pool, client: client, dcID: 4}

	done := make(chan error, 1)
	go func() {
		_, invokeErr := invoker.RPCInvoke(context.Background(), nil, nil)
		done <- invokeErr
	}()
	<-invoked
	if err := client.Disconnect(); err != nil {
		t.Fatal(err)
	}
	close(release)
	if gotErr := <-done; !errors.Is(gotErr, session.ErrSessionClosed) {
		t.Fatalf("RPC error = %v, want %v", gotErr, session.ErrSessionClosed)
	}
	if got := dialer.calls.Load(); got != 0 {
		t.Fatalf("repair dial calls after disconnect = %d, want 0", got)
	}
	if client.IsConnected() {
		t.Fatal("repair reconnected client after disconnect")
	}
}

func TestDCSessionCandidateRejectsLifecycleChangeAfterDial(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatal(err)
	}
	setDCPoolClientConnected(t, client, 2)
	dialer := &gatedPoolRepairDialer{
		entered:    make(chan struct{}),
		release:    make(chan struct{}),
		serverDone: make(chan struct{}),
	}
	client.setTestDialer(dialer)
	_, generation := client.dcSessions.getInitLock(4)

	done := make(chan error, 1)
	go func() {
		_, release, candidateErr := client.createDCSessionCandidate(context.Background(), 4, generation)
		if release != nil {
			release()
		}
		done <- candidateErr
	}()
	<-dialer.entered
	client.state.SetDisconnected(ErrNotConnected)
	close(dialer.release)
	if gotErr := <-done; !errors.Is(gotErr, ErrNotConnected) {
		t.Fatalf("candidate error = %v, want %v", gotErr, ErrNotConnected)
	}
	if got := dialer.calls.Load(); got != 1 {
		t.Fatalf("dial calls = %d, want 1", got)
	}
	select {
	case <-dialer.serverDone:
	case <-time.After(time.Second):
		t.Fatal("candidate transport was not closed")
	}
}

func TestDCConnectionFailureIncludesAuthorizationLoss(t *testing.T) {
	for _, err := range []error{
		tgerr.New(406, "AUTH_KEY_DUPLICATED"),
		tgerr.New(401, "AUTH_KEY_UNREGISTERED"),
		tgerr.New(401, "AUTH_KEY_INVALID"),
		tgerr.New(401, "SESSION_REVOKED"),
		tgerr.New(401, "SESSION_EXPIRED"),
		tgerr.New(401, "AUTH_KEY_PERM_EMPTY"),
		tgerr.New(401, "TEMP_AUTH_KEY_EMPTY"),
		tgerr.New(400, "TEMP_AUTH_KEY_ALREADY_BOUND"),
	} {
		if !isDCConnectionFailure(err) {
			t.Fatalf("%v was not classified as a DC connection failure", err)
		}
	}
}

func TestDCSingleSessionAuthFailureEvictsEntry(t *testing.T) {
	client, _ := NewClient(1, "hash", nil)
	sess, err := session.NewSession(session.DataCenter{ID: 4}, NewMemoryStorage(), "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	closer := &poolCloseTracker{}
	entry := newDCSessionEntry(sess, closer, client)
	if !client.dcSessions.putIfGeneration(4, entry, 0) {
		t.Fatal("failed to seed DC entry")
	}

	invoker := entry.rpc.RPC().(*dcSessionInvoker)
	invoker.retireOnFailure(tgerr.New(406, "AUTH_KEY_DUPLICATED"))
	if _, ok := client.dcSessions.get(4); ok {
		t.Fatal("invalidated DC entry remained cached")
	}
	if got := closer.closed.Load(); got != 1 {
		t.Fatalf("invalidated DC close calls = %d, want 1", got)
	}
}
