package telegram

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/session"
)

type countingCloser struct {
	calls atomic.Int32
}

func (c *countingCloser) Close() error {
	c.calls.Add(1)
	return nil
}

func TestDCSessionsCleanupRejectsInFlightEntryPublication(t *testing.T) {
	dcs := newDCSessions()
	_, generation := dcs.getInitLock(4)
	closer := &countingCloser{}
	candidate := &dcSessionEntry{closer: closer}

	dcs.cleanup()
	if dcs.putIfGeneration(4, candidate, generation) {
		t.Fatal("stale entry was published after cleanup")
	}
	candidate.close()
	if _, ok := dcs.get(4); ok {
		t.Fatal("cleanup registry contains stale entry")
	}
	if got := closer.calls.Load(); got != 1 {
		t.Fatalf("candidate close calls = %d, want 1", got)
	}
}

func TestDCSessionsCleanupStopsInFlightTransport(t *testing.T) {
	dcs := newDCSessions()
	_, generation := dcs.getInitLock(4)
	creation, ctx, ok := dcs.beginCreation(context.Background(), generation)
	if !ok {
		t.Fatal("failed to register DC candidate")
	}
	closer := &countingCloser{}
	if !creation.setCloser(closer) {
		t.Fatal("candidate transport was rejected before cleanup")
	}

	dcs.cleanup()
	if got := closer.calls.Load(); got != 1 {
		t.Fatalf("in-flight transport close calls = %d, want 1", got)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("candidate context = %v, want canceled", ctx.Err())
	}
	dcs.finishCreation(creation)
}

func TestDCSessionsCleanupClosesTransportAcquiredAfterCancellation(t *testing.T) {
	dcs := newDCSessions()
	_, generation := dcs.getInitLock(4)
	creation, ctx, ok := dcs.beginCreation(context.Background(), generation)
	if !ok {
		t.Fatal("failed to register DC candidate")
	}
	cleanupDone := make(chan struct{})
	go func() {
		dcs.cleanup()
		close(cleanupDone)
	}()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("cleanup did not cancel the in-flight dial")
	}
	select {
	case <-cleanupDone:
		t.Fatal("cleanup returned before the dial-phase candidate finished")
	default:
	}

	closer := &countingCloser{}
	if creation.setCloser(closer) {
		t.Fatal("transport acquired by a canceled candidate was accepted")
	}
	if got := closer.calls.Load(); got != 1 {
		t.Fatalf("late transport close calls = %d, want 1", got)
	}
	dcs.finishCreation(creation)
	select {
	case <-cleanupDone:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not wait for the canceled dial-phase candidate")
	}
}

func TestGetSessionRejectsCandidateCreatedAcrossCleanup(t *testing.T) {
	client, _ := NewClient(1, "hash", nil)
	started := make(chan struct{})
	release := make(chan struct{})
	client.setTestSessionFactory(func(context.Context, int, string, int, []byte) (*session.Session, error) {
		close(started)
		<-release
		return session.NewSession(session.DataCenter{ID: 4}, NewMemoryStorage(), "test", "1", "en", "en")
	})

	done := make(chan error, 1)
	go func() {
		_, err := client.GetSession(context.Background(), 4, false, true)
		done <- err
	}()
	<-started
	client.cleanupSessions(false)
	close(release)
	if err := <-done; !errors.Is(err, ErrNotConnected) {
		t.Fatalf("GetSession() = %v, want ErrNotConnected", err)
	}
	client.sessionsMu.Lock()
	defer client.sessionsMu.Unlock()
	if len(client.sessions) != 0 {
		t.Fatal("stale GetSession candidate escaped cleanup")
	}
}

func TestDCSessionsCleanupRejectsInFlightPoolExpansion(t *testing.T) {
	dcs := newDCSessions()
	oldCloser := &countingCloser{}
	oldEntry := &dcSessionEntry{closer: oldCloser}
	pool := &dcSessionPool{entries: []*dcSessionEntry{oldEntry}}
	if !dcs.putPoolIfGeneration(4, pool, 0) {
		t.Fatal("failed to seed pool")
	}
	_, generation := dcs.getInitLock(4)
	newCloser := &countingCloser{}
	newEntry := &dcSessionEntry{closer: newCloser}

	dcs.cleanup()
	if dcs.updatePoolIfGeneration(4, pool, []*dcSessionEntry{oldEntry, newEntry}, generation) {
		t.Fatal("stale expansion mutated a detached pool")
	}
	newEntry.close()
	if _, ok := dcs.getPool(4, 1); ok {
		t.Fatal("cleanup registry contains stale pool")
	}
	if got := pool.len(); got != 1 {
		t.Fatalf("detached pool size = %d, want 1", got)
	}
	if got := oldCloser.calls.Load(); got != 1 {
		t.Fatalf("old entry close calls = %d, want 1", got)
	}
	if got := newCloser.calls.Load(); got != 1 {
		t.Fatalf("new entry close calls = %d, want 1", got)
	}
}

func TestDCSessionsCleanupRejectsInFlightPoolReplacement(t *testing.T) {
	dcs := newDCSessions()
	oldCloser := &countingCloser{}
	oldEntry := &dcSessionEntry{closer: oldCloser}
	pool := &dcSessionPool{entries: []*dcSessionEntry{oldEntry}}
	if !dcs.putPoolIfGeneration(4, pool, 0) {
		t.Fatal("failed to seed pool")
	}
	_, generation := dcs.getInitLock(4)
	newCloser := &countingCloser{}
	newEntry := &dcSessionEntry{closer: newCloser}

	dcs.cleanup()
	_, _, replaced := dcs.replacePoolEntryIfGeneration(4, pool, 0, oldEntry, newEntry, generation)
	if replaced {
		t.Fatal("stale replacement mutated a detached pool")
	}
	newEntry.close()
	if got := pool.entry(0); got != oldEntry {
		t.Fatal("detached pool entry was replaced")
	}
	if got := oldCloser.calls.Load(); got != 1 {
		t.Fatalf("old entry close calls = %d, want 1", got)
	}
	if got := newCloser.calls.Load(); got != 1 {
		t.Fatalf("new entry close calls = %d, want 1", got)
	}
}
