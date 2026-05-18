package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// SendReaction sends one or more emoji reactions to a message in the specified chat.
// Reactions appear below the message and are visible to all chat participants.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat containing the message
//   - messageID: the ID of the message to react to
//   - reaction: one or more reaction types (emoji or custom emoji)
//
// Returns an error if:
//   - the peer cannot be resolved
//   - reactions are disabled in the chat
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	err := client.SendReaction(ctx, chatID, 42, &tg.ReactionEmoji{Emoticon: "👍"})
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) SendReaction(ctx context.Context, chatID int64, messageID int32, reaction ...tg.ReactionClass) error {
	c.Log.Debugf("SendReaction chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	var flags tg.Fields
	flags.Set(0)

	rpc := c.Raw()
	_, err = rpc.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
		Flags:    flags,
		Peer:     peer,
		MsgID:    messageID,
		Reaction: reaction,
	})
	if err != nil {
	}
	return err
}

// SendPaidReaction sends a paid reaction (using Telegram Stars) to a message in the
// specified chat. Paid reactions are typically used for exclusive content or creator
// support. The Stars are deducted from the user's balance immediately.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat containing the message
//   - messageID: the ID of the message to react to
//   - amount: the number of Telegram Stars to spend on this reaction
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the user has insufficient Stars balance
//   - paid reactions are not available for this message
//   - the RPC call fails
func (c *Client) SendPaidReaction(ctx context.Context, chatID int64, messageID int32, amount int64) error {
	c.Log.Debugf("SendPaidReaction chat_id=%d msg_id=%d amount=%d", chatID, messageID, amount)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.MessagesSendPaidReaction(ctx, &tg.MessagesSendPaidReactionRequest{
		Peer:     peer,
		MsgID:    messageID,
		Count:    int32(amount),
		RandomID: c.RandomID(),
	})
	if err != nil {
	}
	return err
}

// VotePoll casts a vote in a poll by selecting one or more answer options. The
// option bytes correspond to the PollAnswer.Option fields returned when the poll
// was originally sent.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat containing the poll message
//   - messageID: the ID of the message containing the poll
//   - options: the raw bytes of the selected answer options (from PollAnswer.Option)
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the poll is already closed
//   - the selected options are invalid
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	err := client.VotePoll(ctx, chatID, 42, [][]byte{{0}, {1}})
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) VotePoll(ctx context.Context, chatID int64, messageID int32, options [][]byte) error {
	c.Log.Debugf("VotePoll chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.MessagesSendVote(ctx, &tg.MessagesSendVoteRequest{
		Peer:    peer,
		MsgID:   messageID,
		Options: options,
	})
	if err != nil {
	}
	return err
}

// StopPoll closes an active poll, preventing further votes. The poll results remain
// visible but users can no longer change or submit their votes. This is implemented
// by editing the poll message media with the Closed flag set.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat containing the poll message
//   - messageID: the ID of the message containing the poll
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the poll is already closed
//   - the user is not the poll creator
//   - the RPC call fails
func (c *Client) StopPoll(ctx context.Context, chatID int64, messageID int32) error {
	c.Log.Debugf("StopPoll chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
		Flags: (1 << 13),
		Peer:  peer,
		ID:    messageID,
		Media: &tg.InputMediaPoll{
			Poll: &tg.Poll{
				ID:       0,
				Closed:   true,
				Question: &tg.TextWithEntities{Text: ""},
				Answers:  []tg.PollAnswerClass{},
			},
		},
	})
	if err != nil {
	}
	return err
}

// RetractVote withdraws the user's previous vote in a poll. Sends an empty vote
// to the server, which resets the user's selections.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat containing the poll message
//   - messageID: the ID of the message containing the poll
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the poll is closed or the user has not voted
//   - the RPC call fails
func (c *Client) RetractVote(ctx context.Context, chatID int64, messageID int32) error {
	c.Log.Debugf("RetractVote chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	rpc := c.Raw()
	_, err = rpc.MessagesSendVote(ctx, &tg.MessagesSendVoteRequest{
		Peer:    peer,
		MsgID:   messageID,
		Options: nil,
	})
	if err != nil {
	}
	return err
}
