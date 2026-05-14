package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// SetBotInfoDescription sets the bot's long description text for the specified language
// code. This description is shown on the bot's profile page. Returns an error if the
// RPC call fails.
//
// Example:
//
//	ctx := context.Background()
//	err := client.SetBotInfoDescription(ctx, "en", "A helpful assistant bot.")
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) SetBotInfoDescription(ctx context.Context, langCode, description string) error {
	c.Log.Debug("SetBotInfoDescription")
	rpc := c.Raw()
	_, err := rpc.BotsSetBotInfo(ctx, &tg.BotsSetBotInfoRequest{
		LangCode:    langCode,
		Description: description,
	})
	return err
}

// GetBotInfoDescription retrieves the bot's long description text for the specified
// language code. Returns the description string, or an error if the RPC call fails or
// an unexpected response type is received.
//
// Example:
//
//	ctx := context.Background()
//	desc, err := client.GetBotInfoDescription(ctx, "en")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Bot description: %s\n", desc)
func (c *Client) GetBotInfoDescription(ctx context.Context, langCode string) (string, error) {
	c.Log.Debug("GetBotInfoDescription")
	rpc := c.Raw()
	result, err := rpc.BotsGetBotInfo(ctx, &tg.BotsGetBotInfoRequest{
		LangCode: langCode,
	})
	if err != nil {
		return "", fmt.Errorf("get bot info: %w", err)
	}
	switch v := result.(type) {
	case *tg.BotsBotInfo:
		return v.Description, nil
	case *tg.BotInfo:
		return v.Description, nil
	default:
		return "", fmt.Errorf("unexpected bot info type %T", result)
	}
}

// SetBotInfoShortDescription sets the bot's short "about" text for the specified language
// code. This text appears below the bot's name in chat lists. Returns an error if the
// RPC call fails.
func (c *Client) SetBotInfoShortDescription(ctx context.Context, langCode, description string) error {
	c.Log.Debug("SetBotInfoShortDescription")
	rpc := c.Raw()
	_, err := rpc.BotsSetBotInfo(ctx, &tg.BotsSetBotInfoRequest{
		LangCode: langCode,
		About:    description,
	})
	return err
}

// GetBotInfoShortDescription retrieves the bot's short "about" text for the specified
// language code. Returns the short description string, or an error if the RPC call fails
// or an unexpected response type is received.
func (c *Client) GetBotInfoShortDescription(ctx context.Context, langCode string) (string, error) {
	c.Log.Debug("GetBotInfoShortDescription")
	rpc := c.Raw()
	result, err := rpc.BotsGetBotInfo(ctx, &tg.BotsGetBotInfoRequest{
		LangCode: langCode,
	})
	if err != nil {
		return "", fmt.Errorf("get bot info: %w", err)
	}
	switch v := result.(type) {
	case *tg.BotsBotInfo:
		return v.About, nil
	case *tg.BotInfo:
		return v.Description, nil
	default:
		return "", fmt.Errorf("unexpected bot info type %T", result)
	}
}

// SetBotName sets the bot's display name for the specified language code. Returns an
// error if the RPC call fails.
func (c *Client) SetBotName(ctx context.Context, langCode, name string) error {
	c.Log.Debug("SetBotName")
	rpc := c.Raw()
	_, err := rpc.BotsSetBotInfo(ctx, &tg.BotsSetBotInfoRequest{
		LangCode: langCode,
		Name:     name,
	})
	return err
}

// GetBotName retrieves the bot's display name for the specified language code. Returns
// the name string, or an error if the RPC call fails or an unexpected response type is
// received.
func (c *Client) GetBotName(ctx context.Context, langCode string) (string, error) {
	c.Log.Debug("GetBotName")
	rpc := c.Raw()
	result, err := rpc.BotsGetBotInfo(ctx, &tg.BotsGetBotInfoRequest{
		LangCode: langCode,
	})
	if err != nil {
		return "", fmt.Errorf("get bot info: %w", err)
	}
	switch v := result.(type) {
	case *tg.BotsBotInfo:
		return v.Name, nil
	case *tg.BotInfo:
		return "", nil
	default:
		return "", fmt.Errorf("unexpected bot info type %T", result)
	}
}
