package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestGetPinnedAndArchivedStories(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	page := func() *tg.StoriesStories {
		return &tg.StoriesStories{
			Count: 1,
			Stories: []tg.StoryItemClass{
				&tg.StoryItem{ID: 9, Pinned: true},
			},
			Chats: []tg.ChatClass{},
			Users: []tg.UserClass{&tg.User{ID: 10, FirstName: "Pinner"}},
		}
	}
	inv.setResult(tg.StoriesGetPinnedStoriesTypeID, page())
	inv.setResult(tg.StoriesGetStoriesArchiveTypeID, &tg.StoriesStories{
		Stories: []tg.StoryItemClass{&tg.StoryItem{ID: 4}},
		Chats:   []tg.ChatClass{},
		Users:   []tg.UserClass{&tg.User{ID: 10, FirstName: "Pinner"}},
	})

	pinned, err := c.GetPinnedStories(context.Background(), 10, 0, 10)
	if err != nil {
		t.Fatalf("GetPinnedStories() error: %v", err)
	}
	if len(pinned) != 1 || pinned[0].ID != 9 {
		t.Fatalf("pinned = %#v, want story 9", pinned)
	}
	req, ok := inv.lastCall().(*tg.StoriesGetPinnedStoriesRequest)
	if !ok || req.Limit != 10 {
		t.Fatalf("request = %#v, want limit 10", inv.lastCall())
	}

	archived, err := c.GetArchivedStories(context.Background(), 10, 0, 0)
	if err != nil {
		t.Fatalf("GetArchivedStories() error: %v", err)
	}
	if len(archived) != 1 || archived[0].ID != 4 {
		t.Fatalf("archived = %#v, want story 4", archived)
	}
}

func TestGetAllStories_Feed(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.StoriesGetAllStoriesTypeID, &tg.StoriesAllStories{
		Count:   3,
		State:   "s1",
		HasMore: true,
		PeerStories: []tg.PeerStoriesClass{
			&tg.PeerStories{
				Peer:    &tg.PeerUser{UserID: 10},
				Stories: []tg.StoryItemClass{&tg.StoryItem{ID: 1}},
			},
			&tg.PeerStories{
				Peer:    &tg.PeerChannel{ChannelID: 100},
				Stories: []tg.StoryItemClass{&tg.StoryItem{ID: 2}},
			},
		},
		Chats: []tg.ChatClass{&tg.Channel{ID: 100, Title: "Chan"}},
		Users: []tg.UserClass{&tg.User{ID: 10, FirstName: "U"}},
	})

	feed, err := c.GetAllStories(context.Background(), "", false, false)
	if err != nil {
		t.Fatalf("GetAllStories() error: %v", err)
	}
	if feed.Count != 3 || !feed.HasMore || feed.State != "s1" {
		t.Fatalf("feed = %#v, want count 3 with more", feed)
	}
	if len(feed.Peers) != 2 {
		t.Fatalf("entries = %d, want 2", len(feed.Peers))
	}
	if feed.Peers[0].Chat == nil || feed.Peers[0].Chat.ID != 10 {
		t.Errorf("entry 0 chat = %#v, want user 10", feed.Peers[0].Chat)
	}
	if len(feed.Peers[0].Stories) != 1 || feed.Peers[0].Stories[0].ID != 1 {
		t.Errorf("entry 0 stories = %#v, want story 1", feed.Peers[0].Stories)
	}
}

func TestGetAllStories_NotModified(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.StoriesGetAllStoriesTypeID, &tg.StoriesAllStoriesNotModified{})

	feed, err := c.GetAllStories(context.Background(), "s1", true, false)
	if err != nil {
		t.Fatalf("GetAllStories() error: %v", err)
	}
	if feed != nil {
		t.Fatalf("feed = %#v, want nil when not modified", feed)
	}
}

func TestActivateStealthMode(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.StoriesActivateStealthModeTypeID, &tg.Updates{})

	if err := c.ActivateStealthMode(context.Background(), true, true); err != nil {
		t.Fatalf("ActivateStealthMode() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.StoriesActivateStealthModeRequest)
	if !ok || !req.Past || !req.Future {
		t.Fatalf("request = %#v, want past and future", inv.lastCall())
	}
}

func TestGetRemainingStoryCount(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.StoriesCanSendStoryTypeID, &tg.StoriesCanSendStoryCount{CountRemains: 3})

	n, err := c.GetRemainingStoryCount(context.Background())
	if err != nil {
		t.Fatalf("GetRemainingStoryCount() error: %v", err)
	}
	if n != 3 {
		t.Fatalf("count = %d, want 3", n)
	}
}

