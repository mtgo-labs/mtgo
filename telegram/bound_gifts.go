package telegram

import (
	"context"

	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) BoundShowGift(ctx context.Context, msgID int32) error {
	return c.ShowGift(ctx, &tg.InputSavedStarGiftUser{MsgID: msgID})
}

func (c *Client) BoundHideGift(ctx context.Context, msgID int32) error {
	return c.HideGift(ctx, &tg.InputSavedStarGiftUser{MsgID: msgID})
}

func (c *Client) BoundConvertGift(ctx context.Context, msgID int32) error {
	return c.ConvertGift(ctx, &tg.InputSavedStarGiftUser{MsgID: msgID})
}

func (c *Client) BoundUpgradeGift(ctx context.Context, msgID int32, keepOriginalDetails bool) error {
	_, err := c.UpgradeGift(ctx, &tg.InputSavedStarGiftUser{MsgID: msgID}, keepOriginalDetails)
	return err
}

func (c *Client) BoundTransferGift(ctx context.Context, msgID int32, toID int64) error {
	_, err := c.TransferGift(ctx, &tg.InputSavedStarGiftUser{MsgID: msgID}, toID)
	return err
}
