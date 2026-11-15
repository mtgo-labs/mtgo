package peers

import (
	"strings"
	"time"

	"github.com/mtgo-labs/mtgo/internal/storage"
	"github.com/mtgo-labs/mtgo/tg"
)

// Ingest caches every entity found in an update batch (or any RPC response
// carrying users/chats). Min entities are cached with a zeroed access hash
// so they never shadow a known full hash (see preserveAccessHash), forbidden
// variants are cached too (the channel one still carries a usable access
// hash), collectible usernames feed the username index, and full metadata is
// persisted when SavePeers is enabled.
//
// This is the funnel for every User/Chat/Channel the client ever observes;
// there must be no other path writing the cache from entities.
func (m *Manager) Ingest(users []tg.UserClass, chats []tg.ChatClass) {
	var entries []*storage.Peer
	for _, u := range users {
		user, ok := u.(*tg.User)
		if !ok {
			continue
		}
		// Min (or hashless) users are still ingested: the cache's
		// preserve-hash guard keeps any known full hash, the username/phone
		// indexes stay fresh across renames, and storage merges only into
		// empty fields. cachedForUse rejects zero-hash entries for RPC use.
		hash := user.AccessHash
		if user.Min {
			hash = 0
		}
		m.Cache(user.ID, &tg.InputPeerUser{UserID: user.ID, AccessHash: hash})
		username := user.Username
		if username != "" {
			m.CacheUsername(username, user.ID)
		}
		m.cacheCollectibleUsernames(user.Usernames, user.ID)
		if user.Phone != "" {
			m.CachePhone(user.Phone, user.ID)
		}
		entries = append(entries, &storage.Peer{
			ID:          user.ID,
			Type:        storage.PeerTypeUser,
			AccessHash:  hash,
			Username:    username,
			Usernames:   joinUsernames(user.Usernames),
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
		case *tg.ChatForbidden:
			// Basic groups need no access hash, so even the forbidden view
			// of a left group keeps resolving.
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
			m.cacheCollectibleUsernames(v.Usernames, v.ID)
			entries = append(entries, &storage.Peer{
				ID:         v.ID,
				Type:       storage.PeerTypeChannel,
				AccessHash: accessHash,
				Username:   username,
				Usernames:  joinUsernames(v.Usernames),
			})
		case *tg.ChannelForbidden:
			// The forbidden variant still carries a usable access hash.
			m.Cache(v.ID, &tg.InputPeerChannel{ChannelID: v.ID, AccessHash: v.AccessHash})
			entries = append(entries, &storage.Peer{
				ID:         v.ID,
				Type:       storage.PeerTypeChannel,
				AccessHash: v.AccessHash,
			})
		}
	}
	m.persist(entries)
}

// cacheCollectibleUsernames replaces the collectible-username set of a peer
// with the active aliases of the entity, so t.me links using an alias
// resolve without a server round-trip and dropped aliases stop resolving.
func (m *Manager) cacheCollectibleUsernames(collectibles []*tg.Username, id int64) {
	var aliases []string
	for _, un := range collectibles {
		if un != nil && un.Active && un.Username != "" {
			aliases = append(aliases, un.Username)
		}
	}
	m.SetAliases(id, aliases)
}

// joinUsernames renders the active collectible usernames as a comma-separated
// list for the storage record. Inactive (already re-sold or dropped) aliases
// are dropped so a persisted index never resolves to a new owner.
func joinUsernames(collectibles []*tg.Username) string {
	var b strings.Builder
	for _, un := range collectibles {
		if un == nil || !un.Active || un.Username == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(un.Username)
	}
	return b.String()
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

// LoadFromStore promotes every persisted peer into the in-memory cache,
// including its username, collectible-username, and phone index entries.
// Called at connect time; a no-op when persistence is disabled or no store
// is configured. Promotion is read-only: the records came from the store
// itself, so they are not written back (that would rewrite the whole table
// and refresh every TTL stamp on each connect).
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
		m.cacheMem(p.ID, peer)
		if p.Username != "" {
			m.CacheUsername(p.Username, p.ID)
		}
		if p.Usernames != "" {
			m.SetAliases(p.ID, strings.Split(p.Usernames, ","))
		}
		if p.PhoneNumber != "" {
			m.CachePhone(p.PhoneNumber, p.ID)
		}
	}
}

// persist writes storage entries when persistence is enabled. Each record is
// stamped with the current time so username lookups served from the store
// can honor the index TTL. Failures are deliberately ignored: the cache is
// the source of truth for this session.
func (m *Manager) persist(entries []*storage.Peer) {
	if !m.savePeers() || len(entries) == 0 {
		return
	}
	if ps := m.store(); ps != nil {
		now := time.Now().Unix()
		for _, entry := range entries {
			entry.LastUpdated = now
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
