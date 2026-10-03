package peers

import (
	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
)

// Ingest caches every entity found in an update batch (or any RPC response
// carrying users/chats). Min entities are cached with a zeroed access hash
// so they never shadow a known full hash (see preserveAccessHash), and full
// metadata is persisted when SavePeers is enabled.
//
// This is the funnel for every User/Chat/Channel the client ever observes;
// there must be no other path writing the cache from entities.
func (m *Manager) Ingest(users []tg.UserClass, chats []tg.ChatClass) {
	var entries []*storage.Peer
	for _, u := range users {
		user, ok := u.(*tg.User)
		if !ok || user.AccessHash == 0 {
			continue
		}
		hash := user.AccessHash
		if user.Min {
			hash = 0
		}
		m.Cache(user.ID, &tg.InputPeerUser{UserID: user.ID, AccessHash: hash})
		username := user.Username
		if username != "" {
			m.CacheUsername(username, user.ID)
		}
		if user.Phone != "" {
			m.CachePhone(user.Phone, user.ID)
		}
		entries = append(entries, &storage.Peer{
			ID:          user.ID,
			Type:        storage.PeerTypeUser,
			AccessHash:  hash,
			Username:    username,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			PhoneNumber: user.Phone,
			IsBot:       user.Bot,
			Language:    user.LangCode,
		})
	}
	for _, ch := range chats {
		switch v := ch.(type) {
		case *tg.Chat:
			m.Cache(v.ID, &tg.InputPeerChat{ChatID: v.ID})
			entries = append(entries, &storage.Peer{
				ID:   v.ID,
				Type: storage.PeerTypeChat,
			})
		case *tg.Channel:
			accessHash := v.AccessHash
			if v.Min {
				accessHash = 0
			}
			m.Cache(v.ID, &tg.InputPeerChannel{ChannelID: v.ID, AccessHash: accessHash})
			username := v.Username
			if username != "" {
				m.CacheUsername(username, v.ID)
			}
			entries = append(entries, &storage.Peer{
				ID:         v.ID,
				Type:       storage.PeerTypeChannel,
				AccessHash: accessHash,
				Username:   username,
			})
		}
	}
	m.persist(entries)
}

// IngestUsers is Ingest for a users-only batch.
func (m *Manager) IngestUsers(users []tg.UserClass) { m.Ingest(users, nil) }

// IngestChats is Ingest for a chats-only batch.
func (m *Manager) IngestChats(chats []tg.ChatClass) { m.Ingest(nil, chats) }

// ingestResolved caches the entities of a contacts.resolveUsername /
// contacts.resolvePhone response. Unlike Ingest, entries without an access
// hash are skipped entirely, matching server semantics for resolve results.
func (m *Manager) ingestResolved(result *tg.ContactsResolvedPeer) {
	var entries []*storage.Peer
	for _, u := range result.Users {
		user, ok := u.(*tg.User)
		if !ok {
			continue
		}
		if user.AccessHash != 0 {
			m.Cache(user.ID, &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash})
			if user.Username != "" {
				m.CacheUsername(user.Username, user.ID)
			}
			if user.Phone != "" {
				m.CachePhone(user.Phone, user.ID)
			}
			entries = append(entries, &storage.Peer{
				ID:          user.ID,
				Type:        storage.PeerTypeUser,
				AccessHash:  user.AccessHash,
				Username:    user.Username,
				FirstName:   user.FirstName,
				LastName:    user.LastName,
				PhoneNumber: user.Phone,
				IsBot:       user.Bot,
				Language:    user.LangCode,
			})
		}
	}
	for _, ch := range result.Chats {
		switch v := ch.(type) {
		case *tg.Chat:
			m.Cache(v.ID, &tg.InputPeerChat{ChatID: v.ID})
			entries = append(entries, &storage.Peer{
				ID:   v.ID,
				Type: storage.PeerTypeChat,
			})
		case *tg.Channel:
			if v.AccessHash != 0 {
				m.Cache(v.ID, &tg.InputPeerChannel{ChannelID: v.ID, AccessHash: v.AccessHash})
				if v.Username != "" {
					m.CacheUsername(v.Username, v.ID)
				}
				entries = append(entries, &storage.Peer{
					ID:         v.ID,
					Type:       storage.PeerTypeChannel,
					AccessHash: v.AccessHash,
					Username:   v.Username,
				})
			}
		}
	}
	m.persist(entries)
}

// LoadFromStore promotes every persisted peer into the in-memory cache.
// Called at connect time; a no-op when persistence is disabled or no store
// is configured.
func (m *Manager) LoadFromStore() {
	if !m.savePeers() {
		return
	}
	ps := m.store()
	if ps == nil {
		return
	}
	peers, err := ps.LoadPeers()
	if err != nil {
		return
	}
	for _, p := range peers {
		peer, err := peerFromStorageEntry(p)
		if err != nil {
			continue
		}
		m.Cache(p.ID, peer)
		if p.Username != "" {
			m.CacheUsername(p.Username, p.ID)
		}
	}
}

// persist writes storage entries when persistence is enabled. Failures are
// deliberately ignored: the cache is the source of truth for this session.
func (m *Manager) persist(entries []*storage.Peer) {
	if !m.savePeers() || len(entries) == 0 {
		return
	}
	if ps := m.store(); ps != nil {
		for _, entry := range entries {
			_ = ps.SavePeer(entry)
		}
	}
}

// peerFromStorageEntry rebuilds an input peer from a persisted record.
func peerFromStorageEntry(p *storage.Peer) (tg.InputPeerClass, error) {
	switch p.Type {
	case storage.PeerTypeUser:
		return &tg.InputPeerUser{UserID: p.ID, AccessHash: p.AccessHash}, nil
	case storage.PeerTypeChat:
		return &tg.InputPeerChat{ChatID: p.ID}, nil
	case storage.PeerTypeChannel:
		return &tg.InputPeerChannel{ChannelID: p.ID, AccessHash: p.AccessHash}, nil
	default:
		return nil, ErrNotFound
	}
}
