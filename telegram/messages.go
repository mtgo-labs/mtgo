package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/parser"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// toParserMode converts a params.ParseMode string to a parser.ParseMode.
// Returns false for default/disabled/empty modes.
func toParserMode(mode params.ParseMode) (parser.ParseMode, bool) {
	switch mode {
	case params.ParseModeHTML:
		return parser.ParseModeHTML, true
	case params.ParseModeMarkdown:
		return parser.ParseModeMarkdown, true
	default:
		return parser.ParseModeDefault, false
	}
}

// parseText resolves the effective parse mode from opts → client default,
// then parses formatted text into plain text + entities.
// If entities are already provided or parse mode is unset, returns text unchanged.
func (c *Client) parseText(text string, optParseMode params.ParseMode) (string, []tg.MessageEntityClass, error) {
	mode := optParseMode
	if mode == "" || mode == params.ParseModeDefault {
		mode = c.cfg.ParseMode
	}
	pm, ok := toParserMode(mode)
	if !ok {
		return text, nil, nil
	}
	return parser.Parse(pm, text)
}

// SendMessage sends a plain text message to the specified chat. It is the primary
// method for delivering text and supports optional parameters for reply targets,
// inline keyboards, formatting entities, scheduling, silent delivery, and more.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat or channel (can be a user ID, group ID, or channel ID)
//   - text: the message body (may contain Markdown/HTML if entities are also provided)
//   - opts: optional SendMessage parameters (reply, markup, entities, schedule, etc.)
//
// Returns the sent message on success.
//
// Returns an error if:
//   - the peer cannot be resolved (invalid or inaccessible chatID)
//   - the RPC call fails (flood wait, chat write forbidden, etc.)
//
// Example:
//
//	ctx := context.Background()
//	msg, err := client.SendMessage(ctx, chatID, "Hello, world!")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(msg.ID)
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, opts ...*params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendMessage chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		// Cache miss — try RPC resolution via contacts/resolveUsername or
		// InputPeerUserFromMessage when we have a reply context.
		peer, err = c.ResolvePeer(ctx, chatID)
		if err != nil && len(opts) > 0 {
			opt := opts[0]
			if opt.ReplyToMessageID != 0 && chatID > 0 {
				peer = &tg.InputPeerUserFromMessage{
					Peer:   &tg.InputPeerUser{UserID: chatID},
					MsgID:  opt.ReplyToMessageID,
					UserID: chatID,
				}
				err = nil
			}
		}
		if err != nil {
			return nil, fmt.Errorf("resolve peer: %w", err)
		}
	}
	opt := params.GetOptDef(&params.SendMessage{}, opts...)

	// Parse formatted text when no pre-built entities are provided.
	sendText := text
	var parsedEntities []tg.MessageEntityClass
	if len(opt.Entities) == 0 {
		parsed, entities, err := c.parseText(text, opt.ParseMode)
		if err != nil {
			return nil, fmt.Errorf("parse text: %w", err)
		}
		sendText = parsed
		parsedEntities = entities
	}

	var flags tg.Fields
	if opt.DisableWebPagePreview {
		flags.Set(1)
	}
	if opt.Silent || opt.DisableNotification {
		flags.Set(5)
	}
	if opt.Background {
		flags.Set(6)
	}
	if opt.ClearDraft {
		flags.Set(7)
	}
	if opt.NoForwards {
		flags.Set(14)
	}
	if opt.InvertMedia {
		flags.Set(27)
	}

	var replyTo tg.InputReplyToClass
	if opt.ReplyTo != nil {
		flags.Set(0)
		replyTo = opt.ReplyTo
	} else if opt.ReplyToMessageID != 0 {
		flags.Set(0)
		replyTo = &tg.InputReplyToMessage{ReplyToMsgID: opt.ReplyToMessageID}
	}
	if opt.ReplyMarkup != nil {
		flags.Set(2)
	}
	entities := opt.Entities
	if len(entities) == 0 {
		entities = parsedEntities
	}
	if len(entities) > 0 {
		flags.Set(3)
	}
	if opt.SendAs != nil {
		flags |= (1 << 13)
	}

	req := &tg.MessagesSendMessageRequest{
		Flags:       flags,
		Silent:      opt.Silent || opt.DisableNotification,
		Background:  opt.Background,
		ClearDraft:  opt.ClearDraft,
		Noforwards:  opt.NoForwards,
		InvertMedia: opt.InvertMedia,
		Peer:        peer,
		ReplyTo:     replyTo,
		Message:     sendText,
		RandomID:    c.RandomID(),
		ReplyMarkup: opt.ReplyMarkup,
		Entities:    entities,
	}
	if opt.ScheduleDate != nil {
		req.ScheduleDate = *opt.ScheduleDate
	}
	if opt.EffectID != nil {
		req.Effect = *opt.EffectID
	}
	if opt.SendAs != nil {
		req.SendAs = opt.SendAs
	}

	rpc := c.Raw()
	result, err := rpc.MessagesSendMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	return extractSingleMessage(result, c)
}

