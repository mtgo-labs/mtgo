package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestConnectStarRef(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	c.CachePeer(99, &tg.InputPeerUser{UserID: 99, AccessHash: 30})

	inv.setResult(tg.PaymentsConnectStarRefBotTypeID, &tg.PaymentsConnectedStarRefBots{
		ConnectedBots: []*tg.ConnectedBotStarRef{{
			URL:                "https://t.me/BotBot?start=ref10",
			Date:               100,
			BotID:              99,
			CommissionPermille: 100,
			Participants:       0,
		}},
		Users: []tg.UserClass{&tg.User{ID: 99, FirstName: "RefBot"}},
	})

	refs, err := c.ConnectStarRef(t.Context(), 10, 99)
	if err != nil {
		t.Fatalf("ConnectStarRef: %v", err)
	}
	if len(refs) != 1 || refs[0].URL != "https://t.me/BotBot?start=ref10" {
		t.Fatalf("refs = %+v", refs)
	}
	if refs[0].Bot == nil || refs[0].Bot.ID != 99 {
		t.Fatalf("bot = %+v", refs[0].Bot)
	}
}

func TestGetSuggestedStarRefs(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})

	inv.setResult(tg.PaymentsGetSuggestedStarRefBotsTypeID, &tg.PaymentsSuggestedStarRefBots{
		Count: 1,
		SuggestedBots: []*tg.StarRefProgram{{
			BotID:              99,
			CommissionPermille: 50,
		}},
		Users:      []tg.UserClass{&tg.User{ID: 99, FirstName: "RefBot"}},
		NextOffset: "next",
	})

	res, err := c.GetSuggestedStarRefs(t.Context(), 10, true, "", 10)
	if err != nil {
		t.Fatalf("GetSuggestedStarRefs: %v", err)
	}
	if res.Count != 1 || len(res.Programs) != 1 || res.NextOffset != "next" {
		t.Fatalf("res = %+v", res)
	}
	req, ok := inv.lastCall().(*tg.PaymentsGetSuggestedStarRefBotsRequest)
	if !ok || !req.OrderByRevenue {
		t.Fatalf("request = %#v", inv.lastCall())
	}
}
