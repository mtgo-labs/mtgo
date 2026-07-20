package telegram

import (
	"context"
	"fmt"
	"math"
	mrand "math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// isAuthLostError reports whether err indicates the auth key has been
// permanently invalidated (revoked, unregistered, duplicated, or invalid).
// Reconnect attempts stop when this returns true.
func isAuthLostError(err error) bool {
	return tgerr.Is(
		err,
		tgerr.ErrAuthKeyUnregistered,
		tgerr.ErrAuthKeyInvalid,
		tgerr.ErrAuthKeyDuplicated,
		tgerr.ErrSessionRevoked,
	)
}

type backoffConfig struct {
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	MaxAttempts int
	Jitter      float64
	Multiplier  float64
}

var defaultBackoffConfig = backoffConfig{
	BaseDelay:   1 * time.Second,
	MaxDelay:    60 * time.Second,
	MaxAttempts: 0,
	Jitter:      0.1,
	Multiplier:  2.0,
}

const (
	reconnectBurstWindow    = 5 * time.Second
	reconnectBurstThreshold = 10
)

func (c backoffConfig) delay(attempt int) time.Duration {
	if attempt <= 0 {
		return c.BaseDelay
	}
	delay := float64(c.BaseDelay) * math.Pow(c.Multiplier, float64(attempt))
	if delay > float64(c.MaxDelay) {
		delay = float64(c.MaxDelay)
	}
	// Apply jitter: randomize within [1-Jitter, 1+Jitter) to desynchronize
	// reconnect schedules across sessions after mass disconnect events.
	if c.Jitter > 0 {
		delay *= 1 + (mrand.Float64()*2-1)*c.Jitter
	}
	return time.Duration(delay)
}

// PerDCBackoff manages independent backoff per DC.
// Ported from td/td/telegram/net/ConnectionCreator.h:229-232 (ClientInfo::Backoff).
type PerDCBackoff struct {
	backoffs  map[int]*backoffState
	baseDelay time.Duration
	maxDelay  time.Duration
	mu        sync.Mutex
}

type backoffState struct {
	currentDelay time.Duration
	lastFailure  time.Time
	lastSuccess  time.Time
}

// NewPerDCBackoff creates a new per-DC backoff manager.
func NewPerDCBackoff(baseDelay, maxDelay time.Duration) *PerDCBackoff {
	return &PerDCBackoff{
		backoffs:  make(map[int]*backoffState),
		baseDelay: baseDelay,
		maxDelay:  maxDelay,
	}
}

// RecordFailure increases backoff for a DC.
func (b *PerDCBackoff) RecordFailure(dcID int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	bs, ok := b.backoffs[dcID]
	if !ok {
		bs = &backoffState{currentDelay: b.baseDelay}
		b.backoffs[dcID] = bs
	}
	bs.currentDelay = bs.currentDelay * 2
	if bs.currentDelay > b.maxDelay {
		bs.currentDelay = b.maxDelay
	}
	bs.lastFailure = time.Now()
}

// RecordSuccess resets backoff for a DC.
func (b *PerDCBackoff) RecordSuccess(dcID int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	bs, ok := b.backoffs[dcID]
	if !ok {
		bs = &backoffState{currentDelay: b.baseDelay}
		b.backoffs[dcID] = bs
	}
	bs.currentDelay = b.baseDelay
	bs.lastSuccess = time.Now()
	bs.lastFailure = time.Time{} // clear failure timestamp
}

// GetDelay returns the current backoff delay for a DC.
func (b *PerDCBackoff) GetDelay(dcID int) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	bs, ok := b.backoffs[dcID]
	if !ok {
		return b.baseDelay
	}
	return bs.currentDelay
}

// ShouldRetry reports whether enough time has passed since the last failure.
func (b *PerDCBackoff) ShouldRetry(dcID int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	bs, ok := b.backoffs[dcID]
	if !ok {
		return true
	}
	return time.Since(bs.lastFailure) >= bs.currentDelay
}

// Cleanup removes all backoff states.
func (b *PerDCBackoff) Cleanup() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.backoffs = make(map[int]*backoffState)
}

type reconnectManager struct {
	client   *Client
	cfg      backoffConfig
	mu       sync.Mutex
	running  atomic.Bool
	cancel   context.CancelFunc
	done     chan struct{}
	wake     chan struct{}
	attempts int

	burstWindowStart time.Time
	burstCount       int
}