// ForwardMessages forwards one or more messages from a source chat to a destination
// chat. Unlike copy, forwarded messages retain a reference to the original sender
// unless DropAuthor is set.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the destination chat where messages will appear
//   - fromChatID: the source chat containing the original messages
//   - messageIDs: the IDs of the messages to forward
//   - opts: optional ForwardMessages parameters (silent, drop author, drop captions, etc.)
//
// Returns the resulting forwarded messages on success.
//
// Returns an error if:
//   - either peer cannot be resolved
//   - the RPC call fails (no forward permission, message deleted, etc.)
//
// Example:
//
//	ctx := context.Background()
//	msgs, err := client.ForwardMessages(ctx, destChatID, srcChatID, []int32{42, 43})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, m := range msgs {
//	    fmt.Println("forwarded:", m.ID)
//	}
func (c *Client) ForwardMessages(ctx context.Context, chatID int64, fromChatID int64, messageIDs []int32, opts ...*params.ForwardMessages) ([]*types.Message, error) {
	c.Log.Debugf("ForwardMessages to=%d from=%d count=%d", chatID, fromChatID, len(messageIDs))
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	fromPeer, err := resolvePeer(c, fromChatID)
	if err != nil {
		return nil, fmt.Errorf("resolve from peer: %w", err)
	}

	opt := params.GetOptDef(&params.ForwardMessages{}, opts...)

	randomIDs := make([]int64, len(messageIDs))
	for i := range randomIDs {
		randomIDs[i] = c.RandomID()
	}

	var flags tg.Fields
	if opt.DisableNotification {
		flags.Set(5)
	}
	if opt.DropAuthor {
		flags.Set(8)
	}
	if opt.DropMediaCaptions {
		flags.Set(9)
	}
	if opt.NoForwards {
		flags.Set(14)
	}

	req := &tg.MessagesForwardMessagesRequest{
		Flags:             flags,
		Silent:            opt.DisableNotification,
		DropAuthor:        opt.DropAuthor,
		DropMediaCaptions: opt.DropMediaCaptions,
		Noforwards:        opt.NoForwards,
		FromPeer:          fromPeer,
		ID:                messageIDs,
		RandomID:          randomIDs,
		ToPeer:            peer,
	}
	if opt.ScheduleDate != nil {
		req.ScheduleDate = *opt.ScheduleDate
	}

	rpc := c.Raw()
	result, err := rpc.MessagesForwardMessages(ctx, req)
	if err != nil {
		return nil, err
	}
	return extractMessages(result, c)
}

// DeleteMessages removes the specified messages from a chat. When Revoke is true the
// messages are deleted for all participants; otherwise only the local copy is removed.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat from which messages should be deleted
//   - messageIDs: the IDs of the messages to delete
//   - opts: optional DeleteMessages parameters (revoke, etc.)
//
// Returns the number of affected pts (position tracking state) on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails (no delete permission, message already gone, etc.)
//
// Example:
//
//	ctx := context.Background()
//	pts, err := client.DeleteMessages(ctx, chatID, []int32{100, 101},
//	    &params.DeleteMessages{Revoke: true},
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("affected pts:", pts)
func (c *Client) DeleteMessages(ctx context.Context, chatID int64, messageIDs []int32, opts ...*params.DeleteMessages) (int, error) {
	c.Log.Debugf("DeleteMessages chat_id=%d count=%d", chatID, len(messageIDs))
	opt := params.GetOptDef(&params.DeleteMessages{}, opts...)

	_, err := resolvePeer(c, chatID)
	if err != nil {
		return 0, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	affected, err := rpc.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
		Revoke: opt.Revoke,
		ID:     messageIDs,
	})
	if err != nil {
		return 0, err
	}
	return int(affected.PTSCount), nil
}

