package telegram

import (
	"context"
	"errors"
	"testing"
)

func TestGetQRCodeLoginToken_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.GetQRCodeLoginToken(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestCheckQRCodeLoginToken_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.CheckQRCodeLoginToken(context.Background(), []byte("token"))
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}
