package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *ChatMethods) GetChatOnlineCount(ctx context.Context, chatID int64) (int, error) {
	c.Log.Debugf("GetChatOnlineCount chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
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

func (c *ChatMethods) GetSendAsChats(ctx context.Context, chatID int64) ([]*types.Chat, error) {
	c.Log.Debugf("GetSendAsChats chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.ChannelsGetSendAs(ctx, &tg.ChannelsGetSendAsRequest{
		Peer: peer,
	})
	if err != nil {
		return nil, fmt.Errorf("get send as: %w", err)
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

func (c *ChatMethods) SetSendAsChat(ctx context.Context, chatID int64, sendAs int64) error {
	c.Log.Debugf("SetSendAsChat chat_id=%d send_as=%d", chatID, sendAs)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	sendAsPeer, err := resolvePeer(ctx, c.Client, sendAs)
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

func (c *ChatMethods) TransferChatOwnership(ctx context.Context, chatID int64, userID int64, password string) error {
	c.Log.Debugf("TransferChatOwnership chat_id=%d user_id=%d", chatID, userID)
	ch, err := resolveChannelID(ctx, c.Client, chatID)
	if err != nil {
		return err
	}
	user, err := resolveUserID(ctx, c.Client, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}

	_, err = c.computeSRP(ctx, password)
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
		return fmt.Errorf("edit admin: %w", err)
	}
	return nil
}

func (c *ChatMethods) GetSuitableDiscussionChats(ctx context.Context) ([]*types.Chat, error) {
	c.Log.Debug("GetSuitableDiscussionChats")
	rpc := c.Raw()
	result, err := rpc.ChannelsGetGroupsForDiscussion(ctx)
	if err != nil {
		return nil, fmt.Errorf("get groups for discussion: %w", err)
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

func (c *ChatMethods) SetChatDiscussionGroup(ctx context.Context, broadcastChatID int64, groupChatID int64) error {
	c.Log.Debugf("SetChatDiscussionGroup broadcast=%d group=%d", broadcastChatID, groupChatID)
	broadcast, err := resolveChannelID(ctx, c.Client, broadcastChatID)
	if err != nil {
		return fmt.Errorf("resolve broadcast channel: %w", err)
	}
	group, err := resolveChannelID(ctx, c.Client, groupChatID)
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

// GetSimilarChannels returns channels similar to the given channel. Pass a
// chatID of 0 to get recommendations derived from the channels the logged-in
// user already follows.
func (c *ChatMethods) GetSimilarChannels(ctx context.Context, chatID int64) ([]*types.Chat, error) {
	c.Log.Debugf("GetSimilarChannels chat_id=%d", chatID)
	req := &tg.ChannelsGetChannelRecommendationsRequest{}
	if chatID != 0 {
		channel, err := resolveChannelID(ctx, c.Client, chatID)
		if err != nil {
			return nil, fmt.Errorf("resolve channel: %w", err)
		}
		req.Channel = channel
	}

	result, err := c.Raw().ChannelsGetChannelRecommendations(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get similar channels: %w", err)
	}
	var raw []tg.ChatClass
	switch v := result.(type) {
	case *tg.MessagesChats:
		raw = v.Chats
	case *tg.MessagesChatsSlice:
		raw = v.Chats
	default:
		return nil, fmt.Errorf("get similar channels: unexpected result %T", result)
	}
	parsed := make([]*types.Chat, 0, len(raw))
	for _, ch := range raw {
		if p := types.ParseChatFromChat(ch); p != nil {
			parsed = append(parsed, p)
		}
	}
	return parsed, nil
}

// SetChatAccentColor sets the accent color palette index and background emoji
// of a channel. Pass color 0 to clear the custom color, backgroundEmojiID 0 to
// clear the background emoji, and forProfile true to change the color shown
// next to the channel's name instead of its message bubbles.
func (c *ChatMethods) SetChatAccentColor(ctx context.Context, chatID int64, color int32, backgroundEmojiID int64, forProfile bool) error {
	c.Log.Debugf("SetChatAccentColor chat_id=%d color=%d emoji=%d profile=%v", chatID, color, backgroundEmojiID, forProfile)
	channel, err := resolveChannelID(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve channel: %w", err)
	}

	_, err = c.Raw().ChannelsUpdateColor(ctx, &tg.ChannelsUpdateColorRequest{
		Channel:           channel,
		Color:             color,
		BackgroundEmojiID: backgroundEmojiID,
		ForProfile:        forProfile,
	})
	if err != nil {
		return fmt.Errorf("set chat accent color: %w", err)
	}
	return nil
}

// TopChatCategory selects which frequently-contacted peers GetTopChats lists.
type TopChatCategory int

// Top-chat categories, mirroring the peer-rating buckets the server keeps.
const (
	// TopChatsPeople lists the people the user chats with most.
	TopChatsPeople TopChatCategory = iota
	// TopChatsBots lists the bots the user chats with most.
	TopChatsBots
	// TopChatsInlineBots lists the inline bots used most.
	TopChatsInlineBots
	// TopChatsGroups lists the groups the user is most active in.
	TopChatsGroups
	// TopChatsChannels lists the channels the user reads most.
	TopChatsChannels
	// TopChatsPhoneCalls lists the peers most often called.
	TopChatsPhoneCalls
	// TopChatsForwardRecipients lists the peers the user forwards to most.
	TopChatsForwardRecipients
	// TopChatsForwardChats lists the chats the user forwards to most.
	TopChatsForwardChats
	// TopChatsMiniApps lists the mini apps used most.
	TopChatsMiniApps
)

// rawCategory maps a top-chat category to its protocol representation.
func (cat TopChatCategory) rawCategory() tg.TopPeerCategoryClass {
	switch cat {
	case TopChatsBots:
		return &tg.TopPeerCategoryBotsPm{}
	case TopChatsInlineBots:
		return &tg.TopPeerCategoryBotsInline{}
	case TopChatsGroups:
		return &tg.TopPeerCategoryGroups{}
	case TopChatsChannels:
		return &tg.TopPeerCategoryChannels{}
	case TopChatsPhoneCalls:
		return &tg.TopPeerCategoryPhoneCalls{}
	case TopChatsForwardRecipients:
		return &tg.TopPeerCategoryForwardUsers{}
	case TopChatsForwardChats:
		return &tg.TopPeerCategoryForwardChats{}
	case TopChatsMiniApps:
		return &tg.TopPeerCategoryBotsApp{}
	default:
		return &tg.TopPeerCategoryCorrespondents{}
	}
}

// GetTopChats returns the most frequently contacted chats in a category,
// ordered by interaction frequency. A limit of 0 or less defaults to 10.
func (c *ChatMethods) GetTopChats(ctx context.Context, category TopChatCategory, limit int) ([]*types.Chat, error) {
	c.Log.Debugf("GetTopChats category=%d limit=%d", category, limit)
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	req := &tg.ContactsGetTopPeersRequest{Limit: int32(limit)}
	switch category {
	case TopChatsBots:
		req.BotsPm = true
	case TopChatsInlineBots:
		req.BotsInline = true
	case TopChatsGroups:
		req.Groups = true
	case TopChatsChannels:
		req.Channels = true
	case TopChatsPhoneCalls:
		req.PhoneCalls = true
	case TopChatsForwardRecipients:
		req.ForwardUsers = true
	case TopChatsForwardChats:
		req.ForwardChats = true
	case TopChatsMiniApps:
		req.BotsApp = true
	default:
		req.Correspondents = true
	}

	result, err := c.Raw().ContactsGetTopPeers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get top chats: %w", err)
	}
	top, ok := result.(*tg.ContactsTopPeers)
	if !ok {
		return nil, fmt.Errorf("get top chats: unexpected result %T", result)
	}
	want := category.rawCategory()
	for _, cp := range top.Categories {
		if cp == nil || cp.Category.ConstructorID() != want.ConstructorID() {
			continue
		}
		pm := types.NewPeerMapFromClasses(top.Users, top.Chats)
		chats := make([]*types.Chat, 0, len(cp.Peers))
		for _, p := range cp.Peers {
			if parsed := types.ParseChatFromPeer(p.Peer, pm); parsed != nil {
				chats = append(chats, parsed)
			}
		}
		return chats, nil
	}
	return nil, nil
}

// GetChatNotificationSettings returns the current notification settings of a
// chat.
func (c *ChatMethods) GetChatNotificationSettings(ctx context.Context, chatID int64) (*tg.PeerNotifySettings, error) {
	c.Log.Debugf("GetChatNotificationSettings chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	result, err := c.Raw().AccountGetNotifySettings(ctx, &tg.AccountGetNotifySettingsRequest{
		Peer: &tg.InputNotifyPeer{Peer: peer},
	})
	if err != nil {
		return nil, fmt.Errorf("get chat notification settings: %w", err)
	}
	return result, nil
}

// SetChatNotificationSettings replaces the notification settings of a chat.
// MuteChat and UnmuteChat cover the common mute cases; this is the general
// form for sound, previews, and custom mute intervals.
func (c *ChatMethods) SetChatNotificationSettings(ctx context.Context, chatID int64, settings *tg.InputPeerNotifySettings) error {
	c.Log.Debugf("SetChatNotificationSettings chat_id=%d", chatID)
	if settings == nil {
		return fmt.Errorf("settings must not be nil")
	}
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	_, err = c.Raw().AccountUpdateNotifySettings(ctx, &tg.AccountUpdateNotifySettingsRequest{
		Peer:     &tg.InputNotifyPeer{Peer: peer},
		Settings: settings,
	})
	if err != nil {
		return fmt.Errorf("set chat notification settings: %w", err)
	}
	return nil
}

// SetChatEmojiStatus sets the custom emoji displayed next to a channel's
// title. Pass 0 as customEmojiID to clear the status.
func (c *ChatMethods) SetChatEmojiStatus(ctx context.Context, chatID int64, customEmojiID int64) error {
	c.Log.Debugf("SetChatEmojiStatus chat_id=%d emoji=%d", chatID, customEmojiID)
	channel, err := resolveChannelID(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve channel: %w", err)
	}

	_, err = c.Raw().ChannelsUpdateEmojiStatus(ctx, &tg.ChannelsUpdateEmojiStatusRequest{
		Channel: channel,
		EmojiStatus: &tg.EmojiStatus{
			DocumentID: customEmojiID,
		},
	})
	if err != nil {
		return fmt.Errorf("set chat emoji status: %w", err)
	}
	return nil
}
