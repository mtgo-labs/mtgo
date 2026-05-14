package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// ReadHistory marks the chat history of the specified chat as read up to and
// including maxID. After this call, the chat will no longer show unread badges
// for messages with IDs at or below maxID.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat whose history to mark as read
//   - maxID: the maximum message ID up to which history is marked read
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
func (c *Client) ReadHistory(ctx context.Context, chatID int64, maxID int32) error {
	c.Log.Debugf("ReadHistory chat_id=%d max_id=%d", chatID, maxID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.MessagesReadHistory(ctx, &tg.MessagesReadHistoryRequest{
		Peer:  peer,
		MaxID: maxID,
	})
	if err != nil {
		c.Log.Warnf("ReadHistory failed err=%v", err)
	}
	return err
}

// ReadMentions marks all unread mentions (@-mentions and quote replies) in the
// specified chat as read. Clears the mention badge for the chat.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat whose mentions to mark as read
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
func (c *Client) ReadMentions(ctx context.Context, chatID int64) error {
	c.Log.Debugf("ReadMentions chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.MessagesReadMentions(ctx, &tg.MessagesReadMentionsRequest{
		Peer: peer,
	})
	if err != nil {
		c.Log.Warnf("ReadMentions failed err=%v", err)
	}
	return err
}

// ReadReactions marks all unread reaction notifications in the specified chat as
// read. Clears the reaction badge counter for the chat.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat whose reactions to mark as read
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
func (c *Client) ReadReactions(ctx context.Context, chatID int64) error {
	c.Log.Debugf("ReadReactions chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.MessagesReadReactions(ctx, &tg.MessagesReadReactionsRequest{
		Peer: peer,
	})
	if err != nil {
		c.Log.Warnf("ReadReactions failed err=%v", err)
	}
	return err
}
