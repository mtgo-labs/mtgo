package telegram

import (
	"context"
	"fmt"

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

func resolvePeer(r PeerResolver, chatID int64) (tg.InputPeerClass, error) {
	if chatID == 0 {
		return &tg.InputPeerSelf{}, nil
	}
	if p, err := r.ResolvePeerCache(chatID); err == nil {
		return p, nil
	}
	return nil, fmt.Errorf("could not resolve peer %d: %w", chatID, ErrPeerNotFound)
}

func resolveUserID(r PeerResolver, userID int64) (tg.InputUserClass, error) {
	if userID == 0 {
		return &tg.InputUserSelf{}, nil
	}
	peer, err := r.ResolvePeerCache(userID)
	if err != nil {
		return nil, fmt.Errorf("could not resolve user ID %d: %w", userID, err)
	}
	return inputPeerToUser(peer)
}

func resolveChannelID(r PeerResolver, chatID int64) (tg.InputChannelClass, error) {
	peer, err := resolvePeer(r, chatID)
	if err != nil {
		return nil, err
	}
	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		return &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash}, nil
	case *tg.InputPeerSelf:
		return &tg.InputChannelEmpty{}, nil
	default:
		return nil, fmt.Errorf("peer %T is not a channel", peer)
	}
}
