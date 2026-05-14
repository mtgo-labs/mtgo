package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// SetBotCommands sets the list of bot commands for the given scope and language code.
// The scope determines where the commands appear (e.g. all private chats, specific
// chat). Returns an error if the RPC call fails.
//
// Example:
//
//	ctx := context.Background()
//	commands := []*tg.BotCommand{
//	    {Command: "start", Description: "Start the bot"},
//	    {Command: "help", Description: "Show help"},
//	}
//	err := client.SetBotCommands(ctx, &tg.BotCommandScopeUsers{}, "", commands)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) SetBotCommands(ctx context.Context, scope tg.BotCommandScopeClass, langCode string, commands []*tg.BotCommand) error {
	c.Log.Debugf("SetBotCommands count=%d", len(commands))
	rpc := c.Raw()
	_, err := rpc.BotsSetBotCommands(ctx, &tg.BotsSetBotCommandsRequest{
		Scope:    scope,
		LangCode: langCode,
		Commands: commands,
	})
	return err
}

// GetBotCommands retrieves the current list of bot commands for the specified language
// code. Returns nil if no commands are found for the given language. Returns an error
// if the RPC call fails or an unexpected response type is received.
//
// Example:
//
//	ctx := context.Background()
//	commands, err := client.GetBotCommands(ctx, "")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, cmd := range commands {
//	    fmt.Printf("/%s — %s\n", cmd.Command, cmd.Description)
//	}
func (c *Client) GetBotCommands(ctx context.Context, langCode string) ([]*tg.BotCommand, error) {
	c.Log.Debug("GetBotCommands")
	rpc := c.Raw()
	result, err := rpc.BotsGetBotInfo(ctx, &tg.BotsGetBotInfoRequest{
		LangCode: langCode,
	})
	if err != nil {
		return nil, fmt.Errorf("get bot info: %w", err)
	}
	switch v := result.(type) {
	case *tg.BotInfo:
		return v.Commands, nil
	case *tg.BotsBotInfo:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected bot info type %T", result)
	}
}

// DeleteBotCommands resets (deletes) all bot commands for the given scope and language
// code. Returns an error if the RPC call fails.
func (c *Client) DeleteBotCommands(ctx context.Context, scope tg.BotCommandScopeClass, langCode string) error {
	c.Log.Debug("DeleteBotCommands")
	rpc := c.Raw()
	_, err := rpc.BotsResetBotCommands(ctx, &tg.BotsResetBotCommandsRequest{
		Scope:    scope,
		LangCode: langCode,
	})
	return err
}
