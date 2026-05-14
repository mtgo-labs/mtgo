package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestGetPaymentForm_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.GetPaymentForm(context.TODO(), 999, 1, nil)
	if err == nil {
		t.Fatal("expected error for unresolvable peer")
	}
}

func TestSendPaymentForm_NilCredentials(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.SendPaymentForm(context.TODO(), 1, 0, 1, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil credentials")
	}
}

func TestSendPaymentForm_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
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
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.GetStarsBalance(context.TODO(), 999)
	if err == nil {
		t.Fatal("expected error for unresolvable peer")
	}
}

func TestSendGift_PeerResolveError(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	err := c.SendGift(context.TODO(), 999, 1, "hello")
	if err == nil {
		t.Fatal("expected error for unresolvable user")
	}
}
