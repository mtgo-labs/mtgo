package telegram

import (
	"context"
	"errors"
	"testing"
)

func TestSearchContacts_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.SearchContacts(context.Background(), "test", 10)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestGetUser_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.GetUser(context.Background(), 0)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}