// EditMessageText modifies the text content of an existing message. Supports updating
// the message body, formatting entities, inline keyboard, and scheduling.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat containing the message to edit
//   - messageID: the ID of the message to edit
//   - text: the new text content for the message
//   - opts: optional EditMessage parameters (entities, reply markup, schedule, etc.)
//
// Returns the edited message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the message does not exist or is too old to edit
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	edited, err := client.EditMessageText(ctx, chatID, 42, "updated text")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("edited message:", edited.ID)
func (c *Client) EditMessageText(ctx context.Context, chatID int64, messageID int32, text string, opts ...*params.EditMessage) (*types.Message, error) {
	c.Log.Debugf("EditMessageText chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	opt := params.GetOptDef(&params.EditMessage{}, opts...)

	// Parse formatted text when no pre-built entities are provided.
	sendText := text
	var parsedEntities []tg.MessageEntityClass
	if len(opt.Entities) == 0 {
		parsed, entities, err := c.parseText(text, opt.ParseMode)
		if err != nil {
			return nil, fmt.Errorf("parse text: %w", err)
		}
		sendText = parsed
		parsedEntities = entities
	}

	var flags tg.Fields
	if opt.DisableWebPagePreview {
		flags.Set(1)
	}
	if opt.ReplyMarkup != nil {
		flags.Set(2)
	}
	entities := opt.Entities
	if len(entities) == 0 {
		entities = parsedEntities
	}
	if len(entities) > 0 {
		flags.Set(3)
	}
	if sendText != "" {
		flags.Set(11)
	}
	if opt.InvertMedia {
		flags.Set(26)
	}

	req := &tg.MessagesEditMessageRequest{
		Flags:       flags,
		InvertMedia: opt.InvertMedia,
		Peer:        peer,
		ID:          messageID,
		Message:     sendText,
		ReplyMarkup: opt.ReplyMarkup,
		Entities:    entities,
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

// GetMessages retrieves one or more messages by their IDs from a specific chat.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat that contains the requested messages
//   - messageIDs: the IDs of the messages to retrieve
//
// Returns the requested messages on success.
//
// Returns an error if:
//   - the RPC call fails (message not found, no access, etc.)
//
// Example:
//
//	ctx := context.Background()
//	msgs, err := client.GetMessages(ctx, chatID, []int32{100, 101})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, m := range msgs {
//	    fmt.Println(m.ID, m.Text)
//	}
func (c *Client) GetMessages(ctx context.Context, chatID int64, messageIDs []int32) ([]*types.Message, error) {
	c.Log.Debugf("GetMessages chat_id=%d count=%d", chatID, len(messageIDs))
	ids := make([]tg.InputMessageClass, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = &tg.InputMessageID{ID: id}
	}

	rpc := c.Raw()
	var result tg.MessagesClass
	var err error
	peer, peerErr := resolvePeer(c, chatID)
	if peerErr != nil {
		peer, peerErr = c.ResolvePeer(ctx, chatID)
	}
	if ch, ok := peer.(*tg.InputPeerChannel); peerErr == nil && ok {
		result, err = rpc.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
			Channel: &tg.InputChannel{ChannelID: ch.ChannelID, AccessHash: ch.AccessHash},
			ID:      ids,
		})
	} else {
		result, err = rpc.MessagesGetMessages(ctx, &tg.MessagesGetMessagesRequest{ID: ids})
	}
	if err != nil {
		return nil, err
	}
	return extractMessagesFromMessagesClass(result, c)
}

// GetChatHistory retrieves a slice of message history from the specified chat,
// paginated by offset ID. Use offsetID=0 to start from the most recent messages.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat whose history to retrieve
//   - limit: maximum number of messages to return (defaults to 100 if <= 0)
//   - offsetID: message ID to start from; use 0 for the newest messages
//
// Returns the messages in reverse chronological order on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	msgs, err := client.GetChatHistory(ctx, chatID, 50, 0)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, m := range msgs {
//	    fmt.Println(m.ID, m.Text)
//	}
func (c *Client) GetChatHistory(ctx context.Context, chatID int64, limit int, offsetID int32) ([]*types.Message, error) {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	if limit <= 0 {
		limit = 100
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:     peer,
		OffsetID: offsetID,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}
	return extractMessagesFromMessagesClass(result, c)
}

// SendMedia sends a media-attached message (photo, document, video, etc.) to the
// specified chat. The media must already be represented as an InputMediaClass
// (e.g. from an upload or a file reference).
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat
//   - media: the media object to send (photo, document, video, etc.)
//   - caption: optional caption text for the media
//   - opts: optional SendMessage parameters (reply, silent, schedule, etc.)
//
// Returns the sent message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the media upload or RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	photo := &tg.InputMediaPhoto{ID: &tg.InputPhotoTL{ID: photoID}}
//	msg, err := client.SendMedia(ctx, chatID, photo, "check this out")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(msg.ID)
func (c *Client) SendMedia(ctx context.Context, chatID int64, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendMedia chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	return c.sendMediaInternal(ctx, peer, media, caption, opt)
}

func (c *Client) sendMediaInternal(ctx context.Context, peer tg.InputPeerClass, media tg.InputMediaClass, caption string, opt *params.SendMessage) (*types.Message, error) {
	// Parse formatted caption when no pre-built entities are provided.
	sendCaption := caption
	var parsedEntities []tg.MessageEntityClass
	if len(opt.Entities) == 0 {
		parsed, entities, err := c.parseText(caption, opt.ParseMode)
		if err != nil {
			return nil, fmt.Errorf("parse caption: %w", err)
		}
		sendCaption = parsed
		parsedEntities = entities
	}

	var flags tg.Fields
	if opt.DisableNotification || opt.Silent {
		flags.Set(5)
	}
	if opt.Background {
		flags.Set(6)
	}
	if opt.ClearDraft {
		flags.Set(7)
	}
	if opt.NoForwards {
		flags.Set(14)
	}
	if opt.InvertMedia {
		flags.Set(27)
	}

	var replyTo tg.InputReplyToClass
	if opt.ReplyTo != nil {
		flags.Set(0)
		replyTo = opt.ReplyTo
	} else if opt.ReplyToMessageID != 0 {
		flags.Set(0)
		replyTo = &tg.InputReplyToMessage{ReplyToMsgID: opt.ReplyToMessageID}
	}
	if opt.ReplyMarkup != nil {
		flags |= (1 << 2)
	}
	entities := opt.Entities
	if len(entities) == 0 {
		entities = parsedEntities
	}
	if len(entities) > 0 {
		flags |= (1 << 3)
	}

	req := &tg.MessagesSendMediaRequest{
		Flags:       flags,
		Silent:      opt.Silent || opt.DisableNotification,
		Background:  opt.Background,
		ClearDraft:  opt.ClearDraft,
		Noforwards:  opt.NoForwards,
		InvertMedia: opt.InvertMedia,
		Peer:        peer,
		ReplyTo:     replyTo,
		Media:       media,
		RandomID:    c.RandomID(),
		Message:     sendCaption,
		ReplyMarkup: opt.ReplyMarkup,
		Entities:    entities,
	}
	if opt.ScheduleDate != nil {
		req.ScheduleDate = *opt.ScheduleDate
	}
	if opt.EffectID != nil {
		req.Effect = *opt.EffectID
	}

	rpc := c.Raw()
	result, err := rpc.MessagesSendMedia(ctx, req)
	if err != nil {
		return nil, err
	}
	return extractSingleMessage(result, c)
}

func extractSingleMessage(result tg.UpdatesClass, binder types.Binder) (*types.Message, error) {
	switch v := result.(type) {
	case *tg.Updates:
		pm := types.NewPeerMapFromClasses(v.Users, v.Chats)
		for _, u := range v.Updates {
			switch upd := u.(type) {
			case *tg.UpdateNewMessage:
				m := types.ParseMessage(upd.Message, pm)
				if m != nil {
					m.SetBinder(binder)
				}
				return m, nil
			case *tg.UpdateNewChannelMessage:
				m := types.ParseMessage(upd.Message, pm)
				if m != nil {
					m.SetBinder(binder)
				}
				return m, nil
			}
		}
		return nil, ErrNoMessageUpdates
	case *tg.UpdateShort:
		pm := &types.PeerMap{
			Users:    make(map[int64]*tg.User),
			Chats:    make(map[int64]*tg.Chat),
			Channels: make(map[int64]*tg.Channel),
		}
		if upd, ok := v.Update.(*tg.UpdateNewMessage); ok {
			m := types.ParseMessage(upd.Message, pm)
			if m != nil {
				m.SetBinder(binder)
			}
			return m, nil
		}
		return nil, ErrNoMessageShort
	case *tg.UpdateShortSentMessage:
		return &types.Message{ID: v.ID}, nil
	default:
		return nil, fmt.Errorf("unexpected updates type %T", result)
	}
}

func extractMessages(result tg.UpdatesClass, binder types.Binder) ([]*types.Message, error) {
	switch v := result.(type) {
	case *tg.Updates:
		pm := types.NewPeerMapFromClasses(v.Users, v.Chats)
		msgs := make([]*types.Message, 0, len(v.Updates))
		for _, u := range v.Updates {
			switch upd := u.(type) {
			case *tg.UpdateNewMessage:
				if m := types.ParseMessage(upd.Message, pm); m != nil {
					m.SetBinder(binder)
					msgs = append(msgs, m)
				}
			case *tg.UpdateNewChannelMessage:
				if m := types.ParseMessage(upd.Message, pm); m != nil {
					m.SetBinder(binder)
					msgs = append(msgs, m)
				}
			}
		}
		return msgs, nil
	default:
		return nil, fmt.Errorf("unexpected updates type %T", result)
	}
}

func extractMessagesFromMessagesClass(result tg.MessagesClass, binder types.Binder) ([]*types.Message, error) {
	switch v := result.(type) {
	case *tg.MessagesMessages:
		return parseMessageClasses(v.Messages, types.NewPeerMapFromClasses(v.Users, v.Chats), binder), nil
	case *tg.MessagesMessagesSlice:
		return parseMessageClasses(v.Messages, types.NewPeerMapFromClasses(v.Users, v.Chats), binder), nil
	case *tg.MessagesChannelMessages:
		return parseMessageClasses(v.Messages, types.NewPeerMapFromClasses(v.Users, v.Chats), binder), nil
	case *tg.MessagesMessagesNotModified:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected messages type %T", result)
	}
}

func parseMessageClasses(messages []tg.MessageClass, pm *types.PeerMap, binder types.Binder) []*types.Message {
	result := make([]*types.Message, 0, len(messages))
	for _, m := range messages {
		if parsed := types.ParseMessage(m, pm); parsed != nil {
			parsed.SetBinder(binder)
			result = append(result, parsed)
		}
	}
	return result
}

// GetMediaGroup retrieves all messages belonging to the same album (grouped media)
// as the specified message. If the message is not part of a group, returns it alone.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat containing the grouped message
//   - messageID: the ID of any message in the media group
//
// Returns all messages in the album on success.
//
// Returns an error if:
//   - the message does not exist
//   - the RPC calls fail
//
// Example:
//
//	ctx := context.Background()
//	album, err := client.GetMediaGroup(ctx, chatID, 55)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, m := range album {
//	    fmt.Println("album item:", m.ID)
//	}
func (c *Client) GetMediaGroup(ctx context.Context, chatID int64, messageID int32) ([]*types.Message, error) {
	msgs, err := c.GetMessages(ctx, chatID, []int32{messageID})
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, ErrNoMessage
	}
	groupedID := msgs[0].GroupedID
	if groupedID == 0 {
		return msgs, nil
	}
	history, err := c.GetChatHistory(ctx, chatID, 10, 0)
	if err != nil {
		return nil, err
	}
	group := make([]*types.Message, 0, len(history))
	for _, m := range history {
		if m.GroupedID == groupedID {
			group = append(group, m)
		}
	}
	if len(group) == 0 {
		return msgs, nil
	}
	return group, nil
}

