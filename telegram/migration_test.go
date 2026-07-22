package telegram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

type migrationRawInvoker struct {
	input  tg.TLObject
	result []byte
}

type migrationCloseSignalTransport struct {
	closed chan struct{}
	once   sync.Once
}

func (*migrationCloseSignalTransport) Send([]byte) error { return nil }
func (*migrationCloseSignalTransport) Recv() ([]byte, error) {
	return nil, errors.New("transport closed")
}
func (t *migrationCloseSignalTransport) Close() error {
	t.once.Do(func() { close(t.closed) })
	return nil
}
func (t *migrationCloseSignalTransport) IsConnected() bool {
	select {
	case <-t.closed:
		return false
	default:
		return true
	}
}
func (*migrationCloseSignalTransport) SetWriteDeadline(time.Time) error { return nil }
func (*migrationCloseSignalTransport) SetReadDeadline(time.Time) error  { return nil }

type migrationRecordingStorage struct {
	*MemoryStorage
	ops []string
}

func (s *migrationRecordingStorage) SetAuthKey(key []byte) error {
	s.ops = append(s.ops, "auth")
	return s.MemoryStorage.SetAuthKey(key)
}

func (s *migrationRecordingStorage) SetDate(date int) error {
	s.ops = append(s.ops, "date")
	return s.MemoryStorage.SetDate(date)
}

func (s *migrationRecordingStorage) SetUserID(userID int64) error {
	s.ops = append(s.ops, "user")
	return s.MemoryStorage.SetUserID(userID)
}

func (s *migrationRecordingStorage) SetDCID(dcID int) error {
	s.ops = append(s.ops, "dc")
	return s.MemoryStorage.SetDCID(dcID)
}

func (s *migrationRecordingStorage) Sync() error {
	s.ops = append(s.ops, "sync")
	return nil
}

func (m *migrationRawInvoker) RPCInvoke(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	return nil, errors.New("unexpected decoded invoke")
}

func (m *migrationRawInvoker) RPCInvokeRaw(_ context.Context, input tg.TLObject) ([]byte, error) {
	m.input = input
	return m.result, nil
}

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

