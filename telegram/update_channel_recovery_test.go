package telegram

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestChannelDiffConcurrencyLimit(t *testing.T) {
	st := newTestStorage()
	c, _ := NewClient(1, "hash", &Config{Storage: st})
	mgr := newUpdateManager(c, st, updateManagerConfig{
		QueueSize:                 4,
		MaxChannelDiffConcurrency: 2,
	})

	var concurrent int32
	var maxConcurrent int32
	var mu sync.Mutex

	rpc := &concurrencyTrackingRPC{
		fn: func(ctx context.Context, req *tg.UpdatesGetChannelDifferenceRequest) (tg.ChannelDifferenceClass, error) {
			cur := atomic.AddInt32(&concurrent, 1)
			mu.Lock()
			if int(cur) > int(maxConcurrent) {
				atomic.StoreInt32(&maxConcurrent, cur)
			}
			mu.Unlock()
			time.Sleep(20 * time.Millisecond)
			atomic.AddInt32(&concurrent, -1)
			return &tg.UpdatesChannelDifferenceEmpty{PTS: req.PTS, Final: true}, nil
		},
	}

	var wg sync.WaitGroup
	for i := int64(1); i <= 6; i++ {
		mgr.mu.Lock()
		mgr.channels[i] = channelState{ChannelID: i, Pts: 1}
		mgr.mu.Unlock()
		wg.Add(1)
		go func(chID int64) {
			defer wg.Done()
			_ = mgr.RecoverChannel(context.Background(), rpc, chID, &tg.InputChannel{ChannelID: chID})
		}(i)
	}
	wg.Wait()

	if mx := atomic.LoadInt32(&maxConcurrent); mx > 2 {
		t.Fatalf("max concurrent getChannelDifference = %d, want <= 2", mx)
	}
}

type concurrencyTrackingRPC struct {
	fn func(ctx context.Context, req *tg.UpdatesGetChannelDifferenceRequest) (tg.ChannelDifferenceClass, error)
}

func (c *concurrencyTrackingRPC) UpdatesGetDifference(ctx context.Context, req *tg.UpdatesGetDifferenceRequest) (tg.DifferenceClass, error) {
	return &tg.UpdatesDifferenceEmpty{}, nil
}

func (c *concurrencyTrackingRPC) UpdatesGetChannelDifference(ctx context.Context, req *tg.UpdatesGetChannelDifferenceRequest) (tg.ChannelDifferenceClass, error) {
	return c.fn(ctx, req)
}
