package telegram

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/transport"
	"github.com/mtgo-labs/mtgo/mtproxy"
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

func (c backoffConfig) delay(attempt int) time.Duration {
	if attempt <= 0 {
		return c.BaseDelay
	}
	delay := float64(c.BaseDelay) * math.Pow(c.Multiplier, float64(attempt))
	if delay > float64(c.MaxDelay) {
		delay = float64(c.MaxDelay)
	}
	return time.Duration(delay)
}

type reconnectManager struct {
	client   *Client
	cfg      backoffConfig
	mu       sync.Mutex
	running  atomic.Bool
	cancel   context.CancelFunc
	done     chan struct{}
	attempts int
}

func newReconnectManager(client *Client, cfg backoffConfig) *reconnectManager {
	return &reconnectManager{
		client: client,
		cfg:    cfg,
	}
}

func (c *Client) triggerReconnect(err error) {
	if c.state.IsClosed() {
		return
	}
	if !c.config().ReconnectEnabled {
		c.state.SetDisconnected(err)
		return
	}
	c.state.SetReconnecting(err)

	c.mu.Lock()
	sess := c.session
	c.session = nil
	c.mu.Unlock()
	if sess != nil {
		sess.Stop()
	}

	c.reconnectMgr.Start(context.Background())
}

func (c *Client) reconnectOnce() error {
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
		return fmt.Errorf("unknown dc_id: %d", dcID)
	}

	sess, err := session.NewSession(dc, st, c.config().Device.DeviceModel, c.config().Device.AppVersion, c.config().Device.SystemLangCode, c.config().Device.LangCode)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	configureSessionDispatch(sess, c.cfg, c.Log)

	timeout := 15 * time.Second
	var sessionTp *sessionTransport

	if useWebSocket(c.cfg) {
		wsAddr := wsDCAddress(dc.ID, dc.TestMode, c.config().WebSocketTLS)
		wsCtx, wsCancel := dialerCtx(timeout)
		defer wsCancel()
		wsConn, err := transport.DialWebsocket(wsCtx, wsAddr)
		if err != nil {
			return fmt.Errorf("ws dial %s: %w", wsAddr, err)
		}
		tp := transport.NewTCPIntermediateNoHeader(wsConn)
		if err := tp.Connect(); err != nil {
			wsConn.Close()
			return fmt.Errorf("ws transport handshake: %w", err)
		}
		sessionTp = newSessionTransport(tp, wsConn)
	} else if c.config().MTProxy != nil {
		mpConn, err := mtproxy.Dial(c.config().MTProxy.Addr, c.config().MTProxy.Secret, dc.ID, timeout)
		if err != nil {
			return fmt.Errorf("mtproxy dial: %w", err)
		}
		tp := transport.NewTCPIntermediateNoHeader(mpConn)
		if err := tp.Connect(); err != nil {
			mpConn.Close()
			return fmt.Errorf("mtproxy transport handshake: %w", err)
		}
		sessionTp = newSessionTransport(tp, mpConn)
	} else {
		addr := fmt.Sprintf("%s:%d", dc.Address(), dc.Port())
		if c.config().ServerAddr != "" {
			addr = c.config().ServerAddr
		}

		d := c.dialer
		if c.testDialer != nil {
			d = c.testDialer
		}
		conn, err := d.Dial("tcp", addr, timeout)
		if err != nil {
			return fmt.Errorf("dial %s: %w", addr, err)
		}

		tp, err := newTCPTransport(c.config().TransportMode, conn)
		if err != nil {
			conn.Close()
			return err
		}
		if err := tp.Connect(); err != nil {
			conn.Close()
			return fmt.Errorf("transport handshake: %w", err)
		}
		sessionTp = newSessionTransport(tp, conn)
	}

	sess.SetUpdateHandler(func(obj tg.TLObject) {
		c.processRawUpdate(obj)
	})
	sess.SetOnPanic(func(r any) {
		c.Log.Errorf("session dispatch panic: %v", r)
	})

	c.mu.Lock()
	c.apiInit = false
	c.mu.Unlock()

	// Configure session ping intervals from client config before starting.
	if c.config().HealthPingInterval > 0 {
		sess.SetPingInterval(c.config().HealthPingInterval)
	}
	if c.config().HealthPongTimeout > 0 {
		sess.SetPongTimeout(c.config().HealthPongTimeout)
	}

	if err := sess.Connect(sessionTp, 30*time.Second); err != nil {
		sessionTp.Close()
		return fmt.Errorf("session start: %w", err)
	}

	c.mu.Lock()
	c.session = sess
	c.mu.Unlock()

	c.state.SetConnected()
	c.state.SetDC(dcID)
	c.state.ResetReconnectCount()

	c.sessionWg.Add(1)

	c.mu.RLock()
	um := c.updateManager
	c.mu.RUnlock()
	if um != nil {
		if err := um.OnReconnect(context.Background(), c.Raw()); err != nil {
			c.Log.Warnf("recover updates after reconnect: %v", err)
		}
	}

	// Watch for session exit and trigger reconnect when it dies.
	go func() {
		defer c.sessionWg.Done()
		<-sess.SessionDone()
		if c.state.IsConnected() {
			c.triggerReconnect(fmt.Errorf("session exited"))
		}
	}()

	return nil
}

func (rm *reconnectManager) Start(ctx context.Context) {
	if !rm.running.CompareAndSwap(false, true) {
		return
	}
	ctx, rm.cancel = context.WithCancel(ctx)
	rm.done = make(chan struct{})
	rm.attempts = 0
	go rm.loop(ctx)
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

func (rm *reconnectManager) loop(ctx context.Context) {
	defer close(rm.done)
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()
	for {
		rm.mu.Lock()
		rm.attempts++
		attempt := rm.attempts
		rm.mu.Unlock()

		if rm.cfg.MaxAttempts > 0 && attempt > rm.cfg.MaxAttempts {
			rm.client.Log.Errorf("reconnect exhausted %d attempts, giving up", attempt-1)
			rm.client.state.SetDisconnected(&ReconnectError{
				Attempts: attempt,
				Err:      ErrReconnectFailed,
			})
			rm.running.Store(false)
			return
		}

		delay := rm.cfg.delay(attempt)
		rm.client.Log.Infof("reconnect attempt %d in %v", attempt, delay)

		timer.Reset(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
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
			rm.client.Log.Info("reconnected successfully")
			rm.running.Store(false)
			return
		}

		if isAuthLostError(err) {
			rm.client.Log.Errorf("auth key invalid, stopping reconnects: %v", err)
			rm.client.state.SetDisconnected(&ReconnectError{
				Attempts: attempt,
				Err:      fmt.Errorf("auth key invalid: %w", err),
			})
			rm.running.Store(false)
			return
		}

		rm.client.Log.Warnf("reconnect attempt %d failed: %v", attempt, err)
		rm.client.state.SetReconnecting(err)

		if ctx.Err() != nil {
			return
		}
	}
}
