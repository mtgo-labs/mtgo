package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// CopyMessage copies a single message from one chat to another without indicating
// it was forwarded. The message appears as if it were originally sent by the
// authenticated user in the destination chat.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the destination chat where the copy will appear
//   - fromChatID: the source chat containing the original message
//   - messageID: the ID of the message to copy
//   - opts: optional CopyMessage parameters (silent, drop author, custom caption, reply, schedule)
//
// Returns the ID of the newly created message on success.
//
// Returns an error if:
//   - either peer cannot be resolved
//   - the source message does not exist
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	newID, err := client.CopyMessage(ctx, destChatID, srcChatID, 100)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("copied as:", newID)
func (c *Client) CopyMessage(ctx context.Context, chatID int64, fromChatID int64, messageID int32, opts ...*params.CopyMessage) (int64, error) {
	c.Log.Debugf("CopyMessage to=%d from=%d", chatID, fromChatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return 0, fmt.Errorf("resolve peer: %w", err)
	}

	fromPeer, err := resolvePeer(c, fromChatID)
	if err != nil {
		return 0, fmt.Errorf("resolve from peer: %w", err)
	}

	opt := params.GetOptDef(&params.CopyMessage{}, opts...)

	var flags tg.Fields
	if opt.DisableNotification {
		flags.Set(5)
	}
	if opt.DropAuthor {
		flags.Set(8)
	}
	if opt.Caption != "" {
		flags.Set(11)
	}

	var replyTo tg.InputReplyToClass
	if opt.ReplyToMessageID != 0 {
		flags.Set(0)
		replyTo = &tg.InputReplyToMessage{ReplyToMsgID: opt.ReplyToMessageID}
	}

	req := &tg.MessagesForwardMessagesRequest{
		Flags:             flags,
		Silent:            opt.DisableNotification,
		DropAuthor:        opt.DropAuthor,
		DropMediaCaptions: opt.Caption != "",
		FromPeer:          fromPeer,
		ID:                []int32{messageID},
		RandomID:          []int64{c.RandomID()},
		ToPeer:            peer,
		ReplyTo:           replyTo,
	}
	if opt.ScheduleDate != nil {
		req.ScheduleDate = *opt.ScheduleDate
	}

	rpc := c.Raw()
	result, err := rpc.MessagesForwardMessages(ctx, req)
	if err != nil {
		return 0, err
	}

	msg, err := extractSingleMessage(result, c)
	if err != nil {
		return 0, nil
	}
	return int64(msg.ID), nil
}

// CopyMediaGroup copies an entire album (grouped media messages) from one chat to another
// as a new album without forwarding attribution. It looks up all messages sharing the
// same GroupedID and forwards them with DropAuthor enabled.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the destination chat where the album will appear
//   - fromChatID: the source chat containing the original album
//   - groupedID: the grouped media ID shared by all messages in the album
//
// Returns the copied messages on success.
//
// Returns an error if:
//   - either peer cannot be resolved
//   - no messages with the given groupedID are found in the source chat
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	msgs, err := client.CopyMediaGroup(ctx, destChatID, srcChatID, groupedID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, m := range msgs {
//	    fmt.Println("copied:", m.ID)
//	}
func (c *Client) CopyMediaGroup(ctx context.Context, chatID int64, fromChatID int64, groupedID int64) ([]*types.Message, error) {
	c.Log.Debugf("CopyMediaGroup to=%d from=%d", chatID, fromChatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	fromPeer, err := resolvePeer(c, fromChatID)
	if err != nil {
		return nil, fmt.Errorf("resolve from peer: %w", err)
	}

	history, err := c.GetChatHistory(ctx, fromChatID, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("get chat history: %w", err)
	}

	var groupedMsgIDs []int32
	for _, m := range history {
		if m.GroupedID == groupedID {
			groupedMsgIDs = append(groupedMsgIDs, m.ID)
		}
	}

	if len(groupedMsgIDs) == 0 {
		return nil, fmt.Errorf("no messages found with grouped ID %d", groupedID)
	}

	randomIDs := make([]int64, len(groupedMsgIDs))
	for i := range randomIDs {
		randomIDs[i] = c.RandomID()
	}

	req := &tg.MessagesForwardMessagesRequest{
		Flags:      (1 << 8),
		DropAuthor: true,
		FromPeer:   fromPeer,
		ID:         groupedMsgIDs,
		RandomID:   randomIDs,
		ToPeer:     peer,
	}

	rpc := c.Raw()
	result, err := rpc.MessagesForwardMessages(ctx, req)
	if err != nil {
		return nil, err
	}
	return extractMessages(result, c)
}
