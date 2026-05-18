package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

func (c *Client) GetStarGiftCatalog(ctx context.Context, hash int32) (tg.StarGiftsClass, error) {
	c.Log.Debug("GetStarGiftCatalog")
	return c.Raw().PaymentsGetStarGifts(ctx, &tg.PaymentsGetStarGiftsRequest{Hash: hash})
}

func (c *Client) GetStarGiftOptions(ctx context.Context) ([]*types.Gift, error) {
	c.Log.Debug("GetStarGiftOptions")
	res, err := c.Raw().PaymentsGetStarGifts(ctx, &tg.PaymentsGetStarGiftsRequest{Hash: 0})
	if err != nil {
		return nil, err
	}
	starGifts, ok := res.(*tg.PaymentsStarGifts)
	if !ok {
		return nil, nil
	}
	out := make([]*types.Gift, 0, len(starGifts.Gifts))
	for _, g := range starGifts.Gifts {
		if v, ok := g.(*tg.StarGift); ok {
			if parsed := types.ParseGift(v); parsed != nil {
				out = append(out, parsed)
			}
		}
	}
	return out, nil
}

func (c *Client) ResolveGiftOffer(ctx context.Context, messageID int32, accept bool) (*types.Message, error) {
	c.Log.Debugf("ResolveGiftOffer msg_id=%d accept=%v", messageID, accept)
	result, err := c.Raw().PaymentsResolveStarGiftOffer(ctx, &tg.PaymentsResolveStarGiftOfferRequest{
		OfferMsgID: messageID,
		Decline:    !accept,
	})
	if err != nil {
		return nil, err
	}
	return extractSingleMessage(result, c)
}

func (c *Client) GetStarGiftUpgradeOptions(ctx context.Context, giftID int64) (*tg.PaymentsStarGiftUpgradeAttributes, error) {
	c.Log.Debugf("GetStarGiftUpgradeOptions gift_id=%d", giftID)
	return c.Raw().PaymentsGetStarGiftUpgradeAttributes(ctx, &tg.PaymentsGetStarGiftUpgradeAttributesRequest{
		GiftID: giftID,
	})
}

