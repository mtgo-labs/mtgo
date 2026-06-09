package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
)

func newTestStorage() storage.Storage {
	st := NewMemoryStorage()
	_ = st.SetSessionID("test-session-42")
	return st
}

func testUpdateManager(t *testing.T) *updateManager {
	t.Helper()
	st := newTestStorage()
	c, err := NewClient(1, "hash", &Config{Storage: st})
	if err != nil {
		t.Fatal(err)
	}
	mgr := newUpdateManager(c, st, updateManagerConfig{QueueSize: 4, MaxHandlerRetry: 1})
	return mgr
}

func TestUpdateManagerEnqueueDoesNotRunHandlerInline(t *testing.T) {
	st := newTestStorage()
	c, _ := NewClient(1, "hash", &Config{Storage: st})
	mgr := newUpdateManager(c, st, updateManagerConfig{QueueSize: 4})

	block := make(chan struct{})
	c.AddHandler(&FuncHandler{Fn: func(ctx *Context) { <-block }})

	if err := mgr.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- mgr.EnqueueLive(&tg.Updates{Updates: []tg.UpdateClass{&tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 1, PTSCount: 1}}})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("EnqueueLive blocked on slow handler")
	}

	close(block)
	mgr.Stop(context.Background())
}

func TestUpdateManagerHandlerPanicDoesNotAdvanceState(t *testing.T) {
	st := newTestStorage()
	c, _ := NewClient(1, "hash", &Config{Storage: st})
	mgr := newUpdateManager(c, st, updateManagerConfig{QueueSize: 4, MaxHandlerRetry: 1})
	c.AddHandler(&FuncHandler{Fn: func(ctx *Context) { panic("boom") }})

	err := mgr.deliverUpdate(&tg.Updates{Updates: []tg.UpdateClass{
		&tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 1, PTSCount: 1},
	}}, &tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 1, PTSCount: 1},
		updateMeta{Pts: 1, PtsCount: 1}, nil, nil, nil)
	if !errors.Is(err, ErrUpdateHandlerFailed) {
		t.Fatalf("deliverUpdate error = %v", err)
	}
	if mgr.state.Pts != 0 {
		t.Fatalf("pts advanced to %d after handler failure", mgr.state.Pts)
	}
}

func TestUpdateManagerAccountPtsGap(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state.Pts = 10
	kind := mgr.classifyUpdate(extractUpdateMeta(&tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 13, PTSCount: 1}))
	if kind != accountPtsGap {
		t.Fatalf("gap kind = %v, want accountPtsGap", kind)
	}
}

func TestUpdateManagerSeqGap(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state.Seq = 5
	kind := classifyAccountUpdate(mgr.state, updateMeta{Seq: 8})
	if kind != accountSeqGap {
		t.Fatalf("gap kind = %v, want accountSeqGap", kind)
	}
}

func TestUpdateManagerDuplicateIgnored(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state.Pts = 10
	err := mgr.applyUpdate(context.Background(), &tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 9, PTSCount: 1}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("duplicate apply error = %v", err)
	}
	if mgr.health.DuplicateCount != 1 {
		t.Fatalf("DuplicateCount = %d, want 1", mgr.health.DuplicateCount)
	}
}

func TestLiveUpdateArrivingWhileRecoveryRunningDoesNotStartDuplicateRecovery(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state.Pts = 10
	mgr.recovering = true
	err := mgr.applyUpdate(context.Background(), &tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 12, PTSCount: 1}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("applyUpdate while recovery running: %v", err)
	}
	if mgr.health.RecoveryCount != 0 {
		t.Fatalf("RecoveryCount = %d, want 0", mgr.health.RecoveryCount)
	}
}

func TestUpdateManagerPersistsStateAfterSuccess(t *testing.T) {
	st := newTestStorage()
	c, _ := NewClient(1, "hash", &Config{Storage: st})
	store := any(st).(storage.UpdateStateStore)
	mgr := newUpdateManager(c, st, updateManagerConfig{QueueSize: 4})
	mgr.advanceState(updateMeta{Pts: 5, PtsCount: 1})

	saved, err := store.LoadUpdateState("test-session-42")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Pts != 5 {
		t.Fatalf("saved pts = %d, want 5", saved.Pts)
	}
}

