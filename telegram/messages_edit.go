package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// EditMessageCaption changes the caption of a media message (photo, video, document, etc.).
// Use this to update the text displayed below a media attachment.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat containing the message to edit
//   - messageID: the ID of the media message whose caption should be changed
//   - caption: the new caption text
//   - opts: optional EditMessage parameters (reply markup, schedule, invert media)
//
// Returns the edited message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the message does not exist or is not a media message
//   - the edit window has expired
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	edited, err := client.EditMessageCaption(ctx, chatID, 42, "new caption")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(edited.ID)
func (c *Client) EditMessageCaption(ctx context.Context, chatID int64, messageID int32, caption string, opts ...*params.EditMessage) (*types.Message, error) {
	c.Log.Debugf("EditMessageCaption chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	opt := params.GetOptDef(&params.EditMessage{}, opts...)

	var flags tg.Fields
	flags.Set(11)
	if opt.InvertMedia {
		flags.Set(26)
	}

	req := &tg.MessagesEditMessageRequest{
		Flags:       flags,
		InvertMedia: opt.InvertMedia,
		Peer:        peer,
		ID:          messageID,
		Message:     caption,
		ReplyMarkup: opt.ReplyMarkup,
	}
	if opt.ScheduleDate != nil {
		req.ScheduleDate = *opt.ScheduleDate
	}

	rpc := c.Raw()
	result, err := rpc.MessagesEditMessage(ctx, req)
	if err != nil {
		return nil, err
	}
	return extractSingleMessage(result, c)
}

// EditMessageMedia replaces the media content of an existing message (photo, video,
// document, etc.) with new media. The message text/caption remains unchanged unless
// also updated through the media object.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat containing the message to edit
//   - messageID: the ID of the message whose media should be replaced
//   - media: the new media to attach (InputMediaPhoto, InputMediaDocument, etc.)
//   - opts: optional EditMessage parameters (reply markup, schedule, invert media)
//
// Returns the edited message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the message does not exist or cannot have its media replaced
//   - the new media upload fails
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	newPhoto := &tg.InputMediaPhoto{ID: &tg.InputPhotoTL{ID: updatedPhotoID}}
//	edited, err := client.EditMessageMedia(ctx, chatID, 42, newPhoto)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(edited.ID)
func (c *Client) EditMessageMedia(ctx context.Context, chatID int64, messageID int32, media tg.InputMediaClass, opts ...*params.EditMessage) (*types.Message, error) {
	c.Log.Debugf("EditMessageMedia chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	opt := params.GetOptDef(&params.EditMessage{}, opts...)

	var flags tg.Fields
	flags.Set(13)
	if opt.InvertMedia {
		flags.Set(26)
	}

	req := &tg.MessagesEditMessageRequest{
		Flags:       flags,
		InvertMedia: opt.InvertMedia,
		Peer:        peer,
		ID:          messageID,
		Media:       media,
		ReplyMarkup: opt.ReplyMarkup,
	}
	if opt.ScheduleDate != nil {
		req.ScheduleDate = *opt.ScheduleDate
	}

	rpc := c.Raw()
	result, err := rpc.MessagesEditMessage(ctx, req)
	if err != nil {
		return nil, err
	}
	return extractSingleMessage(result, c)
}

// EditMessageReplyMarkup changes only the inline keyboard (reply markup) of an
// existing message without modifying its text or media. Useful for updating button
// states (e.g., toggling a button after a user presses it).
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat containing the message to edit
//   - messageID: the ID of the message whose reply markup should be updated
//   - replyMarkup: the new inline keyboard markup to display
//
// Returns the edited message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the message does not exist
//   - the reply markup is invalid
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	keyboard := &tg.ReplyInlineMarkup{Rows: rows}
//	edited, err := client.EditMessageReplyMarkup(ctx, chatID, 42, keyboard)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(edited.ID)
func (c *Client) EditMessageReplyMarkup(ctx context.Context, chatID int64, messageID int32, replyMarkup tg.ReplyMarkupClass) (*types.Message, error) {
	c.Log.Debugf("EditMessageReplyMarkup chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	var flags tg.Fields
	flags.Set(2)

	req := &tg.MessagesEditMessageRequest{
		Flags:       flags,
		Peer:        peer,
		ID:          messageID,
		ReplyMarkup: replyMarkup,
	}

	rpc := c.Raw()
	result, err := rpc.MessagesEditMessage(ctx, req)
	if err != nil {
		return nil, err
	}
	return extractSingleMessage(result, c)
}
