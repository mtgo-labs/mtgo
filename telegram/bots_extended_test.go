package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestBotUserMenuButton_RoundTrip(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.BotsSetBotMenuButtonTypeID, &tg.BoolTrue{})
	inv.setResult(tg.BotsGetBotMenuButtonTypeID, &tg.BotMenuButton{Text: "Open", URL: "https://app.example"})

	btn := &tg.BotMenuButton{Text: "Open", URL: "https://app.example"}
	if err := c.SetBotUserMenuButton(context.Background(), 10, btn); err != nil {
		t.Fatalf("SetBotUserMenuButton() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.BotsSetBotMenuButtonRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotMenuButtonRequest, got %T", inv.lastCall())
	}
	if u, ok := req.UserID.(*tg.InputUser); !ok || u.UserID != 10 {
		t.Errorf("user = %#v, want user 10", req.UserID)
	}

	got, err := c.GetBotUserMenuButton(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetBotUserMenuButton() error: %v", err)
	}
	if b, ok := got.(*tg.BotMenuButton); !ok || b.URL != "https://app.example" {
		t.Errorf("button = %#v, want the set button", got)
	}
}

func TestSetBotDefaultAdminRights_GroupAndBroadcast(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.BotsSetBotGroupDefaultAdminRightsTypeID, &tg.BoolTrue{})
	inv.setResult(tg.BotsSetBotBroadcastDefaultAdminRightsTypeID, &tg.BoolTrue{})

	rights := &tg.ChatAdminRights{PostMessages: true}
	if err := c.SetBotDefaultAdminRights(context.Background(), false, rights); err != nil {
		t.Fatalf("SetBotDefaultAdminRights(groups) error: %v", err)
	}
	if _, ok := inv.lastCall().(*tg.BotsSetBotGroupDefaultAdminRightsRequest); !ok {
		t.Fatalf("expected group rights request, got %T", inv.lastCall())
	}

	if err := c.SetBotDefaultAdminRights(context.Background(), true, rights); err != nil {
		t.Fatalf("SetBotDefaultAdminRights(channels) error: %v", err)
	}
	if _, ok := inv.lastCall().(*tg.BotsSetBotBroadcastDefaultAdminRightsRequest); !ok {
		t.Fatalf("expected broadcast rights request, got %T", inv.lastCall())
	}
}

func TestCanMessageBot_AllowBotToMessage(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.BotsCanSendMessageTypeID, &tg.BoolTrue{})
	inv.setResult(tg.BotsAllowSendMessageTypeID, &tg.Updates{})

	ok, err := c.CanMessageBot(context.Background(), 10)
	if err != nil {
		t.Fatalf("CanMessageBot() error: %v", err)
	}
	if !ok {
		t.Error("CanMessageBot() = false, want true")
	}
	if err := c.AllowBotToMessage(context.Background(), 10); err != nil {
		t.Fatalf("AllowBotToMessage() error: %v", err)
	}
	if _, ok := inv.lastCall().(*tg.BotsAllowSendMessageRequest); !ok {
		t.Fatalf("expected BotsAllowSendMessageRequest, got %T", inv.lastCall())
	}
}

func TestInvokeWebViewCustomMethod(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.BotsInvokeWebViewCustomMethodTypeID, &tg.DataJSON{Data: `{"ok":true}`})

	out, err := c.InvokeWebViewCustomMethod(context.Background(), 10, "saveData", `{"k":1}`)
	if err != nil {
		t.Fatalf("InvokeWebViewCustomMethod() error: %v", err)
	}
	if out != `{"ok":true}` {
		t.Errorf("out = %q, want the JSON reply", out)
	}
	req, ok := inv.lastCall().(*tg.BotsInvokeWebViewCustomMethodRequest)
	if !ok {
		t.Fatalf("expected BotsInvokeWebViewCustomMethodRequest, got %T", inv.lastCall())
	}
	if req.CustomMethod != "saveData" || req.Params.Data != `{"k":1}` {
		t.Errorf("request = %#v, want saveData with params", req)
	}
}

func TestGetPopularAppBots(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.BotsGetPopularAppBotsTypeID, &tg.BotsPopularAppBots{
		NextOffset: "next",
		Users:      []tg.UserClass{&tg.User{ID: 10, FirstName: "AppBot", Username: "appbot"}},
	})

	bots, next, err := c.GetPopularAppBots(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("GetPopularAppBots() error: %v", err)
	}
	if len(bots) != 1 || bots[0].Username != "appbot" {
		t.Fatalf("bots = %#v, want one parsed app bot", bots)
	}
	if next != "next" {
		t.Errorf("next offset = %q, want next", next)
	}
	req, ok := inv.lastCall().(*tg.BotsGetPopularAppBotsRequest)
	if !ok || req.Limit != 10 {
		t.Fatalf("request = %#v, want default limit 10", inv.lastCall())
	}
}

func TestBotUsernames_ReorderAndToggle(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.BotsReorderUsernamesTypeID, &tg.BoolTrue{})
	inv.setResult(tg.BotsToggleUsernameTypeID, &tg.BoolTrue{})

	if err := c.ReorderBotUsernames(context.Background(), 10, []string{"main", "alt"}); err != nil {
		t.Fatalf("ReorderBotUsernames() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.BotsReorderUsernamesRequest)
	if !ok || len(req.Order) != 2 {
		t.Fatalf("request = %#v, want order of two", inv.lastCall())
	}

	if err := c.ToggleBotUsername(context.Background(), 10, "alt", false); err != nil {
		t.Fatalf("ToggleBotUsername() error: %v", err)
	}
	if _, ok := inv.lastCall().(*tg.BotsToggleUsernameRequest); !ok {
		t.Fatalf("expected BotsToggleUsernameRequest, got %T", inv.lastCall())
	}
}

func TestGetBotRecommendations(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.BotsGetBotRecommendationsTypeID, &tg.UsersUsers{
		Users: []tg.UserClass{&tg.User{ID: 11, FirstName: "Sim", Username: "simbot"}},
	})

	recs, err := c.GetBotRecommendations(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetBotRecommendations() error: %v", err)
	}
	if len(recs) != 1 || recs[0].ID != 11 {
		t.Fatalf("recs = %#v, want user 11", recs)
	}
}
