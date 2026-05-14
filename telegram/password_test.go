package telegram

import (
	"context"
	"errors"
	"testing"
)

func TestEnableCloudPassword_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.EnableCloudPassword(context.Background(), "pw", "hint")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestChangeCloudPassword_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.ChangeCloudPassword(context.Background(), "old", "new", "hint")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestRemoveCloudPassword_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.RemoveCloudPassword(context.Background(), "pw")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}
