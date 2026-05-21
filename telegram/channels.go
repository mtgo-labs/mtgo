package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// GetFullChannel retrieves the full information about a channel or supergroup,
// including participant count, admin list, banned list, default permissions,
// and the linked chat/peer. This is useful for admin panels, stats displays,
// or any feature that needs more than the basic Channel object.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the channel or supergroup ID
//
// Returns a ChatFullClass (e.g. *ChannelFull) on success.
//
// Returns an error if:
//   - the peer cannot be resolved or is not a channel
//   - the user is not a member (for private channels)
//   - the RPC call fails
func (c *Client) GetFullChannel(ctx context.Context, chatID int64) (tg.ChatFullClass, error) {
	c.Log.Debugf("GetFullChannel chat_id=%d", chatID)
	channel, err := resolveChannelID(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve channel: %w", err)
	}

	rpc := c.Raw()
	return rpc.ChannelsGetFullChannel(ctx, &tg.ChannelsGetFullChannelRequest{
		Channel: channel,
	})
}
