package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSetBotInfoDescription(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.SetBotInfoDescription(context.Background(), "en", "My bot description")
	if err != nil {
		t.Fatalf("SetBotInfoDescription() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.BotsSetBotInfoRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotInfoRequest, got %T", mock.lastCall())
	}
	if req.Description != "My bot description" {
		t.Errorf("Description = %v, want %q", req.Description, "My bot description")
	}
	if req.LangCode != "en" {
		t.Errorf("LangCode = %q, want %q", req.LangCode, "en")
	}
}

func TestGetBotInfoDescription(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setResult(tg.BotsGetBotInfoTypeID, &tg.BotsBotInfo{
		Name:        "TestBot",
		About:       "Short desc",
		Description: "Full description",
	})

	desc, err := c.GetBotInfoDescription(context.Background(), "en")
	if err != nil {
		t.Fatalf("GetBotInfoDescription() error: %v", err)
	}
	if desc != "Full description" {
		t.Errorf("description = %q, want %q", desc, "Full description")
	}
}

func TestSetBotInfoShortDescription(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.SetBotInfoShortDescription(context.Background(), "en", "Short about text")
	if err != nil {
		t.Fatalf("SetBotInfoShortDescription() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.BotsSetBotInfoRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotInfoRequest, got %T", mock.lastCall())
	}
	if req.About != "Short about text" {
		t.Errorf("About = %v, want %q", req.About, "Short about text")
	}
}

func TestGetBotInfoShortDescription(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setResult(tg.BotsGetBotInfoTypeID, &tg.BotsBotInfo{
		Name:        "TestBot",
		About:       "Short about",
		Description: "Full desc",
	})

	about, err := c.GetBotInfoShortDescription(context.Background(), "en")
	if err != nil {
		t.Fatalf("GetBotInfoShortDescription() error: %v", err)
	}
	if about != "Short about" {
		t.Errorf("about = %q, want %q", about, "Short about")
	}
}

func TestSetBotName(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.SetBotName(context.Background(), "en", "NewBotName")
	if err != nil {
		t.Fatalf("SetBotName() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.BotsSetBotInfoRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotInfoRequest, got %T", mock.lastCall())
	}
	if req.Name != "NewBotName" {
		t.Errorf("Name = %v, want %q", req.Name, "NewBotName")
	}
}

func TestGetBotName(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setResult(tg.BotsGetBotInfoTypeID, &tg.BotsBotInfo{
		Name:        "TestBot",
		About:       "About",
		Description: "Desc",
	})

	name, err := c.GetBotName(context.Background(), "en")
	if err != nil {
		t.Fatalf("GetBotName() error: %v", err)
	}
	if name != "TestBot" {
		t.Errorf("name = %q, want %q", name, "TestBot")
	}
}

func TestBotInfo_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setError(tg.BotsSetBotInfoTypeID, errors.New("rpc error"))

	err := c.SetBotInfoDescription(context.Background(), "en", "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "rpc error" {
		t.Errorf("error = %q, want %q", err.Error(), "rpc error")
	}
}