func TestLiveGapRecoveryTriggersGetDifference(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state.Pts = 10
	rpc := &fakeDifferenceRPC{diffs: []tg.DifferenceClass{
		&tg.UpdatesDifference{
			OtherUpdates: []tg.UpdateClass{&tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 12, PTSCount: 2}},
			State:        &tg.UpdatesState{PTS: 12, Date: 11, Seq: 2},
		},
	}}
	mgr.SetRPC(rpc)

	err := mgr.applyUpdate(context.Background(), &tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 13, PTSCount: 1}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("applyUpdate: %v", err)
	}
	if rpc.calls != 1 {
		t.Fatalf("rpc calls = %d, want 1", rpc.calls)
	}
	if mgr.state.Pts != 13 {
		t.Fatalf("pts = %d, want 13", mgr.state.Pts)
	}
}

func TestChannelUpdateStatePersists(t *testing.T) {
	st := newTestStorage()
	c, _ := NewClient(1, "hash", &Config{Storage: st})
	store := any(st).(storage.UpdateStateStore)
	mgr := newUpdateManager(c, st, updateManagerConfig{QueueSize: 4})
	mgr.advanceState(updateMeta{IsChannel: true, ChannelID: 100, ChannelPts: 15, PtsCount: 1})

	saved, err := store.LoadChannelUpdateState("test-session-42", 100)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Pts != 15 {
		t.Fatalf("channel pts = %d, want 15", saved.Pts)
	}
}

func TestUpdateManagerReplayDurableQueueOnRestart(t *testing.T) {
	st := newTestStorage()
	store := any(st).(storage.UpdateStateStore)
	if err := store.EnqueueDurableUpdate(&storage.DurableUpdate{SessionID: "test-session-42", ID: "u1", Payload: []byte{1, 2, 3}}); err != nil {
		t.Fatal(err)
	}
	c, _ := NewClient(1, "hash", &Config{Storage: st})
	mgr := newUpdateManager(c, st, updateManagerConfig{QueueSize: 4, DurableQueue: true})
	if err := mgr.replayDurableQueue(); err != nil {
		t.Fatalf("replayDurableQueue: %v", err)
	}
	items, _ := store.LoadDurableUpdates("test-session-42", 10)
	if len(items) != 0 {
		t.Fatalf("durable queue after replay = %d items, want 0", len(items))
	}
}

func TestUpdateManagerGracefulShutdown(t *testing.T) {
	st := newTestStorage()
	c, _ := NewClient(1, "hash", &Config{Storage: st})
	mgr := newUpdateManager(c, st, updateManagerConfig{QueueSize: 4})

	if err := mgr.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := mgr.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestUpdateManagerContextCancellation(t *testing.T) {
	st := newTestStorage()
	c, _ := NewClient(1, "hash", &Config{Storage: st})
	mgr := newUpdateManager(c, st, updateManagerConfig{QueueSize: 4})

	if err := mgr.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := mgr.Stop(ctx); err == nil {
		t.Fatal("expected error from cancelled context Stop")
	}
}

func TestUpdateManagerNoGoroutineLeak(t *testing.T) {
	st := newTestStorage()
	c, _ := NewClient(1, "hash", &Config{Storage: st})
	mgr := newUpdateManager(c, st, updateManagerConfig{QueueSize: 4})

	if err := mgr.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		mgr.Stop(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not complete within timeout — goroutine leak likely")
	}
}

func TestUpdateManagerQtsGap(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state.Qts = 5
	kind := classifyAccountUpdate(mgr.state, updateMeta{Qts: 8})
	if kind != accountQtsGap {
		t.Fatalf("gap kind = %v, want accountQtsGap", kind)
	}
}

func TestUpdateManagerQtsDuplicate(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state.Qts = 5
	kind := classifyAccountUpdate(mgr.state, updateMeta{Qts: 3})
	if kind != duplicateUpdate {
		t.Fatalf("gap kind = %v, want duplicateUpdate", kind)
	}
}

func TestUpdateManagerQtsNoGap(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state.Qts = 5
	kind := classifyAccountUpdate(mgr.state, updateMeta{Qts: 6})
	if kind != noGap {
		t.Fatalf("gap kind = %v, want noGap", kind)
	}
}

func TestUpdateChannelTooLongTriggersRecovery(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.channels[100] = channelState{ChannelID: 100, Pts: 10}
	rpc := &fakeChannelDifferenceRPC{result: &tg.UpdatesChannelDifference{Final: true, PTS: 50}}
	mgr.SetRPC(rpc)

	err := mgr.applyUpdate(context.Background(), &tg.UpdateChannelTooLong{ChannelID: 100}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("applyUpdate ChannelTooLong: %v", err)
	}
	if rpc.calls != 1 {
		t.Fatalf("rpc calls = %d, want 1", rpc.calls)
	}
	if mgr.channels[100].Pts != 50 {
		t.Fatalf("channel pts = %d, want 50", mgr.channels[100].Pts)
	}
}

func TestExtractUpdateMetaWebPage(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdateWebPage{PTS: 10, PTSCount: 1})
	if meta.Pts != 10 || meta.PtsCount != 1 {
		t.Fatalf("meta = %+v", meta)
	}
	if meta.IsChannel {
		t.Fatal("UpdateWebPage should not be channel")
	}
}

