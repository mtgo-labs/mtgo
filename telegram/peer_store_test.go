package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
)

type legacyStorageStub struct {
	*MemoryStorage
	peers map[int64]storage.Peer
}

func (s *legacyStorageStub) SavePeer(peer storage.Peer) error {
	if s.peers == nil {
		s.peers = make(map[int64]storage.Peer)
	}
	s.peers[peer.ID] = peer
	return nil
}

func (s *legacyStorageStub) LoadPeers() ([]storage.Peer, error) {
	out := make([]storage.Peer, 0, len(s.peers))
	for _, peer := range s.peers {
		out = append(out, peer)
	}
	return out, nil
}

func (s *legacyStorageStub) DeletePeer(id int64) error {
	delete(s.peers, id)
	return nil
}

func TestPeerStoreSupportsLegacyPeerCache(t *testing.T) {
	c := &Client{storage: &legacyStorageStub{MemoryStorage: NewMemoryStorage()}}
	ps := c.peerStore()
	if ps == nil {
		t.Fatal("peerStore returned nil")
	}

	peer := &storage.Peer{
		ID:         123,
		Type:       storage.PeerTypeChannel,
		AccessHash: 456,
		Username:   "channel",
	}
	if err := ps.SavePeer(peer); err != nil {
		t.Fatalf("SavePeer: %v", err)
	}

	got, err := ps.GetPeer(123)
	if err != nil {
		t.Fatalf("GetPeer: %v", err)
	}
	if got == nil || got.ID != peer.ID || got.AccessHash != peer.AccessHash {
		t.Fatalf("GetPeer = %+v", got)
	}

	got, err = ps.GetPeerByUsername("channel")
	if err != nil {
		t.Fatalf("GetPeerByUsername: %v", err)
	}
	if got == nil || got.ID != peer.ID {
		t.Fatalf("GetPeerByUsername = %+v", got)
	}
}

func TestPublicPeerMethodsSupportLegacyPeerCache(t *testing.T) {
	c := &Client{storage: &legacyStorageStub{MemoryStorage: NewMemoryStorage()}}
	peer := &storage.Peer{
		ID:         456,
		Type:       storage.PeerTypeUser,
		AccessHash: 789,
		Username:   "user",
	}
	if err := c.SavePeer(peer); err != nil {
		t.Fatalf("SavePeer: %v", err)
	}

	got, err := c.GetPeerByUsername("user")
	if err != nil {
		t.Fatalf("GetPeerByUsername: %v", err)
	}
	if got == nil || got.ID != peer.ID {
		t.Fatalf("GetPeerByUsername = %+v", got)
	}

	peers, err := c.LoadPeers()
	if err != nil {
		t.Fatalf("LoadPeers: %v", err)
	}
	if len(peers) != 1 || peers[0].ID != peer.ID {
		t.Fatalf("LoadPeers = %+v", peers)
	}
}

func TestLegacyPeerCachePreservesMetadataOnMinimalSave(t *testing.T) {
	c := &Client{storage: &legacyStorageStub{MemoryStorage: NewMemoryStorage()}}
	full := &storage.Peer{
		ID:         789,
		Type:       storage.PeerTypeUser,
		AccessHash: 111,
		Username:   "channel_bot",
		FirstName:  "Channel",
		IsBot:      true,
	}
	if err := c.SavePeer(full); err != nil {
		t.Fatalf("SavePeer full: %v", err)
	}
	if err := c.SavePeer(&storage.Peer{ID: 789, Type: storage.PeerTypeUser, AccessHash: 222}); err != nil {
		t.Fatalf("SavePeer minimal: %v", err)
	}

	got, err := c.GetPeer(789)
	if err != nil {
		t.Fatalf("GetPeer: %v", err)
	}
	if got == nil || got.AccessHash != 222 || got.Username != full.Username || got.FirstName != full.FirstName || !got.IsBot {
		t.Fatalf("GetPeer = %+v", got)
	}
}

func TestChannelChatIDCacheUsesRawID(t *testing.T) {
	c, err := NewClient(1, "hash", &Config{InMemory: true, SessionName: "test", SavePeers: true, NoUpdates: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.storage = NewMemoryStorage()

	peer := &tg.InputPeerChannel{ChannelID: 1795456711, AccessHash: 9048563865316545949}
	c.CachePeer(-1001795456711, peer)

	got, err := c.ResolvePeerCache(-1001795456711)
	if err != nil {
		t.Fatalf("ResolvePeerCache: %v", err)
	}
	ch, ok := got.(*tg.InputPeerChannel)
	if !ok {
		t.Fatalf("ResolvePeerCache type = %T", got)
	}
	if ch.ChannelID != peer.ChannelID {
		t.Fatalf("ChannelID = %d, want %d", ch.ChannelID, peer.ChannelID)
	}

	stored, err := c.GetPeer(1795456711)
	if err != nil {
		t.Fatalf("GetPeer raw: %v", err)
	}
	if stored == nil || stored.ID != 1795456711 {
		t.Fatalf("stored raw peer = %+v", stored)
	}
	if stale, err := c.GetPeer(-1001795456711); err != nil || stale != nil {
		t.Fatalf("stored bot-api peer = %+v, err=%v", stale, err)
	}
}

func TestResolvePeerCacheNormalizesStaleChannelChatID(t *testing.T) {
	c, err := NewClient(1, "hash", &Config{InMemory: true, SessionName: "test", SavePeers: true, NoUpdates: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.storage = NewMemoryStorage()
	if err := c.SavePeer(&storage.Peer{
		ID:         -1001795456711,
		Type:       storage.PeerTypeChannel,
		AccessHash: 9048563865316545949,
	}); err != nil {
		t.Fatalf("SavePeer: %v", err)
	}

	got, err := c.ResolvePeerCache(-1001795456711)
	if err != nil {
		t.Fatalf("ResolvePeerCache: %v", err)
	}
	ch, ok := got.(*tg.InputPeerChannel)
	if !ok {
		t.Fatalf("ResolvePeerCache type = %T", got)
	}
	if ch.ChannelID != 1795456711 {
		t.Fatalf("ChannelID = %d, want 1795456711", ch.ChannelID)
	}
}
