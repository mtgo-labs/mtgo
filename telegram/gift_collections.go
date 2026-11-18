package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// CreateGiftCollection groups owned gifts into a named showcase collection on
// the giver's profile.
func (c *PaymentMethods) CreateGiftCollection(ctx context.Context, chatID int64, title string, gifts []tg.InputSavedStarGiftClass) (*tg.StarGiftCollection, error) {
	c.Log.Debugf("CreateGiftCollection chat_id=%d title=%q", chatID, title)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().PaymentsCreateStarGiftCollection(ctx, &tg.PaymentsCreateStarGiftCollectionRequest{
		Peer:     peer,
		Title:    title,
		Stargift: gifts,
	})
	if err != nil {
		return nil, fmt.Errorf("create gift collection: %w", err)
	}
	return result, nil
}

// UpdateGiftCollection edits a gift collection: renames it (title), removes
// or adds gifts, or reorders its gifts. Empty values leave each aspect
// untouched.
func (c *PaymentMethods) UpdateGiftCollection(ctx context.Context, chatID int64, collectionID int32, title string, remove, add, order []tg.InputSavedStarGiftClass) (*tg.StarGiftCollection, error) {
	c.Log.Debugf("UpdateGiftCollection chat_id=%d collection=%d", chatID, collectionID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	req := &tg.PaymentsUpdateStarGiftCollectionRequest{
		Peer:         peer,
		CollectionID: collectionID,
	}
	if title != "" {
		req.Title = title
	}
	if len(remove) > 0 {
		req.DeleteStargift = remove
	}
	if len(add) > 0 {
		req.AddStargift = add
	}
	if len(order) > 0 {
		req.Order = order
	}
	result, err := c.Raw().PaymentsUpdateStarGiftCollection(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("update gift collection: %w", err)
	}
	return result, nil
}

// ReorderGiftCollections changes the display order of a profile's gift
// collections. order lists collection IDs in their new order.
func (c *PaymentMethods) ReorderGiftCollections(ctx context.Context, chatID int64, order []int32) error {
	c.Log.Debugf("ReorderGiftCollections chat_id=%d n=%d", chatID, len(order))
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	_, err = c.Raw().PaymentsReorderStarGiftCollections(ctx, &tg.PaymentsReorderStarGiftCollectionsRequest{
		Peer:  peer,
		Order: order,
	})
	if err != nil {
		return fmt.Errorf("reorder gift collections: %w", err)
	}
	return nil
}

// DeleteGiftCollection deletes a gift collection. The gifts inside return to
// the ungrouped showcase.
func (c *PaymentMethods) DeleteGiftCollection(ctx context.Context, chatID int64, collectionID int32) error {
	c.Log.Debugf("DeleteGiftCollection chat_id=%d collection=%d", chatID, collectionID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	_, err = c.Raw().PaymentsDeleteStarGiftCollection(ctx, &tg.PaymentsDeleteStarGiftCollectionRequest{
		Peer:         peer,
		CollectionID: collectionID,
	})
	if err != nil {
		return fmt.Errorf("delete gift collection: %w", err)
	}
	return nil
}

// GetGiftCollections lists a profile's gift collections. Returns nil when
// unchanged since the last hash.
func (c *PaymentMethods) GetGiftCollections(ctx context.Context, chatID int64, hash int64) ([]*tg.StarGiftCollection, error) {
	c.Log.Debugf("GetGiftCollections chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().PaymentsGetStarGiftCollections(ctx, &tg.PaymentsGetStarGiftCollectionsRequest{
		Peer: peer,
		Hash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("get gift collections: %w", err)
	}
	switch v := result.(type) {
	case *tg.PaymentsStarGiftCollections:
		return v.Collections, nil
	case *tg.PaymentsStarGiftCollectionsNotModified:
		return nil, nil
	default:
		return nil, fmt.Errorf("get gift collections: unexpected result %T", result)
	}
}

// GetGiftAuctionState returns the live state of a gift auction; pass the last
// seen version to receive only changes. The result is the auction state, its
// finished form, or NotModified.
func (c *PaymentMethods) GetGiftAuctionState(ctx context.Context, auction tg.InputStarGiftAuctionClass, version int32) (tg.StarGiftAuctionStateClass, error) {
	c.Log.Debugf("GetGiftAuctionState version=%d", version)
	result, err := c.Raw().PaymentsGetStarGiftAuctionState(ctx, &tg.PaymentsGetStarGiftAuctionStateRequest{
		Auction: auction,
		Version: version,
	})
	if err != nil {
		return nil, fmt.Errorf("get gift auction state: %w", err)
	}
	return result, nil
}

// GetGiftAuctionAcquiredGifts lists the gifts acquired through auctions by a
// user's resale bot.
func (c *PaymentMethods) GetGiftAuctionAcquiredGifts(ctx context.Context, giftID int64) (*tg.PaymentsStarGiftAuctionAcquiredGifts, error) {
	c.Log.Debugf("GetGiftAuctionAcquiredGifts gift_id=%d", giftID)
	result, err := c.Raw().PaymentsGetStarGiftAuctionAcquiredGifts(ctx, &tg.PaymentsGetStarGiftAuctionAcquiredGiftsRequest{
		GiftID: giftID,
	})
	if err != nil {
		return nil, fmt.Errorf("get gift auction acquired gifts: %w", err)
	}
	return result, nil
}

// GetActiveGiftAuctions lists currently running gift auctions. Returns nil
// when unchanged since the last hash.
func (c *PaymentMethods) GetActiveGiftAuctions(ctx context.Context, hash int64) ([]*tg.StarGiftActiveAuctionState, error) {
	c.Log.Debug("GetActiveGiftAuctions")
	result, err := c.Raw().PaymentsGetStarGiftActiveAuctions(ctx, &tg.PaymentsGetStarGiftActiveAuctionsRequest{
		Hash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("get active gift auctions: %w", err)
	}
	switch v := result.(type) {
	case *tg.PaymentsStarGiftActiveAuctions:
		return v.Auctions, nil
	case *tg.PaymentsStarGiftActiveAuctionsNotModified:
		return nil, nil
	default:
		return nil, fmt.Errorf("get active gift auctions: unexpected result %T", result)
	}
}

// CheckCanSendGift verifies the logged-in user may send a specific gift
// (age, privacy, and regional restrictions included) before attempting it.
func (c *PaymentMethods) CheckCanSendGift(ctx context.Context, giftID int64) error {
	c.Log.Debugf("CheckCanSendGift gift_id=%d", giftID)
	result, err := c.Raw().PaymentsCheckCanSendGift(ctx, &tg.PaymentsCheckCanSendGiftRequest{
		GiftID: giftID,
	})
	if err != nil {
		return fmt.Errorf("check can send gift: %w", err)
	}
	switch v := result.(type) {
	case *tg.PaymentsCheckCanSendGiftResultOk:
		return nil
	case *tg.PaymentsCheckCanSendGiftResultFail:
		reason := "not allowed"
		if v.Reason != nil {
			reason = v.Reason.Text
		}
		return fmt.Errorf("check can send gift: %s", reason)
	default:
		return fmt.Errorf("check can send gift: unexpected result %T", result)
	}
}

// SetGiftChatNotifications enables or disable notifications about new gifts
// in a chat.
func (c *PaymentMethods) SetGiftChatNotifications(ctx context.Context, chatID int64, enabled bool) error {
	c.Log.Debugf("SetGiftChatNotifications chat_id=%d enabled=%v", chatID, enabled)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	_, err = c.Raw().PaymentsToggleChatStarGiftNotifications(ctx, &tg.PaymentsToggleChatStarGiftNotificationsRequest{
		Enabled: enabled,
		Peer:    peer,
	})
	if err != nil {
		return fmt.Errorf("set gift chat notifications: %w", err)
	}
	return nil
}
