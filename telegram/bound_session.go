package telegram

import (
	"context"

	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) BoundResetSession(hash int64) error {
	ctx := context.Background()
	_, err := c.Raw().AccountResetAuthorization(ctx, &tg.AccountResetAuthorizationRequest{
		Hash: hash,
	})
	return err
}