func newReconnectManager(client *Client, cfg backoffConfig) *reconnectManager {
	return &reconnectManager{
		client: client,
		cfg:    cfg,
		wake:   make(chan struct{}, 1),
	}
}

func (c *Client) triggerReconnect(err error) {
	c.triggerReconnectMode(err, false)
}

func (c *Client) triggerReconnectMode(err error, immediate bool) {
	if c.state.IsClosed() {
		return
	}
	c.mu.Lock()
	sess := c.session
	c.session = nil
	c.mu.Unlock()
	if sess != nil {
		sess.Stop()
	}

	if !c.config().ReconnectEnabled {
		c.connMetrics.recordDisconnected(err)
		c.state.SetDisconnected(err)
		return
	}
	c.connMetrics.recordDisconnected(err)
	c.state.SetReconnecting(err)

	c.reconnectMgr.Start(context.Background(), immediate)
}

// NotifyNetworkChange invalidates connection state tied to the previous local
// network and starts an immediate reconnect. Applications should call it when
// their platform network monitor reports an interface or route change.
func (c *Client) NotifyNetworkChange() {
	if c.state.IsClosed() {
		return
	}
	hadResources := c.hasActiveResources()
	if c.connPool != nil {
		c.connPool.Clear()
	}
	if c.dcOptionPool != nil {
		c.dcOptionPool.ResetHealth()
	}
	if !hadResources {
		return
	}
	c.cleanupSessions(false)
	c.triggerReconnectMode(ErrNetworkChanged, true)
}

// signalReconnect closes the current connChanged channel and replaces it with a
// fresh one, waking every RPC caller blocked in waitForConnect. Must be called
// after the session is live and state is Connected.
func (c *Client) signalReconnect() {
	c.mu.Lock()
	close(c.connChanged)
	c.connChanged = make(chan struct{})
	c.mu.Unlock()
}

func (c *Client) reconnectOnce() error {
	// Serialize with ensureConnected's inline AutoConnect path. Without this,
	// the background reconnector and an RPC-triggered inline reconnect race:
	// both create competing sessions, dial transports, and mutate c.session +
	// state concurrently. Last writer wins; the loser's session/transport is
	// leaked and its session-exit watcher fires a cascade of spurious
	// reconnects. See ensureConnected (client.go) which holds the same lock.
	c.autoConnectMu.Lock()
	defer c.autoConnectMu.Unlock()

	// If ensureConnected already reconnected us inline while we waited for
	// the lock, nothing to do.
	if c.state.IsConnected() {
		return nil
	}

	c.mu.Lock()
	st := c.storage
	c.mu.Unlock()
	if st == nil {
		return ErrNotConnected
	}

	dcID, _ := st.DCID()
	if dcID == 0 {
		dcID = 2
	}

	if err := c.state.SetConnecting(dcID); err != nil {
		return err
	}

	dc := session.DataCenter{
		ID:       dcID,
		TestMode: c.config().TestMode,
		IPv6:     c.config().IPv6,
	}
	if dc.Address() == "" {
		return fmt.Errorf("%w: %d", ErrUnknownDC, dcID)
	}

	sess, err := session.NewSession(dc, st, c.config().Device.DeviceModel, c.config().Device.AppVersion, c.config().Device.SystemLangCode, c.config().Device.LangCode)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	configureSessionDispatch(sess, c)

	timeout := 15 * time.Second
	sessionTp, err := c.dialTransport(dc, timeout, c.testDialer)
	if err != nil {
		return err
	}

	sess.SetUpdateHandler(func(obj tg.TLObject) {
		c.processRawUpdate(obj)
	})
	sess.SetOnPanic(func(r any) {
		c.Log.Errorf("session dispatch panic: %v", r)
	})

	c.apiInit.Store(false)

	// Configure session ping intervals from client config before starting.
	configureSessionHealth(sess, c.config(), c.connMetrics)

	if err := sess.Connect(sessionTp, 30*time.Second); err != nil {
		sessionTp.Close()
		return fmt.Errorf("session start: %w", err)
	}

	// Watch for session exit and trigger reconnect when it dies. Register the
	// watcher immediately after Connect succeeds — before publishing the session
	// or running update recovery — so a session that dies during that window is
	// still observed. Otherwise the client can be left stuck reporting
	// "connected" with no reconnect ever firing.
	c.sessionWg.Add(1)
	go func() {
		defer c.sessionWg.Done()
		<-sess.SessionDone()
		if c.state.IsConnected() && !c.state.IsClosed() {
			if source, _, cause := sess.ShutdownCause(); cause != nil {
				c.triggerReconnect(fmt.Errorf("session exited [%s]: %w", source, cause))
			} else {
				c.triggerReconnect(fmt.Errorf("session exited"))
			}
		}
	}()

	c.mu.Lock()
	c.session = sess
	c.mu.Unlock()

	c.state.SetConnected()
	c.state.SetDC(dcID)
	c.state.ResetReconnectCount()

	// Notify reconnect hooks for plugin-driven gap recovery.
	c.refreshDCOptions(context.Background())
	c.fireReconnect()
	c.signalReconnect()

	return nil
}

