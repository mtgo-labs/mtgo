package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSetBotCommands(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	commands := []*tg.BotCommand{
		{Command: "start", Description: "Start the bot"},
		{Command: "help", Description: "Show help"},
	}
	err := c.SetBotCommands(context.Background(), &tg.BotCommandScopeDefault{}, "en", commands)
	if err != nil {
		t.Fatalf("SetBotCommands() error: %v", err)
	}
	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}
	req, ok := mock.lastCall().(*tg.BotsSetBotCommandsRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotCommandsRequest, got %T", mock.lastCall())
	}
	if len(req.Commands) != 2 {
		t.Errorf("Commands length = %d, want 2", len(req.Commands))
	}
	if req.LangCode != "en" {
		t.Errorf("LangCode = %q, want %q", req.LangCode, "en")
	}
	_, ok = req.Scope.(*tg.BotCommandScopeDefault)
	if !ok {
		t.Errorf("Scope = %T, want BotCommandScopeDefault", req.Scope)
	}
}

func TestGetBotCommands(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	desc := "test description"
	mock.setResult(tg.BotsGetBotInfoTypeID, &tg.BotInfo{
		Flags:       0x06,
		Description: desc,
		Commands: []*tg.BotCommand{
			{Command: "start", Description: "Start"},
		},
	})

	cmds, err := c.GetBotCommands(context.Background(), "en")
	if err != nil {
		t.Fatalf("GetBotCommands() error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].Command != "start" {
		t.Errorf("Command = %q, want %q", cmds[0].Command, "start")
	}
}

func TestDeleteBotCommands(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.DeleteBotCommands(context.Background(), &tg.BotCommandScopeDefault{}, "en")
	if err != nil {
		t.Fatalf("DeleteBotCommands() error: %v", err)
	}
	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}
	req, ok := mock.lastCall().(*tg.BotsResetBotCommandsRequest)
	if !ok {
		t.Fatalf("expected BotsResetBotCommandsRequest, got %T", mock.lastCall())
	}
	if req.LangCode != "en" {
		t.Errorf("LangCode = %q, want %q", req.LangCode, "en")
	}
}

func TestSetBotCommands_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setError(tg.BotsSetBotCommandsTypeID, errors.New("rpc error"))

	err := c.SetBotCommands(context.Background(), &tg.BotCommandScopeDefault{}, "en", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "rpc error" {
		t.Errorf("error = %q, want %q", err.Error(), "rpc error")
	}
}
