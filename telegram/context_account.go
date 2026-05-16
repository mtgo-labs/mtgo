package telegram

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// GetBusinessConnection retrieves a business bot connection by its identifier. Use this
// to inspect the state of a business connection established through the Telegram Business
// feature when handling OnBusinessConnection or OnBusinessMessage updates.
//
// Parameters:
//   - connectionID: unique identifier of the business connection to retrieve
//
// Returns:
//   - *tg.BotBusinessConnection: the business connection details
//   - error: non-nil if the context has no client or the connection cannot be found
func (c *Context) GetBusinessConnection(connectionID string) (*tg.BotBusinessConnection, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.GetBusinessConnection(c.Ctx, connectionID)
}