// GetChatHistoryCount returns the total number of messages in the specified chat.
// It performs a minimal history request (limit=1) to extract the total count from
// the server response.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat to count messages in
//
// Returns the total message count on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
func (c *Client) GetChatHistoryCount(ctx context.Context, chatID int64) (int, error) {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return 0, fmt.Errorf("resolve peer: %w", err)
	}
	rpc := c.Raw()
	result, err := rpc.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: 1,
	})
	if err != nil {
		return 0, err
	}
	switch v := result.(type) {
	case *tg.MessagesMessagesSlice:
		return int(v.Count), nil
	case *tg.MessagesChannelMessages:
		return int(v.Count), nil
	case *tg.MessagesMessages:
		return len(v.Messages), nil
	default:
		return 0, fmt.Errorf("unexpected messages type %T", result)
	}
}

// ForwardMediaGroup forwards a group of media messages (an album) from one chat to another.
// It delegates to ForwardMessages, which handles the grouped forwarding transparently.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the destination chat
//   - fromChatID: the source chat containing the album
//   - messageIDs: the IDs of all messages in the album
//   - opts: optional ForwardMessages parameters
//
// Returns the forwarded messages on success.
//
// Returns an error if the underlying ForwardMessages call fails.
func (c *Client) ForwardMediaGroup(ctx context.Context, chatID int64, fromChatID int64, messageIDs []int32, opts ...*params.ForwardMessages) ([]*types.Message, error) {
	return c.ForwardMessages(ctx, chatID, fromChatID, messageIDs, opts...)
}

