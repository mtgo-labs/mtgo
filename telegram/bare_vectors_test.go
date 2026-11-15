package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestGetOwnedBots(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.BotsGetAdminedBotsTypeID, &tg.GenericVector{
		Items: []tg.TLObject{&tg.User{ID: 10, FirstName: "MyBot", Username: "mybot"}},
	})

	bots, err := c.GetOwnedBots(context.Background())
	if err != nil {
		t.Fatalf("GetOwnedBots() error: %v", err)
	}
	if len(bots) != 1 || bots[0].Username != "mybot" {
		t.Fatalf("bots = %#v, want one parsed bot", bots)
	}
}

func TestGetLeaveFolderSuggestions(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.ChatlistsGetLeaveChatlistSuggestionsTypeID, &tg.GenericVector{
		Items: []tg.TLObject{&tg.PeerChannel{ChannelID: 100}},
	})

	ids, err := c.GetLeaveFolderSuggestions(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetLeaveFolderSuggestions() error: %v", err)
	}
	if len(ids) != 1 || ids[0] != -1000000000100 {
		t.Fatalf("ids = %v, want marked channel 100", ids)
	}
	req, ok := inv.lastCall().(*tg.ChatlistsGetLeaveChatlistSuggestionsRequest)
	if !ok || req.Chatlist.FilterID != 2 {
		t.Fatalf("request = %#v, want folder 2", inv.lastCall())
	}
}
