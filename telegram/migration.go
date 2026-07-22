package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/internal/storage"
)

const primaryMigrationTimeout = 30 * time.Second

type migrationCall struct {
	targetDC int
	done     chan struct{}
	err      error
}

// migrationCoordinator serializes primary-DC changes and coalesces callers
// redirected to the same DC. Its zero value is ready for use.
type migrationCoordinator struct {
	mu      sync.Mutex
	active  *migrationCall
	waiters int
}

func (m *migrationCoordinator) Do(ctx context.Context, targetDC int, migrate func(context.Context) error) error {
	for {
		m.mu.Lock()
		if active := m.active; active != nil {
			m.waiters++
			m.mu.Unlock()
			var err error
			select {
			case <-active.done:
				if active.targetDC == targetDC {
					err = active.err
				}
			case <-ctx.Done():
				err = ctx.Err()
			}
			m.mu.Lock()
			m.waiters--
			m.mu.Unlock()
			if err != nil || active.targetDC == targetDC {
				return err
			}
			continue
		}

		call := &migrationCall{targetDC: targetDC, done: make(chan struct{})}
		m.active = call
		m.mu.Unlock()

		call.err = migrate(ctx)

		m.mu.Lock()
		m.active = nil
		close(call.done)
		m.mu.Unlock()
		return call.err
	}
}

type primaryMigrationSnapshot struct {
	dcID                     int
	authKey                  []byte
	date                     int
	userID                   int64
	configDC                 int
	authKeyOrigin            int32
	sessionStringInvalidated bool
}

func capturePrimaryMigration(st storage.Storage, configDC int) (primaryMigrationSnapshot, error) {
	dcID, err := st.DCID()
	if err != nil {
		return primaryMigrationSnapshot{}, fmt.Errorf("read dc_id: %w", err)
	}
	authKey, err := st.AuthKey()
	if err != nil {
		return primaryMigrationSnapshot{}, fmt.Errorf("read auth key: %w", err)
	}
	userID, err := st.UserID()
	if err != nil {
		return primaryMigrationSnapshot{}, fmt.Errorf("read user id: %w", err)
	}
	date, err := st.Date()
	if err != nil {
		return primaryMigrationSnapshot{}, fmt.Errorf("read auth key date: %w", err)
	}
	return primaryMigrationSnapshot{
		dcID:     dcID,
		authKey:  append([]byte(nil), authKey...),
		date:     date,
		userID:   userID,
		configDC: configDC,
	}, nil
}

func applyPrimaryMigration(st storage.Storage, targetDC int) error {
	if err := st.SetAuthKey(nil); err != nil {
		return fmt.Errorf("clear auth key: %w", err)
	}
	if err := st.SetDate(0); err != nil {
		return fmt.Errorf("clear auth key date: %w", err)
	}
	if err := st.SetUserID(0); err != nil {
		return fmt.Errorf("clear user id: %w", err)
	}
	if err := syncStorage(st); err != nil {
		return fmt.Errorf("sync cleared auth: %w", err)
	}
	if err := st.SetDCID(targetDC); err != nil {
		return fmt.Errorf("save dc_id: %w", err)
	}
	if err := syncStorage(st); err != nil {
		return fmt.Errorf("sync dc_id: %w", err)
	}
	return nil
}

func (c *Client) restorePrimaryMigration(st storage.Storage, snapshot primaryMigrationSnapshot) error {
	// Latching auth loss and restoring the snapshot are mutually exclusive.
	// If loss wins, keep the credentials cleared. If restore wins, a subsequent
	// loss cannot latch until the writes finish and will clear them afterward.
	c.authLossMu.Lock()
	defer c.authLossMu.Unlock()
	if c.authLoss != nil {
		return restorePrimaryMigrationDC(st, snapshot)
	}

	if err := clearPrimaryMigrationCredentials(st); err != nil {
		return err
	}
	if err := restorePrimaryMigrationDCID(st, snapshot); err != nil {
		return err
	}
	authKeyErr := wrapMigrationRestoreError("auth key", st.SetAuthKey(snapshot.authKey))
	dateErr := wrapMigrationRestoreError("auth key date", st.SetDate(snapshot.date))
	userIDErr := wrapMigrationRestoreError("user id", st.SetUserID(snapshot.userID))
	syncErr := wrapMigrationRestoreError("storage sync", syncStorage(st))
	if authKeyErr == nil && dateErr == nil && userIDErr == nil && syncErr == nil {
		c.mainAuthKeyOrigin.Store(snapshot.authKeyOrigin)
		c.sessionStringInvalidated.Store(snapshot.sessionStringInvalidated)
	}
	return errors.Join(authKeyErr, dateErr, userIDErr, syncErr)
}

