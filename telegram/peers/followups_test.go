package peers

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestAnchorResolutionFallback(t *testing.T) {
	m := newTestManager(0, false, nil)
	// Chat 500 is known; user 42 appeared in message 77 there without a hash.
	m.Cache(500, &tg.InputPeerChat{ChatID: 500})
	m.CacheAnchor(42, 500, 77)

	peer, ok := m.anchorInputPeer(42)
	if !ok {
		t.Fatal("anchor not found")
	}
	fromMsg, ok := peer.(*tg.InputPeerUserFromMessage)
	if !ok || fromMsg.UserID != 42 || fromMsg.MsgID != 77 || fromMsg.Peer == nil {
		t.Fatalf("anchor peer = %+v", peer)
	}
}

func TestAnchorRequiresKnownChat(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.CacheAnchor(42, 999, 77) // chat unknown
	if _, ok := m.anchorInputPeer(42); ok {
		t.Fatal("anchor must not resolve when the containing chat is unknown")
	}
}

func TestAnchorBounded(t *testing.T) {
	store := newAnchorStore(2)
	store.CacheAnchor(1, 10, 1)
	store.CacheAnchor(2, 10, 2)
	store.CacheAnchor(3, 10, 3)
	if _, ok := store.get(1); ok {
		t.Fatal("oldest anchor not evicted")
	}
	if _, ok := store.get(3); !ok {
		t.Fatal("newest anchor missing")
	}
}

type fakePhoneStore struct {
	entry *storage.Peer
}

func (f *fakePhoneStore) SavePeer(*storage.Peer) error         { return nil }
func (f *fakePhoneStore) GetPeer(int64) (*storage.Peer, error) { return nil, nil }
func (f *fakePhoneStore) GetPeerByUsername(string) (*storage.Peer, error) {
	return nil, nil
}
func (f *fakePhoneStore) LoadPeers() ([]*storage.Peer, error) { return nil, nil }
func (f *fakePhoneStore) DeletePeer(int64) error              { return nil }

func (f *fakePhoneStore) GetPeerByPhone(phone string) (*storage.Peer, error) {
	if f.entry != nil && f.entry.PhoneNumber == phone {
		return f.entry, nil
	}
	return nil, nil
}

func TestPhoneStoreReadBack(t *testing.T) {
	entry := &storage.Peer{ID: 7, Type: storage.PeerTypeUser, AccessHash: 9, PhoneNumber: "79001112233"}
	m := NewManager(Deps{
		Store:     func() Store { return &fakePhoneStore{entry: entry} },
		SavePeers: func() bool { return true },
	})

	peer, err := m.InputPeerByPhone(t.Context(), "+79001112233")
	if err != nil {
		t.Fatalf("InputPeerByPhone: %v", err)
	}
	p, ok := peer.(*tg.InputPeerUser)
	if !ok || p.UserID != 7 || p.AccessHash != 9 {
		t.Fatalf("phone read-back = %v", peer)
	}
	// Promoted to memory: second lookup is a hit even without the store.
	if _, ok := m.cachedByPhone("79001112233"); !ok {
		t.Fatal("phone not promoted to memory index")
	}
}

func TestIngestResponseMiddleware(t *testing.T) {
	m := newTestManager(0, false, nil)
	called := false
	var next tg.Invoker = tg.InvokerFunc(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
		called = true
		return &tg.ContactsResolvedPeer{
			Peer:  &tg.PeerUser{UserID: 5},
			Users: []tg.UserClass{&tg.User{ID: 5, AccessHash: 11, Username: "five"}},
		}, nil
	})
	inv := m.IngestResponses()(next)

	res, err := inv.RPCInvoke(t.Context(), &tg.ContactsResolveUsernameRequest{Username: "five"}, nil)
	if err != nil || !called {
		t.Fatalf("invoke: %v called=%v", err, called)
	}
	if _, ok := res.(*tg.ContactsResolvedPeer); !ok {
		t.Fatalf("result passthrough broken: %T", res)
	}
	if _, err := m.Cached(5); err != nil {
		t.Fatalf("response entity not ingested: %v", err)
	}
	if m.LookupUsername(5) != "five" {
		t.Fatal("username index not fed by response ingest")
	}
}

func TestIngestResponseNoEntityTypesSkipped(t *testing.T) {
	m := newTestManager(0, false, nil)
	var next tg.Invoker = tg.InvokerFunc(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
		return &tg.True{}, nil
	})
	inv := m.IngestResponses()(next)
	if _, err := inv.RPCInvoke(t.Context(), &tg.HelpGetConfigRequest{}, nil); err != nil {
		t.Fatal(err)
	}
	if len(layoutCache) == 0 && len(noEntities) == 0 {
		t.Fatal("layout cache not populated")
	}
}
