package peers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
)

// recordingStore is a Store stub that records writes and serves them back
// from its maps.
type recordingStore struct {
	mu       sync.Mutex
	saved    []*storage.Peer
	byID     map[int64]*storage.Peer
	byName   map[string]*storage.Peer
	saveHits int
}

func newRecordingStore() *recordingStore {
	return &recordingStore{
		byID:   make(map[int64]*storage.Peer),
		byName: make(map[string]*storage.Peer),
	}
}

func (r *recordingStore) SavePeer(p *storage.Peer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.saved = append(r.saved, &cp)
	r.byID[p.ID] = &cp
	if p.Username != "" {
		r.byName[p.Username] = &cp
	}
	r.saveHits++
	return nil
}

func (r *recordingStore) GetPeer(id int64) (*storage.Peer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.byID[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, nil
}

func (r *recordingStore) GetPeerByUsername(username string) (*storage.Peer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.byName[username]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, nil
}

func (r *recordingStore) LoadPeers() ([]*storage.Peer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*storage.Peer, 0, len(r.byID))
	for _, p := range r.byID {
		cp := *p
		out = append(out, &cp)
	}
	return out, nil
}

func (r *recordingStore) DeletePeer(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.byID[id]; ok {
		delete(r.byName, p.Username)
	}
	delete(r.byID, id)
	return nil
}

func (r *recordingStore) writeCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveHits
}

func TestUsernameResolutionReadsStore(t *testing.T) {
	store := newRecordingStore()
	store.byName["nine"] = &storage.Peer{
		ID: 9, Type: storage.PeerTypeUser, AccessHash: 12,
		Username: "nine", LastUpdated: time.Now().Unix(),
	}
	rpcCalls := 0
	m := NewManager(Deps{
		Invoker: func() *tg.RPCClient {
			return tg.NewRPCClient(tg.InvokerFunc(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
				rpcCalls++
				return nil, errStoreMustNotRPC
			}))
		},
		Store:     func() Store { return store },
		SavePeers: func() bool { return true },
	})

	for i := 0; i < 2; i++ { // second call must be served from the memory promotion
		peer, err := m.InputPeerByUsername(t.Context(), "nine")
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		p, ok := peer.(*tg.InputPeerUser)
		if !ok || p.UserID != 9 || p.AccessHash != 12 {
			t.Fatalf("call %d: peer = %+v", i, peer)
		}
	}
	if rpcCalls != 0 {
		t.Fatalf("store hit should not RPC, got %d calls", rpcCalls)
	}
}

var errStoreMustNotRPC = &storeRPCError{}

type storeRPCError struct{}

func (*storeRPCError) Error() string { return "unexpected RPC while store should serve the lookup" }

func TestUsernameStoreEntryTTLExpiry(t *testing.T) {
	store := newRecordingStore()
	store.byName["stale"] = &storage.Peer{
		ID: 5, Type: storage.PeerTypeUser, AccessHash: 1,
		Username: "stale", LastUpdated: time.Now().Add(-25 * time.Hour).Unix(),
	}
	rpcCalls := 0
	m := NewManager(Deps{
		Invoker: func() *tg.RPCClient {
			return tg.NewRPCClient(tg.InvokerFunc(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
				rpcCalls++
				return &tg.ContactsResolvedPeer{
					Peer:  &tg.PeerUser{UserID: 5},
					Users: []tg.UserClass{&tg.User{ID: 5, AccessHash: 2, Username: "stale"}},
				}, nil
			}))
		},
		Store:     func() Store { return store },
		SavePeers: func() bool { return true },
	})

	peer, err := m.InputPeerByUsername(t.Context(), "stale")
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := peer.(*tg.InputPeerUser); !ok || p.AccessHash != 2 {
		t.Fatalf("peer = %+v, want fresh RPC resolution", peer)
	}
	if rpcCalls != 1 {
		t.Fatalf("expired store entry must fall through to RPC, calls = %d", rpcCalls)
	}
	// The re-resolution re-stamps the record so the next lookup is a store hit.
	entry, _ := store.GetPeerByUsername("stale")
	if entry == nil || time.Since(time.Unix(entry.LastUpdated, 0)) > time.Minute {
		t.Fatalf("record not re-stamped: %+v", entry)
	}
}

func TestUsernameStoreZeroHashSkipped(t *testing.T) {
	store := newRecordingStore()
	store.byName["hashless"] = &storage.Peer{
		ID: 3, Type: storage.PeerTypeUser, AccessHash: 0,
		Username: "hashless", LastUpdated: time.Now().Unix(),
	}
	rpcCalls := 0
	m := NewManager(Deps{
		Invoker: func() *tg.RPCClient {
			return tg.NewRPCClient(tg.InvokerFunc(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
				rpcCalls++
				return &tg.ContactsResolvedPeer{
					Peer:  &tg.PeerUser{UserID: 3},
					Users: []tg.UserClass{&tg.User{ID: 3, AccessHash: 4, Username: "hashless"}},
				}, nil
			}))
		},
		Store:     func() Store { return store },
		SavePeers: func() bool { return true },
	})

	peer, err := m.InputPeerByUsername(t.Context(), "hashless")
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := peer.(*tg.InputPeerUser); !ok || p.AccessHash != 4 {
		t.Fatalf("peer = %+v", peer)
	}
	if rpcCalls != 1 {
		t.Fatalf("zero-hash record must be skipped for RPC, calls = %d", rpcCalls)
	}
}

