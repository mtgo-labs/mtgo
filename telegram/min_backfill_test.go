package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func newMinBackfillClient(t *testing.T) *Client {
	t.Helper()
	c, err := NewClient(1, "hash", &Config{SavePeers: true, NoUpdates: true})
	if err != nil {
		t.Fatal(err)
	}
	c.storage = NewMemoryStorage()
	return c
}

func TestCachePeersMinChannelDoesNotOverwriteFullHash(t *testing.T) {
	c := newMinBackfillClient(t)

	fullHash := int64(9048563865316545949)
	c.cachePeersFromUpdates(nil, []tg.ChatClass{
		&tg.Channel{ID: 100, AccessHash: fullHash},
	})

	got, err := c.peerStore().GetPeer(100)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.AccessHash != fullHash {
		t.Fatalf("seed: stored peer = %v, want hash %d", got, fullHash)
	}

	minHash := int64(1111)
	c.cachePeersFromUpdates(nil, []tg.ChatClass{
		&tg.Channel{ID: 100, AccessHash: minHash, Min: true},
	})

	got, err = c.peerStore().GetPeer(100)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.AccessHash != fullHash {
		t.Fatalf("after min: stored peer = %v, want hash %d (full hash preserved)", got, fullHash)
	}
}

func TestCachePeersMinUserDoesNotOverwriteFullHash(t *testing.T) {
	c := newMinBackfillClient(t)

	fullHash := int64(555555)
	c.cachePeersFromUpdates([]tg.UserClass{
		&tg.User{ID: 50, AccessHash: fullHash},
	}, nil)

	got, err := c.peerStore().GetPeer(50)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.AccessHash != fullHash {
		t.Fatalf("seed: stored peer = %v, want hash %d", got, fullHash)
	}

	minHash := int64(222)
	c.cachePeersFromUpdates([]tg.UserClass{
		&tg.User{ID: 50, AccessHash: minHash, Min: true},
	}, nil)

	got, err = c.peerStore().GetPeer(50)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.AccessHash != fullHash {
		t.Fatalf("after min: stored peer = %v, want hash %d (full hash preserved)", got, fullHash)
	}
}

func TestBackfillMinAccessHashesChannel(t *testing.T) {
	c := newMinBackfillClient(t)

	fullHash := int64(9048563865316545949)
	c.cachePeersFromUpdates(nil, []tg.ChatClass{
		&tg.Channel{ID: 200, AccessHash: fullHash},
	})

	minHash := int64(777)
	minCh := types.ParseChatFromChat(&tg.Channel{ID: 200, AccessHash: minHash, Min: true})
	chatMap := map[int64]*types.Chat{minCh.ID: minCh}

	c.backfillMinAccessHashes(chatMap, nil)

	ch := chatMap[minCh.ID]
	if ch.AccessHash != fullHash {
		t.Fatalf("channel hash after backfill = %d, want %d", ch.AccessHash, fullHash)
	}
}

func TestBackfillMinAccessHashesChannelNoStoredHash(t *testing.T) {
	c := newMinBackfillClient(t)

	minHash := int64(777)
	minCh := types.ParseChatFromChat(&tg.Channel{ID: 300, AccessHash: minHash, Min: true})
	chatMap := map[int64]*types.Chat{minCh.ID: minCh}

	c.backfillMinAccessHashes(chatMap, nil)

	ch := chatMap[minCh.ID]
	if ch.AccessHash != minHash {
		t.Fatalf("channel hash = %d, want %d (unchanged, no stored hash)", ch.AccessHash, minHash)
	}
}

func TestBackfillMinAccessHashesUser(t *testing.T) {
	c := newMinBackfillClient(t)

	fullHash := int64(555555)
	c.cachePeersFromUpdates([]tg.UserClass{
		&tg.User{ID: 60, AccessHash: fullHash},
	}, nil)

	minHash := int64(444)
	userMap := map[int64]*types.User{
		60: {
			ID:         60,
			IsMin:      true,
			AccessHash: minHash,
			Raw:        &tg.User{ID: 60, AccessHash: minHash, Min: true},
		},
	}

	c.backfillMinAccessHashes(nil, userMap)

	u := userMap[60]
	if u.AccessHash != fullHash {
		t.Fatalf("user hash after backfill = %d, want %d", u.AccessHash, fullHash)
	}
}

func TestBackfillMinAccessHashesNonMinUnchanged(t *testing.T) {
	c := newMinBackfillClient(t)

	original := int64(999999)
	ch := types.ParseChatFromChat(&tg.Channel{ID: 400, AccessHash: original})
	chatMap := map[int64]*types.Chat{ch.ID: ch}

	c.backfillMinAccessHashes(chatMap, nil)

	if chatMap[ch.ID].AccessHash != original {
		t.Fatalf("non-min channel hash changed = %d, want %d", chatMap[ch.ID].AccessHash, original)
	}
}
