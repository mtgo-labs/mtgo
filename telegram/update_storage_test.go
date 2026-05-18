package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/internal/storage"
)

func TestMemoryStorageUpdateState(t *testing.T) {
	st := NewMemoryStorage()
	store, ok := any(st).(storage.UpdateStateStore)
	if !ok {
		t.Fatal("MemoryStorage must implement storage.UpdateStateStore")
	}

	state := &storage.UpdateState{SessionID: "s1", Pts: 10, Qts: 2, Date: 100, Seq: 7}
	if err := store.SaveUpdateState(state); err != nil {
		t.Fatalf("SaveUpdateState: %v", err)
	}
	got, err := store.LoadUpdateState("s1")
	if err != nil {
		t.Fatalf("LoadUpdateState: %v", err)
	}
	if got.Pts != 10 || got.Qts != 2 || got.Date != 100 || got.Seq != 7 {
		t.Fatalf("LoadUpdateState = %+v", got)
	}
}

func TestMemoryStorageChannelUpdateState(t *testing.T) {
	st := NewMemoryStorage()
	store := any(st).(storage.UpdateStateStore)

	if err := store.SaveChannelUpdateState(&storage.ChannelUpdateState{SessionID: "s1", ChannelID: 99, Pts: 123}); err != nil {
		t.Fatalf("SaveChannelUpdateState: %v", err)
	}
	got, err := store.LoadChannelUpdateState("s1", 99)
	if err != nil {
		t.Fatalf("LoadChannelUpdateState: %v", err)
	}
	if got.Pts != 123 {
		t.Fatalf("channel pts = %d, want 123", got.Pts)
	}
}

func TestMemoryStorageDedupAndDurableQueue(t *testing.T) {
	st := NewMemoryStorage()
	store := any(st).(storage.UpdateStateStore)

	inserted, err := store.SaveUpdateDedupKey("s1", "msg:1:100")
	if err != nil || !inserted {
		t.Fatalf("first SaveUpdateDedupKey inserted=%v err=%v", inserted, err)
	}
	inserted, err = store.SaveUpdateDedupKey("s1", "msg:1:100")
	if err != nil || inserted {
		t.Fatalf("second SaveUpdateDedupKey inserted=%v err=%v", inserted, err)
	}

	u := &storage.DurableUpdate{SessionID: "s1", ID: "u1", Payload: []byte{1, 2, 3}}
	if err := store.EnqueueDurableUpdate(u); err != nil {
		t.Fatalf("EnqueueDurableUpdate: %v", err)
	}
	items, err := store.LoadDurableUpdates("s1", 10)
	if err != nil {
		t.Fatalf("LoadDurableUpdates: %v", err)
	}
	if len(items) != 1 || items[0].ID != "u1" {
		t.Fatalf("durable updates = %+v", items)
	}
	if err := store.DeleteDurableUpdate("s1", "u1"); err != nil {
		t.Fatalf("DeleteDurableUpdate: %v", err)
	}
	items, _ = store.LoadDurableUpdates("s1", 10)
	if len(items) != 0 {
		t.Fatalf("durable queue after delete = %+v", items)
	}
}
