package peers

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

func TestInvalidateDropsAllForms(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.Cache(100, &tg.InputPeerChannel{ChannelID: 100, AccessHash: 5})
	m.CacheUsername("news", 100)
	m.CachePhone("79001112233", 100)

	m.Invalidate(100)

	if _, err := m.Cached(100); err == nil {
		t.Error("peer entry survived Invalidate")
	}
	if m.LookupUsername(100) != "" {
		t.Error("username index survived Invalidate")
	}
	if _, ok := m.cachedByPhone("79001112233"); ok {
		t.Error("phone index survived Invalidate")
	}
}

func TestInvalidateNormalizesMarkedChannelID(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.Cache(200, &tg.InputPeerChannel{ChannelID: 200, AccessHash: 5})

	m.Invalidate(-1000000000200)

	if _, err := m.Cached(200); err == nil {
		t.Error("marked-form Invalidate did not drop the raw-keyed entry")
	}
}

func TestRequestPeerIDs(t *testing.T) {
	req := &tg.MessagesSendMessageRequest{
		Peer: &tg.InputPeerChannel{ChannelID: 55, AccessHash: 1},
	}
	ids := requestPeerIDs(req)
	if len(ids) != 1 || ids[0] != 55 {
		t.Fatalf("ids = %v, want [55]", ids)
	}
}

func TestStaleHashInvokerInvalidatesAndReplays(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.Cache(55, &tg.InputPeerChannel{ChannelID: 55, AccessHash: 1})

	calls := 0
	var next tg.Invoker = tg.InvokerFunc(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
		calls++
		if calls == 1 {
			return nil, tgerr.New(400, "PEER_ID_INVALID")
		}
		return &tg.True{}, nil
	})
	inv := m.InvalidateOnStaleHash()(next)

	_, err := inv.RPCInvoke(t.Context(),
		&tg.MessagesSendMessageRequest{Peer: &tg.InputPeerChannel{ChannelID: 55}},
		nil)
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2 (reject + replay)", calls)
	}
	if _, cacheErr := m.Cached(55); cacheErr == nil {
		t.Error("stale entry not invalidated before replay")
	}
}

func TestStaleHashInvokerPassesOtherErrors(t *testing.T) {
	m := newTestManager(0, false, nil)
	calls := 0
	var next tg.Invoker = tg.InvokerFunc(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
		calls++
		return nil, tgerr.New(420, "FLOOD_WAIT_5")
	})
	inv := m.InvalidateOnStaleHash()(next)

	_, err := inv.RPCInvoke(t.Context(),
		&tg.MessagesSendMessageRequest{Peer: &tg.InputPeerChannel{ChannelID: 55}},
		nil)
	if err == nil {
		t.Fatal("expected flood error to propagate")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 (no replay on flood)", calls)
	}
}
