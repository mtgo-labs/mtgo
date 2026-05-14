package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/mtgo-labs/mtgo/tg"
)

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.TrimPrefix(phone, "00")
	return phone
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
	switch p := peer.(type) {
	case *tg.PeerUser:
		for _, u := range users {
			user, ok := u.(*tg.User)
			if !ok {
				continue
			}
			if user.ID == p.UserID {
				if user.AccessHash != 0 {
					return &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}, nil
				}
				return &tg.InputPeerSelf{}, nil
			}
		}
		if p.UserID == 0 {
			return &tg.InputPeerSelf{}, nil
		}
		return nil, fmt.Errorf("user %d not found in resolved peers", p.UserID)
	case *tg.PeerChat:
		for _, c := range chats {
			chat, ok := c.(*tg.Chat)
			if !ok {
				continue
			}
			if chat.ID == p.ChatID {
				return &tg.InputPeerChat{ChatID: chat.ID}, nil
			}
		}
		return nil, fmt.Errorf("chat %d not found in resolved peers", p.ChatID)
	case *tg.PeerChannel:
		for _, c := range chats {
			ch, ok := c.(*tg.Channel)
			if !ok {
				continue
			}
			if ch.ID == p.ChannelID {
				return &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}, nil
			}
		}
		return nil, fmt.Errorf("channel %d not found in resolved peers", p.ChannelID)
	default:
		return nil, fmt.Errorf("unsupported peer type %T", peer)
	}
}

func (c *Client) resolveAndCache(result *tg.ContactsResolvedPeer) {
	for _, u := range result.Users {
		user, ok := u.(*tg.User)
		if !ok {
			continue
		}
		if user.AccessHash != 0 {
			c.CachePeer(user.ID, &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash})
			if user.Username != "" {
				c.peerCacheMu.Lock()
				c.usernameCache[user.Username] = user.ID
				c.peerCacheMu.Unlock()
			}
		}
	}
	for _, ch := range result.Chats {
		channel, ok := ch.(*tg.Channel)
		if !ok {
			continue
		}
		if channel.AccessHash != 0 {
			c.CachePeer(channel.ID, &tg.InputPeerChannel{ChannelID: channel.ID, AccessHash: channel.AccessHash})
			if channel.Username != "" {
				c.peerCacheMu.Lock()
				c.usernameCache[channel.Username] = channel.ID
				c.peerCacheMu.Unlock()
			}
		}
	}
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
	phone = normalizePhone(phone)
	c.Log.Debug("ResolvePhone")
	rpc := c.Raw()
	result, err := rpc.ContactsResolvePhone(ctx, &tg.ContactsResolvePhoneRequest{
		Phone: phone,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: resolve phone %s: %v", ErrPeerNotFound, phone, err)
	}
	c.resolveAndCache(result)
	inputPeer, err := PeerToInputPeer(result.Peer, result.Users, result.Chats)
	if err != nil {
		return nil, fmt.Errorf("%w: phone %s: %v", ErrPeerNotFound, phone, err)
	}
	return inputPeer, nil
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
	username = strings.TrimPrefix(username, "@")
	c.Log.Debugf("ResolveUsername @%s", username)
	c.peerCacheMu.RLock()
	if cachedID, ok := c.usernameCache[username]; ok {
		if p, ok2 := c.peerCache[cachedID]; ok2 {
			c.peerCacheMu.RUnlock()
			c.Log.Tracef("ResolveUsername cache hit @%s", username)
			return p, nil
		}
	}
	c.peerCacheMu.RUnlock()

	rpc := c.Raw()
	result, err := rpc.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: resolve @%s: %v", ErrPeerNotFound, username, err)
	}
	c.resolveAndCache(result)
	inputPeer, err := PeerToInputPeer(result.Peer, result.Users, result.Chats)
	if err != nil {
		return nil, fmt.Errorf("%w: @%s: %v", ErrPeerNotFound, username, err)
	}
	return inputPeer, nil
}
