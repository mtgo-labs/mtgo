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
