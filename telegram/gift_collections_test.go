package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestGiftCollection_CRUD(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.PaymentsCreateStarGiftCollectionTypeID, &tg.StarGiftCollection{CollectionID: 3, Title: "Best"})
	inv.setResult(tg.PaymentsGetStarGiftCollectionsTypeID, &tg.PaymentsStarGiftCollections{
		Collections: []*tg.StarGiftCollection{{CollectionID: 3, Title: "Best"}},
	})

	created, err := c.CreateGiftCollection(context.Background(), 10, "Best", nil)
	if err != nil {
		t.Fatalf("CreateGiftCollection() error: %v", err)
	}
	if created.CollectionID != 3 || created.Title != "Best" {
		t.Fatalf("created = %#v, want collection 3", created)
	}

	cols, err := c.GetGiftCollections(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("GetGiftCollections() error: %v", err)
	}
	if len(cols) != 1 || cols[0].CollectionID != 3 {
		t.Fatalf("collections = %#v, want one", cols)
	}
}

func TestGetGiftCollections_NotModified(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.PaymentsGetStarGiftCollectionsTypeID, &tg.PaymentsStarGiftCollectionsNotModified{})

	cols, err := c.GetGiftCollections(context.Background(), 10, 7)
	if err != nil {
		t.Fatalf("GetGiftCollections() error: %v", err)
	}
	if cols != nil {
		t.Fatalf("collections = %#v, want nil when not modified", cols)
	}
}

func TestUpdateGiftCollection_PartialEdits(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.PaymentsUpdateStarGiftCollectionTypeID, &tg.StarGiftCollection{CollectionID: 3})

	if _, err := c.UpdateGiftCollection(context.Background(), 10, 3, "Renamed", nil, nil, nil); err != nil {
		t.Fatalf("UpdateGiftCollection() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.PaymentsUpdateStarGiftCollectionRequest)
	if !ok {
		t.Fatalf("expected PaymentsUpdateStarGiftCollectionRequest, got %T", inv.lastCall())
	}
	if req.Title != "Renamed" || req.DeleteStargift != nil || req.AddStargift != nil || req.Order != nil {
		t.Errorf("request = %#v, want title-only edit", req)
	}
}

func TestGiftAuctions(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.PaymentsGetStarGiftAuctionStateTypeID, &tg.StarGiftAuctionState{})
	inv.setResult(tg.PaymentsGetStarGiftActiveAuctionsTypeID, &tg.PaymentsStarGiftActiveAuctions{
		Auctions: []*tg.StarGiftActiveAuctionState{{}},
	})

	state, err := c.GetGiftAuctionState(context.Background(), nil, 0)
	if err != nil {
		t.Fatalf("GetGiftAuctionState() error: %v", err)
	}
	if _, ok := state.(*tg.StarGiftAuctionState); !ok {
		t.Fatalf("state = %T, want StarGiftAuctionState", state)
	}

	auctions, err := c.GetActiveGiftAuctions(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetActiveGiftAuctions() error: %v", err)
	}
	if len(auctions) != 1 {
		t.Fatalf("auctions = %#v, want one", auctions)
	}
}

func TestCheckCanSendGift(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.PaymentsCheckCanSendGiftTypeID, &tg.PaymentsCheckCanSendGiftResultFail{
		Reason: &tg.TextWithEntities{Text: "recipient too young"},
	})

	err := c.CheckCanSendGift(context.Background(), 55)
	if err == nil {
		t.Fatal("expected failure error")
	}
	req, ok := inv.lastCall().(*tg.PaymentsCheckCanSendGiftRequest)
	if !ok || req.GiftID != 55 {
		t.Fatalf("request = %#v, want gift 55", inv.lastCall())
	}
}

func TestSetGiftChatNotifications(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.PaymentsToggleChatStarGiftNotificationsTypeID, &tg.BoolTrue{})

	if err := c.SetGiftChatNotifications(context.Background(), 10, false); err != nil {
		t.Fatalf("SetGiftChatNotifications() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.PaymentsToggleChatStarGiftNotificationsRequest)
	if !ok || req.Enabled {
		t.Fatalf("request = %#v, want disabled", inv.lastCall())
	}
}
