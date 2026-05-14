package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// SetChatMenuButton sets the menu button for the specified user in the bot's chat.
// The userID identifies the target user, and button defines the new menu button configuration.
// Returns an error if the user cannot be resolved or the RPC call fails.
//
// Example:
//
//	ctx := context.Background()
//	err := client.SetChatMenuButton(ctx, userID, &tg.BotMenuButtonCommands{})
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) SetChatMenuButton(ctx context.Context, userID int64, button tg.BotMenuButtonClass) error {
	c.Log.Debugf("SetChatMenuButton user_id=%d", userID)
	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.BotsSetBotMenuButton(ctx, &tg.BotsSetBotMenuButtonRequest{
		UserID: user,
		Button: button,
	})
	return err
}

// GetChatMenuButton returns the current menu button configured for the specified user.
// Returns the BotMenuButtonClass representing the menu button, or an error if the user
// cannot be resolved or the RPC call fails.
//
// Example:
//
//	ctx := context.Background()
//	button, err := client.GetChatMenuButton(ctx, userID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Menu button type: %T\n", button)
func (c *Client) GetChatMenuButton(ctx context.Context, userID int64) (tg.BotMenuButtonClass, error) {
	c.Log.Debugf("GetChatMenuButton user_id=%d", userID)
	user, err := resolveUserID(c, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	rpc := c.Raw()
	return rpc.BotsGetBotMenuButton(ctx, &tg.BotsGetBotMenuButtonRequest{
		UserID: user,
	})
}
