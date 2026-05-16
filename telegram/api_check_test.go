package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestAPISurface_Stories(t *testing.T) {
	if testing.Short() {
		return
	}
	c := &Client{state: newConnectionState(), testResolver: &mockPeerResolver{}}
	var peer int64
	var user int64
	var media tg.InputMediaClass

	_, _ = c.SendStory(context.Background(), peer, media)
	_, _ = c.EditStoryCaption(context.Background(), peer, 1, "caption")
	_, _ = c.EditStoryMedia(context.Background(), peer, 1, media)
	_ = c.DeleteStories(context.Background(), peer, []int32{1})
	_, _ = c.GetStories(context.Background(), user, []int32{1})
	_, _ = c.GetChatStories(context.Background(), peer)
	_, _ = c.GetStoryViews(context.Background(), peer, []int32{1})
	_, _ = c.ForwardStory(context.Background(), peer, peer, 1)
	_ = c.PinChatStories(context.Background(), peer, []int32{1})
	_ = c.ReadChatStories(context.Background(), peer, []int32{1})
}

func TestAPISurface_Premium(t *testing.T) {
	if testing.Short() {
		return
	}
	c := &Client{state: newConnectionState(), testResolver: &mockPeerResolver{}}
	var peer int64

	_, _ = c.ApplyBoost(context.Background(), peer)
	_, _ = c.GetBoostsStatus(context.Background(), peer)
	_, _ = c.GetBoosts(context.Background())
}

func TestAPISurface_Payments(t *testing.T) {
	if testing.Short() {
		return
	}
	c := &Client{state: newConnectionState(), testResolver: &mockPeerResolver{}}
	var peer int64
	var user int64
	var creds tg.InputPaymentCredentialsClass

	_, _ = c.GetPaymentForm(context.Background(), peer, 1, nil)
	_, _ = c.SendPaymentForm(context.Background(), 1, peer, 1, creds, nil)
	_, _ = c.GetStarsBalance(context.Background(), peer)
	_, _ = c.SendGift(context.Background(), user, 1, "hello")
}

func TestAPISurface_Business(t *testing.T) {
	if testing.Short() {
		return
	}
	c := &Client{state: newConnectionState(), testResolver: &mockPeerResolver{}}

	_, _ = c.GetBusinessConnection(context.Background(), "conn_123")
}

func TestAPISurface_ReturnTypes(t *testing.T) {
	var (
		_ *types.Story              = nil
		_ *types.Message            = nil
		_ []*types.Story            = nil
		_ []*tg.StoryViews          = nil
		_ []*tg.MyBoost             = nil
		_ *tg.PremiumBoostsStatus   = nil
		_ tg.PaymentFormClass       = nil
		_ tg.PaymentResultClass     = nil
		_ *tg.BotBusinessConnection = nil
	)
}
