package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSendStory_NilMedia(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.SendStory(context.Background(), 1, nil)
	if err == nil {
		t.Fatal("expected error for nil media")
	}
}

func TestSendStory_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	media := &tg.InputMediaPhoto{}
	_, err := c.SendStory(context.Background(), 999, media)
	if err == nil {
		t.Fatal("expected error for peer resolve failure")
	}
}

func TestSendStory_Options(t *testing.T) {
	cfg := &SendStoryOption{
		Pinned:       true,
		NoForwards:   true,
		PrivacyRules: []tg.InputPrivacyRuleClass{&tg.InputPrivacyValueAllowContacts{}},
	}
	if !cfg.Pinned {
		t.Error("expected Pinned to be true")
	}
	if !cfg.NoForwards {
		t.Error("expected NoForwards to be true")
	}
	if len(cfg.PrivacyRules) != 1 {
		t.Error("expected 1 privacy rule")
	}
}

func TestEditStoryCaption_EmptyCaption(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.EditStoryCaption(context.Background(), 1, 1, "")
	if err == nil {
		t.Fatal("expected error for empty caption")
	}
}

func TestEditStoryCaption_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.EditStoryCaption(context.Background(), 999, 1, "hello")
	if err == nil {
		t.Fatal("expected error for peer resolve failure")
	}
}

func TestEditStoryMedia_NilMedia(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.EditStoryMedia(context.Background(), 1, 1, nil)
	if err == nil {
		t.Fatal("expected error for nil media")
	}
}

func TestEditStoryMedia_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	media := &tg.InputMediaPhoto{}
	_, err := c.EditStoryMedia(context.Background(), 999, 1, media)
	if err == nil {
		t.Fatal("expected error for peer resolve failure")
	}
}

func TestDeleteStories_NotSupported(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	err := c.DeleteStories(context.Background(), 1, []int32{1})
	if err == nil {
		t.Fatal("expected error for delete stories")
	}
}

func TestPinChatStories_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	err := c.PinChatStories(context.Background(), 999, []int32{1})
	if err == nil {
		t.Fatal("expected error for peer resolve failure")
	}
}

func TestReadChatStories_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	err := c.ReadChatStories(context.Background(), 999, []int32{1})
	if err == nil {
		t.Fatal("expected error for peer resolve failure")
	}
}

func TestGetStories_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.GetStories(context.Background(), 999, []int32{1})
	if err == nil {
		t.Fatal("expected error for peer resolve failure")
	}
}

func TestGetStories_EmptyIDs(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.GetStories(context.Background(), 1, []int32{})
	if err == nil {
		t.Fatal("expected error for empty story IDs")
	}
}

func TestGetChatStories_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.GetChatStories(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for peer resolve failure")
	}
}

func TestGetStoryViews_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.GetStoryViews(context.Background(), 999, []int32{1})
	if err == nil {
		t.Fatal("expected error for peer resolve failure")
	}
}

func TestGetStoryViews_EmptyIDs(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.GetStoryViews(context.Background(), 1, []int32{})
	if err == nil {
		t.Fatal("expected error for empty story IDs")
	}
}

func TestForwardStory_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.ForwardStory(context.Background(), 999, 1, 1)
	if err == nil {
		t.Fatal("expected error for target peer resolve failure")
	}
}

func TestForwardStory_SourcePeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{
		peers: map[int64]tg.InputPeerClass{
			1: &tg.InputPeerUser{UserID: 1, AccessHash: 1},
		},
	}}
	_, err := c.ForwardStory(context.Background(), 1, 999, 1)
	if err == nil {
		t.Fatal("expected error for source peer resolve failure")
	}
}