func restorePrimaryMigrationDC(st storage.Storage, snapshot primaryMigrationSnapshot) error {
	if err := clearPrimaryMigrationCredentials(st); err != nil {
		return err
	}
	return restorePrimaryMigrationDCID(st, snapshot)
}

func clearPrimaryMigrationCredentials(st storage.Storage) error {
	authKeyErr := wrapMigrationRestoreError("target auth key", st.SetAuthKey(nil))
	dateErr := wrapMigrationRestoreError("target auth key date", st.SetDate(0))
	userIDErr := wrapMigrationRestoreError("target user id", st.SetUserID(0))
	if err := errors.Join(authKeyErr, dateErr, userIDErr); err != nil {
		return err
	}
	return wrapMigrationRestoreError("cleared target credentials sync", syncStorage(st))
}

func restorePrimaryMigrationDCID(st storage.Storage, snapshot primaryMigrationSnapshot) error {
	if err := st.SetDCID(snapshot.dcID); err != nil {
		return wrapMigrationRestoreError("dc_id", err)
	}
	return wrapMigrationRestoreError("dc_id sync", syncStorage(st))
}

func wrapMigrationRestoreError(field string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("restore %s: %w", field, err)
}

func migrationTimeout(ctx context.Context) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, context.DeadlineExceeded
		}
		return min(remaining, primaryMigrationTimeout), nil
	}
	return primaryMigrationTimeout, nil
}

func (c *Client) switchPrimaryDC(
	ctx context.Context,
	targetDC int,
	st storage.Storage,
	connect func(time.Duration) error,
) error {
	timeout, err := migrationTimeout(ctx)
	if err != nil {
		return err
	}
	if c.reconnectMgr != nil {
		c.reconnectMgr.Stop()
	}
	c.autoConnectMu.Lock()
	c.mu.RLock()
	sourceSession := c.session
	c.mu.RUnlock()
	sourceAuthGeneration := c.authGeneration.Load()
	if sourceSession != nil {
		sourceSession.Stop()
	}
	c.authDecisionMu.Lock()
	if loss, accepted := c.latchSessionShutdownAuthLoss(sourceSession, sourceAuthGeneration); accepted {
		c.finishMainAuthInvalidation(loss)
		authErr := c.authLossResult(loss)
		c.authDecisionMu.Unlock()
		c.autoConnectMu.Unlock()
		return authErr
	}
	snapshot, err := capturePrimaryMigration(st, c.config().DC)
	if err == nil {
		snapshot.authKeyOrigin = c.mainAuthKeyOrigin.Load()
		snapshot.sessionStringInvalidated = c.sessionStringInvalidated.Load()
		err = c.advanceAuthGeneration()
	}
	if err != nil {
		c.authDecisionMu.Unlock()
		c.autoConnectMu.Unlock()
		return err
	}

	// Snapshot, generation advance, and source detach are one authorization
	// decision. A concurrent successful login can be entirely before or after
	// the migration, but can never be omitted from the rollback snapshot.
	c.sessionStringInvalidated.Store(true)
	c.cleanupSessionsLockedMode(false, false)
	c.authDecisionMu.Unlock()
	c.sessionWg.Wait()
	err = applyPrimaryMigration(st, targetDC)
	if err != nil {
		c.autoConnectMu.Unlock()
		return errors.Join(err, c.rollbackPrimaryMigration(st, snapshot, connect, nil))
	}
	c.updateConfig(func(cfg *Config) { cfg.DC = targetDC })

	c.migratingDC.Store(true)
	var connectErr error
	if connect == nil {
		connectErr = c.connectTransportLocked(timeout)
		if connectErr != nil && c.shouldInvalidateMainAuth(connectErr) {
			connectErr = c.invalidateMainAuthLocked(connectErr)
		}
		var sess *session.Session
		var ready *connectionReadiness
		if connectErr == nil {
			sess, ready = c.mainSessionReadiness()
		}
		c.autoConnectMu.Unlock()
		if connectErr == nil {
			connectErr = c.completeConnect(timeout, sess, ready)
		}
	} else {
		c.autoConnectMu.Unlock()
		connectErr = connect(timeout)
	}
	c.migratingDC.Store(false)
	if connectErr == nil {
		return nil
	}

	rollbackErr := c.rollbackPrimaryMigration(st, snapshot, connect, connectErr)
	return errors.Join(fmt.Errorf("connect target dc: %w", connectErr), rollbackErr)
}

