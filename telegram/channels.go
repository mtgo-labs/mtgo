package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) GetFullChannel(ctx context.Context, chatID int64) (*types.Chat, error) {
	c.Log.Debugf("GetFullChannel chat_id=%d", chatID)
	channel, err := resolveChannelID(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve channel: %w", err)
	}

	rpc := c.Raw()
	full, err := rpc.ChannelsGetFullChannel(ctx, &tg.ChannelsGetFullChannelRequest{
		Channel: channel,
	})
	if err != nil {
		return nil, err
	}

	chat := &types.Chat{
		ID:   chatID,
		Type: types.ChatTypeChannel,
	}
	types.EnrichChatFull(chat, full)
	return chat, nil
}
