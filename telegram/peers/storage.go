package peers

import "github.com/mtgo-labs/mtgo/internal/storage"

// Store is the persistent peer storage contract used by the Manager. It is
// satisfied by storage backends implementing storage.PeerStore; see
// [StoreFrom] for adaptation.
type Store interface {
	SavePeer(*storage.Peer) error
	GetPeer(id int64) (*storage.Peer, error)
	GetPeerByUsername(username string) (*storage.Peer, error)
	LoadPeers() ([]*storage.Peer, error)
	DeletePeer(id int64) error
}

// legacyPeerCache is the pre-pointer storage contract (value-based
// SavePeer); implemented by older storage backends.
type legacyPeerCache interface {
	SavePeer(storage.Peer) error
	LoadPeers() ([]storage.Peer, error)
	DeletePeer(id int64) error
}

// legacyPeerStore adapts a legacyPeerCache to the Store interface with
// read-modify-write merging.
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
	return s.cache.SavePeer(*storage.MergePeer(existing, peer))
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

// StoreFrom adapts a storage backend to the Store contract. Backends
// implementing storage.PeerStore are used directly; older value-based
// backends are wrapped in a merging adapter; anything else (including nil)
// yields nil, disabling persistence.
func StoreFrom(s storage.Storage) Store {
	if s == nil {
		return nil
	}
	if ps, ok := s.(Store); ok {
		return ps
	}
	if ps, ok := s.(legacyPeerCache); ok {
		return legacyPeerStore{cache: ps}
	}
	return nil
}

// PhoneStore is an optional extension Store backends may implement to make
// persisted phone numbers queryable; when absent, phone lookups rely on the
// in-memory index only.
type PhoneStore interface {
	GetPeerByPhone(phone string) (*storage.Peer, error)
}

// phoneStoreFrom returns the store's phone-query extension, or nil.
func phoneStoreFrom(s Store) PhoneStore {
	if s == nil {
		return nil
	}
	if ps, ok := s.(PhoneStore); ok {
		return ps
	}
	return nil
}
