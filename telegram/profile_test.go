package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSetProfilePhoto_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.SetProfilePhoto(context.Background(), &tg.InputFile{})
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSetUsername_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.SetUsername(context.Background(), "newuser")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestSetBio_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.SetBio(context.Background(), "new bio")
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestDeleteProfilePhoto_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	err := c.DeleteProfilePhoto(context.Background(), 123456)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestGetProfilePhotos_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.GetProfilePhotos(context.Background(), 0)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}