func TestSwitchPrimaryDCStopsSourceBeforeDrainingAuthDecisions(t *testing.T) {
	client, err := NewClient(1, "hash", nil)
	if err != nil {
		t.Fatal(err)
	}
	st := populatedAuthStorage(t)
	sess, err := session.NewSession(session.DataCenter{ID: 2}, st, "test", "1", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	transport := &migrationCloseSignalTransport{closed: make(chan struct{})}
	sess.SetTransport(transport)
	client.mu.Lock()
	client.storage = st
	client.session = sess
	client.mu.Unlock()
	client.state.SetConnecting(2)
	client.state.SetConnected()

	client.authDecisionMu.RLock()
	done := make(chan error, 1)
	go func() {
		done <- client.switchPrimaryDC(context.Background(), 4, st, func(time.Duration) error {
			return errors.New("test connect failure")
		})
	}()
	select {
	case <-transport.closed:
	case <-time.After(time.Second):
		client.authDecisionMu.RUnlock()
		<-done
		t.Fatal("source session was not stopped before waiting for in-flight auth decisions")
	}
	client.authDecisionMu.RUnlock()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("migration did not finish after auth decisions drained")
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

func TestApplyPrimaryMigrationClearsAndSyncsKeyBeforeChangingDC(t *testing.T) {
	st := &migrationRecordingStorage{MemoryStorage: NewMemoryStorage()}
	if err := st.MemoryStorage.SetDCID(2); err != nil {
		t.Fatal(err)
	}
	if err := st.MemoryStorage.SetAuthKey([]byte("source-key")); err != nil {
		t.Fatal(err)
	}
	if err := st.MemoryStorage.SetDate(123); err != nil {
		t.Fatal(err)
	}
	if err := st.MemoryStorage.SetUserID(42); err != nil {
		t.Fatal(err)
	}

	if err := applyPrimaryMigration(st, 4); err != nil {
		t.Fatalf("applyPrimaryMigration() = %v", err)
	}
	wantOps := []string{"auth", "date", "user", "sync", "dc", "sync"}
	if fmt.Sprint(st.ops) != fmt.Sprint(wantOps) {
		t.Fatalf("operations = %v, want %v", st.ops, wantOps)
	}
	key, _ := st.AuthKey()
	date, _ := st.Date()
	userID, _ := st.UserID()
	dcID, _ := st.DCID()
	if len(key) != 0 || date != 0 || userID != 0 || dcID != 4 {
		t.Fatalf("migration state = dc:%d key:%d date:%d user:%d", dcID, len(key), date, userID)
	}
}

func TestRestorePrimaryMigrationClearsTargetKeyBeforeRestoringSourceDC(t *testing.T) {
	st := &migrationRecordingStorage{MemoryStorage: NewMemoryStorage()}
	if err := st.MemoryStorage.SetDCID(4); err != nil {
		t.Fatal(err)
	}
	if err := st.MemoryStorage.SetAuthKey(bytes.Repeat([]byte{0x44}, 256)); err != nil {
		t.Fatal(err)
	}
	client, _ := NewClient(12345, "hash", &Config{InMemory: true, DC: 4})
	snapshot := primaryMigrationSnapshot{
		dcID:          2,
		authKey:       bytes.Repeat([]byte{0x22}, 256),
		date:          1234,
		userID:        42,
		authKeyOrigin: authKeyOriginLoaded,
	}

	if err := client.restorePrimaryMigration(st, snapshot); err != nil {
		t.Fatalf("restorePrimaryMigration() = %v", err)
	}
	wantOps := []string{"auth", "date", "user", "sync", "dc", "sync", "auth", "date", "user", "sync"}
	if fmt.Sprint(st.ops) != fmt.Sprint(wantOps) {
		t.Fatalf("operations = %v, want %v", st.ops, wantOps)
	}
	key, _ := st.AuthKey()
	dcID, _ := st.DCID()
	if dcID != 2 || !bytes.Equal(key, snapshot.authKey) {
		t.Fatalf("restored state = dc:%d key:%x", dcID, key)
	}
}

func TestSwitchPrimaryDCRollsBackFailedTarget(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true, DC: 2, SessionString: "source-session"})
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
	if err := st.SetDate(1234); err != nil {
		t.Fatal(err)
	}
	client.storage = st
	client.mainAuthKeyOrigin.Store(authKeyOriginLoaded)
	client.state.SetConnecting(2)
	client.state.SetConnected()

	var connects atomic.Int32
	targetErr := errors.New("target unavailable")
	err := client.switchPrimaryDC(context.Background(), 4, st, func(time.Duration) error {
		switch connects.Add(1) {
		case 1:
			client.mainAuthKeyOrigin.Store(authKeyOriginFresh)
			if err := st.SetDate(9876); err != nil {
				t.Fatal(err)
			}
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
	if client.sessionStringInvalidated.Load() {
		t.Fatal("safe rollback left the source session string invalidated")
	}
	if origin := client.mainAuthKeyOrigin.Load(); origin != authKeyOriginLoaded {
		t.Fatalf("rollback auth key origin = %d, want loaded", origin)
	}
	if date, _ := st.Date(); date != 1234 {
		t.Fatalf("rollback auth key date = %d, want 1234", date)
	}
}

func TestSwitchPrimaryDCRollbackDoesNotRestoreInvalidatedAuth(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true, DC: 2})
	st := NewMemoryStorage()
	oldKey := []byte("rejected-auth-key")
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
	err := client.switchPrimaryDC(context.Background(), 4, st, func(time.Duration) error {
		connects.Add(1)
		_ = client.invalidateMainAuth(tgerr.New(406, "AUTH_KEY_DUPLICATED"))
		return errors.New("target rejected auth key")
	})
	if !errors.Is(err, ErrAuthKeyInvalidated) {
		t.Fatalf("switchPrimaryDC() = %v, want ErrAuthKeyInvalidated", err)
	}
	if got := connects.Load(); got != 1 {
		t.Fatalf("connect calls = %d, want no rollback reconnect", got)
	}
	dcID, _ := st.DCID()
	authKey, _ := st.AuthKey()
	userID, _ := st.UserID()
	if dcID != 2 || len(authKey) != 0 || userID != 0 || client.config().DC != 2 {
		t.Fatalf("rollback state = dc:%d key:%d user:%d config:%d", dcID, len(authKey), userID, client.config().DC)
	}
}

func TestRollbackRelatchesClearedTargetAuthLossBeforeRestoringSourceDC(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true, DC: 4})
	st := &migrationRecordingStorage{MemoryStorage: NewMemoryStorage()}
	freshTargetKey := bytes.Repeat([]byte{0x44}, 256)
	if err := st.MemoryStorage.SetDCID(4); err != nil {
		t.Fatal(err)
	}
	if err := st.MemoryStorage.SetAuthKey(freshTargetKey); err != nil {
		t.Fatal(err)
	}
	if err := st.MemoryStorage.SetDate(9876); err != nil {
		t.Fatal(err)
	}
	if err := st.MemoryStorage.SetUserID(99); err != nil {
		t.Fatal(err)
	}
	client.storage = st
	client.state.SetConnecting(4)
	client.state.SetConnected()
	snapshot := primaryMigrationSnapshot{dcID: 2, configDC: 2, authKey: bytes.Repeat([]byte{0x22}, 256), date: 1234, userID: 42}
	targetErr := &AuthKeyInvalidatedError{Cause: tgerr.New(406, "AUTH_KEY_DUPLICATED")}

	err := client.rollbackPrimaryMigration(st, snapshot, func(time.Duration) error {
		t.Fatal("invalidated target must not reconnect the source")
		return nil
	}, targetErr)
	if !errors.Is(err, ErrAuthKeyInvalidated) {
		t.Fatalf("rollbackPrimaryMigration() = %v, want ErrAuthKeyInvalidated", err)
	}
	dcID, _ := st.DCID()
	key, _ := st.AuthKey()
	date, _ := st.Date()
	userID, _ := st.UserID()
	if dcID != 2 || len(key) != 0 || date != 0 || userID != 0 {
		t.Fatalf("rollback state = dc:%d key:%d date:%d user:%d", dcID, len(key), date, userID)
	}
	wantPrefix := []string{"auth", "date", "user", "sync", "dc", "sync"}
	if len(st.ops) < len(wantPrefix) || fmt.Sprint(st.ops[:len(wantPrefix)]) != fmt.Sprint(wantPrefix) {
		t.Fatalf("rollback operations = %v, want prefix %v", st.ops, wantPrefix)
	}
}

