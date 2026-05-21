package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) SendReaction(ctx context.Context, chatID int64, messageID int32, reactions []types.Reaction, opts ...*params.SendReactionOption) error {
	c.Log.Debugf("SendReaction chat_id=%d msg_id=%d", chatID, messageID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	opt := params.GetOptDef(&params.SendReactionOption{}, opts...)

	rpc := c.Raw()
	_, err = rpc.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
		Big:         opt.Big,
		AddToRecent: opt.AddToRecent,
		Peer:        peer,
		MsgID:       messageID,
		Reaction:    types.ReactionsToTL(reactions),
	})
	return err
}

func (c *Client) SendPaidReaction(ctx context.Context, chatID int64, messageID int32, amount int64, opts ...*params.SendPaidReactionOption) error {
	c.Log.Debugf("SendPaidReaction chat_id=%d msg_id=%d amount=%d", chatID, messageID, amount)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	opt := params.GetOptDef(&params.SendPaidReactionOption{}, opts...)

	req := &tg.MessagesSendPaidReactionRequest{
		Peer:     peer,
		MsgID:    messageID,
		Count:    int32(amount),
		RandomID: c.RandomID(),
	}
	if opt.Private {
		req.Private = &tg.PaidReactionPrivacyAnonymous{}
	}

	rpc := c.Raw()
	_, err = rpc.MessagesSendPaidReaction(ctx, req)
	return err
}

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
	return err
}

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
	return err
}

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
	return err
}

func (c *Client) GetMessagesViews(ctx context.Context, chatID int64, messageIDs []int32, increment bool) ([]int32, error) {
	c.Log.Debugf("GetMessagesViews chat_id=%d count=%d increment=%v", chatID, len(messageIDs), increment)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetMessagesViews(ctx, &tg.MessagesGetMessagesViewsRequest{
		Peer:      peer,
		ID:        messageIDs,
		Increment: increment,
	})
	if err != nil {
		return nil, err
	}

	viewsResult, ok := result.(*tg.MessagesMessageViews)
	if !ok {
		return nil, fmt.Errorf("unexpected MessageViews type: %T", result)
	}

	views := make([]int32, len(viewsResult.Views))
	for i, v := range viewsResult.Views {
		if mv, ok := v.(*tg.MessageViews); ok {
			views[i] = mv.Views
		}
	}
	return views, nil
}

func (c *Client) GetMessageReactionsList(ctx context.Context, chatID int64, messageID int32, reaction *types.Reaction, offset string, limit int32) (*types.PeerReactionList, error) {
	c.Log.Debugf("GetMessageReactionsList chat_id=%d msg_id=%d limit=%d", chatID, messageID, limit)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	if limit <= 0 {
		limit = 100
	}

	var tlReaction tg.ReactionClass
	if reaction != nil {
		tlReaction = types.ReactionsToTL([]types.Reaction{*reaction})[0]
	}

	rpc := c.Raw()
	raw, err := rpc.MessagesGetMessageReactionsList(ctx, &tg.MessagesGetMessageReactionsListRequest{
		Peer:     peer,
		ID:       messageID,
		Reaction: tlReaction,
		Offset:   offset,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}

	out := &types.PeerReactionList{
		Count:      raw.Count,
		NextOffset: raw.NextOffset,
	}
	for _, r := range raw.Reactions {
		if r == nil {
			continue
		}
		pr := types.PeerReaction{
			Date:     r.Date,
			IsBig:    r.Big,
			IsUnread: r.Unread,
			IsMine:   r.My,
		}
		pr.ChatID = types.ExtractChatID(r.PeerID)
		if r.Reaction != nil {
			parsed := types.ParseReaction(r.Reaction)
			if parsed != nil {
				pr.Reaction = *parsed
			}
		}
		out.Reactions = append(out.Reactions, pr)
	}
	return out, nil
}

func (c *Client) GetAvailableReactions(ctx context.Context, hash int32) (*types.ReactionList, error) {
	c.Log.Debugf("GetAvailableReactions hash=%d", hash)
	rpc := c.Raw()
	raw, err := rpc.MessagesGetAvailableReactions(ctx, &tg.MessagesGetAvailableReactionsRequest{
		Hash: hash,
	})
	if err != nil {
		return nil, err
	}
	return types.ParseAvailableReactions(raw), nil
}