func TestExtractUpdateMetaPinnedMessages(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdatePinnedMessages{Messages: []int32{1, 2}, PTS: 5, PTSCount: 2})
	if meta.Pts != 5 || meta.PtsCount != 2 {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestExtractUpdateMetaChannelWebPage(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdateChannelWebPage{ChannelID: 200, PTS: 15, PTSCount: 1})
	if !meta.IsChannel || meta.ChannelID != 200 || meta.ChannelPts != 15 {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestExtractUpdateMetaPinnedChannelMessages(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdatePinnedChannelMessages{ChannelID: 300, Messages: []int32{1}, PTS: 20, PTSCount: 1})
	if !meta.IsChannel || meta.ChannelID != 300 || meta.ChannelPts != 20 {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestExtractUpdateMetaNewEncryptedMessage(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdateNewEncryptedMessage{Qts: 7})
	if meta.Qts != 7 {
		t.Fatalf("meta.Qts = %d, want 7", meta.Qts)
	}
	if meta.IsChannel {
		t.Fatal("encrypted message should not be channel")
	}
}

func TestExtractUpdateMetaChannelTooLong(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdateChannelTooLong{ChannelID: 400, PTS: 42})
	if !meta.IsChannel || meta.ChannelID != 400 || meta.ChannelPts != 42 {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestExtractUpdateMetaChannelTooLongNoPts(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdateChannelTooLong{ChannelID: 400})
	if !meta.IsChannel || meta.ChannelID != 400 || meta.ChannelPts != 0 {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestOnRecoverChannelSweep(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state = updateState{Pts: 5, Date: 10, Seq: 1}
	mgr.channels[100] = channelState{ChannelID: 100, Pts: 10}
	mgr.channels[200] = channelState{ChannelID: 200, Pts: 20}

	channelRPC := &fakeChannelDifferenceRPC{result: &tg.UpdatesChannelDifference{Final: true, PTS: 15}}
	rpc := &compositeFakeRPC{
		diff:    &fakeDifferenceRPC{diffs: []tg.DifferenceClass{&tg.UpdatesDifferenceEmpty{Date: 11, Seq: 2}}},
		channel: channelRPC,
	}

	if err := mgr.OnReconnect(context.Background(), rpc); err != nil {
		t.Fatalf("OnReconnect: %v", err)
	}
	if channelRPC.calls != 2 {
		t.Fatalf("channel RPC calls = %d, want 2", channelRPC.calls)
	}
}

type compositeFakeRPC struct {
	diff    *fakeDifferenceRPC
	channel *fakeChannelDifferenceRPC
}

func (c *compositeFakeRPC) UpdatesGetDifference(ctx context.Context, req *tg.UpdatesGetDifferenceRequest) (tg.DifferenceClass, error) {
	return c.diff.UpdatesGetDifference(ctx, req)
}

func (c *compositeFakeRPC) UpdatesGetChannelDifference(ctx context.Context, req *tg.UpdatesGetChannelDifferenceRequest) (tg.ChannelDifferenceClass, error) {
	return c.channel.UpdatesGetChannelDifference(ctx, req)
}

func TestStartLoadsPersistedChannelStates(t *testing.T) {
	st := newTestStorage()
	store := any(st).(storage.UpdateStateStore)
	store.SaveUpdateState(&storage.UpdateState{SessionID: "test-session-42", Pts: 10, Qts: 0, Date: 5, Seq: 1})
	store.SaveChannelUpdateState(&storage.ChannelUpdateState{SessionID: "test-session-42", ChannelID: 100, Pts: 50})
	store.SaveChannelUpdateState(&storage.ChannelUpdateState{SessionID: "test-session-42", ChannelID: 200, Pts: 75})

	c, _ := NewClient(1, "hash", &Config{Storage: st})
	mgr := newUpdateManager(c, st, updateManagerConfig{QueueSize: 4})
	if err := mgr.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer mgr.Stop(context.Background())

	if mgr.channels[100].Pts != 50 {
		t.Fatalf("channel 100 pts = %d, want 50", mgr.channels[100].Pts)
	}
	if mgr.channels[200].Pts != 75 {
		t.Fatalf("channel 200 pts = %d, want 75", mgr.channels[200].Pts)
	}
	if mgr.state.Pts != 10 {
		t.Fatalf("state pts = %d, want 10", mgr.state.Pts)
	}
}
