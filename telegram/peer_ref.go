package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mtgo-labs/mtgo/tg"
)

// ChatRef is an opaque reference to a chat that can be resolved to an
// [tg.InputPeerClass]. It supports resolution by numeric ID, username, phone
// number, or a pre-built peer object. Use one of the constructor functions
// ([ChatID], [Username], [ChatPhone], [ChatPeer], [ChatRefFrom]) to create
// instances.
//
// Example:
//
//	ref := telegram.ChatID(12345678)
//	peer, err := ref.Resolve(ctx, client)
type ChatRef struct {
	id       int64
	username string
	phone    string
	peer     tg.InputPeerClass
}

// ChatID creates a ChatRef that resolves to the chat with the given numeric
// identifier.
//
// Example:
//
//	ref := telegram.ChatID(12345678)
func ChatID(id int64) ChatRef {
	return ChatRef{id: id}
}

// Username creates a ChatRef that resolves by looking up the given public
// username (with or without the @ prefix).
//
// Example:
//
//	ref := telegram.Username("@durov")
func Username(username string) ChatRef {
	return ChatRef{username: strings.TrimPrefix(username, "@")}
}

// ChatPhone creates a ChatRef that resolves by looking up the given phone
// number in the user's contacts.
func ChatPhone(phone string) ChatRef {
	return ChatRef{phone: phone}
}

// ChatPeer creates a ChatRef that wraps a pre-resolved [tg.InputPeerClass],
// bypassing any resolution lookup.
func ChatPeer(peer tg.InputPeerClass) ChatRef {
	return ChatRef{peer: peer}
}

// ChatRefFrom parses a string and creates the appropriate ChatRef. It
// recognizes phone numbers (starting with "+" or "00"), numeric IDs, t.me URLs,
// and plain usernames.
func ChatRefFrom(peer string) ChatRef {
	if strings.HasPrefix(peer, "+") || strings.HasPrefix(peer, "00") {
		return ChatPhone(peer)
	}
	if id, err := strconv.ParseInt(peer, 10, 64); err == nil {
		return ChatID(id)
	}
	if isPeerURL(peer) {
		return ChatRef{username: extractUsernameFromURL(peer)}
	}
	return Username(peer)
}

func (r ChatRef) resolve(ctx context.Context, res PeerResolver) (tg.InputPeerClass, error) {
	if r.peer != nil {
		return r.peer, nil
	}
	if r.phone != "" {
		return res.ResolvePhone(ctx, r.phone)
	}
	if r.username != "" {
		return res.ResolveUsername(ctx, r.username)
	}
	if r.id == 0 {
		return &tg.InputPeerSelf{}, nil
	}
	if p, err := res.ResolvePeerCache(r.id); err == nil {
		return p, nil
	}
	return nil, fmt.Errorf("could not resolve chat: %w", ErrPeerNotFound)
}

// UserRef is an opaque reference to a user that can be resolved to a
// [tg.InputUserClass]. It supports resolution by numeric ID, username, phone
// number, or a pre-built input user object. Use one of the constructor
// functions ([UserID], [UserUsername], [UserPhone], [UserInput]) to create
// instances.
type UserRef struct {
	id       int64
	username string
	phone    string
	user     tg.InputUserClass
}

// UserID creates a UserRef that resolves to the user with the given numeric
// identifier.
func UserID(id int64) UserRef {
	return UserRef{id: id}
}

// UserUsername creates a UserRef that resolves by looking up the given public
// username (with or without the @ prefix).
func UserUsername(username string) UserRef {
	return UserRef{username: strings.TrimPrefix(username, "@")}
}

// UserPhone creates a UserRef that resolves by looking up the given phone
// number in the user's contacts.
func UserPhone(phone string) UserRef {
	return UserRef{phone: phone}
}

// UserInput creates a UserRef that wraps a pre-resolved [tg.InputUserClass],
// bypassing any resolution lookup.
func UserInput(user tg.InputUserClass) UserRef {
	return UserRef{user: user}
}

func (r UserRef) resolve(ctx context.Context, res PeerResolver) (tg.InputUserClass, error) {
	if r.user != nil {
		return r.user, nil
	}
	if r.phone != "" {
		peer, err := res.ResolvePhone(ctx, r.phone)
		if err != nil {
			return nil, err
		}
		return inputPeerToUser(peer)
	}
	if r.username != "" {
		peer, err := res.ResolveUsername(ctx, r.username)
		if err != nil {
			return nil, err
		}
		return inputPeerToUser(peer)
	}
	if r.id == 0 {
		return &tg.InputUserSelf{}, nil
	}
	peer, err := res.ResolvePeerCache(r.id)
	if err != nil {
		return nil, fmt.Errorf("could not resolve user ID %d: %w", r.id, err)
	}
	return inputPeerToUser(peer)
}

func inputPeerToUser(peer tg.InputPeerClass) (tg.InputUserClass, error) {
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		return &tg.InputUser{UserID: p.UserID, AccessHash: p.AccessHash}, nil
	case *tg.InputPeerSelf:
		return &tg.InputUserSelf{}, nil
	default:
		return nil, fmt.Errorf("peer %T is not a user", peer)
	}
}

func isPeerURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "t.me/") ||
		strings.HasPrefix(s, "telegram.me/") ||
		strings.HasPrefix(s, "telegram.dog/")
}

func extractUsernameFromURL(raw string) string {
	s := raw
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "t.me/")
	s = strings.TrimPrefix(s, "telegram.me/")
	s = strings.TrimPrefix(s, "telegram.dog/")
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}
	if idx := strings.Index(s, "?"); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimPrefix(s, "@")
}