func TestGetChatsToSendStories(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.StoriesGetChatsToSendTypeID, &tg.MessagesChats{
		Chats: []tg.ChatClass{&tg.Channel{ID: 100, Title: "MyChannel"}},
	})

	chats, err := c.GetChatsToSendStories(context.Background())
	if err != nil {
		t.Fatalf("GetChatsToSendStories() error: %v", err)
	}
	if len(chats) != 1 || chats[0].ID != -1000000000100 {
		t.Fatalf("chats = %#v, want one channel", chats)
	}
}

func TestExportStoryLink(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.StoriesExportStoryLinkTypeID, &tg.ExportedStoryLink{Link: "https://t.me/story/9"})

	link, err := c.ExportStoryLink(context.Background(), 10, 9)
	if err != nil {
		t.Fatalf("ExportStoryLink() error: %v", err)
	}
	if link != "https://t.me/story/9" {
		t.Fatalf("link = %q", link)
	}
}

func TestToggleStoriesHidden(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.StoriesTogglePeerStoriesHiddenTypeID, &tg.BoolTrue{})
	inv.setResult(tg.StoriesToggleAllStoriesHiddenTypeID, &tg.BoolTrue{})

	if err := c.ToggleStoriesHidden(context.Background(), 10, true); err != nil {
		t.Fatalf("ToggleStoriesHidden() error: %v", err)
	}
	if req, ok := inv.lastCall().(*tg.StoriesTogglePeerStoriesHiddenRequest); !ok || !req.Hidden {
		t.Fatalf("request = %#v, want hidden", inv.lastCall())
	}
	if err := c.ToggleAllStoriesHidden(context.Background(), true); err != nil {
		t.Fatalf("ToggleAllStoriesHidden() error: %v", err)
	}
}

func TestGetStoryViewers(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.StoriesGetStoryViewsListTypeID, &tg.StoriesStoryViewsList{
		Count:          10,
		ViewsCount:     8,
		ForwardsCount:  1,
		ReactionsCount: 1,
		Views:          []tg.StoryViewClass{&tg.StoryView{UserID: 11}},
		NextOffset:     "n2",
	})

	page, err := c.GetStoryViewers(context.Background(), 10, 9, "", true, false, false, "", 0)
	if err != nil {
		t.Fatalf("GetStoryViewers() error: %v", err)
	}
	if page.Count != 10 || page.ViewsCount != 8 || page.NextOffset != "n2" {
		t.Fatalf("page = %#v, want full counters", page)
	}
	req, ok := inv.lastCall().(*tg.StoriesGetStoryViewsListRequest)
	if !ok {
		t.Fatalf("expected StoriesGetStoryViewsListRequest, got %T", inv.lastCall())
	}
	if !req.JustContacts || req.ID != 9 || req.Limit != 10 {
		t.Errorf("request = %#v, want just-contacts story 9 default limit", req)
	}
}

func TestGetStoryReactions(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.StoriesGetStoryReactionsListTypeID, &tg.StoriesStoryReactionsList{
		Count:     2,
		Reactions: []tg.StoryReactionClass{&tg.StoryReaction{PeerID: &tg.PeerUser{UserID: 11}, Reaction: &tg.ReactionEmoji{Emoticon: "🔥"}}},
	})

	page, err := c.GetStoryReactions(context.Background(), 10, 9, nil, "", 0, false)
	if err != nil {
		t.Fatalf("GetStoryReactions() error: %v", err)
	}
	if page.Count != 2 || len(page.Reactions) != 1 {
		t.Fatalf("page = %#v, want one of two reactions", page)
	}
}

func TestReportStory(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.StoriesReportTypeID, &tg.ReportResultReported{})

	if err := c.ReportStory(context.Background(), 10, []int32{9}, []byte{1}, "spam"); err != nil {
		t.Fatalf("ReportStory() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.StoriesReportRequest)
	if !ok {
		t.Fatalf("expected StoriesReportRequest, got %T", inv.lastCall())
	}
	if len(req.ID) != 1 || string(req.Option) != "\x01" || req.Message != "spam" {
		t.Errorf("request = %#v, want story 9 with option and message", req)
	}
}
