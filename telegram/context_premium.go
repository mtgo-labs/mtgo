package telegram

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// ApplyBoost applies one or more Premium boosts to the specified chat. Boosting a chat
// unlocks additional features for its members. Requires an active Telegram Premium subscription.
//
// Parameters:
//   - chatID: the chat ID to apply the boost to
//   - opts: optional [ApplyBoostOption] parameters for boost configuration
//
// Returns:
//   - []*tg.MyBoost: the list of boosts applied
//   - error: non-nil if the context has no client, the user has no Premium, or the boost fails
func (c *Context) ApplyBoost(chatID int64, opts ...*ApplyBoostOption) ([]*tg.MyBoost, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("context: no client")
	}
	return c.Client.ApplyBoost(c.Ctx, chatID, opts...)
}

// GetBoostsStatus retrieves the current boost status for the specified chat, including
// the number of boosts applied and the level reached.
//
// Parameters:
//   - chatID: the chat ID to check boost status for
//
// Returns:
//   - *tg.PremiumBoostsStatus: the boost status details
//   - error: non-nil if the context has no client or the request fails
func (c *Context) GetBoostsStatus(chatID int64) (*tg.PremiumBoostsStatus, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("context: no client")
	}
	return c.Client.GetBoostsStatus(c.Ctx, chatID)
}

// GetBoosts retrieves the list of boosts that the current user has applied across all chats.
//
// Parameters:
//   - opts: optional [GetBoostsOption] parameters for filtering and pagination
//
// Returns:
//   - []*tg.MyBoost: the list of the user's active boosts
//   - error: non-nil if the context has no client or the request fails
func (c *Context) GetBoosts(opts ...*GetBoostsOption) ([]*tg.MyBoost, error) {
	if c.Client == nil {
		return nil, fmt.Errorf("context: no client")
	}
	return c.Client.GetBoosts(c.Ctx, opts...)
}
