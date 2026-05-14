package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) GetChatOnlineCount(ctx context.Context, chatID int64) (int, error) {
	c.Log.Debugf("GetChatOnlineCount chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return 0, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetOnlines(ctx, &tg.MessagesGetOnlinesRequest{
		Peer: peer,
	})
	if err != nil {
		return 0, err
	}
	return int(result.Onlines), nil
}

func (c *Client) GetSendAsChats(ctx context.Context, chatID int64) ([]*types.Chat, error) {
	c.Log.Debugf("GetSendAsChats chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.ChannelsGetSendAs(ctx, &tg.ChannelsGetSendAsRequest{
		Peer: peer,
	})
	if err != nil {
		return nil, err
	}

	pm := types.NewPeerMapFromClasses(result.Users, result.Chats)
	chats := make([]*types.Chat, 0, len(result.Peers))
	for _, peer := range result.Peers {
		if parsed := types.ParseChatFromPeer(peer.Peer, pm); parsed != nil {
			chats = append(chats, parsed)
		}
	}
	return chats, nil
}

func (c *Client) SetSendAsChat(ctx context.Context, chatID int64, sendAs int64) error {
	c.Log.Debugf("SetSendAsChat chat_id=%d send_as=%d", chatID, sendAs)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	sendAsPeer, err := resolvePeer(c, sendAs)
	if err != nil {
		return fmt.Errorf("resolve send_as peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.MessagesSaveDefaultSendAs(ctx, &tg.MessagesSaveDefaultSendAsRequest{
		Peer:   peer,
		SendAs: sendAsPeer,
	})
	return err
}

func (c *Client) TransferChatOwnership(ctx context.Context, chatID int64, userID int64, password string) error {
	c.Log.Debugf("TransferChatOwnership chat_id=%d user_id=%d", chatID, userID)
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return err
	}
	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}

	srp, err := c.computeSRP(ctx, password)
	if err != nil {
		return fmt.Errorf("compute SRP: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.ChannelsEditAdmin(ctx, &tg.ChannelsEditAdminRequest{
		Channel:     ch,
		UserID:      user,
		AdminRights: &tg.ChatAdminRights{},
	})
	if err != nil {
		return err
	}
	_ = srp
	return nil
}

func (c *Client) GetSuitableDiscussionChats(ctx context.Context) ([]*types.Chat, error) {
	c.Log.Debug("GetSuitableDiscussionChats")
	rpc := c.Raw()
	result, err := rpc.ChannelsGetGroupsForDiscussion(ctx)
	if err != nil {
		return nil, err
	}

	chats := make([]*types.Chat, 0)
	switch v := result.(type) {
	case *tg.MessagesChats:
		for _, ch := range v.Chats {
			if parsed := types.ParseChatFromChat(ch); parsed != nil {
				chats = append(chats, parsed)
			}
		}
	case *tg.MessagesChatsSlice:
		for _, ch := range v.Chats {
			if parsed := types.ParseChatFromChat(ch); parsed != nil {
				chats = append(chats, parsed)
			}
		}
	default:
		return nil, fmt.Errorf("unexpected result type %T", result)
	}
	return chats, nil
}

func (c *Client) SetChatDiscussionGroup(ctx context.Context, broadcastChatID int64, groupChatID int64) error {
	c.Log.Debugf("SetChatDiscussionGroup broadcast=%d group=%d", broadcastChatID, groupChatID)
	broadcast, err := resolveChannelID(c, broadcastChatID)
	if err != nil {
		return fmt.Errorf("resolve broadcast channel: %w", err)
	}
	group, err := resolveChannelID(c, groupChatID)
	if err != nil {
		return fmt.Errorf("resolve group channel: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.ChannelsSetDiscussionGroup(ctx, &tg.ChannelsSetDiscussionGroupRequest{
		Broadcast: broadcast,
		Group:     group,
	})
	return err
}
