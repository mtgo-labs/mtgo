package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Context) GetStarGiftCatalog(hash int32) (tg.StarGiftsClass, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.GetStarGiftCatalog(c.Ctx, hash)
}

func (c *Context) GetStarGiftOptions() ([]*types.Gift, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.GetStarGiftOptions(c.Ctx)
}

func (c *Context) ResolveGiftOffer(messageID int32, accept bool) (*types.Message, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.ResolveGiftOffer(c.Ctx, messageID, accept)
}

func (c *Context) GetStarGiftUpgradeOptions(giftID int64) (*tg.PaymentsStarGiftUpgradeAttributes, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.GetStarGiftUpgradeOptions(c.Ctx, giftID)
}

func (c *Context) GetSavedGifts(peerID int64, opts ...*params.GetGifts) (*tg.PaymentsSavedStarGifts, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.GetSavedGifts(c.Ctx, peerID, opts...)
}

func (c *Context) GetSavedGiftByID(gifts ...tg.InputSavedStarGiftClass) (*tg.PaymentsSavedStarGifts, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.GetSavedGiftByID(c.Ctx, gifts...)
}

func (c *Context) GetUniqueGift(slug string) (*tg.PaymentsUniqueStarGift, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.GetUniqueGift(c.Ctx, slug)
}

func (c *Context) GetGiftValue(slug string) (*tg.PaymentsUniqueStarGiftValueInfo, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.GetGiftValue(c.Ctx, slug)
}

func (c *Context) GetResaleGifts(giftID int64, opts ...*params.GetGifts) (*tg.PaymentsResaleStarGifts, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.GetResaleGifts(c.Ctx, giftID, opts...)
}

func (c *Context) TransferGift(gift tg.InputSavedStarGiftClass, recipientID int64) (*types.Message, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.TransferGift(c.Ctx, gift, recipientID)
}

func (c *Context) UpgradeGift(gift tg.InputSavedStarGiftClass, keepOriginalDetails bool) (*types.Message, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.UpgradeGift(c.Ctx, gift, keepOriginalDetails)
}

func (c *Context) PrepayGiftUpgrade(peerID int64, hash string) (*types.Message, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.PrepayGiftUpgrade(c.Ctx, peerID, hash)
}

func (c *Context) BuyResaleGift(slug string, recipientID int64, opts ...*params.BuyGift) (*types.Message, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.BuyResaleGift(c.Ctx, slug, recipientID, opts...)
}

func (c *Context) SetGiftResalePrice(gift tg.InputSavedStarGiftClass, stars int64) error {
	if c.Client == nil {
		return ErrContextNoClient
	}
	return c.Client.SetGiftResalePrice(c.Ctx, gift, stars)
}

func (c *Context) ShowGift(gift tg.InputSavedStarGiftClass) error {
	if c.Client == nil {
		return ErrContextNoClient
	}
	return c.Client.ShowGift(c.Ctx, gift)
}

func (c *Context) HideGift(gift tg.InputSavedStarGiftClass) error {
	if c.Client == nil {
		return ErrContextNoClient
	}
	return c.Client.HideGift(c.Ctx, gift)
}

func (c *Context) ConvertGift(gift tg.InputSavedStarGiftClass) error {
	if c.Client == nil {
		return ErrContextNoClient
	}
	return c.Client.ConvertGift(c.Ctx, gift)
}

func (c *Context) PinGifts(gifts []tg.InputSavedStarGiftClass, peerID int64) error {
	if c.Client == nil {
		return ErrContextNoClient
	}
	return c.Client.PinGifts(c.Ctx, gifts, peerID)
}

func (c *Context) SendGiftOffer(peerID int64, slug string, price int64, duration int32) (*types.Message, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.SendGiftOffer(c.Ctx, peerID, slug, price, duration)
}

func (c *Context) GetGiftWithdrawalURL(gift tg.InputSavedStarGiftClass, password string) (string, error) {
	if c.Client == nil {
		return "", ErrContextNoClient
	}
	return c.Client.GetGiftWithdrawalURL(c.Ctx, gift, password)
}
