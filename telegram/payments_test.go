package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestGetPaymentForm_PeerResolveError(t *testing.T) {
	c := initClient(&Client{testResolver: &mockPeerResolver{}})
	_, err := c.GetPaymentForm(context.TODO(), 999, 1, nil)
	if err == nil {
		t.Fatal("expected error for unresolvable peer")
	}
}

func TestSendPaymentForm_NilCredentials(t *testing.T) {
	c := initClient(&Client{testResolver: &mockPeerResolver{}})
	_, err := c.SendPaymentForm(context.TODO(), 1, 0, 1, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil credentials")
	}
}

func TestSendPaymentForm_PeerResolveError(t *testing.T) {
	c := initClient(&Client{testResolver: &mockPeerResolver{}})
	creds := &tg.InputPaymentCredentials{Data: &tg.DataJSON{Data: "{}"}}
	_, err := c.SendPaymentForm(context.TODO(), 1, 999, 1, creds, nil)
	if err == nil {
		t.Fatal("expected error for unresolvable peer")
	}
}

func TestGetPaymentForm_ThemeOption(t *testing.T) {
	theme := `{"bg_color":"#fff"}`
	cfg := &GetPaymentFormOption{ThemeParams: &theme}
	if cfg.ThemeParams == nil {
		t.Error("expected non-nil ThemeParams")
	}
}

func TestSendPaymentForm_TipOption(t *testing.T) {
	tip := int64(100)
	cfg := &SendPaymentFormOption{TipAmount: &tip}
	if cfg.TipAmount == nil || *cfg.TipAmount != 100 {
		t.Error("expected TipAmount=100")
	}
}

func TestGetStarsBalance_PeerResolveError(t *testing.T) {
	c := initClient(&Client{testResolver: &mockPeerResolver{}})
	_, err := c.GetStarsBalance(context.TODO(), 999)
	if err == nil {
		t.Fatal("expected error for unresolvable peer")
	}
}

func TestSendGift_PeerResolveError(t *testing.T) {
	c := initClient(&Client{testResolver: &mockPeerResolver{}})
	_, err := c.SendGift(context.TODO(), 999, 1, "hello")
	if err == nil {
		t.Fatal("expected error for unresolvable user")
	}
}

func TestPayMessageFlow(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})

	inv.setResult(tg.PaymentsGetPaymentFormTypeID, &tg.PaymentsPaymentForm{
		FormID:     77,
		BotID:      99,
		Title:      "Widget Pro",
		ProviderID: 42,
	})
	inv.setResult(tg.PaymentsValidateRequestedInfoTypeID, &tg.PaymentsValidatedRequestedInfo{
		ID: "info-1",
		ShippingOptions: []*tg.ShippingOption{{
			ID:    "express",
			Title: "Express",
		}},
	})
	inv.setResult(tg.PaymentsSendPaymentFormTypeID, &tg.PaymentsPaymentResult{
		Updates: &tg.Updates{
			Updates: []tg.UpdateClass{&tg.UpdateNewMessage{
				Message: &tg.Message{ID: 500, PeerID: &tg.PeerUser{UserID: 10}},
			}},
			Users: []tg.UserClass{},
			Chats: []tg.ChatClass{},
		},
	})

	receipt, err := c.PayMessage(t.Context(), 10, 55, &params.Pay{
		RequestedInfo:    &tg.PaymentRequestedInfo{Name: "Sadiq"},
		SaveInfo:         true,
		ShippingOptionID: "express",
		CardData:         &tg.DataJSON{Data: `{"type":"card"}`},
	})
	if err != nil {
		t.Fatalf("PayMessage: %v", err)
	}
	if receipt == nil || receipt.ID != 500 {
		t.Fatalf("receipt = %+v", receipt)
	}
	req, ok := inv.lastCall().(*tg.PaymentsSendPaymentFormRequest)
	if !ok {
		t.Fatalf("last call = %T", inv.lastCall())
	}
	if req.FormID != 77 || req.RequestedInfoID != "info-1" || req.ShippingOptionID != "express" {
		t.Fatalf("form request = %#v", req)
	}
	if creds, ok := req.Credentials.(*tg.InputPaymentCredentials); !ok || creds.Data.Data != `{"type":"card"}` {
		t.Fatalf("credentials = %#v", req.Credentials)
	}
	if invoice, ok := req.Invoice.(*tg.InputInvoiceMessage); !ok || invoice.MsgID != 55 {
		t.Fatalf("invoice = %#v", req.Invoice)
	}
}

func TestPayMessageVerificationNeeded(t *testing.T) {
	c, inv := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.PaymentsGetPaymentFormTypeID, &tg.PaymentsPaymentForm{FormID: 1})
	inv.setResult(tg.PaymentsSendPaymentFormTypeID, &tg.PaymentsPaymentVerificationNeeded{URL: "https://verify.example"})

	_, err := c.PayMessage(t.Context(), 10, 55)
	var verify *PaymentVerificationNeededError
	if !errors.As(err, &verify) {
		t.Fatalf("err = %v, want PaymentVerificationNeededError", err)
	}
	if verify.URL != "https://verify.example" {
		t.Fatalf("url = %q", verify.URL)
	}
}
