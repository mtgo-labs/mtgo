package telegram

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

type healthCheckConfig struct {
	PingInterval time.Duration
	PongTimeout  time.Duration
}

var defaultHealthCheckConfig = healthCheckConfig{
	PingInterval: 60 * time.Second,
	PongTimeout:  30 * time.Second,
}

type healthChecker struct {
	client  *Client
	cfg     healthCheckConfig
	running atomic.Bool
	cancel  context.CancelFunc
	done    chan struct{}
}

func newHealthChecker(client *Client, cfg healthCheckConfig) *healthChecker {
	return &healthChecker{
		client: client,
		cfg:    cfg,
	}
}

func (hc *healthChecker) Start(ctx context.Context) {
	if !hc.running.CompareAndSwap(false, true) {
		return
	}
	ctx, hc.cancel = context.WithCancel(ctx)
	hc.done = make(chan struct{})
	go hc.loop(ctx)
}

func (hc *healthChecker) Stop() {
	if !hc.running.CompareAndSwap(true, false) {
		return
	}
	if hc.cancel != nil {
		hc.cancel()
	}
	if hc.done != nil {
		<-hc.done
	}
}

func (hc *healthChecker) IsRunning() bool {
	return hc.running.Load()
}

func (hc *healthChecker) loop(ctx context.Context) {
	defer close(hc.done)
	ticker := time.NewTicker(hc.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !hc.client.state.IsConnected() || hc.client.state.IsClosed() {
				continue
			}

			hc.client.state.RecordPing()
			hc.client.Log.Debug("health check: sending ping")

			err := hc.sendPing(ctx)
			if err != nil {
				hc.client.Log.Warnf("health check: ping failed: %v", err)
				hc.client.triggerReconnect(err)
				return
			}

			hc.client.state.RecordPong()
		}
	}
}

func (hc *healthChecker) sendPing(ctx context.Context) error {
	c := hc.client
	c.mu.RLock()
	sess := c.session
	c.mu.RUnlock()
	if sess == nil {
		return ErrNotConnected
	}

	pingCtx, cancel := context.WithTimeout(ctx, hc.cfg.PongTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := sess.Invoke(pingCtx, &tg.PingDelayDisconnectRequest{
			PingID:          time.Now().UnixNano(),
			DisconnectDelay: 65,
		}, 1, hc.cfg.PongTimeout)
		done <- err
	}()

	select {
	case <-pingCtx.Done():
		return ErrHealthTimeout
	case err := <-done:
		return err
	}
}
