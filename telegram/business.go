package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// GetBusinessConnection retrieves a business connection by its ID.
// It returns the BotBusinessConnection associated with the given connectionID,
// or an error if the connection ID is empty, the request fails, or no
// matching connection is found in the response updates.
//
// Example:
//
//	conn, err := client.GetBusinessConnection(ctx, "conn_12345")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Business connection: %+v\n", conn)
func (c *Client) GetBusinessConnection(ctx context.Context, connectionID string) (*tg.BotBusinessConnection, error) {
	if connectionID == "" {
		return nil, ErrBusinessConnIDRequired
	}

	c.Log.Debugf("GetBusinessConnection id=%s", connectionID)

	rpc := c.Raw()
	result, err := rpc.AccountGetBotBusinessConnection(ctx, &tg.AccountGetBotBusinessConnectionRequest{
		ConnectionID: connectionID,
	})
	if err != nil {
		return nil, err
	}

	switch v := result.(type) {
	case *tg.Updates:
		for _, u := range v.Updates {
			if upd, ok := u.(*tg.UpdateBotBusinessConnect); ok {
				return upd.Connection, nil
			}
		}
		return nil, ErrNoBusinessConnection
	default:
		return nil, fmt.Errorf("business: unexpected updates type %T", result)
	}
}
