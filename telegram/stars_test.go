package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestGetStarsTopupOptions(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.PaymentsGetStarsTopupOptionsTypeID, &tg.GenericVector{
		Items: []tg.TLObject{
			&tg.StarsTopupOption{Stars: 100, Currency: "USD", Amount: 199},
			&tg.StarsTopupOption{Stars: 500, Currency: "USD", Amount: 899},
		},
	})

	options, err := c.GetStarsTopupOptions(context.Background())
	if err != nil {
		t.Fatalf("GetStarsTopupOptions() error: %v", err)
	}
	if len(options) != 2 || options[0].Stars != 100 {
		t.Fatalf("options = %#v, want two packages", options)
	}
}

func TestGetStarsSubscriptions(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.PaymentsGetStarsSubscriptionsTypeID, &tg.PaymentsStarsStatus{
		Subscriptions: []*tg.StarsSubscription{{ID: "sub1"}},
	})

	subs, err := c.GetStarsSubscriptions(context.Background(), "", false)
	if err != nil {
		t.Fatalf("GetStarsSubscriptions() error: %v", err)
	}
	if len(subs.Subscriptions) != 1 || subs.Subscriptions[0].ID != "sub1" {
		t.Fatalf("subscriptions = %#v, want sub1", subs)
	}
	req, ok := inv.lastCall().(*tg.PaymentsGetStarsSubscriptionsRequest)
	if !ok {
		t.Fatalf("expected PaymentsGetStarsSubscriptionsRequest, got %T", inv.lastCall())
	}
	if p, ok := req.Peer.(*tg.InputPeerSelf); !ok || p == nil {
		t.Errorf("peer = %#v, want self", req.Peer)
	}
}

func TestChangeStarsSubscription(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.PaymentsChangeStarsSubscriptionTypeID, &tg.BoolTrue{})

	if err := c.ChangeStarsSubscription(context.Background(), 10, "sub1", true); err != nil {
		t.Fatalf("ChangeStarsSubscription() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.PaymentsChangeStarsSubscriptionRequest)
	if !ok {
		t.Fatalf("expected PaymentsChangeStarsSubscriptionRequest, got %T", inv.lastCall())
	}
	if req.SubscriptionID != "sub1" || !req.Canceled {
		t.Errorf("request = %#v, want canceled sub1", req)
	}
}

func TestStarsRevenue(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(-1000000000200, &tg.InputPeerChannel{ChannelID: 200, AccessHash: 7})
	inv.setResult(tg.PaymentsGetStarsRevenueStatsTypeID, &tg.PaymentsStarsRevenueStats{})
	inv.setResult(tg.PaymentsGetStarsRevenueWithdrawalURLTypeID, &tg.PaymentsStarsRevenueWithdrawalURL{URL: "https://fragment.com/withdraw"})
	inv.setResult(tg.PaymentsGetStarsRevenueAdsAccountURLTypeID, &tg.PaymentsStarsRevenueAdsAccountURL{URL: "https://ads.example"})

	if _, err := c.GetStarsRevenueStats(context.Background(), -1000000000200, true); err != nil {
		t.Fatalf("GetStarsRevenueStats() error: %v", err)
	}

	url, err := c.WithdrawStarsRevenue(context.Background(), -1000000000200, 750, "")
	if err != nil {
		t.Fatalf("WithdrawStarsRevenue() error: %v", err)
	}
	if url != "https://fragment.com/withdraw" {
		t.Fatalf("url = %q", url)
	}
	withdrawReq, ok := inv.lastCall().(*tg.PaymentsGetStarsRevenueWithdrawalURLRequest)
	if !ok || withdrawReq.Amount != 750 || withdrawReq.Password != nil {
		t.Fatalf("request = %#v, want 750 stars without SRP", inv.lastCall())
	}

	adsURL, err := c.GetStarsRevenueAdsAccountURL(context.Background(), -1000000000200)
	if err != nil || adsURL != "https://ads.example" {
		t.Fatalf("GetStarsRevenueAdsAccountURL() = %q err=%v", adsURL, err)
	}
}

func TestGetPremiumGiftCodeOptions(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.PaymentsGetPremiumGiftCodeOptionsTypeID, &tg.GenericVector{
		Items: []tg.TLObject{&tg.PremiumGiftCodeOption{Users: 1, Months: 3, Currency: "USD"}},
	})

	options, err := c.GetPremiumGiftCodeOptions(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetPremiumGiftCodeOptions() error: %v", err)
	}
	if len(options) != 1 || options[0].Months != 3 {
		t.Fatalf("options = %#v, want one 3-month package", options)
	}
}

func TestCheckGiftCode(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.PaymentsCheckGiftCodeTypeID, &tg.PaymentsCheckedGiftCode{
		UsedDate: 0,
		Days:     30,
	})

	checked, err := c.CheckGiftCode(context.Background(), "slug")
	if err != nil {
		t.Fatalf("CheckGiftCode() error: %v", err)
	}
	if checked.Days != 30 {
		t.Fatalf("checked = %#v, want 30 days", checked)
	}
}

func TestGetGiveawayInfo(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(-1000000000200, &tg.InputPeerChannel{ChannelID: 200, AccessHash: 7})
	inv.setResult(tg.PaymentsGetGiveawayInfoTypeID, &tg.PaymentsGiveawayInfoResults{
		WinnersCount: 10,
	})

	info, err := c.GetGiveawayInfo(context.Background(), -1000000000200, 55)
	if err != nil {
		t.Fatalf("GetGiveawayInfo() error: %v", err)
	}
	if res, ok := info.(*tg.PaymentsGiveawayInfoResults); !ok || res.WinnersCount != 10 {
		t.Fatalf("info = %#v, want results", info)
	}
}

func TestGetPaymentReceipt(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.PaymentsGetPaymentReceiptTypeID, &tg.PaymentsPaymentReceiptStars{
		Currency: "XTR",
	})

	receipt, err := c.GetPaymentReceipt(context.Background(), 10, 55)
	if err != nil {
		t.Fatalf("GetPaymentReceipt() error: %v", err)
	}
	if r, ok := receipt.(*tg.PaymentsPaymentReceiptStars); !ok || r.Currency != "XTR" {
		t.Fatalf("receipt = %#v, want stars receipt", receipt)
	}
}

func TestValidateRequestedInfo(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.PaymentsValidateRequestedInfoTypeID, &tg.PaymentsValidatedRequestedInfo{
		ID: "req1",
	})

	info := &tg.PaymentRequestedInfo{Name: "Alice"}
	validated, err := c.ValidateRequestedInfo(context.Background(), &tg.InputInvoiceMessage{}, info, true)
	if err != nil {
		t.Fatalf("ValidateRequestedInfo() error: %v", err)
	}
	if validated.ID != "req1" {
		t.Fatalf("validated = %#v", validated)
	}
	req, ok := inv.lastCall().(*tg.PaymentsValidateRequestedInfoRequest)
	if !ok || !req.Save {
		t.Fatalf("request = %#v, want save set", inv.lastCall())
	}
}

func TestSavedPaymentInfo(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.PaymentsGetSavedInfoTypeID, &tg.PaymentsSavedInfo{})

	if _, err := c.GetSavedPaymentInfo(context.Background()); err != nil {
		t.Fatalf("GetSavedPaymentInfo() error: %v", err)
	}
}
