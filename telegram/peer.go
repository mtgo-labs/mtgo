package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/peers"
	"github.com/mtgo-labs/mtgo/tg"
)

// Peers returns the peer resolution engine backing this client's Resolve*,
// CachePeer, and storage-facade methods. Use it for the richer surface
// (Ingest, Invalidate, EnsureUsable, typed kind errors) that the frozen
// Client API does not expose.
func (c *Client) Peers() *peers.Manager {
	return c.peersManager()
}

// peersManager lazily constructs the peer resolution engine. All peer
// resolution, caching, and persistence flows through it; the exported
// methods on Client are thin compatibility delegates.
func (c *Client) peersManager() *peers.Manager {
	c.peerMgrOnce.Do(func() {
		c.peerMgr = peers.NewManager(peers.Deps{
			Invoker:   c.Raw,
			Store:     func() peers.Store { return peers.StoreFrom(c.storage) },
			CacheSize: func() int { return c.config().PeerCacheSize },
			SavePeers: func() bool { return c.config().SavePeers },
			IsBot:     c.IsBot,
			Debugf:    c.Log.Debugf,
		})
	})
	return c.peerMgr
}

// peerStore adapts the configured storage backend to the peers.Store
// contract, or returns nil when persistence is unavailable.
func (c *Client) peerStore() peers.Store {
	return peers.StoreFrom(c.storage)
}

// cacheUsername records a username→ID mapping in the peer cache.
func (c *Client) cacheUsername(username string, id int64) {
	c.peersManager().CacheUsername(username, id)
}

// ResolvePeer resolves a ChatRef (ID, username, or InputPeer) into an InputPeerClass
// suitable for use in API calls.
//
// Returns ErrNotConnected if the client is not connected, or ErrPeerNotFound if the
// peer cannot be resolved.
//
// Example:
//
//		ctx := context.Background()
//		peer, err := client.ResolvePeer(ctx, "@durov")
//		if err != nil {
//		    log.Fatal(err)
//	}
//
//	fmt.Println(peer)
func (c *Client) ResolvePeer(ctx context.Context, peerID any) (tg.InputPeerClass, error) {
	if err := c.ensureConnectedContext(ctx); err != nil {
		return nil, err
	}
	switch p := peerID.(type) {
	case tg.InputPeerClass:
		return c.peersManager().InputPeer(ctx, peers.Ref{Peer: p})
	case int64:
		return c.peersManager().InputPeer(ctx, peers.Ref{ID: p})
	case int:
		return c.peersManager().InputPeer(ctx, peers.Ref{ID: int64(p)})
	case string:
		return c.resolveChatRefFull(ctx, ChatRefFrom(p))
	case ChatRef:
		return c.resolveChatRefFull(ctx, p)
	case UserRef:
		user, err := c.peersManager().InputUser(ctx, peers.Ref{
			ID:       p.id,
			Username: p.username,
			Phone:    p.phone,
		})
		if err != nil {
			return nil, err
		}
		return inputUserToPeer(user)
	default:
		return nil, fmt.Errorf("%w: unsupported peer type %T", ErrPeerNotFound, peerID)
	}
}

// resolveChatRefFull routes a ChatRef through the manager cascade so that
// numeric ChatRefs enjoy the same RPC fallbacks as bare numeric IDs.
func (c *Client) resolveChatRefFull(ctx context.Context, r ChatRef) (tg.InputPeerClass, error) {
	if r.inviteHash != "" {
		return nil, fmt.Errorf("could not resolve chat: invite link hash cannot be resolved directly, use JoinChat: %w", ErrPeerNotFound)
	}
	return c.peersManager().InputPeer(ctx, peers.Ref{
		ID:       r.id,
		Username: r.username,
		Phone:    r.phone,
		Peer:     r.peer,
	})
}

// ResolveUsername resolves a Telegram username (with or without the leading "@")
// to an [tg.InputPeerClass]. It first checks the local username-to-ID cache; on a
// miss it queries the Telegram server via contacts.resolveUsername.
//
// Successful lookups are cached both by numeric peer ID and by username, so
// repeated resolutions of the same username are served from memory.
//
// Returns the resolved InputPeer on success, or an error wrapping
// [ErrPeerNotFound] if the username does not exist or is inaccessible.
//
// Example:
//
//	ctx := context.Background()
//	peer, err := client.ResolveUsername(ctx, "durov")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Resolved @durov to %T\n", peer)
func (c *Client) ResolveUsername(ctx context.Context, username string) (tg.InputPeerClass, error) {
	return c.peersManager().InputPeerByUsername(ctx, username)
}

// ResolvePhone resolves a phone number to an [tg.InputPeerClass]. The phone
// string is normalised automatically (leading "+" and "00" prefixes are stripped).
//
// The result is cached internally so subsequent calls for the same peer do not
// hit the server again.
//
// Returns the resolved InputPeer on success, or an error wrapping
// [ErrPeerNotFound] if the phone number does not correspond to a known Telegram
// user.
//
// Example:
//
//	ctx := context.Background()
//	peer, err := client.ResolvePhone(ctx, "+1234567890")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Resolved phone to %T\n", peer)
func (c *Client) ResolvePhone(ctx context.Context, phone string) (tg.InputPeerClass, error) {
	return c.peersManager().InputPeerByPhone(ctx, phone)
}

// ResolvePeerCache looks up a previously cached InputPeer by its numeric ID.
// Returns the cached peer or ErrPeerNotFound if not present.
func (c *Client) ResolvePeerCache(id int64) (tg.InputPeerClass, error) {
	return c.peersManager().Cached(id)
}

// CachePeer stores an InputPeer in the peer cache under the given numeric ID,
// canonicalizing the key and preserving known access hashes.
func (c *Client) CachePeer(id int64, peer tg.InputPeerClass) {
	c.peersManager().Cache(id, peer)
}

// PeerToInputPeer converts a high-level [tg.PeerClass] (as returned by Telegram
// updates or API responses) into an [tg.InputPeerClass] suitable for use as an
// input parameter in subsequent API calls.
//
// The users and chats slices provide the access-hash and metadata needed to build
// the correct InputPeer variant. Without the matching entry the function cannot
// produce a valid InputPeer and returns an error.
//
// Returns:
//   - *[tg.InputPeerSelf]   when the peer references the current user (user ID 0 or self).
//   - *[tg.InputPeerUser]   for a known user with an access hash.
//   - *[tg.InputPeerChat]   for a basic group chat.
//   - *[tg.InputPeerChannel] for a channel or supergroup.
//
// Returns an error if the peer is not found in the provided user/chat slices or if
// the peer type is unsupported.
//
// Example:
//
//	inputPeer, err := telegram.PeerToInputPeer(peer, users, chats)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Resolved to %T\n", inputPeer)
func PeerToInputPeer(peer tg.PeerClass, users []tg.UserClass, chats []tg.ChatClass) (tg.InputPeerClass, error) {
	return peers.PeerToInputPeer(peer, users, chats)
}
