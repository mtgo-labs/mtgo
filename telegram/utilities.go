package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// GetDialogs retrieves a slice of the user's dialogs (conversations). A limit of 0 or
// less defaults to 100. offsetDate can be used for pagination by passing the date of
// the last received dialog. Returns nil (no error) when the dialog list has not been
// modified since the last call.
func (c *Client) GetDialogs(ctx context.Context, limit int, offsetDate int32) ([]*types.Chat, error) {
	if limit <= 0 {
		limit = 100
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetDate: offsetDate,
		Limit:      int32(limit),
		OffsetPeer: &tg.InputPeerEmpty{},
	})
	if err != nil {
		return nil, err
	}

	var chats []tg.ChatClass
	var users []tg.UserClass
	switch v := result.(type) {
	case *tg.MessagesDialogs:
		chats = v.Chats
		users = v.Users
	case *tg.MessagesDialogsSlice:
		chats = v.Chats
		users = v.Users
	case *tg.MessagesDialogsNotModified:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected dialogs type %T", result)
	}

	c.cachePeersFromUpdates(users, chats)

	dialogs := make([]*types.Chat, 0, len(chats))
	for _, chat := range chats {
		switch ch := chat.(type) {
		case *tg.Chat:
			dialogs = append(dialogs, types.ParseChatFromChat(ch))
		case *tg.Channel:
			pm := &types.PeerMap{
				Channels: make(map[int64]*tg.Channel),
			}
			pm.Channels[ch.ID] = ch
			dialogs = append(dialogs, types.ParseChatFromPeer(&tg.PeerChannel{ChannelID: ch.ID}, pm))
		}
	}
	return dialogs, nil
}
