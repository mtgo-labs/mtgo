package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// PinMessage pins a message to the top of the specified chat. Pinned messages are
// displayed prominently at the top of the chat view for all participants (or just
// the current user in private chats).
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat where the message should be pinned
//   - messageID: the ID of the message to pin
//   - opts: optional PinMessage parameters (silent pinning)
//
// Returns the pinned message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the message does not exist
//   - the user lacks pin permission in the chat
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	pinned, err := client.PinMessage(ctx, chatID, 42,
//	    &params.PinMessage{Silent: true},
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("pinned:", pinned.ID)
func (c *Client) PinMessage(ctx context.Context, chatID int64, messageID int32, opts ...*params.PinMessage) (*types.Message, error) {
	c.Log.Debugf("PinMessage chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	opt := params.GetOptDef(&params.PinMessage{}, opts...)

	var flags tg.Fields
	flags.Set(0)
	if opt.Silent {
		flags.Set(5)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesUpdatePinnedMessage(ctx, &tg.MessagesUpdatePinnedMessageRequest{
		Flags:  flags,
		Silent: opt.Silent,
		Peer:   peer,
		ID:     messageID,
	})
	if err != nil {
		c.Log.Warnf("PinMessage failed err=%v", err)
		return nil, err
	}
	return extractSingleMessage(result, c)
}

// UnpinMessage removes a single message from the pinned messages list in the
// specified chat. The message itself is not deleted, only its pinned status is removed.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat where the message is pinned
//   - messageID: the ID of the pinned message to unpin
//
// Returns the updated message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the message is not pinned or does not exist
//   - the user lacks pin permission in the chat
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	msg, err := client.UnpinMessage(ctx, chatID, 42)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("unpinned:", msg.ID)
func (c *Client) UnpinMessage(ctx context.Context, chatID int64, messageID int32) (*types.Message, error) {
	c.Log.Debugf("UnpinMessage chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesUpdatePinnedMessage(ctx, &tg.MessagesUpdatePinnedMessageRequest{
		Flags: (1 << 0) | (1 << 1),
		Unpin: true,
		Peer:  peer,
		ID:    messageID,
	})
	if err != nil {
		c.Log.Warnf("UnpinMessage failed err=%v", err)
		return nil, err
	}
	return extractSingleMessage(result, c)
}

// UnpinAllMessages removes all pinned messages from the specified chat in a single
// operation. This is more efficient than calling UnpinMessage for each pinned message.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat whose pinned messages should all be cleared
//
// Returns the pts count of affected changes on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the user lacks pin permission in the chat
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	pts, err := client.UnpinAllMessages(ctx, chatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("affected pts:", pts)
func (c *Client) UnpinAllMessages(ctx context.Context, chatID int64) (int, error) {
	c.Log.Debugf("UnpinAllMessages chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return 0, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesUnpinAllMessages(ctx, &tg.MessagesUnpinAllMessagesRequest{
		Peer: peer,
	})
	if err != nil {
		c.Log.Warnf("UnpinAllMessages failed err=%v", err)
		return 0, err
	}
	return int(result.PTSCount), nil
}
