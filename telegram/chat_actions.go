package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// SendChatAction sends a chat action indicator to the specified chat, such as
// "typing" or "uploading photo". These indicators are displayed to the other
// party as a status hint. Use this before long-running operations to give the
// user visual feedback.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: target chat identifier
//   - action: the action to display (e.g. [*tg.SendMessageTypingAction],
//     [*tg.SendMessageUploadPhotoAction])
//
// Returns an error if the peer cannot be resolved or the RPC call fails.
func (c *Client) SendChatAction(ctx context.Context, chatID int64, action tg.SendMessageActionClass) error {
	c.Log.Debugf("SendChatAction chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.MessagesSetTyping(ctx, &tg.MessagesSetTypingRequest{
		Peer:   peer,
		Action: action,
	})
	return err
}
