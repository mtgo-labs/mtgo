package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSetEmojiStatus(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountUpdateEmojiStatusTypeID, &tg.BoolTrue{})

	if err := c.SetEmojiStatus(context.Background(), 777, 0); err != nil {
		t.Fatalf("SetEmojiStatus() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.AccountUpdateEmojiStatusRequest)
	if !ok {
		t.Fatalf("expected AccountUpdateEmojiStatusRequest, got %T", inv.lastCall())
	}
	st, ok := req.EmojiStatus.(*tg.EmojiStatus)
	if !ok {
		t.Fatalf("status = %T, want *tg.EmojiStatus", req.EmojiStatus)
	}
	if st.DocumentID != 777 || st.Until != 0 {
		t.Errorf("status = %#v, want document 777 without expiry", st)
	}
}

func TestSetBirthday_SetsAndClears(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountUpdateBirthdayTypeID, &tg.BoolTrue{})

	if err := c.SetBirthday(context.Background(), 3, 14, 1990); err != nil {
		t.Fatalf("SetBirthday() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.AccountUpdateBirthdayRequest)
	if !ok {
		t.Fatalf("expected AccountUpdateBirthdayRequest, got %T", inv.lastCall())
	}
	if req.Birthday == nil || req.Birthday.Month != 3 || req.Birthday.Day != 14 || req.Birthday.Year != 1990 {
		t.Errorf("birthday = %#v, want March 14 1990", req.Birthday)
	}

	if err := c.SetBirthday(context.Background(), 0, 0, 0); err != nil {
		t.Fatalf("SetBirthday() clear error: %v", err)
	}
	req, ok = inv.lastCall().(*tg.AccountUpdateBirthdayRequest)
	if !ok {
		t.Fatalf("expected AccountUpdateBirthdayRequest, got %T", inv.lastCall())
	}
	if req.Birthday != nil {
		t.Errorf("birthday = %#v, want nil to clear", req.Birthday)
	}
}

func TestReorderActiveUsernames(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountReorderUsernamesTypeID, &tg.BoolTrue{})

	if err := c.ReorderActiveUsernames(context.Background(), []string{"durov", "durovs"}); err != nil {
		t.Fatalf("ReorderActiveUsernames() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.AccountReorderUsernamesRequest)
	if !ok {
		t.Fatalf("expected AccountReorderUsernamesRequest, got %T", inv.lastCall())
	}
	if len(req.Order) != 2 || req.Order[0] != "durov" {
		t.Errorf("order = %v, want [durov durovs]", req.Order)
	}
}

func TestToggleUsername(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountToggleUsernameTypeID, &tg.BoolTrue{})

	if err := c.ToggleUsername(context.Background(), "alt", false); err != nil {
		t.Fatalf("ToggleUsername() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.AccountToggleUsernameRequest)
	if !ok {
		t.Fatalf("expected AccountToggleUsernameRequest, got %T", inv.lastCall())
	}
	if req.Username != "alt" || req.Active {
		t.Errorf("request = %#v, want alt deactivated", req)
	}
}

func TestSetChatEmojiStatus(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(-1000000000200, &tg.InputPeerChannel{ChannelID: 200, AccessHash: 7})
	inv.setResult(tg.ChannelsUpdateEmojiStatusTypeID, &tg.Updates{})

	if err := c.SetChatEmojiStatus(context.Background(), -1000000000200, 555); err != nil {
		t.Fatalf("SetChatEmojiStatus() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.ChannelsUpdateEmojiStatusRequest)
	if !ok {
		t.Fatalf("expected ChannelsUpdateEmojiStatusRequest, got %T", inv.lastCall())
	}
	st, ok := req.EmojiStatus.(*tg.EmojiStatus)
	if !ok {
		t.Fatalf("status = %T, want *tg.EmojiStatus", req.EmojiStatus)
	}
	if st.DocumentID != 555 {
		t.Errorf("status = %#v, want document 555", st)
	}
}

func TestGetTermsOfServiceUpdate(t *testing.T) {
	c, inv := newClientWithMock(t)
	tos := &tg.HelpTermsOfService{ID: &tg.DataJSON{Data: `{"id":2}`}, Text: "terms"}
	inv.setResult(tg.HelpGetTermsOfServiceUpdateTypeID, &tg.HelpTermsOfServiceUpdate{TermsOfService: tos})

	got, err := c.GetTermsOfServiceUpdate(context.Background())
	if err != nil {
		t.Fatalf("GetTermsOfServiceUpdate() error: %v", err)
	}
	if got != tos {
		t.Fatalf("terms = %#v, want the pending revision", got)
	}
}

func TestGetTermsOfServiceUpdate_Empty(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.HelpGetTermsOfServiceUpdateTypeID, &tg.HelpTermsOfServiceUpdateEmpty{})

	got, err := c.GetTermsOfServiceUpdate(context.Background())
	if err != nil {
		t.Fatalf("GetTermsOfServiceUpdate() error: %v", err)
	}
	if got != nil {
		t.Fatalf("terms = %#v, want nil when up to date", got)
	}
}

func TestAcceptTermsOfService(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.HelpAcceptTermsOfServiceTypeID, &tg.BoolTrue{})

	tos := &tg.HelpTermsOfService{ID: &tg.DataJSON{Data: `{"id":3}`}}
	if err := c.AcceptTermsOfService(context.Background(), tos); err != nil {
		t.Fatalf("AcceptTermsOfService() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.HelpAcceptTermsOfServiceRequest)
	if !ok {
		t.Fatalf("expected HelpAcceptTermsOfServiceRequest, got %T", inv.lastCall())
	}
	if req.ID != tos.ID {
		t.Errorf("id = %#v, want the revision id", req.ID)
	}

	if err := c.AcceptTermsOfService(context.Background(), nil); err == nil {
		t.Error("expected error for nil terms")
	}
	if inv.callCount() != 1 {
		t.Errorf("RPC calls = %d, want 1 (nil guard makes no call)", inv.callCount())
	}
}
