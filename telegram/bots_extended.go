package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// SetBotUserMenuButton sets the menu button a specific user sees for the
// logged-in bot.
func (c *BotsMethods) SetBotUserMenuButton(ctx context.Context, userID int64, button tg.BotMenuButtonClass) error {
	c.Log.Debugf("SetBotUserMenuButton user_id=%d", userID)
	user, err := resolveUserID(ctx, c.Client, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}
	_, err = c.Raw().BotsSetBotMenuButton(ctx, &tg.BotsSetBotMenuButtonRequest{
		UserID: user,
		Button: button,
	})
	if err != nil {
		return fmt.Errorf("set bot user menu button: %w", err)
	}
	return nil
}

// GetBotUserMenuButton returns the menu button a specific user sees for the
// logged-in bot.
func (c *BotsMethods) GetBotUserMenuButton(ctx context.Context, userID int64) (tg.BotMenuButtonClass, error) {
	c.Log.Debugf("GetBotUserMenuButton user_id=%d", userID)
	user, err := resolveUserID(ctx, c.Client, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}
	result, err := c.Raw().BotsGetBotMenuButton(ctx, &tg.BotsGetBotMenuButtonRequest{
		UserID: user,
	})
	if err != nil {
		return nil, fmt.Errorf("get bot user menu button: %w", err)
	}
	return result, nil
}

// SetBotDefaultAdminRights sets the default administrator rights the logged-in
// bot requests when added to channels (forChannels true) or groups.
func (c *BotsMethods) SetBotDefaultAdminRights(ctx context.Context, forChannels bool, rights *tg.ChatAdminRights) error {
	c.Log.Debugf("SetBotDefaultAdminRights for_channels=%v", forChannels)
	if rights == nil {
		rights = &tg.ChatAdminRights{}
	}
	var err error
	if forChannels {
		_, err = c.Raw().BotsSetBotBroadcastDefaultAdminRights(ctx, &tg.BotsSetBotBroadcastDefaultAdminRightsRequest{
			AdminRights: rights,
		})
	} else {
		_, err = c.Raw().BotsSetBotGroupDefaultAdminRights(ctx, &tg.BotsSetBotGroupDefaultAdminRightsRequest{
			AdminRights: rights,
		})
	}
	if err != nil {
		return fmt.Errorf("set bot default admin rights: %w", err)
	}
	return nil
}

// CanMessageBot reports whether the logged-in user may message the given
// business bot.
func (c *BotsMethods) CanMessageBot(ctx context.Context, botID int64) (bool, error) {
	c.Log.Debugf("CanMessageBot bot_id=%d", botID)
	bot, err := resolveUserID(ctx, c.Client, botID)
	if err != nil {
		return false, fmt.Errorf("resolve bot: %w", err)
	}
	result, err := c.Raw().BotsCanSendMessage(ctx, &tg.BotsCanSendMessageRequest{
		Bot: bot,
	})
	if err != nil {
		return false, fmt.Errorf("can message bot: %w", err)
	}
	return result, nil
}

// AllowBotToMessage grants the given business bot permission to message the
// logged-in user.
func (c *BotsMethods) AllowBotToMessage(ctx context.Context, botID int64) error {
	c.Log.Debugf("AllowBotToMessage bot_id=%d", botID)
	bot, err := resolveUserID(ctx, c.Client, botID)
	if err != nil {
		return fmt.Errorf("resolve bot: %w", err)
	}
	_, err = c.Raw().BotsAllowSendMessage(ctx, &tg.BotsAllowSendMessageRequest{
		Bot: bot,
	})
	if err != nil {
		return fmt.Errorf("allow bot to message: %w", err)
	}
	return nil
}

// InvokeWebViewCustomMethod invokes a custom mini-app method on behalf of the
// logged-in mini app with the given bot. params is a JSON string; the raw JSON
// reply is returned as a string.
func (c *BotsMethods) InvokeWebViewCustomMethod(ctx context.Context, botID int64, method, params string) (string, error) {
	c.Log.Debugf("InvokeWebViewCustomMethod bot_id=%d method=%s", botID, method)
	bot, err := resolveUserID(ctx, c.Client, botID)
	if err != nil {
		return "", fmt.Errorf("resolve bot: %w", err)
	}
	result, err := c.Raw().BotsInvokeWebViewCustomMethod(ctx, &tg.BotsInvokeWebViewCustomMethodRequest{
		Bot:          bot,
		CustomMethod: method,
		Params:       &tg.DataJSON{Data: params},
	})
	if err != nil {
		return "", fmt.Errorf("invoke web view custom method: %w", err)
	}
	return result.Data, nil
}

