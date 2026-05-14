package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

type fakeChannelDifferenceRPC struct {
	result tg.ChannelDifferenceClass
	calls  int
}

func (f *fakeChannelDifferenceRPC) UpdatesGetDifference(ctx context.Context, req *tg.UpdatesGetDifferenceRequest) (tg.DifferenceClass, error) {
	return &tg.UpdatesDifferenceEmpty{}, nil
}

func (f *fakeChannelDifferenceRPC) UpdatesGetChannelDifference(ctx context.Context, req *tg.UpdatesGetChannelDifferenceRequest) (tg.ChannelDifferenceClass, error) {
	f.calls++
	return f.result, nil
}

func TestChannelGapRecovery(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.channels[100] = channelState{ChannelID: 100, Pts: 10}
	rpc := &fakeChannelDifferenceRPC{result: &tg.UpdatesChannelDifference{Final: true, PTS: 12}}
	if err := mgr.RecoverChannel(context.Background(), rpc, 100, &tg.InputChannel{ChannelID: 100}); err != nil {
		t.Fatalf("RecoverChannel: %v", err)
	}
	if got := mgr.channels[100].Pts; got != 12 {
		t.Fatalf("channel pts = %d, want 12", got)
	}
}

func TestGetChannelDifferenceTooLong(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.channels[100] = channelState{ChannelID: 100, Pts: 10}
	rpc := &fakeChannelDifferenceRPC{result: &tg.UpdatesChannelDifferenceTooLong{
		Final:  true,
		Dialog: &tg.Dialog{PTS: 50},
	}}
	if err := mgr.RecoverChannel(context.Background(), rpc, 100, &tg.InputChannel{ChannelID: 100}); err != nil {
		t.Fatalf("RecoverChannel: %v", err)
	}
	if got := mgr.channels[100].Pts; got != 50 {
		t.Fatalf("channel pts = %d, want 50", got)
	}
}

func TestSingleFlightChannelRecovery(t *testing.T) {
	mgr := testUpdateManager(t)
	mgr.channels[100] = channelState{ChannelID: 100, Pts: 10, recovering: true}
	rpc := &fakeChannelDifferenceRPC{result: &tg.UpdatesChannelDifference{Final: true, PTS: 12}}
	if err := mgr.RecoverChannel(context.Background(), rpc, 100, &tg.InputChannel{ChannelID: 100}); err != nil {
		t.Fatalf("RecoverChannel while recovering: %v", err)
	}
	if rpc.calls != 0 {
		t.Fatalf("calls = %d, want 0 (should be no-op)", rpc.calls)
	}
}
