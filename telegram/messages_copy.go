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
// When a Caption (or CaptionEntities) is provided the copy is sent with the
// replacement caption: media is re-sent by file reference and text-only
// messages use the caption as the new body. messages.forwardMessages cannot
// carry a replacement caption, so this path fetches the source message first.
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
	peer, err := resolvePeer(ctx, c, chatID)
	if err != nil {
		return 0, fmt.Errorf("resolve peer: %w", err)
	}

	fromPeer, err := resolvePeer(ctx, c, fromChatID)
	if err != nil {
		return 0, fmt.Errorf("resolve from peer: %w", err)
	}

	opt := params.GetOptDef(&params.CopyMessage{}, opts...)

	// messages.forwardMessages cannot carry a replacement caption. When one is
	// requested, fetch the source message and re-send it with the new caption
	// instead (see copyMessageWithCaption).
	if opt.Caption != "" || len(opt.CaptionEntities) > 0 {
		return c.copyMessageWithCaption(ctx, chatID, peer, messageID, opt)
	}

	var replyTo tg.InputReplyToClass
	if opt.ReplyToMessageID != 0 {
		replyTo = &tg.InputReplyToMessage{ReplyToMsgID: opt.ReplyToMessageID}
	}

	req := &tg.MessagesForwardMessagesRequest{
		Silent:     opt.DisableNotification,
		DropAuthor: opt.DropAuthor,
		FromPeer:   fromPeer,
		ID:         []int32{messageID},
		RandomID:   []int64{c.RandomID()},
		ToPeer:     peer,
		ReplyTo:    replyTo,
	}
	if opt.ScheduleDate != nil {
		req.ScheduleDate = *opt.ScheduleDate
	}
	req.SetFlags()

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

// copyMessageWithCaption implements CopyMessage when a replacement caption (or
// caption entities) is requested. messages.forwardMessages has no caption
// field, so the source message is fetched via messages.getMessages and re-sent:
// photos and documents are re-sent by raw file reference with the new caption,
// and text-only messages are sent as plain text using the caption as the body.
func (c *Client) copyMessageWithCaption(ctx context.Context, chatID int64, peer tg.InputPeerClass, messageID int32, opt *params.CopyMessage) (int64, error) {
	res, err := c.Raw().MessagesGetMessages(ctx, &tg.MessagesGetMessagesRequest{
		ID: []tg.InputMessageClass{&tg.InputMessageID{ID: messageID}},
	})
	if err != nil {
		return 0, fmt.Errorf("get source message: %w", err)
	}
	msgs, err := extractMessagesFromMessagesClass(res, c)
	if err != nil {
		return 0, err
	}
	if len(msgs) == 0 {
		return 0, fmt.Errorf("copy message: source message %d not found", messageID)
	}
	src := msgs[0]

	sendOpts := &params.SendMessage{
		DisableNotification:   opt.DisableNotification,
		ReplyToMessageID:      opt.ReplyToMessageID,
		ReplyMarkup:           opt.ReplyMarkup,
		Entities:              opt.CaptionEntities,
		ParseMode:             opt.ParseMode,
		ScheduleDate:          opt.ScheduleDate,
		NoForwards:            opt.NoForwards,
		ShowCaptionAboveMedia: opt.ShowCaptionAboveMedia,
		ProtectContent:        opt.ProtectContent,
		MessageThreadID:       opt.MessageThreadID,
		AllowPaidBroadcast:    opt.AllowPaidBroadcast,
		PaidMessageStarCount:  opt.PaidMessageStarCount,
	}

	var sent *types.Message
	switch {
	case src.Photo != nil:
		media := &tg.InputMediaPhoto{
			ID: &tg.InputPhoto{
				ID:            src.Photo.ID,
				AccessHash:    src.Photo.AccessHash,
				FileReference: src.Photo.FileReference,
			},
			TTLSeconds: int32(src.Photo.TTLSeconds),
		}
		if pm, ok := src.Media.(*types.PhotoMedia); ok {
			media.Spoiler = pm.IsSpoiler
		}
		sent, err = c.sendMediaInternal(ctx, peer, media, opt.Caption, sendOpts)
	case src.Document != nil && src.Document.RawDocument != nil:
		raw := src.Document.RawDocument
		media := &tg.InputMediaDocument{
			ID: &tg.InputDocument{
				ID:            raw.ID,
				AccessHash:    raw.AccessHash,
				FileReference: raw.FileReference,
			},
			TTLSeconds: int32(src.Document.TTLSeconds),
			Spoiler:    src.Document.IsSpoiler,
		}
		sent, err = c.sendMediaInternal(ctx, peer, media, opt.Caption, sendOpts)
	default:
		text := opt.Caption
		if text == "" {
			text = src.Text
		}
		sent, err = c.SendMessage(ctx, chatID, text, sendOpts)
	}
	if err != nil {
		return 0, err
	}
	return int64(sent.ID), nil
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
	peer, err := resolvePeer(ctx, c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	fromPeer, err := resolvePeer(ctx, c, fromChatID)
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
		DropAuthor: true,
		FromPeer:   fromPeer,
		ID:         groupedMsgIDs,
		RandomID:   randomIDs,
		ToPeer:     peer,
	}
	req.SetFlags()

	rpc := c.Raw()
	result, err := rpc.MessagesForwardMessages(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("forward messages: %w", err)
	}
	return extractMessages(result, c)
}