func TestLoadFromStoreRestoresIndexes(t *testing.T) {
	store := newRecordingStore()
	store.byID[1] = &storage.Peer{
		ID: 1, Type: storage.PeerTypeUser, AccessHash: 3,
		Username: "main", Usernames: "alias_a,alias_b", PhoneNumber: "79001112233",
	}
	store.byName["main"] = store.byID[1]
	store.byID[2] = &storage.Peer{ID: 2, Type: storage.PeerTypeChat}

	m := newTestManager(0, true, store)
	m.LoadFromStore()

	if _, err := m.Cached(1); err != nil {
		t.Fatalf("Cached(1): %v", err)
	}
	if _, ok := m.cachedByUsername("main"); !ok {
		t.Fatal("primary username not restored")
	}
	for _, alias := range []string{"alias_a", "alias_b"} {
		if _, ok := m.cachedByUsername(alias); !ok {
			t.Fatalf("collectible username %q not restored", alias)
		}
	}
	if _, ok := m.cachedByPhone("79001112233"); !ok {
		t.Fatal("phone index not restored")
	}
	if _, err := m.Cached(2); err != nil {
		t.Fatalf("Cached(2): %v", err)
	}
	if got := store.writeCount(); got != 0 {
		t.Fatalf("LoadFromStore rewrote the store %d times; promotion must be read-only", got)
	}
}

func TestIngestForbiddenEntities(t *testing.T) {
	store := newRecordingStore()
	m := newTestManager(0, true, store)
	m.Ingest(nil, []tg.ChatClass{
		&tg.ChatForbidden{ID: 77},
		&tg.ChannelForbidden{ID: 88, AccessHash: 3},
	})

	peer, err := m.Cached(77)
	if err != nil {
		t.Fatalf("Cached(77): %v", err)
	}
	if _, ok := peer.(*tg.InputPeerChat); !ok {
		t.Fatalf("forbidden chat cached as %T", peer)
	}
	peer, err = m.Cached(88)
	if err != nil {
		t.Fatalf("Cached(88): %v", err)
	}
	ch, ok := peer.(*tg.InputPeerChannel)
	if !ok || ch.AccessHash != 3 {
		t.Fatalf("forbidden channel cached as %+v", peer)
	}
	entry, _ := store.GetPeer(88)
	if entry == nil || entry.Type != storage.PeerTypeChannel || entry.AccessHash != 3 {
		t.Fatalf("forbidden channel record = %+v", entry)
	}
}

func TestIngestResponseVectorEntities(t *testing.T) {
	m := newTestManager(0, false, nil)
	var next tg.Invoker = tg.InvokerFunc(func(context.Context, tg.TLObject, func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
		return &tg.GenericVector{Items: []tg.TLObject{
			&tg.User{ID: 5, AccessHash: 11, Username: "vec"},
			&tg.Chat{ID: 6},
		}}, nil
	})
	inv := m.IngestResponses()(next)

	if _, err := inv.RPCInvoke(t.Context(), &tg.UsersGetUsersRequest{}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Cached(5); err != nil {
		t.Fatalf("vector user not ingested: %v", err)
	}
	if m.LookupUsername(5) != "vec" {
		t.Fatal("vector user username index not fed")
	}
	if _, err := m.Cached(6); err != nil {
		t.Fatalf("vector chat not ingested: %v", err)
	}
}

func TestIngestCollectibleUsernames(t *testing.T) {
	store := newRecordingStore()
	m := newTestManager(0, true, store)
	m.Ingest([]tg.UserClass{&tg.User{
		ID: 4, AccessHash: 1,
		Username:  "primary",
		Usernames: []*tg.Username{{Username: "alias", Active: true}, {Username: "dropped"}},
	}}, nil)

	if _, ok := m.cachedByUsername("alias"); !ok {
		t.Fatal("active collectible username not indexed")
	}
	if _, ok := m.cachedByUsername("dropped"); ok {
		t.Fatal("inactive collectible username must not be indexed")
	}
	entry, _ := store.GetPeer(4)
	if entry == nil || entry.Usernames != "alias" {
		t.Fatalf("collectible usernames not persisted: %+v", entry)
	}
}

func TestPersistStampsLastUpdated(t *testing.T) {
	store := newRecordingStore()
	m := newTestManager(0, true, store)
	m.Ingest([]tg.UserClass{&tg.User{ID: 8, AccessHash: 2, Username: "stamped"}}, nil)

	if len(store.saved) == 0 {
		t.Fatal("nothing persisted")
	}
	for _, entry := range store.saved {
		if entry.LastUpdated == 0 {
			t.Fatalf("entry %+v has no last-update stamp", entry)
		}
	}
}
