package telegram

import "github.com/mtgo-labs/mtgo/internal/storage"

type peerStore interface {
	SavePeer(*storage.Peer) error
	GetPeer(id int64) (*storage.Peer, error)
	GetPeerByUsername(username string) (*storage.Peer, error)
	LoadPeers() ([]*storage.Peer, error)
	DeletePeer(id int64) error
}

type legacyPeerCache interface {
	SavePeer(storage.Peer) error
	LoadPeers() ([]storage.Peer, error)
	DeletePeer(id int64) error
}

type legacyPeerStore struct {
	cache legacyPeerCache
}

func (s legacyPeerStore) SavePeer(peer *storage.Peer) error {
	if peer == nil {
		return nil
	}
	existing, err := s.GetPeer(peer.ID)
	if err != nil {
		return err
	}
	return s.cache.SavePeer(*mergePeer(existing, peer))
}

func (s legacyPeerStore) GetPeer(id int64) (*storage.Peer, error) {
	peers, err := s.LoadPeers()
	if err != nil {
		return nil, err
	}
	for _, peer := range peers {
		if peer.ID == id {
			return peer, nil
		}
	}
	return nil, nil
}

func (s legacyPeerStore) GetPeerByUsername(username string) (*storage.Peer, error) {
	peers, err := s.LoadPeers()
	if err != nil {
		return nil, err
	}
	for _, peer := range peers {
		if peer.Username == username {
			return peer, nil
		}
	}
	return nil, nil
}

func (s legacyPeerStore) LoadPeers() ([]*storage.Peer, error) {
	peers, err := s.cache.LoadPeers()
	if err != nil {
		return nil, err
	}
	out := make([]*storage.Peer, len(peers))
	for i := range peers {
		peer := peers[i]
		out[i] = &peer
	}
	return out, nil
}

func (s legacyPeerStore) DeletePeer(id int64) error {
	return s.cache.DeletePeer(id)
}

func (c *Client) peerStore() peerStore {
	if c.storage == nil {
		return nil
	}
	if ps, ok := c.storage.(storage.PeerStore); ok {
		return ps
	}
	if ps, ok := c.storage.(legacyPeerCache); ok {
		return legacyPeerStore{cache: ps}
	}
	return nil
}

func mergePeer(existing, incoming *storage.Peer) *storage.Peer {
	if incoming == nil {
		return existing
	}
	if existing == nil {
		cp := *incoming
		return &cp
	}
	merged := *incoming
	if merged.AccessHash == 0 {
		merged.AccessHash = existing.AccessHash
	}
	if merged.Username == "" {
		merged.Username = existing.Username
	}
	if merged.Usernames == "" {
		merged.Usernames = existing.Usernames
	}
	if merged.FirstName == "" {
		merged.FirstName = existing.FirstName
	}
	if merged.LastName == "" {
		merged.LastName = existing.LastName
	}
	if merged.PhoneNumber == "" {
		merged.PhoneNumber = existing.PhoneNumber
	}
	if !merged.IsBot {
		merged.IsBot = existing.IsBot
	}
	if merged.PhotoID == 0 {
		merged.PhotoID = existing.PhotoID
	}
	if merged.Language == "" {
		merged.Language = existing.Language
	}
	if merged.LastUpdated == 0 {
		merged.LastUpdated = existing.LastUpdated
	}
	return &merged
}
