package peers

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/peerid"
	"github.com/mtgo-labs/mtgo/tg"
)

func newTestManager(cacheSize int, save bool, store Store) *Manager {
	return NewManager(Deps{
		CacheSize: func() int { return cacheSize },
		SavePeers: func() bool { return save },
		Store:     func() Store { return store },
	})
}

func TestCacheRoundtrip(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.Cache(42, &tg.InputPeerUser{UserID: 42, AccessHash: 7})

	peer, err := m.Cached(42)
	if err != nil {
		t.Fatalf("Cached(42) error: %v", err)
	}
	if p, ok := peer.(*tg.InputPeerUser); !ok || p.AccessHash != 7 {
		t.Fatalf("Cached(42) = %T %+v, want InputPeerUser hash 7", peer, peer)
	}
}

func TestCachedByMarkedChannelID(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.Cache(peerid.MarkChannel(1234), &tg.InputPeerChannel{ChannelID: 1234, AccessHash: 5})

	// Both the raw and the marked form must find the entry.
	for _, id := range []int64{1234, peerid.MarkChannel(1234)} {
		if _, err := m.Cached(id); err != nil {
			t.Errorf("Cached(%d) error: %v", id, err)
		}
	}
}

func TestCachedMissIsErrNotFound(t *testing.T) {
	m := newTestManager(0, false, nil)
	if _, err := m.Cached(1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Cached miss err = %v, want ErrNotFound", err)
	}
}

func TestCachePreservesFullHashOverMin(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.Cache(5, &tg.InputPeerUser{UserID: 5, AccessHash: 9})
	// A min entity arrives with zero hash and must not poison the entry.
	m.Cache(5, &tg.InputPeerUser{UserID: 5, AccessHash: 0})

	peer, err := m.Cached(5)
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := peer.(*tg.InputPeerUser); !ok || p.AccessHash != 9 {
		t.Fatalf("cached hash = %v, want 9 preserved", peer)
	}
}

func TestCacheUsernameEviction(t *testing.T) {
	m := newTestManager(2, false, nil)
	m.CacheUsername("alice", 1)
	m.CacheUsername("bob", 2)
	m.CacheUsername("carol", 3)

	if m.LookupUsername(1) != "" {
		t.Error("alice should have been evicted first")
	}
	if m.LookupUsername(3) != "carol" {
		t.Error("carol should be cached")
	}
}

func TestIngestZeroHashUserCachedButUnusable(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.IngestUsers([]tg.UserClass{&tg.User{ID: 10, AccessHash: 0}})

	// The entry exists (indexes stay fresh)...
	peer, err := m.Cached(10)
	if err != nil {
		t.Fatalf("Cached(10): %v", err)
	}
	if p, ok := peer.(*tg.InputPeerUser); !ok || p.AccessHash != 0 {
		t.Fatalf("cached zero-hash user = %v", peer)
	}
	// ...but is rejected for RPC use by the cascade.
	if _, err := m.cachedForUse(10); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cachedForUse(10) err = %v, want ErrNotFound", err)
	}
}

func TestIngestMinUserZeroesHash(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.IngestUsers([]tg.UserClass{&tg.User{ID: 10, AccessHash: 77, Min: true}})

	peer, err := m.Cached(10)
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := peer.(*tg.InputPeerUser); !ok || p.AccessHash != 0 {
		t.Fatalf("min user hash = %v, want 0", peer)
	}
}

func TestIngestChannelAndUsername(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.IngestChats([]tg.ChatClass{&tg.Channel{ID: 100, AccessHash: 5, Username: "news"}})

	if m.LookupUsername(100) != "news" {
		t.Error("channel username not cached")
	}
	if p, err := m.Cached(100); err != nil {
		t.Fatalf("Cached(100): %v", err)
	} else if ip, ok := p.(*tg.InputPeerChannel); !ok || ip.AccessHash != 5 {
		t.Fatalf("Cached(100) = %v", p)
	}
}

