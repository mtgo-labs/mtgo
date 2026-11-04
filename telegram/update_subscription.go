package telegram

import (
	"context"
	"time"

	"github.com/mtgo-labs/mtgo/tgerr"
)

// Update-subscription restoration.
//
// Per https://core.telegram.org/api/updates, update events are pushed to an
// authorized user's *last active connection*, and "to start receiving updates
// the client needs to init connection and call API method" on it. That
// identification lives in the server-side session. When the server replaces
// the session mid-connection (new_session_created — MTProto server garbage
// collection or reboot) or the client reconnects onto a fresh session, the
// identification and update-subscription state are gone: pings (service
// messages) keep being answered, but no updates are pushed until the client
// re-identifies and calls an API method again. A passive client that only
// listens for updates then goes permanently deaf
// (mtgo-bug-session-death-update-dispatch-deaf.md).

// markUpdateReceived records that an update batch arrived, resetting the
// update-silence watchdog timer.
func (c *Client) markUpdateReceived() {
	c.lastUpdateAt.Store(time.Now().UnixNano())
}

// resubscribeUpdates restores the client's update subscription on the current
// server-side session after reason ("new_session_created", "reconnect",
// "update-silence"). It:
//
//  1. clears the client-level API-init flag so the next RPC re-runs
//     invokeWithLayer+initConnection on the (new) server-side session;
//  2. sends updates.getState — both an API call that re-establishes the push
//     subscription and a re-seed of the update state — unless NoUpdates;
//  3. fires the reconnect hooks so the application's update-recovery plugin
//     can run updates.getDifference for the gap
//     (https://core.telegram.org/api/updates lists server-side session loss
//     as a mandatory gap-recovery trigger).
//
// Single-flight: concurrent triggers coalesce into one resubscribe.
func (c *Client) resubscribeUpdates(reason string) {
	if !c.resubscribing.CompareAndSwap(false, true) {
		c.Log.Debugf("update resubscribe already in flight (trigger: %s)", reason)
		return
	}
	defer c.resubscribing.Store(false)

	// The server-side session was (or may have been) replaced, which resets
	// its identification state server-side. Forget the client-level flag so
	// the next RPC re-identifies the session.
	c.apiInit.Store(false)

	if c.config().NoUpdates {
		c.fireReconnect()
		return
	}

	c.Log.Debugf("update resubscribe (%s): re-identifying session and fetching update state", reason)
	rpc := c.Raw()
	if _, err := rpc.UpdatesGetState(context.Background()); err != nil {
		if rpcErr, ok := tgerr.As(err); ok && rpcErr.Code == 401 {
			c.Log.Debug("update resubscribe skipped: not authorized (", rpcErr.Type, ")")
		} else {
			// Non-fatal: log and continue — the update-silence watchdog
			// retries on the next window.
			c.Log.Warnf("update resubscribe (%s) get state: %v", reason, err)
		}
	} else {
		c.Log.Infof("update resubscribe (%s): session re-identified and update state fetched", reason)
	}
	c.fireReconnect()
}

// startUpdateSilenceWatchdog starts the update-silence loop, which
// re-subscribes when no update has been received for
// Config.UpdateSilenceTimeout. The Telegram update documentation lists "no
// updates for 15 minutes or longer" as a mandatory gap-recovery trigger;
// without it a wedged subscription stays invisible indefinitely because the
// transport, ping loop, and RPCs all remain healthy.
//
// The loop is client-level: it no-ops while the client is disconnected and is
// torn down only on Disconnect/Close. Idempotent.
func (c *Client) startUpdateSilenceWatchdog() {
	cfg := c.config()
	if cfg.NoUpdates || cfg.UpdateSilenceTimeout <= 0 {
		return
	}
	c.postConnectMu.Lock()
	defer c.postConnectMu.Unlock()
	if c.updateSilenceCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	c.updateSilenceCancel = cancel
	c.updateSilenceDone = done
	go c.updateSilenceLoop(ctx, cfg.UpdateSilenceTimeout, done)
	c.Log.Debugf("update silence watchdog started (%v)", cfg.UpdateSilenceTimeout)
}

// stopUpdateSilenceWatchdog cancels the update-silence loop and waits for it
// to exit. Must NOT be called while holding autoConnectMu: the loop may be
// blocked acquiring that mutex inside an RPC.
func (c *Client) stopUpdateSilenceWatchdog() {
	c.postConnectMu.Lock()
	cancel := c.updateSilenceCancel
	done := c.updateSilenceDone
	c.updateSilenceCancel = nil
	c.updateSilenceDone = nil
	c.postConnectMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// updateSilenceLoop ticks until ctx is cancelled and re-subscribes once per
// silence window while the client stays connected.
func (c *Client) updateSilenceLoop(ctx context.Context, timeout time.Duration, done chan struct{}) {
	defer close(done)
	interval := timeout / 4
	if interval < 25*time.Millisecond {
		interval = 25 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !c.state.IsConnected() {
				continue
			}
			last := c.lastUpdateAt.Load()
			if last == 0 || time.Since(time.Unix(0, last)) < timeout {
				continue
			}
			// Stamp first so one missed window triggers exactly one
			// resubscribe, not a storm.
			c.markUpdateReceived()
			c.Log.Warnf("no updates for %v — re-establishing update subscription", timeout)
			c.resubscribeUpdates("update-silence")
		}
	}
}
