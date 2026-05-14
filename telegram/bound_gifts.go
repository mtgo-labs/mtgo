package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) BoundShowGift(msgID int32) error {
	ctx := context.Background()
	_, err := c.Raw().PaymentsSaveStarGift(ctx, &tg.PaymentsSaveStarGiftRequest{
		Unsave: false,
		Stargift: &tg.InputSavedStarGiftUser{
			MsgID: msgID,
		},
	})
	return err
}

func (c *Client) BoundHideGift(msgID int32) error {
	ctx := context.Background()
	_, err := c.Raw().PaymentsSaveStarGift(ctx, &tg.PaymentsSaveStarGiftRequest{
		Unsave: true,
		Stargift: &tg.InputSavedStarGiftUser{
			MsgID: msgID,
		},
	})
	return err
}

func (c *Client) BoundConvertGift(msgID int32) error {
	ctx := context.Background()
	_, err := c.Raw().PaymentsConvertStarGift(ctx, &tg.PaymentsConvertStarGiftRequest{
		Stargift: &tg.InputSavedStarGiftUser{
			MsgID: msgID,
		},
	})
	return err
}

func (c *Client) BoundUpgradeGift(msgID int32, keepOriginalDetails bool) error {
	ctx := context.Background()
	_, err := c.Raw().PaymentsUpgradeStarGift(ctx, &tg.PaymentsUpgradeStarGiftRequest{
		KeepOriginalDetails: keepOriginalDetails,
		Stargift: &tg.InputSavedStarGiftUser{
			MsgID: msgID,
		},
	})
	return err
}

func (c *Client) BoundTransferGift(msgID int32, toID int64) error {
	peer, err := resolvePeer(c, toID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	ctx := context.Background()
	_, err = c.Raw().PaymentsTransferStarGift(ctx, &tg.PaymentsTransferStarGiftRequest{
		Stargift: &tg.InputSavedStarGiftUser{
			MsgID: msgID,
		},
		ToID: peer,
	})
	return err
}

func (c *Client) BoundPinGifts(gifts []tg.InputSavedStarGiftClass) error {
	ctx := context.Background()
	peer, err := resolvePeer(c, c.Me().ID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	_, err = c.Raw().PaymentsToggleStarGiftsPinnedToTop(ctx, &tg.PaymentsToggleStarGiftsPinnedToTopRequest{
		Peer:     peer,
		Stargift: gifts,
	})
	return err
}

func (c *Client) BoundGetGifts(peerID int64, limit int32, opts ...*params.GetGifts) (*tg.PaymentsSavedStarGifts, error) {
	opt := params.GetOptDef(&params.GetGifts{Limit: limit}, opts...)
	ctx := context.Background()
	peer, err := resolvePeer(c, peerID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	req := &tg.PaymentsGetSavedStarGiftsRequest{
		ExcludeUnsaved:      opt.ExcludeUnsaved,
		ExcludeSaved:        opt.ExcludeSaved,
		ExcludeUnlimited:    opt.ExcludeUnlimited,
		ExcludeUnique:       opt.ExcludeUnique,
		SortByValue:         opt.SortByValue,
		ExcludeUpgradable:   opt.ExcludeUpgradable,
		ExcludeUnupgradable: opt.ExcludeUnupgradable,
		PeerColorAvailable:  opt.PeerColorAvailable,
		ExcludeHosted:       opt.ExcludeHosted,
		Peer:                peer,
		Offset:              opt.Offset,
		Limit:               opt.Limit,
	}
	if opt.CollectionID != 0 {
		req.CollectionID = opt.CollectionID
	}
	return c.Raw().PaymentsGetSavedStarGifts(ctx, req)
}

func (c *Client) BoundGetUniqueGift(slug string) (*tg.PaymentsUniqueStarGift, error) {
	ctx := context.Background()
	return c.Raw().PaymentsGetUniqueStarGift(ctx, &tg.PaymentsGetUniqueStarGiftRequest{
		Slug: slug,
	})
}

func (c *Client) BoundGetUniqueGiftValueInfo(slug string) (*tg.PaymentsUniqueStarGiftValueInfo, error) {
	ctx := context.Background()
	return c.Raw().PaymentsGetUniqueStarGiftValueInfo(ctx, &tg.PaymentsGetUniqueStarGiftValueInfoRequest{
		Slug: slug,
	})
}

func (c *Client) BoundGetGiftCatalog(hash int32) (tg.StarGiftsClass, error) {
	ctx := context.Background()
	return c.Raw().PaymentsGetStarGifts(ctx, &tg.PaymentsGetStarGiftsRequest{
		Hash: hash,
	})
}