func (c *Client) rollbackPrimaryMigration(
	st storage.Storage,
	snapshot primaryMigrationSnapshot,
	connect func(time.Duration) error,
	targetConnectErr error,
) error {
	c.migratingDC.Store(false)
	if c.reconnectMgr != nil {
		c.reconnectMgr.Stop()
	}
	c.autoConnectMu.Lock()
	// Retire the target transport before starting the restored source credential
	// epoch. RequestStop closes transports synchronously; authDecisionMu then
	// drains every target RPC/classifier before restoration.
	c.cleanupSessionsLockedMode(false, false)
	c.updateConfig(func(cfg *Config) { cfg.DC = snapshot.configDC })
	c.authDecisionMu.Lock()
	generationErr := c.advanceAuthGeneration()
	var restoreErr error
	targetAuthInvalidated := errors.Is(targetConnectErr, ErrAuthKeyInvalidated)
	if generationErr == nil && !targetAuthInvalidated {
		restoreErr = c.restorePrimaryMigration(st, snapshot)
	} else {
		restoreErr = restorePrimaryMigrationDC(st, snapshot)
	}
	authErr := c.authLossError()
	var restoreLoss *authLossState
	if targetAuthInvalidated && authErr == nil {
		restoreLoss, _ = c.latchMainAuthLoss(targetConnectErr)
		authErr = c.authLossResult(restoreLoss)
	} else if restoreErr != nil && authErr == nil {
		restoreLoss, _ = c.latchMainAuthLoss(fmt.Errorf("unsafe migration rollback: %w", restoreErr))
		authErr = c.authLossResult(restoreLoss)
	}
	c.authDecisionMu.Unlock()
	c.sessionWg.Wait()
	c.authLossMu.Lock()
	loss := c.authLoss
	c.authLossMu.Unlock()
	if loss != nil {
		c.finishMainAuthInvalidation(loss)
		authErr = c.authLossResult(loss)
	}
	if authErr != nil {
		c.autoConnectMu.Unlock()
		return errors.Join(restoreErr, authErr)
	}
	if generationErr != nil {
		c.autoConnectMu.Unlock()
		return errors.Join(generationErr, restoreErr)
	}
	if targetAuthInvalidated {
		c.autoConnectMu.Unlock()
		return targetConnectErr
	}
	var reconnectErr error
	if restoreErr != nil {
		c.autoConnectMu.Unlock()
		return restoreErr
	}
	if connect == nil {
		reconnectErr = c.connectTransportLocked(primaryMigrationTimeout)
		if reconnectErr != nil && c.shouldInvalidateMainAuth(reconnectErr) {
			reconnectErr = c.invalidateMainAuthLocked(reconnectErr)
		}
		var sess *session.Session
		var ready *connectionReadiness
		if reconnectErr == nil {
			sess, ready = c.mainSessionReadiness()
		}
		c.autoConnectMu.Unlock()
		if reconnectErr == nil {
			reconnectErr = c.completeConnect(primaryMigrationTimeout, sess, ready)
		}
	} else {
		c.autoConnectMu.Unlock()
		reconnectErr = connect(primaryMigrationTimeout)
	}
	if reconnectErr != nil {
		reconnectErr = fmt.Errorf("reconnect previous dc: %w", reconnectErr)
	}
	return errors.Join(restoreErr, reconnectErr)
}