func (rm *reconnectManager) Start(ctx context.Context, immediate ...bool) {
	if !rm.running.CompareAndSwap(false, true) {
		if len(immediate) > 0 && immediate[0] {
			rm.Wake()
		}
		return
	}
	ctx, rm.cancel = context.WithCancel(ctx)
	rm.done = make(chan struct{})
	rm.mu.Lock()
	rm.attempts = 0
	rm.burstWindowStart = time.Time{}
	rm.burstCount = 0
	rm.mu.Unlock()
	go rm.loop(ctx, len(immediate) > 0 && immediate[0])
}

// Wake interrupts the current reconnect backoff without starting another loop.
func (rm *reconnectManager) Wake() {
	select {
	case rm.wake <- struct{}{}:
	default:
	}
}

func (rm *reconnectManager) Stop() {
	if !rm.running.CompareAndSwap(true, false) {
		return
	}
	if rm.cancel != nil {
		rm.cancel()
	}
	if rm.done != nil {
		<-rm.done
	}
}

func (rm *reconnectManager) IsRunning() bool {
	return rm.running.Load()
}

func (rm *reconnectManager) Attempts() int {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.attempts
}

func (rm *reconnectManager) recordAttempt(now time.Time) (int, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.attempts++
	if rm.burstWindowStart.IsZero() || now.Sub(rm.burstWindowStart) > reconnectBurstWindow {
		rm.burstWindowStart = now
		rm.burstCount = 1
	} else {
		rm.burstCount++
	}

	return rm.attempts, rm.burstCount > reconnectBurstThreshold
}

func (rm *reconnectManager) loop(ctx context.Context, immediate bool) {
	defer func() {
		rm.running.Store(false)
		close(rm.done)
	}()
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()
	for {
		attempt, burstExceeded := rm.recordAttempt(time.Now())
		if burstExceeded {
			err := fmt.Errorf("%w: rapid reconnect loop", ErrReconnectFailed)
			rm.client.Log.Errorf("reconnect exceeded burst threshold (%d attempts in %v), giving up", reconnectBurstThreshold, reconnectBurstWindow)
			rm.client.connMetrics.recordDisconnected(err)
			rm.client.state.SetDisconnected(&ReconnectError{
				Attempts: attempt,
				Err:      err,
			})
			rm.client.signalReconnect()
			return
		}

		if rm.cfg.MaxAttempts > 0 && attempt > rm.cfg.MaxAttempts {
			rm.client.Log.Errorf("reconnect exhausted %d attempts, giving up", attempt-1)
			rm.client.connMetrics.recordDisconnected(ErrReconnectFailed)
			rm.client.state.SetDisconnected(&ReconnectError{
				Attempts: attempt,
				Err:      ErrReconnectFailed,
			})
			rm.client.signalReconnect()
			return
		}

		delay := rm.cfg.delay(attempt)
		if immediate && attempt == 1 {
			delay = 0
		}
		rm.client.Log.Infof("reconnect attempt %d in %v", attempt, delay)

		timer.Reset(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-rm.wake:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		if !rm.client.state.CanReconnect() {
			return
		}

		err := rm.client.reconnectOnce()
		if err == nil {
			rm.client.connMetrics.recordConnected()
			rm.client.Log.Info("reconnected successfully")
			return
		}

		if isAuthLostError(err) {
			rm.client.Log.Errorf("auth key invalid, stopping reconnects: %v", err)
			rm.client.connMetrics.recordDisconnected(err)
			rm.client.state.SetDisconnected(&ReconnectError{
				Attempts: attempt,
				Err:      fmt.Errorf("auth key invalid: %w", err),
			})
			rm.client.signalReconnect()
			return
		}

		rm.client.Log.Warnf("reconnect attempt %d failed: %v", attempt, err)
		rm.client.state.SetReconnecting(err)

		if ctx.Err() != nil {
			return
		}
	}
}
