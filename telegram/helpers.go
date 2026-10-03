package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/peers"
	"github.com/mtgo-labs/mtgo/tg"
)

// PeerResolver abstracts peer resolution across multiple strategies: in-memory
// cache lookup, username resolution via the Telegram API, and phone number
// resolution. Client implements this interface so helper functions can resolve
// peers without depending directly on the Client type.
type PeerResolver interface {
	// ResolvePeerCache returns a cached InputPeerClass for the given numeric ID
	// without making network requests. Returns an error when the ID is unknown.
	ResolvePeerCache(id int64) (tg.InputPeerClass, error)
	// ResolveUsername resolves a @username to an InputPeerClass via the Telegram
	// API. Blocks until the RPC completes or the context is cancelled.
	ResolveUsername(ctx context.Context, username string) (tg.InputPeerClass, error)
	// ResolvePhone resolves a phone number to an InputPeerClass via the Telegram
	// API. Blocks until the RPC completes or the context is cancelled.
	ResolvePhone(ctx context.Context, phone string) (tg.InputPeerClass, error)
}

// RemoveFunc is a cancellation callback returned by event handler registration
// methods. Calling it unregisters the previously added handler so it no longer
// receives updates.
type RemoveFunc func()

// resolvePeer resolves a numeric chat ID through the full cascade: memory
// cache, persistent store, then the bot/account RPC fallbacks (hash
// completion, dialog preload). An injected test resolver (c.testResolver) is
// honored with cache-only semantics instead.
func resolvePeer(ctx context.Context, c *Client, chatID int64) (tg.InputPeerClass, error) {
	if chatID == 0 {
		return &tg.InputPeerSelf{}, nil
	}
	if r := c.testResolver; r != nil {
		if p, err := r.ResolvePeerCache(chatID); err == nil {
			return p, nil
		}
		return nil, fmt.Errorf("could not resolve peer %d: %w", chatID, ErrPeerNotFound)
	}
	return c.peersManager().InputPeer(ctx, peers.Ref{ID: chatID})
}

// resolveUserID resolves a numeric user ID to an input user through the full
// cascade. See resolvePeer for the lookup order.
func resolveUserID(ctx context.Context, c *Client, userID int64) (tg.InputUserClass, error) {
	if userID == 0 {
		return &tg.InputUserSelf{}, nil
	}
	if r := c.testResolver; r != nil {
		peer, err := r.ResolvePeerCache(userID)
		if err != nil {
			return nil, fmt.Errorf("could not resolve user ID %d: %w", userID, err)
		}
		return inputPeerToUser(peer)
	}
	return c.peersManager().InputUser(ctx, peers.Ref{ID: userID})
}

// resolveChannelID resolves a numeric channel ID to an input channel through
// the full cascade. See resolvePeer for the lookup order.
func resolveChannelID(ctx context.Context, c *Client, channelID int64) (tg.InputChannelClass, error) {
	peer, err := resolvePeer(ctx, c, channelID)
	if err != nil {
		return nil, err
	}
	return peers.InputPeerToChannel(peer)
}
