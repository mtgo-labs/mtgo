package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"

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
	userMap := makeUserMap(users)
	chatMap := makeChatMap(chats)
	switch p := peer.(type) {
	case *tg.PeerUser:
		user, ok := userMap[p.UserID]
		if !ok {
			if p.UserID == 0 {
				return &tg.InputPeerSelf{}, nil
			}
			return nil, fmt.Errorf("user %d not found in resolved peers", p.UserID)
		}
		if user.AccessHash != 0 {
			return &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}, nil
		}
		return &tg.InputPeerSelf{}, nil
	case *tg.PeerChat:
		if chat, ok := chatMap[p.ChatID]; ok {
			return &tg.InputPeerChat{ChatID: chat.id}, nil
		}
		return nil, fmt.Errorf("chat %d not found in resolved peers", p.ChatID)
	case *tg.PeerChannel:
		ch, ok := chatMap[p.ChannelID]
		if !ok {
			return nil, fmt.Errorf("channel %d not found in resolved peers", p.ChannelID)
		}
		return &tg.InputPeerChannel{ChannelID: ch.id, AccessHash: ch.accessHash}, nil
	default:
		return nil, fmt.Errorf("unsupported peer type %T", peer)
	}
}

func makeUserMap(users []tg.UserClass) map[int64]*tg.User {
	m := make(map[int64]*tg.User, len(users))
	for _, u := range users {
		user, ok := u.(*tg.User)
		if ok && user.ID != 0 {
			m[user.ID] = user
		}
	}
	return m
}

func makeChatMap(chats []tg.ChatClass) map[int64]chatInfo {
	m := make(map[int64]chatInfo, len(chats))
	for _, c := range chats {
		switch v := c.(type) {
		case *tg.Chat:
			m[v.ID] = chatInfo{id: v.ID}
		case *tg.Channel:
			m[v.ID] = chatInfo{id: v.ID, accessHash: v.AccessHash}
		}
	}
	return m
}

type chatInfo struct {
	id         int64
	accessHash int64
}

// resolveCoalescer prevents duplicate concurrent RPC calls for the same
// username or phone number. Multiple goroutines resolving the same peer
// share a single in-flight RPC call and all receive the same result.
type resolveCoalescer struct {
	mu       sync.Mutex
	inFlight map[string][]chan resolveResult
}

type resolveResult struct {
	peer tg.InputPeerClass
	err  error
}

func (r *resolveCoalescer) Do(key string, fn func() (tg.InputPeerClass, error)) (tg.InputPeerClass, error) {
	r.mu.Lock()
	if waiters, ok := r.inFlight[key]; ok {
		ch := make(chan resolveResult, 1)
		r.inFlight[key] = append(waiters, ch)
		r.mu.Unlock()
		res := <-ch
		return res.peer, res.err
	}
	r.inFlight[key] = nil
	r.mu.Unlock()

	peer, err := fn()

	r.mu.Lock()
	waiters := r.inFlight[key]
	delete(r.inFlight, key)
	r.mu.Unlock()

	res := resolveResult{peer: peer, err: err}
	for _, ch := range waiters {
		ch <- res
	}
	return peer, err
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
				c.cacheUsernameLocked(user.Username, user.ID)
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
				c.cacheUsernameLocked(channel.Username, channel.ID)
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
	return c.resolveCoalescer.Do("phone:"+phone, func() (tg.InputPeerClass, error) {
		rpc := c.Raw()
		result, err := rpc.ContactsResolvePhone(context.Background(), &tg.ContactsResolvePhoneRequest{
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

	return c.resolveCoalescer.Do("username:"+username, func() (tg.InputPeerClass, error) {
		// Double-check cache inside the coalescer — another goroutine
		// may have resolved it while we were waiting for the lock.
		c.peerCacheMu.RLock()
		if cachedID, ok := c.usernameCache[username]; ok {
			if p, ok2 := c.peerCache[cachedID]; ok2 {
				c.peerCacheMu.RUnlock()
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
	})
}