func TestSwitchPrimaryDCRollbackDrainsTargetGenerationBeforeRestore(t *testing.T) {
	client, _ := NewClient(12345, "hash", &Config{InMemory: true, DC: 2})
	st := NewMemoryStorage()
	oldKey := bytes.Repeat([]byte{0x21}, 256)
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

	rpcEntered := make(chan struct{})
	rpcRelease := make(chan struct{})
	rpcDone := make(chan error, 1)
	targetErr := errors.New("target unavailable")
	var connects atomic.Int32
	err := client.switchPrimaryDC(context.Background(), 4, st, func(time.Duration) error {
		if connects.Add(1) != 1 {
			t.Fatal("rollback reconnected after target auth loss")
		}
		if err := st.SetAuthKey(bytes.Repeat([]byte{0x42}, 256)); err != nil {
			t.Fatal(err)
		}
		targetSession, err := session.NewSession(session.DataCenter{ID: 4}, st, "test", "1", "en", "en")
		if err != nil {
			t.Fatal(err)
		}
		client.mu.Lock()
		client.session = targetSession
		client.mu.Unlock()
		client.state.SetConnecting(4)
		client.state.SetConnected()
		go func() {
			rpcDone <- client.retrySessionErr(context.Background(), func(source *session.Session) error {
				if source != targetSession {
					return fmt.Errorf("source session = %p, want target %p", source, targetSession)
				}
				close(rpcEntered)
				<-rpcRelease
				return tgerr.New(406, "AUTH_KEY_DUPLICATED")
			})
		}()
		<-rpcEntered
		go func() {
			for client.Session() != nil {
				time.Sleep(time.Millisecond)
			}
			close(rpcRelease)
		}()
		return targetErr
	})

	if !errors.Is(err, ErrAuthKeyInvalidated) || !errors.Is(err, targetErr) {
		t.Fatalf("switchPrimaryDC() = %v, want target error and ErrAuthKeyInvalidated", err)
	}
	if rpcErr := <-rpcDone; !errors.Is(rpcErr, ErrAuthKeyInvalidated) {
		t.Fatalf("late target RPC = %v, want ErrAuthKeyInvalidated", rpcErr)
	}
	authKey, keyErr := st.AuthKey()
	if keyErr != nil {
		t.Fatal(keyErr)
	}
	if len(authKey) != 0 {
		t.Fatalf("rollback restored source key after target auth loss: %q", authKey)
	}
	if got := connects.Load(); got != 1 {
		t.Fatalf("connect calls = %d, want no rollback reconnect", got)
	}
}

func TestHandleRawMigrationErrorReturnsTargetRawResult(t *testing.T) {
	client, err := NewClient(12345, "hash", &Config{InMemory: true, DC: 2, DCPoolSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	client.state.SetConnecting(2)
	client.state.SetConnected()
	client.storage = NewMemoryStorage()

	want := []byte{1, 2, 3, 4}
	invoker := &migrationRawInvoker{result: want}
	if !client.dcSessions.putIfGeneration(4, &dcSessionEntry{rpc: tg.NewRPCClient(invoker)}, 0) {
		t.Fatal("failed to seed DC session")
	}
	query := &tg.UploadGetFileRequest{}

	got, err := client.handleRawMigrationError(context.Background(), tgerr.New(303, "FILE_MIGRATE_4"), query)
	if err != nil {
		t.Fatalf("handleRawMigrationError: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("raw result = %x, want %x", got, want)
	}
	if invoker.input != query {
		t.Fatalf("target query = %T, want original query", invoker.input)
	}
}
