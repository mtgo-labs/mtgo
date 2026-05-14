package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) GetDiscussionMessage(ctx context.Context, chatID int64, messageID int32) (*types.Message, error) {
	c.Log.Debugf("GetDiscussionMessage chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetDiscussionMessage(ctx, &tg.MessagesGetDiscussionMessageRequest{
		Peer:  peer,
		MsgID: messageID,
	})
	if err != nil {
		return nil, err
	}

	if len(result.Messages) == 0 {
		return nil, nil
	}
	pm := types.NewPeerMapFromClasses(result.Users, result.Chats)
	m := types.ParseMessage(result.Messages[0], pm)
	if m != nil {
		m.SetBinder(c)
	}
	return m, nil
}

func (c *Client) GetDiscussionReplies(ctx context.Context, chatID int64, messageID int32, limit int, offsetID int32) ([]*types.Message, error) {
	c.Log.Debugf("GetDiscussionReplies chat_id=%d msg_id=%d limit=%d", chatID, messageID, limit)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	if limit <= 0 {
		limit = 100
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetReplies(ctx, &tg.MessagesGetRepliesRequest{
		Peer:     peer,
		MsgID:    messageID,
		OffsetID: offsetID,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}
	return extractMessagesFromMessagesClass(result, c)
}

func (c *Client) GetDiscussionRepliesCount(ctx context.Context, chatID int64, messageID int32) (int, error) {
	c.Log.Debugf("GetDiscussionRepliesCount chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return 0, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetReplies(ctx, &tg.MessagesGetRepliesRequest{
		Peer:  peer,
		MsgID: messageID,
		Limit: 1,
	})
	if err != nil {
		return 0, err
	}
	count, err := extractMessagesCount(result)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (c *Client) SendMessageDraft(ctx context.Context, chatID int64, text string, opts ...*params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendMessageDraft chat_id=%d", chatID)
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	opt.ClearDraft = true
	return c.SendMessage(ctx, chatID, text, opt)
}
