package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

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
	dcID     int
	authKey  []byte
	userID   int64
	configDC int
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
	return primaryMigrationSnapshot{
		dcID:     dcID,
		authKey:  append([]byte(nil), authKey...),
		userID:   userID,
		configDC: configDC,
	}, nil
}

func applyPrimaryMigration(st storage.Storage, targetDC int) error {
	if err := st.SetDCID(targetDC); err != nil {
		return fmt.Errorf("save dc_id: %w", err)
	}
	if err := st.SetAuthKey(nil); err != nil {
		return fmt.Errorf("clear auth key: %w", err)
	}
	if err := st.SetUserID(0); err != nil {
		return fmt.Errorf("clear user id: %w", err)
	}
	return nil
}

func restorePrimaryMigration(st storage.Storage, snapshot primaryMigrationSnapshot) error {
	return errors.Join(
		wrapMigrationRestoreError("dc_id", st.SetDCID(snapshot.dcID)),
		wrapMigrationRestoreError("auth key", st.SetAuthKey(snapshot.authKey)),
		wrapMigrationRestoreError("user id", st.SetUserID(snapshot.userID)),
	)
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
	snapshot, err := capturePrimaryMigration(st, c.config().DC)
	if err != nil {
		return err
	}

	c.cleanupSessions(false)
	if err := applyPrimaryMigration(st, targetDC); err != nil {
		return errors.Join(err, c.rollbackPrimaryMigration(st, snapshot, connect))
	}
	c.updateConfig(func(cfg *Config) { cfg.DC = targetDC })

	c.migratingDC.Store(true)
	connectErr := func() error {
		defer c.migratingDC.Store(false)
		return connect(timeout)
	}()
	if connectErr == nil {
		return nil
	}

	rollbackErr := c.rollbackPrimaryMigration(st, snapshot, connect)
	return errors.Join(fmt.Errorf("connect target dc: %w", connectErr), rollbackErr)
}

func (c *Client) rollbackPrimaryMigration(
	st storage.Storage,
	snapshot primaryMigrationSnapshot,
	connect func(time.Duration) error,
) error {
	c.migratingDC.Store(false)
	c.cleanupSessions(false)
	c.updateConfig(func(cfg *Config) { cfg.DC = snapshot.configDC })
	restoreErr := restorePrimaryMigration(st, snapshot)
	reconnectErr := connect(primaryMigrationTimeout)
	if reconnectErr != nil {
		reconnectErr = fmt.Errorf("reconnect previous dc: %w", reconnectErr)
	}
	return errors.Join(restoreErr, reconnectErr)
}