// SendMediaGroup sends an album of media items (multiple photos, videos, or documents)
// as a single grouped message to the specified chat.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat
//   - items: the media items to send as a group (each wrapped in InputSingleMedia)
//   - opts: optional SendMessage parameters (silent, background, reply, schedule, etc.)
//
// Returns the sent messages on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
func (c *Client) SendMediaGroup(ctx context.Context, chatID int64, items []*tg.InputSingleMedia, opts ...*params.SendMessage) ([]*types.Message, error) {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	opt := params.GetOptDef(&params.SendMessage{}, opts...)

	var flags tg.Fields
	if opt.Silent || opt.DisableNotification {
		flags.Set(5)
	}
	if opt.Background {
		flags.Set(6)
	}

	var replyTo tg.InputReplyToClass
	if opt.ReplyToMessageID != 0 {
		flags.Set(0)
		replyTo = &tg.InputReplyToMessage{ReplyToMsgID: opt.ReplyToMessageID}
	}

	rpc := c.Raw()
	req := &tg.MessagesSendMultiMediaRequest{
		Flags:      flags,
		Silent:     opt.Silent || opt.DisableNotification,
		Background: opt.Background,
		Peer:       peer,
		ReplyTo:    replyTo,
		MultiMedia: items,
	}
	if opt.ScheduleDate != nil {
		req.ScheduleDate = *opt.ScheduleDate
	}
	result, err := rpc.MessagesSendMultiMedia(ctx, req)
	if err != nil {
		return nil, err
	}
	return extractMessages(result, c)
}

// DeleteChatHistory deletes the entire message history of a chat for the current user.
// When revoke is true, messages are also deleted for all other participants (if permitted).
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the chat whose history should be cleared
//   - maxID: delete all messages up to and including this ID; use 0 to delete all
//   - revoke: when true, delete for all participants rather than just locally
//
// Returns the pts count of affected changes on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails (insufficient permissions, etc.)
func (c *Client) DeleteChatHistory(ctx context.Context, chatID int64, maxID int32, revoke bool) (int, error) {
	c.Log.Debugf("DeleteChatHistory chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return 0, fmt.Errorf("resolve peer: %w", err)
	}
	rpc := c.Raw()
	result, err := rpc.MessagesDeleteHistory(ctx, &tg.MessagesDeleteHistoryRequest{
		Revoke: revoke,
		Peer:   peer,
		MaxID:  maxID,
	})
	if err != nil {
		return 0, err
	}
	return int(result.PTSCount), nil
}
