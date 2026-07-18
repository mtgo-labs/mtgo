package telegram

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMigrationCoordinatorCoalescesSameTarget(t *testing.T) {
	var coordinator migrationCoordinator
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	ownerDone := make(chan error, 1)
	go func() {
		ownerDone <- coordinator.Do(context.Background(), 4, func(context.Context) error {
			calls.Add(1)
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	const workers = 100
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- coordinator.Do(context.Background(), 4, func(context.Context) error {
				calls.Add(1)
				return nil
			})
		}()
	}

	deadline := time.Now().Add(time.Second)
	for {
		coordinator.mu.Lock()
		waiters := coordinator.waiters
		coordinator.mu.Unlock()
		if waiters == workers {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("coordinator waiters = %d, want %d", waiters, workers)
		}
		time.Sleep(time.Millisecond)
	}
	close(release)
	if err := <-ownerDone; err != nil {
		t.Fatalf("owner Do() = %v", err)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("Do() = %v", err)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("migration calls = %d, want 1", got)
	}
}

func TestMigrationCoordinatorSerializesDifferentTargets(t *testing.T) {
	var coordinator migrationCoordinator
	var active atomic.Int32
	var maxActive atomic.Int32
	started := make(chan struct{}, 2)
	release := make(chan struct{})

	run := func(targetDC int) <-chan error {
		done := make(chan error, 1)
		go func() {
			done <- coordinator.Do(context.Background(), targetDC, func(context.Context) error {
				current := active.Add(1)
				defer active.Add(-1)
				for current > maxActive.Load() && !maxActive.CompareAndSwap(maxActive.Load(), current) {
				}
				started <- struct{}{}
				<-release
				return nil
			})
		}()
		return done
	}

	done4 := run(4)
	<-started
	done5 := run(5)
	select {
	case <-started:
		t.Fatal("different target started before active migration completed")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if err := <-done4; err != nil {
		t.Fatalf("DC 4 migration: %v", err)
	}
	if err := <-done5; err != nil {
		t.Fatalf("DC 5 migration: %v", err)
	}
	if got := maxActive.Load(); got != 1 {
		t.Fatalf("maximum concurrent migrations = %d, want 1", got)
	}
}

func TestMigrationCoordinatorWaiterCanCancel(t *testing.T) {
	var coordinator migrationCoordinator
	started := make(chan struct{})
	release := make(chan struct{})
	ownerDone := make(chan error, 1)
	go func() {
		ownerDone <- coordinator.Do(context.Background(), 4, func(context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := coordinator.Do(ctx, 4, func(context.Context) error {
		t.Fatal("canceled waiter must not run migration")
		return nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("waiter error = %v, want context.Canceled", err)
	}

	close(release)
	if err := <-ownerDone; err != nil {
		t.Fatalf("owner migration = %v", err)
	}
}

func TestSwitchPrimaryDCRollsBackFailedTarget(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true, DC: 2})
	st := NewMemoryStorage()
	oldKey := []byte("old-auth-key")
	if err := st.SetDCID(2); err != nil {
		t.Fatal(err)
	}
	if err := st.SetAuthKey(oldKey); err != nil {
		t.Fatal(err)
	}
	if err := st.SetUserID(42); err != nil {
		t.Fatal(err)
	}
	client.storage = st
	client.state.SetConnecting(2)
	client.state.SetConnected()

	var connects atomic.Int32
	targetErr := errors.New("target unavailable")
	err := client.switchPrimaryDC(context.Background(), 4, st, func(time.Duration) error {
		switch connects.Add(1) {
		case 1:
			dcID, _ := st.DCID()
			authKey, _ := st.AuthKey()
			userID, _ := st.UserID()
			if dcID != 4 || len(authKey) != 0 || userID != 0 || client.config().DC != 4 {
				t.Fatalf("target state = dc:%d key:%d user:%d config:%d", dcID, len(authKey), userID, client.config().DC)
			}
			return targetErr
		case 2:
			return nil
		default:
			t.Fatalf("unexpected connect call %d", connects.Load())
			return nil
		}
	})
	if !errors.Is(err, targetErr) {
		t.Fatalf("switchPrimaryDC() = %v, want target error", err)
	}
	if got := connects.Load(); got != 2 {
		t.Fatalf("connect calls = %d, want target and rollback reconnect", got)
	}
	dcID, _ := st.DCID()
	authKey, _ := st.AuthKey()
	userID, _ := st.UserID()
	if dcID != 2 || string(authKey) != string(oldKey) || userID != 42 || client.config().DC != 2 {
		t.Fatalf("rollback state = dc:%d key:%q user:%d config:%d", dcID, authKey, userID, client.config().DC)
	}
}