// GetPopularAppBots searches popular mini-app bots by query. An empty offset
// starts from the beginning; the returned next offset fetches the next page.
func (c *BotsMethods) GetPopularAppBots(ctx context.Context, offset string, limit int) ([]*types.User, string, error) {
	c.Log.Debugf("GetPopularAppBots offset=%q limit=%d", offset, limit)
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	result, err := c.Raw().BotsGetPopularAppBots(ctx, &tg.BotsGetPopularAppBotsRequest{
		Offset: offset,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, "", fmt.Errorf("get popular app bots: %w", err)
	}
	bots := make([]*types.User, 0, len(result.Users))
	for _, u := range result.Users {
		if parsed := types.ParseUser(u); parsed != nil {
			bots = append(bots, parsed)
		}
	}
	return bots, result.NextOffset, nil
}

// ReorderBotUsernames changes the display order of an owned bot's active
// usernames.
func (c *BotsMethods) ReorderBotUsernames(ctx context.Context, botID int64, order []string) error {
	c.Log.Debugf("ReorderBotUsernames bot_id=%d n=%d", botID, len(order))
	bot, err := resolveUserID(ctx, c.Client, botID)
	if err != nil {
		return fmt.Errorf("resolve bot: %w", err)
	}
	_, err = c.Raw().BotsReorderUsernames(ctx, &tg.BotsReorderUsernamesRequest{
		Bot:   bot,
		Order: order,
	})
	if err != nil {
		return fmt.Errorf("reorder bot usernames: %w", err)
	}
	return nil
}

// ToggleBotUsername activates or deactivates one of an owned bot's usernames.
func (c *BotsMethods) ToggleBotUsername(ctx context.Context, botID int64, username string, active bool) error {
	c.Log.Debugf("ToggleBotUsername bot_id=%d username=%s active=%v", botID, username, active)
	bot, err := resolveUserID(ctx, c.Client, botID)
	if err != nil {
		return fmt.Errorf("resolve bot: %w", err)
	}
	_, err = c.Raw().BotsToggleUsername(ctx, &tg.BotsToggleUsernameRequest{
		Bot:      bot,
		Username: username,
		Active:   active,
	})
	if err != nil {
		return fmt.Errorf("toggle bot username: %w", err)
	}
	return nil
}

// GetBotRecommendations returns bots recommended alongside the given bot.
func (c *BotsMethods) GetBotRecommendations(ctx context.Context, botID int64) ([]*types.User, error) {
	c.Log.Debugf("GetBotRecommendations bot_id=%d", botID)
	bot, err := resolveUserID(ctx, c.Client, botID)
	if err != nil {
		return nil, fmt.Errorf("resolve bot: %w", err)
	}
	result, err := c.Raw().BotsGetBotRecommendations(ctx, &tg.BotsGetBotRecommendationsRequest{
		Bot: bot,
	})
	if err != nil {
		return nil, fmt.Errorf("get bot recommendations: %w", err)
	}
	var raw []tg.UserClass
	switch v := result.(type) {
	case *tg.UsersUsers:
		raw = v.Users
	case *tg.UsersUsersSlice:
		raw = v.Users
	default:
		return nil, fmt.Errorf("get bot recommendations: unexpected result %T", result)
	}
	users := make([]*types.User, 0, len(raw))
	for _, u := range raw {
		if parsed := types.ParseUser(u); parsed != nil {
			users = append(users, parsed)
		}
	}
	return users, nil
}

// GetOwnedBots lists the bots created by the logged-in user.
func (c *BotsMethods) GetOwnedBots(ctx context.Context) ([]*types.User, error) {
	c.Log.Debug("GetOwnedBots")
	raw, err := c.Raw().BotsGetAdminedBots(ctx)
	if err != nil {
		return nil, fmt.Errorf("get owned bots: %w", err)
	}
	vector, ok := raw.(*tg.GenericVector)
	if !ok {
		return nil, fmt.Errorf("get owned bots: unexpected result %T", raw)
	}
	bots := make([]*types.User, 0, len(vector.Items))
	for _, item := range vector.Items {
		if u, ok := item.(tg.UserClass); ok {
			if parsed := types.ParseUser(u); parsed != nil {
				bots = append(bots, parsed)
			}
		}
	}
	return bots, nil
}