func (c *Client) GetSavedGifts(ctx context.Context, peerID int64, opts ...*params.GetGifts) (*tg.PaymentsSavedStarGifts, error) {
	c.Log.Debugf("GetSavedGifts peer_id=%d", peerID)
	opt := params.GetOptDef(&params.GetGifts{Limit: 100}, opts...)
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

func (c *Client) GetSavedGiftByID(ctx context.Context, gifts ...tg.InputSavedStarGiftClass) (*tg.PaymentsSavedStarGifts, error) {
	c.Log.Debug("GetSavedGiftByID")
	return c.Raw().PaymentsGetSavedStarGift(ctx, &tg.PaymentsGetSavedStarGiftRequest{
		Stargift: gifts,
	})
}

func (c *Client) GetUniqueGift(ctx context.Context, slug string) (*tg.PaymentsUniqueStarGift, error) {
	c.Log.Debugf("GetUniqueGift slug=%s", slug)
	return c.Raw().PaymentsGetUniqueStarGift(ctx, &tg.PaymentsGetUniqueStarGiftRequest{Slug: slug})
}

func (c *Client) GetGiftValue(ctx context.Context, slug string) (*tg.PaymentsUniqueStarGiftValueInfo, error) {
	c.Log.Debugf("GetGiftValue slug=%s", slug)
	return c.Raw().PaymentsGetUniqueStarGiftValueInfo(ctx, &tg.PaymentsGetUniqueStarGiftValueInfoRequest{Slug: slug})
}

func (c *Client) GetResaleGifts(ctx context.Context, giftID int64, opts ...*params.GetGifts) (*tg.PaymentsResaleStarGifts, error) {
	c.Log.Debugf("GetResaleGifts gift_id=%d", giftID)
	opt := params.GetOptDef(&params.GetGifts{Limit: 100}, opts...)
	req := &tg.PaymentsGetResaleStarGiftsRequest{
		GiftID: giftID,
		Offset: opt.Offset,
		Limit:  opt.Limit,
	}
	return c.Raw().PaymentsGetResaleStarGifts(ctx, req)
}

func (c *Client) SendGift(ctx context.Context, userID int64, giftID int64, message string, opts ...*params.GiftSend) (*types.Message, error) {
	c.Log.Debugf("SendGift user_id=%d gift_id=%d", userID, giftID)
	opt := params.GetOptDef(&params.GiftSend{}, opts...)
	peer, err := resolvePeer(c, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	invoice := &tg.InputInvoiceStarGift{
		Peer:           peer,
		GiftID:         giftID,
		HideName:       opt.IsPrivate,
		IncludeUpgrade: opt.PayForUpgrade,
	}
	if message != "" || opt.Text != "" {
		text := message
		if text == "" {
			text = opt.Text
		}
		invoice.Message = &tg.TextWithEntities{Text: text}
	}

	formID, err := c.getPaymentFormID(ctx, invoice)
	if err != nil {
		return nil, err
	}
	return c.payForUpdates(ctx, formID, invoice)
}

func (c *Client) TransferGift(ctx context.Context, gift tg.InputSavedStarGiftClass, recipientID int64) (*types.Message, error) {
	c.Log.Debugf("TransferGift recipient_id=%d", recipientID)
	peer, err := resolvePeer(c, recipientID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	invoice := &tg.InputInvoiceStarGiftTransfer{
		Stargift: gift,
		ToID:     peer,
	}

	updates, err := c.payOrFreeFallback(ctx, invoice, func() (tg.UpdatesClass, error) {
		return c.Raw().PaymentsTransferStarGift(ctx, &tg.PaymentsTransferStarGiftRequest{
			Stargift: gift,
			ToID:     peer,
		})
	})
	if err != nil {
		return nil, err
	}
	if updates == nil {
		return nil, nil
	}
	return extractSingleMessage(updates, c)
}

func (c *Client) UpgradeGift(ctx context.Context, gift tg.InputSavedStarGiftClass, keepOriginalDetails bool) (*types.Message, error) {
	c.Log.Debug("UpgradeGift")

	invoice := &tg.InputInvoiceStarGiftUpgrade{
		Stargift:            gift,
		KeepOriginalDetails: keepOriginalDetails,
	}

	updates, err := c.payOrFreeFallback(ctx, invoice, func() (tg.UpdatesClass, error) {
		return c.Raw().PaymentsUpgradeStarGift(ctx, &tg.PaymentsUpgradeStarGiftRequest{
			Stargift:            gift,
			KeepOriginalDetails: keepOriginalDetails,
		})
	})
	if err != nil {
		return nil, err
	}
	if updates == nil {
		return nil, nil
	}
	return extractSingleMessage(updates, c)
}

func (c *Client) PrepayGiftUpgrade(ctx context.Context, peerID int64, hash string) (*types.Message, error) {
	c.Log.Debugf("PrepayGiftUpgrade peer_id=%d", peerID)
	peer, err := resolvePeer(c, peerID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	invoice := &tg.InputInvoiceStarGiftPrepaidUpgrade{
		Peer: peer,
		Hash: hash,
	}

	formID, err := c.getPaymentFormID(ctx, invoice)
	if err != nil {
		return nil, err
	}
	return c.payForUpdates(ctx, formID, invoice)
}

func (c *Client) BuyResaleGift(ctx context.Context, slug string, recipientID int64, opts ...*params.BuyGift) (*types.Message, error) {
	c.Log.Debugf("BuyResaleGift slug=%s", slug)
	opt := params.GetOptDef(&params.BuyGift{}, opts...)
	peer, err := resolvePeer(c, recipientID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	invoice := &tg.InputInvoiceStarGiftResale{
		Slug: slug,
		ToID: peer,
		Ton:  opt.Ton,
	}

	formID, err := c.getPaymentFormID(ctx, invoice)
	if err != nil {
		return nil, err
	}
	return c.payForUpdates(ctx, formID, invoice)
}

func (c *Client) SetGiftResalePrice(ctx context.Context, gift tg.InputSavedStarGiftClass, stars int64) error {
	c.Log.Debugf("SetGiftResalePrice stars=%d", stars)
	var amount tg.StarsAmount
	if stars <= 0 {
		amount = tg.StarsAmount{Amount: 0, Nanos: 0}
	} else {
		amount = tg.StarsAmount{Amount: stars, Nanos: 0}
	}
	_, err := c.Raw().PaymentsUpdateStarGiftPrice(ctx, &tg.PaymentsUpdateStarGiftPriceRequest{
		Stargift:     gift,
		ResellAmount: &amount,
	})
	return err
}

func (c *Client) ShowGift(ctx context.Context, gift tg.InputSavedStarGiftClass) error {
	c.Log.Debug("ShowGift")
	_, err := c.Raw().PaymentsSaveStarGift(ctx, &tg.PaymentsSaveStarGiftRequest{
		Unsave:   false,
		Stargift: gift,
	})
	return err
}

func (c *Client) HideGift(ctx context.Context, gift tg.InputSavedStarGiftClass) error {
	c.Log.Debug("HideGift")
	_, err := c.Raw().PaymentsSaveStarGift(ctx, &tg.PaymentsSaveStarGiftRequest{
		Unsave:   true,
		Stargift: gift,
	})
	return err
}

func (c *Client) ConvertGift(ctx context.Context, gift tg.InputSavedStarGiftClass) error {
	c.Log.Debug("ConvertGift")
	_, err := c.Raw().PaymentsConvertStarGift(ctx, &tg.PaymentsConvertStarGiftRequest{
		Stargift: gift,
	})
	return err
}

func (c *Client) PinGifts(ctx context.Context, gifts []tg.InputSavedStarGiftClass, peerID int64) error {
	c.Log.Debugf("PinGifts peer_id=%d count=%d", peerID, len(gifts))
	peer, err := resolvePeer(c, peerID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	_, err = c.Raw().PaymentsToggleStarGiftsPinnedToTop(ctx, &tg.PaymentsToggleStarGiftsPinnedToTopRequest{
		Peer:     peer,
		Stargift: gifts,
	})
	return err
}

func (c *Client) SendGiftOffer(ctx context.Context, peerID int64, slug string, price int64, duration int32) (*types.Message, error) {
	c.Log.Debugf("SendGiftOffer peer_id=%d slug=%s", peerID, slug)
	peer, err := resolvePeer(c, peerID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().PaymentsSendStarGiftOffer(ctx, &tg.PaymentsSendStarGiftOfferRequest{
		Peer:     peer,
		Slug:     slug,
		Price:    &tg.StarsAmount{Amount: price, Nanos: 0},
		Duration: duration,
		RandomID: c.RandomID(),
	})
	if err != nil {
		return nil, err
	}
	return extractSingleMessage(result, c)
}

func (c *Client) GetGiftWithdrawalURL(ctx context.Context, gift tg.InputSavedStarGiftClass, password string) (string, error) {
	c.Log.Debug("GetGiftWithdrawalURL")
	srp, err := c.computeSRP(ctx, password)
	if err != nil {
		return "", fmt.Errorf("compute SRP: %w", err)
	}
	res, err := c.Raw().PaymentsGetStarGiftWithdrawalURL(ctx, &tg.PaymentsGetStarGiftWithdrawalURLRequest{
		Stargift: gift,
		Password: srp,
	})
	if err != nil {
		return "", err
	}
	return res.URL, nil
}

func (c *Client) getPaymentFormID(ctx context.Context, invoice tg.InputInvoiceClass) (int64, error) {
	form, err := c.Raw().PaymentsGetPaymentForm(ctx, &tg.PaymentsGetPaymentFormRequest{
		Invoice: invoice,
	})
	if err != nil {
		return 0, fmt.Errorf("get payment form: %w", err)
	}
	switch f := form.(type) {
	case *tg.PaymentsPaymentForm:
		return f.FormID, nil
	case *tg.PaymentsPaymentFormStars:
		return f.FormID, nil
	default:
		return 0, fmt.Errorf("unexpected payment form type %T", form)
	}
}

func (c *Client) payForUpdates(ctx context.Context, formID int64, invoice tg.InputInvoiceClass) (*types.Message, error) {
	res, err := c.Raw().PaymentsSendStarsForm(ctx, &tg.PaymentsSendStarsFormRequest{
		FormID:  formID,
		Invoice: invoice,
	})
	if err != nil {
		return nil, fmt.Errorf("send stars form: %w", err)
	}
	paymentResult, ok := res.(*tg.PaymentsPaymentResult)
	if !ok {
		return nil, nil
	}
	return extractSingleMessage(paymentResult.Updates, c)
}

func (c *Client) payOrFreeFallback(ctx context.Context, invoice tg.InputInvoiceClass, freeFn func() (tg.UpdatesClass, error)) (tg.UpdatesClass, error) {
	form, err := c.Raw().PaymentsGetPaymentForm(ctx, &tg.PaymentsGetPaymentFormRequest{
		Invoice: invoice,
	})
	if err != nil {
		if tgerr.Is(err, tgerr.ErrNoPaymentNeeded) {
			return freeFn()
		}
		return nil, fmt.Errorf("get payment form: %w", err)
	}

	var formID int64
	switch f := form.(type) {
	case *tg.PaymentsPaymentForm:
		formID = f.FormID
	case *tg.PaymentsPaymentFormStars:
		formID = f.FormID
	default:
		return nil, fmt.Errorf("unexpected payment form type %T", form)
	}

	res, err := c.Raw().PaymentsSendStarsForm(ctx, &tg.PaymentsSendStarsFormRequest{
		FormID:  formID,
		Invoice: invoice,
	})
	if err != nil {
		return nil, fmt.Errorf("send stars form: %w", err)
	}

	paymentResult, ok := res.(*tg.PaymentsPaymentResult)
	if !ok {
		return nil, nil
	}
	return paymentResult.Updates, nil
}
