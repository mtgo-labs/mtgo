package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestGetSimilarChannels_WithChannel(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(-1000000000200, &tg.InputPeerChannel{ChannelID: 200, AccessHash: 7})
	inv.setResult(tg.ChannelsGetChannelRecommendationsTypeID, &tg.MessagesChats{
		Chats: []tg.ChatClass{&tg.Channel{ID: 201, Title: "Similar"}},
	})

	chats, err := c.GetSimilarChannels(context.Background(), -1000000000200)
	if err != nil {
		t.Fatalf("GetSimilarChannels() error: %v", err)
	}
	if len(chats) != 1 || chats[0].ID != -1000000000201 {
		t.Fatalf("chats = %#v, want one parsed channel", chats)
	}
	req, ok := inv.lastCall().(*tg.ChannelsGetChannelRecommendationsRequest)
	if !ok {
		t.Fatalf("expected ChannelsGetChannelRecommendationsRequest, got %T", inv.lastCall())
	}
	ch, ok := req.Channel.(*tg.InputChannel)
	if !ok || ch.ChannelID != 200 {
		t.Errorf("channel = %#v, want channel 200", req.Channel)
	}
}

func TestGetSimilarChannels_ServerWide(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.ChannelsGetChannelRecommendationsTypeID, &tg.MessagesChatsSlice{
		Chats: []tg.ChatClass{&tg.Channel{ID: 202, Title: "Pick"}},
	})

	chats, err := c.GetSimilarChannels(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetSimilarChannels() error: %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("chats = %#v, want one parsed channel", chats)
	}
	req, ok := inv.lastCall().(*tg.ChannelsGetChannelRecommendationsRequest)
	if !ok {
		t.Fatalf("expected ChannelsGetChannelRecommendationsRequest, got %T", inv.lastCall())
	}
	if req.Channel != nil {
		t.Errorf("channel = %#v, want nil for server-wide recommendations", req.Channel)
	}
}

func TestSetChatAccentColor(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(-1000000000200, &tg.InputPeerChannel{ChannelID: 200, AccessHash: 7})
	inv.setResult(tg.ChannelsUpdateColorTypeID, &tg.Updates{})

	if err := c.SetChatAccentColor(context.Background(), -1000000000200, 5, 999, true); err != nil {
		t.Fatalf("SetChatAccentColor() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.ChannelsUpdateColorRequest)
	if !ok {
		t.Fatalf("expected ChannelsUpdateColorRequest, got %T", inv.lastCall())
	}
	if req.Color != 5 || req.BackgroundEmojiID != 999 || !req.ForProfile {
		t.Errorf("request = %#v, want color 5 emoji 999 forProfile", req)
	}
}

func TestGetTopChats_PeopleCategory(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.ContactsGetTopPeersTypeID, &tg.ContactsTopPeers{
		Categories: []*tg.TopPeerCategoryPeers{{
			Category: &tg.TopPeerCategoryCorrespondents{},
			Peers: []*tg.TopPeer{
				{Peer: &tg.PeerUser{UserID: 10}, Rating: 0.9},
				{Peer: &tg.PeerUser{UserID: 11}, Rating: 0.5},
			},
		}},
		Users: []tg.UserClass{
			&tg.User{ID: 10, FirstName: "A"},
			&tg.User{ID: 11, FirstName: "B"},
		},
	})

	chats, err := c.GetTopChats(context.Background(), TopChatsPeople, 10)
	if err != nil {
		t.Fatalf("GetTopChats() error: %v", err)
	}
	if len(chats) != 2 || chats[0].ID != 10 || chats[1].ID != 11 {
		t.Fatalf("chats = %#v, want users 10 and 11 in rating order", chats)
	}
	req, ok := inv.lastCall().(*tg.ContactsGetTopPeersRequest)
	if !ok {
		t.Fatalf("expected ContactsGetTopPeersRequest, got %T", inv.lastCall())
	}
	if !req.Correspondents || req.Limit != 10 {
		t.Errorf("request = %#v, want correspondents flag with limit 10", req)
	}
}

func TestGetTopChats_CategoryFlagMapping(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.ContactsGetTopPeersTypeID, &tg.ContactsTopPeers{
		Categories: []*tg.TopPeerCategoryPeers{{
			Category: &tg.TopPeerCategoryChannels{},
			Peers:    []*tg.TopPeer{{Peer: &tg.PeerChannel{ChannelID: 300}}},
		}},
		Chats: []tg.ChatClass{&tg.Channel{ID: 300, Title: "News"}},
	})

	chats, err := c.GetTopChats(context.Background(), TopChatsChannels, 0)
	if err != nil {
		t.Fatalf("GetTopChats() error: %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("chats = %#v, want one channel", chats)
	}
	req, ok := inv.lastCall().(*tg.ContactsGetTopPeersRequest)
	if !ok {
		t.Fatalf("expected ContactsGetTopPeersRequest, got %T", inv.lastCall())
	}
	if !req.Channels || req.Limit != 10 {
		t.Errorf("request = %#v, want channels flag with default limit 10", req)
	}
}

func TestGetTopChats_NoMatchingCategory(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.ContactsGetTopPeersTypeID, &tg.ContactsTopPeers{})

	chats, err := c.GetTopChats(context.Background(), TopChatsGroups, 10)
	if err != nil {
		t.Fatalf("GetTopChats() error: %v", err)
	}
	if chats != nil {
		t.Fatalf("chats = %#v, want nil for absent category", chats)
	}
}

func TestGetChatNotificationSettings(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.AccountGetNotifySettingsTypeID, &tg.PeerNotifySettings{MuteUntil: 100})

	settings, err := c.GetChatNotificationSettings(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetChatNotificationSettings() error: %v", err)
	}
	if settings.MuteUntil != 100 {
		t.Errorf("settings = %#v, want MuteUntil 100", settings)
	}
	req, ok := inv.lastCall().(*tg.AccountGetNotifySettingsRequest)
	if !ok {
		t.Fatalf("expected AccountGetNotifySettingsRequest, got %T", inv.lastCall())
	}
	if np, ok := req.Peer.(*tg.InputNotifyPeer); !ok {
		t.Errorf("peer = %#v, want InputNotifyPeer", req.Peer)
	} else if p, ok := np.Peer.(*tg.InputPeerUser); !ok || p.UserID != 10 {
		t.Errorf("notify peer = %#v, want user 10", np.Peer)
	}
}

func TestSetChatNotificationSettings(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.AccountUpdateNotifySettingsTypeID, &tg.BoolTrue{})

	if err := c.SetChatNotificationSettings(context.Background(), 10, &tg.InputPeerNotifySettings{MuteUntil: 500}); err != nil {
		t.Fatalf("SetChatNotificationSettings() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.AccountUpdateNotifySettingsRequest)
	if !ok {
		t.Fatalf("expected AccountUpdateNotifySettingsRequest, got %T", inv.lastCall())
	}
	if req.Settings == nil || req.Settings.MuteUntil != 500 {
		t.Errorf("settings = %#v, want MuteUntil 500", req.Settings)
	}
}

func TestSetChatNotificationSettings_NilRejected(t *testing.T) {
	c, inv := newClientWithMock(t)

	if err := c.SetChatNotificationSettings(context.Background(), 10, nil); err == nil {
		t.Fatal("expected error for nil settings")
	}
	if inv.callCount() != 0 {
		t.Errorf("RPC calls = %d, want 0", inv.callCount())
	}
}
