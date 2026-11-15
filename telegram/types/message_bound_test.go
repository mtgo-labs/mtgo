package types

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

func boundMessage(t *testing.T) (*Message, *mockBinder) {
	t.Helper()
	m := &Message{ID: 42, ChatID: 10}
	b := &mockBinder{}
	m.SetBinder(b)
	return m, b
}

func TestMessageEditLiveLocation(t *testing.T) {
	m, _ := boundMessage(t)
	updated, err := m.EditLiveLocationContext(context.Background(), 52.5, 13.4)
	if err != nil {
		t.Fatalf("EditLiveLocationContext() error: %v", err)
	}
	if updated == nil || updated.ID != 42 {
		t.Fatalf("updated = %#v", updated)
	}
}

func TestMessageStopLiveLocation(t *testing.T) {
	m, _ := boundMessage(t)
	if _, err := m.StopLiveLocation(); err != nil {
		t.Fatalf("StopLiveLocation() error: %v", err)
	}
}

func TestMessageClick(t *testing.T) {
	m, b := boundMessage(t)
	m.ReplyMarkup = &ReplyMarkup{
		Type: ReplyMarkupInline,
		Rows: [][]Button{
			{{Type: ButtonCallback, Text: "A", Data: []byte("cb0")}},
			{{Type: ButtonURL, Text: "Site", URL: "https://example"}, {Type: ButtonCallback, Text: "B", Data: []byte("cb1")}},
		},
	}

	answer, err := m.Click(2)
	if err != nil {
		t.Fatalf("Click() error: %v", err)
	}
	if answer == nil || answer.Message != "ok" {
		t.Fatalf("answer = %#v", answer)
	}
	if string(b.clickedData) != "cb1" {
		t.Errorf("clicked data = %q, want cb1", b.clickedData)
	}

	if _, err := m.Click(1); err == nil {
		t.Error("expected error clicking a URL button")
	}
	if _, err := m.Click(99); err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestMessageReplyInvoice(t *testing.T) {
	m, b := boundMessage(t)
	invoice := &tg.InputMediaInvoice{Title: "Pro"}
	msg, err := m.ReplyInvoice(invoice, "pay up")
	if err != nil {
		t.Fatalf("ReplyInvoice() error: %v", err)
	}
	if msg == nil || msg.ID != 1 {
		t.Fatalf("msg = %#v", msg)
	}
	if b.mediaReplyTo != 42 {
		t.Errorf("reply-to = %d, want 42", b.mediaReplyTo)
	}
}

func TestMessageReplyPaidMedia(t *testing.T) {
	m, b := boundMessage(t)
	media := &tg.InputMediaPaidMedia{StarsAmount: 25}
	msg, err := m.ReplyPaidMedia(media, "unlock", &params.SendMessage{})
	if err != nil {
		t.Fatalf("ReplyPaidMedia() error: %v", err)
	}
	if msg == nil || msg.ID != 100 {
		t.Fatalf("msg = %#v", msg)
	}
	if b.mediaReplyTo != 42 {
		t.Errorf("reply-to = %d, want 42", b.mediaReplyTo)
	}
}

func TestMessageGiftPurchaseOffer(t *testing.T) {
	m, b := boundMessage(t)
	if _, err := m.AcceptGiftPurchaseOffer(); err != nil {
		t.Fatalf("AcceptGiftPurchaseOffer() error: %v", err)
	}
	if b.giftOfferDeclined {
		t.Error("accept must not set decline")
	}
	if _, err := m.RejectGiftPurchaseOffer(); err != nil {
		t.Fatalf("RejectGiftPurchaseOffer() error: %v", err)
	}
	if !b.giftOfferDeclined {
		t.Error("reject must set decline")
	}
}
