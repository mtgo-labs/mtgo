package telegram

import (
	"context"

	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) BoundShowGift(msgID int32) error {
	return c.ShowGift(context.Background(), &tg.InputSavedStarGiftUser{MsgID: msgID})
}

func (c *Client) BoundHideGift(msgID int32) error {
	return c.HideGift(context.Background(), &tg.InputSavedStarGiftUser{MsgID: msgID})
}

func (c *Client) BoundConvertGift(msgID int32) error {
	return c.ConvertGift(context.Background(), &tg.InputSavedStarGiftUser{MsgID: msgID})
}

func (c *Client) BoundUpgradeGift(msgID int32, keepOriginalDetails bool) error {
	_, err := c.UpgradeGift(context.Background(), &tg.InputSavedStarGiftUser{MsgID: msgID}, keepOriginalDetails)
	return err
}

func (c *Client) BoundTransferGift(msgID int32, toID int64) error {
	_, err := c.TransferGift(context.Background(), &tg.InputSavedStarGiftUser{MsgID: msgID}, toID)
	return err
}
