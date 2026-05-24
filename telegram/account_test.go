package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSetPrivacy_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.SetPrivacy(context.Background(), &tg.InputPrivacyKeyStatusTimestamp{}, nil)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestGetPrivacy_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.GetPrivacy(context.Background(), &tg.InputPrivacyKeyStatusTimestamp{})
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSetGlobalPrivacySettings_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.SetGlobalPrivacySettings(context.Background(), &tg.GlobalPrivacySettings{})
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestGetGlobalPrivacySettings_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.GetGlobalPrivacySettings(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSetGlobalPrivacySettings_NilSettings(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	c.state.setConnected(true)
	err := c.SetGlobalPrivacySettings(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil settings")
	}
}

func TestSetAccountTTL_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.SetAccountTTL(context.Background(), 180)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestGetAccountTTL_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.GetAccountTTL(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSetPrivacy_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountSetPrivacyTypeID, &tg.AccountPrivacyRules{
		Rules: []tg.PrivacyRuleClass{&tg.PrivacyValueAllowAll{}},
	})

	err := c.SetPrivacy(context.Background(), &tg.InputPrivacyKeyStatusTimestamp{},
		[]tg.InputPrivacyRuleClass{&tg.InputPrivacyValueAllowAll{}})
	if err != nil {
		t.Fatalf("SetPrivacy() error: %v", err)
	}

	req, ok := mock.lastCall().(*tg.AccountSetPrivacyRequest)
	if !ok {
		t.Fatalf("expected AccountSetPrivacyRequest, got %T", mock.lastCall())
	}
	if len(req.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(req.Rules))
	}
}

func TestSetPrivacy_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AccountSetPrivacyTypeID, errors.New("rpc error"))

	err := c.SetPrivacy(context.Background(), &tg.InputPrivacyKeyStatusTimestamp{}, nil)
	if err == nil {
		t.Fatal("expected error from RPC")
	}
}

func TestGetPrivacy_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountGetPrivacyTypeID, &tg.AccountPrivacyRules{
		Rules: []tg.PrivacyRuleClass{
			&tg.PrivacyValueAllowAll{},
			&tg.PrivacyValueDisallowContacts{},
		},
	})

	rules, err := c.GetPrivacy(context.Background(), &tg.InputPrivacyKeyStatusTimestamp{})
	if err != nil {
		t.Fatalf("GetPrivacy() error: %v", err)
	}
	if len(rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules))
	}
}

func TestGetPrivacy_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.AccountGetPrivacyTypeID, errors.New("rpc error"))

	_, err := c.GetPrivacy(context.Background(), &tg.InputPrivacyKeyStatusTimestamp{})
	if err == nil {
		t.Fatal("expected error from RPC")
	}
}

func TestSetGlobalPrivacySettings_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	settings := &tg.GlobalPrivacySettings{ArchiveAndMuteNewNoncontactPeers: true}
	mock.setResult(tg.AccountSetGlobalPrivacySettingsTypeID, &tg.GlobalPrivacySettings{})

	err := c.SetGlobalPrivacySettings(context.Background(), settings)
	if err != nil {
		t.Fatalf("SetGlobalPrivacySettings() error: %v", err)
	}

	req, ok := mock.lastCall().(*tg.AccountSetGlobalPrivacySettingsRequest)
	if !ok {
		t.Fatalf("expected AccountSetGlobalPrivacySettingsRequest, got %T", mock.lastCall())
	}
	if req.Settings != settings {
		t.Error("expected settings to be passed through")
	}
}

func TestGetGlobalPrivacySettings_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	expected := &tg.GlobalPrivacySettings{ArchiveAndMuteNewNoncontactPeers: true}
	mock.setResult(tg.AccountGetGlobalPrivacySettingsTypeID, expected)

	result, err := c.GetGlobalPrivacySettings(context.Background())
	if err != nil {
		t.Fatalf("GetGlobalPrivacySettings() error: %v", err)
	}
	if !result.ArchiveAndMuteNewNoncontactPeers {
		t.Error("expected ArchiveAndMuteNewNoncontactPeers to be true")
	}
}

func TestSetAccountTTL_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountSetAccountTTLTypeID, &tg.BoolTrue{})

	err := c.SetAccountTTL(context.Background(), 365)
	if err != nil {
		t.Fatalf("SetAccountTTL() error: %v", err)
	}

	req, ok := mock.lastCall().(*tg.AccountSetAccountTTLRequest)
	if !ok {
		t.Fatalf("expected AccountSetAccountTTLRequest, got %T", mock.lastCall())
	}
	if req.TTL == nil {
		t.Fatal("TTL should not be nil")
	}
	if req.TTL.Days != 365 {
		t.Errorf("TTL.Days = %d, want 365", req.TTL.Days)
	}
}

func TestGetAccountTTL_Success(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.AccountGetAccountTTLTypeID, &tg.AccountDaysTTL{Days: 180})

	days, err := c.GetAccountTTL(context.Background())
	if err != nil {
		t.Fatalf("GetAccountTTL() error: %v", err)
	}
	if days != 180 {
		t.Errorf("days = %d, want 180", days)
	}
}