func TestPhoneIndex(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.CachePhone("+7900123", 77) // normalized automatically
	m.Cache(77, &tg.InputPeerUser{UserID: 77, AccessHash: 3})

	peer, err := m.InputPeerByPhone(t.Context(), "+7900123")
	if err != nil {
		t.Fatalf("InputPeerByPhone: %v", err)
	}
	if p, ok := peer.(*tg.InputPeerUser); !ok || p.UserID != 77 {
		t.Fatalf("phone resolve = %v, want user 77", peer)
	}

	m.IngestUsers([]tg.UserClass{&tg.User{ID: 8, AccessHash: 1, Phone: "7900555"}})
	if _, err := m.InputPeerByPhone(t.Context(), "7900555"); err != nil {
		t.Errorf("ingested phone not indexed: %v", err)
	}
}

func TestUsernameReverseIndex(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.CacheUsername("alice", 1)
	if got := m.LookupUsername(1); got != "alice" {
		t.Fatalf("LookupUsername(1) = %q, want alice", got)
	}
	// Re-binding the ID to a new username drops the old one.
	m.CacheUsername("alice_new", 1)
	if got := m.LookupUsername(1); got != "alice_new" {
		t.Fatalf("LookupUsername(1) = %q, want alice_new", got)
	}
}

func TestReset(t *testing.T) {
	m := newTestManager(0, false, nil)
	m.Cache(1, &tg.InputPeerChat{ChatID: 1})
	m.CacheUsername("a", 1)
	m.Reset()

	if _, err := m.Cached(1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after Reset err = %v, want ErrNotFound", err)
	}
	if m.LookupUsername(1) != "" {
		t.Error("username cache not reset")
	}
}

func TestPeerToInputPeerNotFound(t *testing.T) {
	_, err := PeerToInputPeer(&tg.PeerUser{UserID: 99}, nil, nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPeerToInputPeerSelfForZeroUser(t *testing.T) {
	peer, err := PeerToInputPeer(&tg.PeerUser{UserID: 0}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := peer.(*tg.InputPeerSelf); !ok {
		t.Fatalf("peer = %T, want InputPeerSelf", peer)
	}
}

func TestInputUserRejectsNonUser(t *testing.T) {
	if _, err := newTestManager(0, false, nil).InputUser(t.Context(), Ref{Peer: &tg.InputPeerChat{ChatID: 1}}); err == nil {
		t.Fatal("InputUser on chat peer should fail")
	}
}

func TestInputChannelRejectsNonChannel(t *testing.T) {
	if _, err := newTestManager(0, false, nil).InputChannel(t.Context(), Ref{Peer: &tg.InputPeerUser{UserID: 1}}); err == nil {
		t.Fatal("InputChannel on user peer should fail")
	}
}

func TestInputPeerSelfForZeroRef(t *testing.T) {
	peer, err := newTestManager(0, false, nil).InputPeer(t.Context(), Ref{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := peer.(*tg.InputPeerSelf); !ok {
		t.Fatalf("peer = %T, want InputPeerSelf", peer)
	}
}

func TestCoalescerSingleManagerDedup(t *testing.T) {
	m := newTestManager(0, false, nil)
	var calls atomic.Int64
	release := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = coalesce(m, t.Context(), "phone:+7", func() (tg.InputPeerClass, error) {
				calls.Add(1)
				<-release
				return &tg.InputPeerSelf{}, nil
			})
		}()
	}
	// Wait until one leader is blocked in fn and the rest are queued as waiters.
	deadline := time.Now().Add(2 * time.Second)
	for {
		m.coalescer.mu.Lock()
		waiters := 0
		if w, ok := m.coalescer.inFlight["phone:+7"]; ok {
			waiters = len(w)
		}
		m.coalescer.mu.Unlock()
		if waiters == 7 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("waiters did not accumulate")
		}
		time.Sleep(time.Millisecond)
	}
	close(release)
	wg.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("fn called %d times, want 1", got)
	}
}
