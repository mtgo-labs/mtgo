package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSetBusinessWorkHours(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountUpdateBusinessWorkHoursTypeID, &tg.BoolTrue{})

	hours := &tg.BusinessWorkHours{
		TimezoneID: "Europe/Berlin",
		WeeklyOpen: []*tg.BusinessWeeklyOpen{{StartMinute: 540, EndMinute: 1080}},
	}
	if err := c.SetBusinessWorkHours(context.Background(), hours); err != nil {
		t.Fatalf("SetBusinessWorkHours() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.AccountUpdateBusinessWorkHoursRequest)
	if !ok {
		t.Fatalf("expected AccountUpdateBusinessWorkHoursRequest, got %T", inv.lastCall())
	}
	if req.BusinessWorkHours != hours {
		t.Errorf("request = %#v, want the given hours", req)
	}

	if err := c.SetBusinessWorkHours(context.Background(), nil); err != nil {
		t.Fatalf("SetBusinessWorkHours(clear) error: %v", err)
	}
	if req, ok := inv.lastCall().(*tg.AccountUpdateBusinessWorkHoursRequest); !ok || req.BusinessWorkHours != nil {
		t.Errorf("clear request = %#v, want nil hours", req)
	}
}

func TestSetBusinessLocation(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountUpdateBusinessLocationTypeID, &tg.BoolTrue{})

	if err := c.SetBusinessLocation(context.Background(), "Main St 1", 52.5, 13.4); err != nil {
		t.Fatalf("SetBusinessLocation() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.AccountUpdateBusinessLocationRequest)
	if !ok {
		t.Fatalf("expected AccountUpdateBusinessLocationRequest, got %T", inv.lastCall())
	}
	if req.Address != "Main St 1" {
		t.Errorf("address = %q, want Main St 1", req.Address)
	}
	if geo, ok := req.GeoPoint.(*tg.InputGeoPoint); !ok || geo.Lat != 52.5 {
		t.Errorf("geo = %#v, want 52.5", req.GeoPoint)
	}
}

func TestSetBusinessMessagesAndIntro(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountUpdateBusinessGreetingMessageTypeID, &tg.BoolTrue{})
	inv.setResult(tg.AccountUpdateBusinessAwayMessageTypeID, &tg.BoolTrue{})
	inv.setResult(tg.AccountUpdateBusinessIntroTypeID, &tg.BoolTrue{})

	greeting := &tg.InputBusinessGreetingMessage{ShortcutID: 3, NoActivityDays: 7}
	if err := c.SetBusinessGreetingMessage(context.Background(), greeting); err != nil {
		t.Fatalf("SetBusinessGreetingMessage() error: %v", err)
	}
	away := &tg.InputBusinessAwayMessage{ShortcutID: 4}
	if err := c.SetBusinessAwayMessage(context.Background(), away); err != nil {
		t.Fatalf("SetBusinessAwayMessage() error: %v", err)
	}
	intro := &tg.InputBusinessIntro{Title: "Welcome", Description: "Hi"}
	if err := c.SetBusinessIntro(context.Background(), intro); err != nil {
		t.Fatalf("SetBusinessIntro() error: %v", err)
	}
	if _, ok := inv.lastCall().(*tg.AccountUpdateBusinessIntroRequest); !ok {
		t.Fatalf("expected AccountUpdateBusinessIntroRequest, got %T", inv.lastCall())
	}
}

func TestBusinessChatLink_CRUD(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountGetBusinessChatLinksTypeID, &tg.AccountBusinessChatLinks{
		Links: []*tg.BusinessChatLink{{Link: "https://t.me/business?slug=a", Title: "Main"}},
	})
	inv.setResult(tg.AccountCreateBusinessChatLinkTypeID, &tg.BusinessChatLink{Link: "https://t.me/business?slug=b"})
	inv.setResult(tg.AccountEditBusinessChatLinkTypeID, &tg.BusinessChatLink{Link: "https://t.me/business?slug=b", Title: "Edited"})
	inv.setResult(tg.AccountDeleteBusinessChatLinkTypeID, &tg.BoolTrue{})
	inv.setResult(tg.AccountResolveBusinessChatLinkTypeID, &tg.AccountResolvedBusinessChatLinks{
		Peer:    &tg.PeerUser{UserID: 10},
		Message: "hello",
	})

	links, err := c.GetBusinessChatLinks(context.Background())
	if err != nil || len(links) != 1 || links[0].Title != "Main" {
		t.Fatalf("GetBusinessChatLinks() = %#v err=%v, want one Main link", links, err)
	}
	def := &tg.InputBusinessChatLink{Message: "hello"}
	created, err := c.CreateBusinessChatLink(context.Background(), def)
	if err != nil || created.Link != "https://t.me/business?slug=b" {
		t.Fatalf("CreateBusinessChatLink() = %#v err=%v", created, err)
	}
	edited, err := c.EditBusinessChatLink(context.Background(), "b", def)
	if err != nil || edited.Title != "Edited" {
		t.Fatalf("EditBusinessChatLink() = %#v err=%v", edited, err)
	}
	if err := c.DeleteBusinessChatLink(context.Background(), "b"); err != nil {
		t.Fatalf("DeleteBusinessChatLink() error: %v", err)
	}
	if req, ok := inv.lastCall().(*tg.AccountDeleteBusinessChatLinkRequest); !ok || req.Slug != "b" {
		t.Fatalf("request = %#v, want slug b", inv.lastCall())
	}
	resolved, err := c.ResolveBusinessChatLink(context.Background(), "b")
	if err != nil || resolved.Message != "hello" {
		t.Fatalf("ResolveBusinessChatLink() = %#v err=%v", resolved, err)
	}
}
