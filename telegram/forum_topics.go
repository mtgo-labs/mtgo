package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) CreateForumTopic(ctx context.Context, chatID int64, title string, iconColor *int32, iconEmojiID *int64) (*types.ForumTopic, error) {
	c.Log.Debugf("CreateForumTopic chat_id=%d title=%s", chatID, title)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	var iconColorValue int32
	if iconColor != nil {
		iconColorValue = *iconColor
	}
	var iconEmojiIDValue int64
	if iconEmojiID != nil {
		iconEmojiIDValue = *iconEmojiID
	}
	result, err := rpc.MessagesCreateForumTopic(ctx, &tg.MessagesCreateForumTopicRequest{
		Peer:        peer,
		Title:       title,
		IconColor:   iconColorValue,
		IconEmojiID: iconEmojiIDValue,
		RandomID:    c.RandomID(),
	})
	if err != nil {
		return nil, err
	}

	return extractForumTopicFromUpdates(result)
}

func (c *Client) EditForumTopic(ctx context.Context, chatID int64, topicID int32, title *string, iconEmojiID *int64) error {
	c.Log.Debugf("EditForumTopic chat_id=%d topic_id=%d", chatID, topicID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	var titleValue string
	if title != nil {
		titleValue = *title
	}
	var iconEmojiIDValue int64
	if iconEmojiID != nil {
		iconEmojiIDValue = *iconEmojiID
	}
	_, err = rpc.MessagesEditForumTopic(ctx, &tg.MessagesEditForumTopicRequest{
		Peer:        peer,
		TopicID:     topicID,
		Title:       titleValue,
		IconEmojiID: iconEmojiIDValue,
	})
	return err
}

func (c *Client) CloseForumTopic(ctx context.Context, chatID int64, topicID int32) error {
	c.Log.Debugf("CloseForumTopic chat_id=%d topic_id=%d", chatID, topicID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.MessagesEditForumTopic(ctx, &tg.MessagesEditForumTopicRequest{
		Peer:    peer,
		TopicID: topicID,
		Closed:  true,
	})
	return err
}

func (c *Client) ReopenForumTopic(ctx context.Context, chatID int64, topicID int32) error {
	c.Log.Debugf("ReopenForumTopic chat_id=%d topic_id=%d", chatID, topicID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	req := &tg.MessagesEditForumTopicRequest{
		Peer:    peer,
		TopicID: topicID,
		Closed:  false,
	}
	req.Flags.Set(2)
	_, err = rpc.MessagesEditForumTopic(ctx, req)
	return err
}

func (c *Client) DeleteForumTopic(ctx context.Context, chatID int64, topicID int32) error {
	c.Log.Debugf("DeleteForumTopic chat_id=%d topic_id=%d", chatID, topicID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.MessagesDeleteTopicHistory(ctx, &tg.MessagesDeleteTopicHistoryRequest{
		Peer:     peer,
		TopMsgID: topicID,
	})
	return err
}

func (c *Client) GetForumTopics(ctx context.Context, chatID int64, query string, limit int, offsetDate int32, offsetTopic int32) ([]*types.ForumTopic, error) {
	c.Log.Debugf("GetForumTopics chat_id=%d limit=%d", chatID, limit)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	if limit <= 0 {
		limit = 100
	}

	var q string
	if query != "" {
		q = query
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetForumTopics(ctx, &tg.MessagesGetForumTopicsRequest{
		Peer:        peer,
		Q:           q,
		OffsetDate:  offsetDate,
		OffsetTopic: offsetTopic,
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, err
	}

	topics := make([]*types.ForumTopic, 0, len(result.Topics))
	for _, t := range result.Topics {
		if ft := types.ParseForumTopic(t); ft != nil {
			topics = append(topics, ft)
		}
	}
	return topics, nil
}

func extractForumTopicFromUpdates(result tg.UpdatesClass) (*types.ForumTopic, error) {
	switch v := result.(type) {
	case *tg.Updates:
		for _, u := range v.Updates {
			if upd, ok := u.(*tg.UpdateNewChannelMessage); ok {
				if msg, ok := upd.Message.(*tg.MessageService); ok && msg.Action != nil {
					if action, ok := msg.Action.(*tg.MessageActionTopicCreate); ok {
						return &types.ForumTopic{
							ID:        msg.ID,
							Date:      time.Unix(int64(msg.Date), 0),
							Title:     action.Title,
							IconColor: action.IconColor,
						}, nil
					}
				}
			}
		}
		return nil, ErrForumTopicNotFound
	default:
		return nil, fmt.Errorf("unexpected updates type %T", result)
	}
}
