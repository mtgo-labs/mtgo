package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSetChatMenuButton(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	err := c.SetChatMenuButton(context.Background(), 42, &tg.BotMenuButtonCommands{})
	if err != nil {
		t.Fatalf("SetChatMenuButton() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.BotsSetBotMenuButtonRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotMenuButtonRequest, got %T", mock.lastCall())
	}
	_, ok = req.Button.(*tg.BotMenuButtonCommands)
	if !ok {
		t.Errorf("Button = %T, want BotMenuButtonCommands", req.Button)
	}
}

func TestGetChatMenuButton(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	mock.setResult(tg.BotsGetBotMenuButtonTypeID, &tg.BotMenuButtonCommands{})

	button, err := c.GetChatMenuButton(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetChatMenuButton() error: %v", err)
	}
	if _, ok := button.(*tg.BotMenuButtonCommands); !ok {
		t.Errorf("Button = %T, want BotMenuButtonCommands", button)
	}
}

func TestSetChatMenuButton_DefaultButton(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	err := c.SetChatMenuButton(context.Background(), 42, &tg.BotMenuButtonDefault{})
	if err != nil {
		t.Fatalf("SetChatMenuButton() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.BotsSetBotMenuButtonRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotMenuButtonRequest, got %T", mock.lastCall())
	}
	_, ok = req.Button.(*tg.BotMenuButtonDefault)
	if !ok {
		t.Errorf("Button = %T, want BotMenuButtonDefault", req.Button)
	}
}

func TestSetChatMenuButton_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	mock.setError(tg.BotsSetBotMenuButtonTypeID, errors.New("rpc error"))

	err := c.SetChatMenuButton(context.Background(), 42, &tg.BotMenuButtonCommands{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetChatMenuButton_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	mock.setError(tg.BotsGetBotMenuButtonTypeID, errors.New("rpc error"))

	_, err := c.GetChatMenuButton(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
