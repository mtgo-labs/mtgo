package telegram

import (
	"context"
	"errors"
	"fmt"
	"math"
	mrand "math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

var errTemporaryAuthKeyRejected = errors.New("telegram: temporary auth key rejected")

const (
	authKeyOriginUnknown int32 = iota
	authKeyOriginLoaded
	authKeyOriginFresh
)

type authLossState struct {
	cause  error
	result *AuthKeyInvalidatedError
	done   chan struct{}
	closed bool
}

// isAuthLostError reports whether err indicates the auth key has been
// permanently invalidated (revoked, unregistered, duplicated, or invalid).
// Reconnect attempts stop when this returns true.
func isAuthLostError(err error) bool {
	if errors.Is(err, errTemporaryAuthKeyRejected) {
		return false
	}
	return errors.Is(err, session.ErrBindRequiresKeyRotation) ||
		transport.IsTransportError(err, transport.ErrCodeAuthKeyNotFound) || tgerr.Is(
		err,
		tgerr.ErrAuthKeyUnregistered,
		tgerr.ErrAuthKeyInvalid,
		tgerr.ErrAuthKeyDuplicated,
		tgerr.ErrSessionRevoked,
		tgerr.ErrSessionExpired,
	)
}

func isTemporaryAuthRejection(err error) bool {
	return transport.IsTransportError(err, transport.ErrCodeAuthKeyNotFound) || tgerr.Is(
		err,
		tgerr.ErrAuthKeyPermEmpty,
		tgerr.ErrTempAuthKeyEmpty,
		tgerr.ErrTempAuthKeyAlreadyBound,
	)
}

func (c *Client) shouldInvalidateMainAuth(err error) bool {
	c.mu.RLock()
	sess := c.session
	c.mu.RUnlock()
	return c.shouldInvalidateMainAuthFrom(sess, err)
}

func (c *Client) shouldInvalidateMainAuthFrom(sess *session.Session, err error) bool {
	if errors.Is(err, errTemporaryAuthKeyRejected) {
		return false
	}
	if c.activePFSAuthRejection(sess, err) {
		return false
	}
	if errors.Is(err, session.ErrBindRequiresKeyRotation) ||
		transport.IsTransportError(err, transport.ErrCodeAuthKeyNotFound) || tgerr.Is(
		err,
		tgerr.ErrAuthKeyDuplicated,
		tgerr.ErrAuthKeyInvalid,
		tgerr.ErrSessionRevoked,
		tgerr.ErrSessionExpired,
	) {
		return true
	}
	if !tgerr.Is(err, tgerr.ErrAuthKeyUnregistered) {
		return false
	}
	c.mu.RLock()
	authorized := c.me != nil
	st := c.storage
	c.mu.RUnlock()
	if authorized || c.mainAuthKeyOrigin.Load() == authKeyOriginLoaded {
		return true
	}
	if st == nil {
		return authorized
	}
	userID, userErr := st.UserID()
	return userErr == nil && userID != 0
}

func (c *Client) activePFSAuthRejection(sess *session.Session, err error) bool {
	if sess == nil || sess.PFS() == nil || errors.Is(err, errTemporaryAuthKeyRejected) {
		return false
	}
	return transport.IsTransportError(err, transport.ErrCodeAuthKeyNotFound) || tgerr.Is(
		err,
		tgerr.ErrAuthKeyPermEmpty,
		tgerr.ErrTempAuthKeyEmpty,
		tgerr.ErrTempAuthKeyAlreadyBound,
	)
}

func (c *Client) rejectActivePFSKey(sess *session.Session, err error) error {
	if !c.activePFSAuthRejection(sess, err) {
		return err
	}
	rejected := fmt.Errorf("%w: %v", errTemporaryAuthKeyRejected, err)
	// Serialize the ownership check, detach, synchronous stop, and reconnect
	// publication with every other lifecycle transition. A replacement cannot
	// appear after the check and then be marked Reconnecting by this stale
	// temporary-key result.
	c.autoConnectMu.Lock()
	defer c.autoConnectMu.Unlock()
	if c.state.IsClosed() || c.explicitLogout.Load() || c.authLossError() != nil {
		return rejected
	}
	c.mu.Lock()
	if c.session != sess {
		c.mu.Unlock()
		return rejected
	}
	c.session = nil
	c.mu.Unlock()
	c.detachMainReadinessForReconnect(sess)
	sess.Stop()
	c.beginReconnect(rejected, true)
	return rejected
}

func (c *Client) authLossError() error {
	c.authLossMu.Lock()
	defer c.authLossMu.Unlock()
	if c.authLoss == nil {
		return nil
	}
	return c.authLoss.result
}

// advanceAuthGeneration starts a new permanent-credential epoch atomically
// with terminal-loss latching. Callers must abort the transition if a loss has
// already won.
func (c *Client) advanceAuthGeneration() error {
	c.authLossMu.Lock()
	defer c.authLossMu.Unlock()
	if c.authLoss != nil {
		return c.authLoss.result
	}
	c.authGeneration.Add(1)
	return nil
}

// prepareExplicitAuthRecovery permits an explicit Connect/Start after terminal
// auth loss only when cleanup completed and the rejected key is no longer in
// storage. AutoConnect never calls this helper.
func (c *Client) prepareExplicitAuthRecovery() error {
	c.authLossMu.Lock()
	loss := c.authLoss
	c.authLossMu.Unlock()
	if loss == nil {
		return nil
	}
	<-loss.done

	c.authLossMu.Lock()
	defer c.authLossMu.Unlock()
	if c.authLoss != loss {
		return nil
	}
	if loss.result.Cleanup != nil {
		return loss.result
	}
	c.mu.RLock()
	st := c.storage
	c.mu.RUnlock()
	if st == nil {
		// Successful cleanup was already durably synced before Disconnect
		// released storage. A fresh storage instance may now be initialized.
		c.authLoss = nil
		return nil
	}
	authKey, err := st.AuthKey()
	if err != nil {
		return errors.Join(loss.result, fmt.Errorf("verify cleared auth key: %w", err))
	}
	if len(authKey) != 0 {
		return loss.result
	}
	c.authLoss = nil
	return nil
}

func (c *Client) invalidateMainAuth(cause error) error {
	loss, first := c.latchMainAuthLoss(cause)
	return c.completeMainAuthInvalidation(loss, first)
}

func (c *Client) completeMainAuthInvalidation(loss *authLossState, first bool) error {
	if !first {
		return c.authLossResult(loss)
	}
	if c.autoConnectMu.TryLock() {
		c.finishMainAuthInvalidation(loss)
		c.autoConnectMu.Unlock()
		result := c.authLossResult(loss)
		c.connMetrics.recordDisconnected(result)
		return result
	}
	go func() {
		c.autoConnectMu.Lock()
		c.finishMainAuthInvalidation(loss)
		c.autoConnectMu.Unlock()
		c.connMetrics.recordDisconnected(c.authLossResult(loss))
	}()
	return c.authLossResult(loss)
}

func (c *Client) authLossResult(loss *authLossState) *AuthKeyInvalidatedError {
	c.authLossMu.Lock()
	defer c.authLossMu.Unlock()
	return loss.result
}

// invalidateMainAuthLocked latches and completes terminal cleanup while the
// caller owns autoConnectMu. Connection paths use it before releasing lifecycle
// ownership, leaving no window for another dial with the rejected key.
func (c *Client) invalidateMainAuthLocked(cause error) error {
	loss, _ := c.latchMainAuthLoss(cause)
	c.finishMainAuthInvalidation(loss)
	return c.authLossError()
}

func (c *Client) latchMainAuthLoss(cause error) (*authLossState, bool) {
	c.authLossMu.Lock()
	if c.authLoss != nil {
		loss := c.authLoss
		c.authLossMu.Unlock()
		return loss, false
	}
	loss := &authLossState{
		cause:  cause,
		result: &AuthKeyInvalidatedError{Cause: cause},
		done:   make(chan struct{}),
	}
	c.authLoss = loss
	c.authGeneration.Add(1)
	c.authLossMu.Unlock()

	// Latch the terminal state before waiting for lifecycle ownership. This
	// prevents AutoConnect and reconnectOnce from publishing another session.
	c.state.SetDisconnected(loss.result)
	return loss, true
}

// latchMainAuthLossFrom accepts a terminal result only while the permanent-key
// generation that produced it is still current. The source must be a real main
// session; same-key reconnects deliberately remain in the same generation.
func (c *Client) latchMainAuthLossFrom(sess *session.Session, authGeneration uint64, cause error) (*authLossState, bool, bool) {
	c.authLossMu.Lock()
	if sess == nil || c.authGeneration.Load() != authGeneration {
		c.authLossMu.Unlock()
		return nil, false, false
	}
	if c.authLoss != nil {
		loss := c.authLoss
		c.authLossMu.Unlock()
		return loss, false, true
	}
	loss := &authLossState{
		cause:  cause,
		result: &AuthKeyInvalidatedError{Cause: cause},
		done:   make(chan struct{}),
	}
	c.authLoss = loss
	c.authGeneration.Add(1)
	c.authLossMu.Unlock()
	c.state.SetDisconnected(loss.result)
	return loss, true, true
}

// finishMainAuthInvalidation runs with autoConnectMu held. It deliberately
// does not Stop reconnectMgr because it may be executing on that manager's
// goroutine; the terminal latch and disconnected state make the loop exit.
func (c *Client) finishMainAuthInvalidation(loss *authLossState) {
	c.authLossMu.Lock()
	if loss.closed {
		c.authLossMu.Unlock()
		return
	}
	c.authLossMu.Unlock()

	c.abortSessionsLocked()
	if c.dcAuthManager != nil {
		c.dcAuthManager.Cleanup()
	}
	if c.connPool != nil {
		c.connPool.Clear()
	}

	c.mu.RLock()
	st := c.storage
	c.mu.RUnlock()
	var cleanupErr error
	if st == nil {
		cleanupErr = errors.New("auth cleanup: storage unavailable")
	} else {
		cleanupErr = errors.Join(
			wrapAuthCleanupError("auth key", st.SetAuthKey(nil)),
			wrapAuthCleanupError("auth key date", st.SetDate(0)),
			wrapAuthCleanupError("user id", st.SetUserID(0)),
			wrapAuthCleanupError("bot flag", st.SetIsBot(false)),
			wrapAuthCleanupError("first name", st.SetFirstName("")),
			wrapAuthCleanupError("last name", st.SetLastName("")),
			wrapAuthCleanupError("username", st.SetUsername("")),
			wrapAuthCleanupError("update state", st.SetState(nil)),
			wrapAuthCleanupError("storage sync", syncStorage(st)),
		)
	}

	c.peerCacheMu.Lock()
	c.peerCache = make(map[int64]tg.InputPeerClass)
	c.usernameCache = make(map[string]int64)
	c.peerCacheOrder = nil
	c.usernameCacheOrder = nil
	c.peerCacheMu.Unlock()
	c.apiInit.Store(false)
	c.sessionStringInvalidated.Store(true)
	c.mainAuthKeyOrigin.Store(authKeyOriginUnknown)

	result := loss.result
	if cleanupErr != nil {
		result = &AuthKeyInvalidatedError{Cause: loss.cause, Cleanup: cleanupErr}
	}
	c.authLossMu.Lock()
	if c.authLoss == loss && !loss.closed {
		loss.result = result
		loss.closed = true
		close(loss.done)
	}
	c.authLossMu.Unlock()
	c.state.SetDisconnected(result)
	c.signalReconnect()
	if c.Log != nil {
		c.Log.Errorf("auth key invalidated; automatic reconnect disabled: %v", result)
	}
}

func wrapAuthCleanupError(field string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("clear %s: %w", field, err)
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
	restart  bool
	stoppers int
	attempts int

	burstWindowStart time.Time
	burstCount       int
}

func newReconnectManager(client *Client, cfg backoffConfig) *reconnectManager {
	return &reconnectManager{
		client: client,
		cfg:    cfg,
	}
}

func (c *Client) triggerReconnect(err error) {
	c.triggerReconnectMode(err, false)
}

func (c *Client) triggerReconnectMode(err error, immediate bool) {
	if c.state.IsClosed() || c.explicitLogout.Load() {
		return
	}
	c.autoConnectMu.Lock()
	defer c.autoConnectMu.Unlock()
	if c.state.IsClosed() || c.explicitLogout.Load() {
		return
	}
	c.state.setConnected(false)
	c.mu.Lock()
	sess := c.session
	c.session = nil
	c.mu.Unlock()
	if sess != nil {
		c.detachMainReadinessForReconnect(sess)
		sess.Stop()
	}
	c.detachAuxSessions(true)
	if c.dcSessions != nil {
		c.dcSessions.cleanup(true)
	}
	c.beginReconnect(err, immediate)
}

func (c *Client) beginReconnect(err error, immediate bool) {
	if c.state.IsClosed() {
		return
	}
	if c.explicitLogout.Load() {
		c.connMetrics.recordDisconnected(ErrNotConnected)
		c.state.SetDisconnected(ErrNotConnected)
		c.finishCurrentMainReadiness(ErrNotConnected)
		return
	}
	if authErr := c.authLossError(); authErr != nil {
		c.connMetrics.recordDisconnected(authErr)
		c.state.SetDisconnected(authErr)
		c.finishCurrentMainReadiness(authErr)
		return
	}
	if !c.config().ReconnectEnabled {
		c.connMetrics.recordDisconnected(err)
		c.state.SetDisconnected(err)
		c.finishCurrentMainReadiness(err)
		return
	}
	c.connMetrics.recordDisconnected(err)
	c.state.SetReconnecting(err)

	c.reconnectMgr.Start(context.Background(), immediate)
}

// watchMainSession observes one published main session. Ownership is checked
// when it exits so a stale watcher can never detach or stop a replacement.
func (c *Client) watchMainSession(sess *session.Session) {
	c.sessionWg.Go(func() {
		<-sess.SessionDone()
		c.handleMainSessionExit(sess)
	})
}

func (c *Client) ownsMainSession(sess *session.Session) bool {
	if sess == nil || !c.state.IsConnected() {
		return false
	}
	c.mu.RLock()
	owned := c.session == sess
	c.mu.RUnlock()
	return owned
}

func (c *Client) handleReconnectRefreshResult(sess *session.Session, authGeneration uint64, err error) bool {
	if err == nil {
		return true
	}
	if c.activePFSAuthRejection(sess, err) {
		c.rejectActivePFSKey(sess, err)
		return false
	}
	if c.shouldInvalidateMainAuthFrom(sess, err) {
		loss, first, accepted := c.latchMainAuthLossFrom(sess, authGeneration, err)
		if accepted {
			c.completeMainAuthInvalidation(loss, first)
		}
		return false
	}
	return true
}

// latchSessionShutdownAuthLoss classifies a session's recorded terminal cause
// before lifecycle teardown can make its watcher fail the ownership check.
func (c *Client) latchSessionShutdownAuthLoss(sess *session.Session, authGeneration uint64) (*authLossState, bool) {
	if sess == nil {
		return nil, false
	}
	source, _, cause := sess.ShutdownCause()
	if cause == nil {
		return nil, false
	}
	exitErr := fmt.Errorf("session exited [%s]: %w", source, cause)
	return c.latchSessionAuthLossFromError(sess, authGeneration, exitErr)
}

func (c *Client) latchSessionAuthLossFromError(sess *session.Session, authGeneration uint64, exitErr error) (*authLossState, bool) {
	if c.activePFSAuthRejection(sess, exitErr) || !c.shouldInvalidateMainAuthFrom(sess, exitErr) {
		return nil, false
	}
	loss, _, accepted := c.latchMainAuthLossFrom(sess, authGeneration, exitErr)
	return loss, accepted
}

func (c *Client) handleMainSessionExit(sess *session.Session) {
	if sess == nil || c.state.IsClosed() {
		return
	}
	c.mu.Lock()
	if c.session != sess || !c.state.IsConnected() {
		c.mu.Unlock()
		return
	}
	sourceAuthGeneration := c.authGeneration.Load()
	c.session = nil
	c.mu.Unlock()

	exitErr := error(nil)
	if source, _, cause := sess.ShutdownCause(); cause != nil {
		exitErr = fmt.Errorf("session exited [%s]: %w", source, cause)
	} else {
		exitErr = fmt.Errorf("session exited")
	}
	c.handleDetachedMainSessionExit(sess, sourceAuthGeneration, exitErr)
}

func (c *Client) handleDetachedMainSessionExit(sess *session.Session, sourceAuthGeneration uint64, exitErr error) {
	// Classify against the exact session that exited. A missing PFS temp key is
	// recoverable; a permanent-key error must be durably cleared immediately,
	// even when automatic reconnect is disabled.
	c.authDecisionMu.RLock()
	pfsRejected := c.activePFSAuthRejection(sess, exitErr)
	terminalAuthLoss := !pfsRejected && c.shouldInvalidateMainAuthFrom(sess, exitErr)
	var loss *authLossState
	var firstLoss bool
	var accepted bool
	if terminalAuthLoss {
		loss, firstLoss, accepted = c.latchMainAuthLossFrom(sess, sourceAuthGeneration, exitErr)
	}
	c.authDecisionMu.RUnlock()
	if accepted {
		authErr := c.completeMainAuthInvalidation(loss, firstLoss)
		c.finishMainReadiness(sess, authErr)
		return
	}
	// A terminal result from a retired credential generation belongs only to
	// that generation. Do not turn it into a generic reconnect signal that can
	// detach a fresh replacement installed while classification was waiting.
	if terminalAuthLoss {
		return
	}
	if pfsRejected {
		exitErr = fmt.Errorf("%w: %v", errTemporaryAuthKeyRejected, exitErr)
	}
	c.beginReconnectAfterMainSessionExit(sess, sourceAuthGeneration, exitErr)
}

// beginReconnectAfterMainSessionExit publishes a reconnect only if the
// detached session still owns the current lifecycle gap. It cannot block a
// session watcher behind cleanupSessionsLocked's sessionWg.Wait.
func (c *Client) beginReconnectAfterMainSessionExit(sess *session.Session, sourceAuthGeneration uint64, exitErr error) {
	transition := func() {
		defer c.autoConnectMu.Unlock()
		if c.state.IsClosed() || !c.state.IsConnected() ||
			c.explicitLogout.Load() || c.authLossError() != nil ||
			c.authGeneration.Load() != sourceAuthGeneration {
			return
		}
		c.mu.RLock()
		replacementPublished := c.session != nil
		c.mu.RUnlock()
		if replacementPublished {
			return
		}
		c.detachMainReadinessForReconnect(sess)
		c.beginReconnect(exitErr, false)
	}

	if c.autoConnectMu.TryLock() {
		transition()
		return
	}
	go func() {
		c.autoConnectMu.Lock()
		transition()
	}()
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
	c.triggerReconnectMode(ErrNetworkChanged, true)
}

// signalReconnect closes the current connChanged channel and replaces it with a
// fresh one, waking every RPC caller blocked in waitForConnect. Must be called
// after the session is live and state is Connected.
func (c *Client) signalReconnect() {
	c.mu.Lock()
	select {
	case <-c.connChanged:
	default:
		close(c.connChanged)
	}
	if !c.state.IsClosed() {
		c.connChanged = make(chan struct{})
	}
	c.mu.Unlock()
}

func (c *Client) reconnectOnce() (retErr error) {
	// Serialize with ensureConnected's inline AutoConnect path. Without this,
	// the background reconnector and an RPC-triggered inline reconnect race:
	// both create competing sessions, dial transports, and mutate c.session +
	// state concurrently. Last writer wins; the loser's session/transport is
	// leaked and its session-exit watcher fires a cascade of spurious
	// reconnects. See ensureConnected (client.go) which holds the same lock.
	c.autoConnectMu.Lock()
	locked := true
	defer func() {
		if locked {
			if retErr != nil && c.shouldInvalidateMainAuth(retErr) {
				retErr = c.invalidateMainAuthLocked(retErr)
			}
			c.autoConnectMu.Unlock()
		}
	}()
	if err := c.authLossError(); err != nil {
		return err
	}
	if c.explicitLogout.Load() {
		return ErrNotConnected
	}

	// If ensureConnected already reconnected us inline while we waited for
	// the lock, nothing to do.
	if c.state.IsConnected() {
		return nil
	}
	// Reconnect is strictly disconnect-then-connect. Defensively detach and
	// stop any stale main session before a replacement transport is dialed.
	c.mu.Lock()
	previous := c.session
	c.session = nil
	c.mu.Unlock()
	if previous != nil {
		c.detachMainReadinessForReconnect(previous)
		previous.Stop()
	}
	c.detachAuxSessions(true)
	if c.dcSessions != nil {
		c.dcSessions.cleanup(true)
	}
	if !c.state.CanReconnect() {
		return ErrNotConnected
	}

	c.mu.Lock()
	st := c.storage
	c.mu.Unlock()
	if st == nil {
		return ErrNotConnected
	}
	if authKey, err := st.AuthKey(); err == nil && len(authKey) != 0 {
		c.mainAuthKeyOrigin.CompareAndSwap(authKeyOriginUnknown, authKeyOriginLoaded)
	}

	dcID, err := c.initialDCID(st)
	if err != nil {
		return fmt.Errorf("reconnect resolve dc: %w", err)
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
	if err := c.performPFS(sess, st, dc, sessionTp); err != nil {
		sessionTp.Close()
		return err
	}
	usingPFS := sess.PFS() != nil

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
		if usingPFS && isTemporaryAuthRejection(err) {
			return fmt.Errorf("%w: %v", errTemporaryAuthKeyRejected, err)
		}
		return fmt.Errorf("session start: %w", err)
	}
	if err := c.bindPFS(sess); err != nil {
		sess.Stop()
		sessionTp.Close()
		if usingPFS && isTemporaryAuthRejection(err) {
			return fmt.Errorf("%w: %v", errTemporaryAuthKeyRejected, err)
		}
		return err
	}

	if err := c.publishMainSession(sess, dcID, false); err != nil {
		sess.Stop()
		sessionTp.Close()
		return err
	}
	if err := c.activateMainSession(sess); err != nil {
		c.cleanupSessionsLocked(false)
		if usingPFS && isTemporaryAuthRejection(err) {
			return fmt.Errorf("%w: %v", errTemporaryAuthKeyRejected, err)
		}
		return err
	}
	requiresAuth := c.mainReadinessRequiresAuth(sess)
	if !requiresAuth {
		c.finishMainReadiness(sess, nil)
	}
	c.state.ResetReconnectCount()
	c.autoConnectMu.Unlock()
	locked = false

	return nil
}

func (rm *reconnectManager) Start(ctx context.Context, immediate ...bool) {
	wakeNow := len(immediate) > 0 && immediate[0]
	rm.mu.Lock()
	if rm.stoppers > 0 {
		rm.mu.Unlock()
		return
	}
	if rm.running.Load() {
		rm.restart = true
		wake := rm.wake
		rm.mu.Unlock()
		if wakeNow && wake != nil {
			select {
			case wake <- struct{}{}:
			default:
			}
		}
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	wake := make(chan struct{}, 1)
	rm.cancel = cancel
	rm.done = done
	rm.wake = wake
	rm.restart = false
	rm.attempts = 0
	rm.burstWindowStart = time.Time{}
	rm.burstCount = 0
	rm.running.Store(true)
	rm.mu.Unlock()
	go rm.loop(runCtx, wakeNow, done, wake)
}

// Wake interrupts the current reconnect backoff without starting another loop.
func (rm *reconnectManager) Wake() {
	rm.mu.Lock()
	wake := rm.wake
	running := rm.running.Load()
	rm.mu.Unlock()
	if !running || wake == nil {
		return
	}
	select {
	case wake <- struct{}{}:
	default:
	}
}

func (rm *reconnectManager) Stop() {
	rm.mu.Lock()
	rm.stoppers++
	cancel := rm.cancel
	done := rm.done
	if cancel != nil {
		cancel()
	}
	rm.mu.Unlock()
	if done != nil {
		<-done
	}
	rm.mu.Lock()
	rm.stoppers--
	rm.mu.Unlock()
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

// publishLifecycleError mutates connection state only while this reconnect
// generation still owns a reconnectable lifecycle gap. A concurrent Connect
// cannot be overwritten by a stale loop after it publishes a replacement.
func (rm *reconnectManager) publishLifecycleError(done chan struct{}, publish func()) bool {
	rm.client.autoConnectMu.Lock()
	defer rm.client.autoConnectMu.Unlock()
	rm.mu.Lock()
	current := rm.done == done
	rm.mu.Unlock()
	if !current || !rm.client.state.CanReconnect() {
		return false
	}
	publish()
	return true
}

func (rm *reconnectManager) loop(ctx context.Context, immediate bool, done chan struct{}, wake <-chan struct{}) {
	postReconnect := false
	var postSession *session.Session
	defer func() {
		// Hold the auth-decision read barrier before Stop can release waiters.
		// Disconnect then waits for this internal refresh/classification before it
		// closes storage. Telemetry is suppressed below to keep Close hooks
		// reentrant while the barrier is held.
		refreshLocked := false
		rm.mu.Lock()
		current := rm.done == done
		rm.mu.Unlock()
		if postReconnect && current && rm.client.ownsMainSession(postSession) {
			rm.client.authDecisionMu.RLock()
			refreshLocked = true
		}

		rm.mu.Lock()
		current = rm.done == done
		if current {
			rm.cancel = nil
			rm.done = nil
			rm.wake = nil
			rm.restart = false
			rm.running.Store(false)
		}
		close(done)
		rm.mu.Unlock()
		// Reconnect callbacks run only after this manager generation is fully
		// retired. Hooks may safely call Disconnect or Close without waiting on
		// their own reconnect loop.
		refreshOK := false
		pfsRejected := false
		terminalAuthLoss := false
		var refreshLoss *authLossState
		var firstRefreshLoss bool
		var acceptedRefreshLoss bool
		var postRefreshErr error
		if refreshLocked && current && rm.client.ownsMainSession(postSession) {
			refreshAuthGeneration := rm.client.authGeneration.Load()
			rpc := tg.NewRPCClient(&dcSessionInvoker{sess: postSession, client: rm.client, suppressTelemetry: true})
			refreshErr := rm.client.refreshDCOptionsRPC(context.Background(), rpc)
			postRefreshErr = refreshErr
			switch {
			case refreshErr == nil:
				refreshOK = true
			case rm.client.activePFSAuthRejection(postSession, refreshErr):
				pfsRejected = true
			case rm.client.shouldInvalidateMainAuthFrom(postSession, refreshErr):
				terminalAuthLoss = true
				refreshLoss, firstRefreshLoss, acceptedRefreshLoss = rm.client.latchMainAuthLossFrom(
					postSession,
					refreshAuthGeneration,
					refreshErr,
				)
			default:
				refreshOK = true
			}
		}
		if refreshLocked {
			rm.client.authDecisionMu.RUnlock()
		}
		// PFS rejection takes lifecycle ownership only after releasing the auth
		// decision barrier. Disconnect owns the locks in the opposite phase.
		if pfsRejected {
			rm.client.rejectActivePFSKey(postSession, postRefreshErr)
			return
		}
		if terminalAuthLoss {
			if acceptedRefreshLoss {
				rm.client.completeMainAuthInvalidation(refreshLoss, firstRefreshLoss)
			}
			return
		}
		if refreshOK {
			// DC option refresh is best-effort. A terminal auth error latches
			// authLoss and fails ownership; transient refresh failures must not
			// suppress reconnect hooks or update-gap recovery.
			if rm.client.authLossError() == nil && rm.client.ownsMainSession(postSession) {
				rm.client.fireReconnect()
			}
		}
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
			terminalErr := &ReconnectError{
				Attempts: attempt,
				Err:      err,
			}
			rm.publishLifecycleError(done, func() {
				rm.client.connMetrics.recordDisconnected(err)
				rm.client.state.SetDisconnected(terminalErr)
				rm.client.finishCurrentMainReadiness(terminalErr)
				rm.client.signalReconnect()
			})
			return
		}

		if rm.cfg.MaxAttempts > 0 && attempt > rm.cfg.MaxAttempts {
			rm.client.Log.Errorf("reconnect exhausted %d attempts, giving up", attempt-1)
			terminalErr := &ReconnectError{
				Attempts: attempt,
				Err:      ErrReconnectFailed,
			}
			rm.publishLifecycleError(done, func() {
				rm.client.connMetrics.recordDisconnected(ErrReconnectFailed)
				rm.client.state.SetDisconnected(terminalErr)
				rm.client.finishCurrentMainReadiness(terminalErr)
				rm.client.signalReconnect()
			})
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
		case <-wake:
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
			if ctx.Err() != nil {
				return
			}
			rm.mu.Lock()
			restart := rm.restart || !rm.client.state.IsConnected()
			rm.restart = false
			if !restart && rm.done == done {
				// Retire atomically with Start: a session exit after this point
				// observes running=false and creates the next generation.
				rm.running.Store(false)
			}
			rm.mu.Unlock()
			if restart {
				continue
			}
			rm.client.Log.Info("reconnected successfully")
			rm.client.mu.RLock()
			postSession = rm.client.session
			rm.client.mu.RUnlock()
			postReconnect = true
			return
		}

		if isAuthLostError(err) {
			// reconnectOnce classifies terminal errors synchronously while it owns
			// the lifecycle gate. Do not re-latch that completed result after an
			// explicit recovery has advanced to a fresh credential generation.
			if errors.Is(err, ErrAuthKeyInvalidated) {
				return
			}
			if rm.client.shouldInvalidateMainAuth(err) {
				rm.client.invalidateMainAuth(err)
				return
			}
			rm.client.Log.Errorf("auth key invalid, stopping reconnects: %v", err)
			terminalErr := &ReconnectError{
				Attempts: attempt,
				Err:      fmt.Errorf("auth key invalid: %w", err),
			}
			rm.publishLifecycleError(done, func() {
				rm.client.connMetrics.recordDisconnected(err)
				rm.client.state.SetDisconnected(terminalErr)
				rm.client.finishCurrentMainReadiness(terminalErr)
				rm.client.signalReconnect()
			})
			return
		}

		rm.client.Log.Warnf("reconnect attempt %d failed: %v", attempt, err)
		if !rm.publishLifecycleError(done, func() {
			rm.client.state.SetReconnecting(err)
		}) {
			return
		}

		if ctx.Err() != nil {
			return
		}
	}
}
