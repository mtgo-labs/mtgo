package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
)

type fakeDifferenceRPC struct {
	diffs []tg.DifferenceClass
	calls int
}

func (f *fakeDifferenceRPC) UpdatesGetDifference(ctx context.Context, req *tg.UpdatesGetDifferenceRequest) (tg.DifferenceClass, error) {
	f.calls++
	out := f.diffs[0]
	f.diffs = f.diffs[1:]
	return out, nil
}

func (f *fakeDifferenceRPC) UpdatesGetChannelDifference(ctx context.Context, req *tg.UpdatesGetChannelDifferenceRequest) (tg.ChannelDifferenceClass, error) {
	return nil, nil
}

func TestStartupRecoveryWithSavedState(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state = updateState{Pts: 5, Qts: 0, Date: 10, Seq: 1}
	rpc := &fakeDifferenceRPC{diffs: []tg.DifferenceClass{
		&tg.UpdatesDifference{
			OtherUpdates: []tg.UpdateClass{&tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 6, PTSCount: 1}},
			State:        &tg.UpdatesState{PTS: 6, Qts: 0, Date: 11, Seq: 2},
		},
	}}
	if err := mgr.RecoverAccount(context.Background(), rpc); err != nil {
		t.Fatalf("RecoverAccount: %v", err)
	}
	if mgr.state.Pts != 6 || mgr.state.Seq != 2 {
		t.Fatalf("state = %+v", mgr.state)
	}
	if rpc.calls != 1 {
		t.Fatalf("calls = %d, want 1", rpc.calls)
	}
}

func TestStartupWithNoSavedStateUsesGetState(t *testing.T) {
	st := newTestStorage()
	store := any(st).(storage.UpdateStateStore)
	state, err := store.LoadUpdateState("test-session-42")
	if err != nil {
		t.Fatal(err)
	}
	if state != nil {
		t.Fatalf("state = %+v, want nil", state)
	}
}

func TestGetDifferenceEmpty(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state = updateState{Pts: 5, Date: 10, Seq: 1}
	rpc := &fakeDifferenceRPC{diffs: []tg.DifferenceClass{
		&tg.UpdatesDifferenceEmpty{Date: 11, Seq: 2},
	}}
	if err := mgr.RecoverAccount(context.Background(), rpc); err != nil {
		t.Fatalf("RecoverAccount: %v", err)
	}
	if mgr.state.Date != 11 || mgr.state.Seq != 2 {
		t.Fatalf("state = %+v", mgr.state)
	}
}

func TestGetDifferenceSliceLoopsUntilFinal(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state = updateState{Pts: 5, Date: 10, Seq: 1}
	rpc := &fakeDifferenceRPC{diffs: []tg.DifferenceClass{
		&tg.UpdatesDifferenceSlice{
			OtherUpdates:      []tg.UpdateClass{&tg.UpdateDeleteMessages{Messages: []int32{1}, PTS: 6, PTSCount: 1}},
			IntermediateState: &tg.UpdatesState{PTS: 6, Date: 11, Seq: 2},
		},
		&tg.UpdatesDifference{
			State: &tg.UpdatesState{PTS: 7, Qts: 0, Date: 12, Seq: 3},
		},
	}}
	if err := mgr.RecoverAccount(context.Background(), rpc); err != nil {
		t.Fatalf("RecoverAccount: %v", err)
	}
	if mgr.state.Pts != 7 || mgr.state.Seq != 3 {
		t.Fatalf("state = %+v", mgr.state)
	}
	if rpc.calls != 2 {
		t.Fatalf("calls = %d, want 2", rpc.calls)
	}
}

func TestGetDifferenceTooLongRecovery(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state = updateState{Pts: 5, Date: 10, Seq: 1}
	rpc := &fakeDifferenceRPC{diffs: []tg.DifferenceClass{
		&tg.UpdatesDifferenceTooLong{PTS: 100},
		&tg.UpdatesDifferenceEmpty{Date: 12, Seq: 3},
	}}
	if err := mgr.RecoverAccount(context.Background(), rpc); err != nil {
		t.Fatalf("RecoverAccount: %v", err)
	}
	if mgr.state.Pts != 100 {
		t.Fatalf("pts = %d, want 100", mgr.state.Pts)
	}
	if rpc.calls != 2 {
		t.Fatalf("calls = %d, want 2", rpc.calls)
	}
}

func TestRecoverAccountSingleFlight(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state = updateState{Pts: 5, Date: 10, Seq: 1}
	rpc := &fakeDifferenceRPC{diffs: []tg.DifferenceClass{
		&tg.UpdatesDifferenceEmpty{Date: 11, Seq: 2},
	}}

	err := mgr.RecoverAccount(context.Background(), rpc)
	if err != nil {
		t.Fatalf("first RecoverAccount: %v", err)
	}

	mgr.mu.Lock()
	mgr.recovering = true
	mgr.mu.Unlock()

	err = mgr.RecoverAccount(context.Background(), rpc)
	if err != nil {
		t.Fatalf("second RecoverAccount while recovering: %v", err)
	}
	if rpc.calls != 1 {
		t.Fatalf("calls = %d, want 1 (second call should be no-op)", rpc.calls)
	}
}

func TestReconnectTriggersDifferenceRecovery(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.state = updateState{Pts: 5, Date: 10, Seq: 1}
	rpc := &fakeDifferenceRPC{diffs: []tg.DifferenceClass{
		&tg.UpdatesDifferenceEmpty{Date: 11, Seq: 2},
	}}
	if err := mgr.OnReconnect(context.Background(), rpc); err != nil {
		t.Fatalf("OnReconnect: %v", err)
	}
	if rpc.calls != 1 {
		t.Fatalf("calls = %d, want 1", rpc.calls)
	}
}
