package generic

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// AsMessage extracts the first [types.Message] from a [tg.UpdatesClass]
// response. This mirrors [telegram]'s internal extractSingleMessage but
// produces messages with a nil Binder, which is acceptable for standalone
// use outside the bound-method chain.
//
// Returns [ErrNoMessageUpdates] if the response contains no message update.
func AsMessage(result tg.UpdatesClass) (*types.Message, error) {
	switch v := result.(type) {
	case *tg.Updates:
		pm := types.NewPeerMapFromClasses(v.Users, v.Chats)
		for _, u := range v.Updates {
			switch upd := u.(type) {
			case *tg.UpdateNewMessage:
				return types.ParseMessage(upd.Message, pm), nil
			case *tg.UpdateNewChannelMessage:
				return types.ParseMessage(upd.Message, pm), nil
			case *tg.UpdateEditMessage:
				return types.ParseMessage(upd.Message, pm), nil
			case *tg.UpdateEditChannelMessage:
				return types.ParseMessage(upd.Message, pm), nil
			}
		}
		return nil, ErrNoMessageUpdates
	case *tg.UpdateShort:
		pm := &types.PeerMap{
			Users:    make(map[int64]*tg.User),
			Chats:    make(map[int64]*tg.Chat),
			Channels: make(map[int64]*tg.Channel),
		}
		if upd, ok := v.Update.(*tg.UpdateNewMessage); ok {
			return types.ParseMessage(upd.Message, pm), nil
		}
		if upd, ok := v.Update.(*tg.UpdateEditMessage); ok {
			return types.ParseMessage(upd.Message, pm), nil
		}
		return nil, ErrNoMessageUpdates
	case *tg.UpdateShortSentMessage:
		return &types.Message{ID: v.ID}, nil
	default:
		return nil, fmt.Errorf("unexpected updates type %T", result)
	}
}

// AsMessages extracts all [types.Message] values from a [tg.UpdatesClass]
// response. Messages have a nil Binder.
func AsMessages(result tg.UpdatesClass) ([]*types.Message, error) {
	switch v := result.(type) {
	case *tg.Updates:
		pm := types.NewPeerMapFromClasses(v.Users, v.Chats)
		msgs := make([]*types.Message, 0, len(v.Updates))
		for _, u := range v.Updates {
			switch upd := u.(type) {
			case *tg.UpdateNewMessage:
				if m := types.ParseMessage(upd.Message, pm); m != nil {
					msgs = append(msgs, m)
				}
			case *tg.UpdateNewChannelMessage:
				if m := types.ParseMessage(upd.Message, pm); m != nil {
					msgs = append(msgs, m)
				}
			}
		}
		return msgs, nil
	default:
		return nil, fmt.Errorf("unexpected updates type %T", result)
	}
}

// ExtractUsers collects all [types.User] entities from a [tg.UpdatesClass]
// response, deduplicating by user ID.
func ExtractUsers(result tg.UpdatesClass) ([]*types.User, error) {
	users, _, _ := extractEntities(result)
	return users, nil
}

// ExtractChats collects all [types.Chat] entities (basic groups, channels, and
// supergroups) from a [tg.UpdatesClass] response, deduplicating by chat ID.
func ExtractChats(result tg.UpdatesClass) ([]*types.Chat, error) {
	_, chats, _ := extractEntities(result)
	return chats, nil
}

// extractEntities collects users and chats from the carrier objects embedded
// in an [tg.UpdatesClass]. Returns nil slices when the update type has no
// carrier.
func extractEntities(result tg.UpdatesClass) (users []*types.User, chats []*types.Chat, err error) {
	var (
		rawUsers []tg.UserClass
		rawChats []tg.ChatClass
	)
	switch v := result.(type) {
	case *tg.Updates:
		rawUsers = v.Users
		rawChats = v.Chats
	default:
		return nil, nil, fmt.Errorf("unexpected updates type %T", result)
	}

	userSeen := make(map[int64]struct{})
	for _, u := range rawUsers {
		parsed := types.ParseUser(u)
		if parsed == nil {
			continue
		}
		if _, dup := userSeen[parsed.ID]; dup {
			continue
		}
		userSeen[parsed.ID] = struct{}{}
		users = append(users, parsed)
	}

	chatSeen := make(map[int64]struct{})
	for _, ch := range rawChats {
		parsed := types.ParseChatFromChat(ch)
		if parsed == nil {
			continue
		}
		if _, dup := chatSeen[parsed.ID]; dup {
			continue
		}
		chatSeen[parsed.ID] = struct{}{}
		chats = append(chats, parsed)
	}
	return users, chats, nil
}
