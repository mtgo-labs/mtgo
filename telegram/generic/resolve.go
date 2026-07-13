package generic

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
)

// PeerInput constrains the accepted identifier types for generic wrappers.
// int, int64 (numeric IDs) and string (username, phone, t.me URL, or numeric string).
type PeerInput interface {
	~int | ~int64 | ~string
}

// Caller is the type constraint for the generic wrappers' first argument.
// Both *telegram.Client and *telegram.Context satisfy this constraint.
// *telegram.Client uses context.Background() internally.
// *telegram.Context propagates its Ctx field and Client field.
type Caller interface {
	*telegram.Client | *telegram.Context
}

// extractClient resolves a Caller to its underlying *telegram.Client and
// context.Context.
func extractClient[C Caller](c C) (*telegram.Client, context.Context) {
	switch v := any(c).(type) {
	case *telegram.Client:
		return v, context.Background()
	case *telegram.Context:
		return v.Client, v.Ctx
	}
	panic("unreachable")
}

// resolveID converts a [PeerInput] value to a raw int64 chat/user identifier.
//
// For numeric types the conversion is direct.
// For strings the function first attempts a numeric parse; if that fails it
// delegates to [telegram.Client.ResolvePeer] (which handles usernames, phone
// numbers and t.me URLs) and extracts the underlying ID from the resolved
// [tg.InputPeerClass].
func resolveID[T PeerInput](ctx context.Context, c *telegram.Client, v T) (int64, error) {
	switch val := any(v).(type) {
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	case string:
		if id, err := strconv.ParseInt(val, 10, 64); err == nil {
			return id, nil
		}
		peer, err := c.ResolvePeer(ctx, val)
		if err != nil {
			return 0, fmt.Errorf("resolve peer %q: %w", val, err)
		}
		return inputPeerToID(peer), nil
	default:
		return 0, fmt.Errorf("unsupported peer type %T", v)
	}
}

// inputPeerToID extracts the canonical int64 identifier from an
// [tg.InputPeerClass], using the same sign convention as the rest of the
// library: positive for users, negative for basic groups, -100-prefixed for
// channels and supergroups.
func inputPeerToID(peer tg.InputPeerClass) int64 {
	const channelPrefix int64 = -1000000000000
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		return p.UserID
	case *tg.InputPeerSelf:
		return 0
	case *tg.InputPeerChat:
		return -p.ChatID
	case *tg.InputPeerChannel:
		return channelPrefix - p.ChannelID
	default:
		return 0
	}
}
