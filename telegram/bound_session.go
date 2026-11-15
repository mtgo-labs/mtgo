package telegram

import (
	"context"

	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) BoundResetSession(ctx context.Context, hash int64) error {
	_, err := c.Raw().AccountResetAuthorization(ctx, &tg.AccountResetAuthorizationRequest{
		Hash: hash,
	})
	return err
}
