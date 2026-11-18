package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestResendPhoneNumberCode(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AuthResendCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "hash2",
		Type:          &tg.AuthSentCodeTypeCall{},
	})

	result, err := c.ResendPhoneNumberCode(context.Background(), "+1234567890", "hash1", "user request")
	if err != nil {
		t.Fatalf("ResendPhoneNumberCode() error: %v", err)
	}
	if result.PhoneCodeHash != "hash2" {
		t.Fatalf("result = %#v, want hash2", result)
	}
	req, ok := inv.lastCall().(*tg.AuthResendCodeRequest)
	if !ok {
		t.Fatalf("expected AuthResendCodeRequest, got %T", inv.lastCall())
	}
	if req.PhoneNumber != "+1234567890" || req.PhoneCodeHash != "hash1" || req.Reason != "user request" {
		t.Errorf("request = %#v, want resend for hash1", req)
	}
}

func TestSendChangePhoneNumberCode(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountSendChangePhoneCodeTypeID, &tg.AuthSentCode{
		PhoneCodeHash: "chash",
		Type:          &tg.AuthSentCodeTypeSms{},
	})

	result, err := c.SendChangePhoneNumberCode(context.Background(), "+15550001111")
	if err != nil {
		t.Fatalf("SendChangePhoneNumberCode() error: %v", err)
	}
	if result.PhoneCodeHash != "chash" {
		t.Fatalf("result = %#v, want chash", result)
	}
	req, ok := inv.lastCall().(*tg.AccountSendChangePhoneCodeRequest)
	if !ok {
		t.Fatalf("expected AccountSendChangePhoneCodeRequest, got %T", inv.lastCall())
	}
	if req.PhoneNumber != "+15550001111" || req.Settings == nil {
		t.Errorf("request = %#v, want the new number with default settings", req)
	}
}

func TestChangePhoneNumber(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountChangePhoneTypeID, &tg.User{ID: 5, FirstName: "New", Username: "u"})

	user, err := c.ChangePhoneNumber(context.Background(), "+15550001111", "chash", "54321")
	if err != nil {
		t.Fatalf("ChangePhoneNumber() error: %v", err)
	}
	if user == nil || user.ID != 5 || user.FirstName != "New" {
		t.Fatalf("user = %#v, want parsed user 5", user)
	}
	req, ok := inv.lastCall().(*tg.AccountChangePhoneRequest)
	if !ok {
		t.Fatalf("expected AccountChangePhoneRequest, got %T", inv.lastCall())
	}
	if req.PhoneCode != "54321" || req.PhoneCodeHash != "chash" {
		t.Errorf("request = %#v, want code 54321 for chash", req)
	}
}

func TestResetCloudPassword_Scheduled(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountResetPasswordTypeID, &tg.AccountResetPasswordRequestedWait{UntilDate: 1700000000})

	result, err := c.ResetCloudPassword(context.Background())
	if err != nil {
		t.Fatalf("ResetCloudPassword() error: %v", err)
	}
	if result.OK || result.UntilDate != 1700000000 || result.RetryDate != 0 {
		t.Fatalf("result = %#v, want scheduled reset at 1700000000", result)
	}
}

func TestResetCloudPassword_Immediate(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.AccountResetPasswordTypeID, &tg.AccountResetPasswordOk{})

	result, err := c.ResetCloudPassword(context.Background())
	if err != nil {
		t.Fatalf("ResetCloudPassword() error: %v", err)
	}
	if !result.OK {
		t.Fatalf("result = %#v, want immediate OK", result)
	}
}
