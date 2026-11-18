package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// GetStarsTopupOptions returns the available Telegram Stars top-up packages.
func (c *PaymentMethods) GetStarsTopupOptions(ctx context.Context) ([]*tg.StarsTopupOption, error) {
	c.Log.Debug("GetStarsTopupOptions")
	raw, err := c.Raw().PaymentsGetStarsTopupOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get stars topup options: %w", err)
	}
	options, err := unwrapStarsVector[*tg.StarsTopupOption](raw, "topup options")
	if err != nil {
		return nil, fmt.Errorf("get stars topup options: %w", err)
	}
	return options, nil
}

// GetStarsGiveawayOptions returns the available Stars-giveaway boost packages
// for channels.
func (c *PaymentMethods) GetStarsGiveawayOptions(ctx context.Context) ([]*tg.StarsGiveawayOption, error) {
	c.Log.Debug("GetStarsGiveawayOptions")
	raw, err := c.Raw().PaymentsGetStarsGiveawayOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get stars giveaway options: %w", err)
	}
	options, err := unwrapStarsVector[*tg.StarsGiveawayOption](raw, "giveaway options")
	if err != nil {
		return nil, fmt.Errorf("get stars giveaway options: %w", err)
	}
	return options, nil
}

// unwrapStarsVector converts a generic bare-vector RPC result into typed
// entries.
func unwrapStarsVector[T any](raw tg.TLObject, what string) ([]T, error) {
	vector, ok := raw.(*tg.GenericVector)
	if !ok {
		return nil, fmt.Errorf("unexpected %s result %T", what, raw)
	}
	items := make([]T, 0, len(vector.Items))
	for _, item := range vector.Items {
		typed, ok := item.(T)
		if !ok {
			return nil, fmt.Errorf("unexpected %s entry %T", what, item)
		}
		items = append(items, typed)
	}
	return items, nil
}

// StarsSubscriptions is one page of the logged-in user's Stars subscriptions.
type StarsSubscriptions struct {
	// Subscriptions are the subscriptions on this page.
	Subscriptions []*tg.StarsSubscription
	// NextOffset fetches the next page; empty on the last page.
	NextOffset string
	// MissingBalance is the total Stars shortfall of active subscriptions.
	MissingBalance int64
}

// GetStarsSubscriptions lists the logged-in user's Stars subscriptions;
// missingBalance true lists only subscriptions the balance can no longer
// cover.
func (c *PaymentMethods) GetStarsSubscriptions(ctx context.Context, offset string, missingBalance bool) (*StarsSubscriptions, error) {
	c.Log.Debugf("GetStarsSubscriptions offset=%q", offset)
	result, err := c.Raw().PaymentsGetStarsSubscriptions(ctx, &tg.PaymentsGetStarsSubscriptionsRequest{
		MissingBalance: missingBalance,
		Peer:           &tg.InputPeerSelf{},
		Offset:         offset,
	})
	if err != nil {
		return nil, fmt.Errorf("get stars subscriptions: %w", err)
	}
	return &StarsSubscriptions{
		Subscriptions:  result.Subscriptions,
		NextOffset:     result.SubscriptionsNextOffset,
		MissingBalance: result.SubscriptionsMissingBalance,
	}, nil
}

// ChangeStarsSubscription cancels (or re-enables) one of the logged-in user's
// Stars subscriptions with a bot.
func (c *PaymentMethods) ChangeStarsSubscription(ctx context.Context, botID int64, subscriptionID string, canceled bool) error {
	c.Log.Debugf("ChangeStarsSubscription bot_id=%d canceled=%v", botID, canceled)
	peer, err := resolvePeer(ctx, c.Client, botID)
	if err != nil {
		return fmt.Errorf("resolve bot: %w", err)
	}
	_, err = c.Raw().PaymentsChangeStarsSubscription(ctx, &tg.PaymentsChangeStarsSubscriptionRequest{
		Peer:           peer,
		SubscriptionID: subscriptionID,
		Canceled:       canceled,
	})
	if err != nil {
		return fmt.Errorf("change stars subscription: %w", err)
	}
	return nil
}

// FulfillStarsSubscription charges the subscriber for the next period of a
// bot's Stars subscription.
func (c *PaymentMethods) FulfillStarsSubscription(ctx context.Context, subscriberID int64, subscriptionID string) error {
	c.Log.Debugf("FulfillStarsSubscription user_id=%d", subscriberID)
	peer, err := resolvePeer(ctx, c.Client, subscriberID)
	if err != nil {
		return fmt.Errorf("resolve subscriber: %w", err)
	}
	_, err = c.Raw().PaymentsFulfillStarsSubscription(ctx, &tg.PaymentsFulfillStarsSubscriptionRequest{
		Peer:           peer,
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		return fmt.Errorf("fulfill stars subscription: %w", err)
	}
	return nil
}

// CancelStarsSubscriptionByBot cancels a user's subscription to the logged-in
// bot's paid channel; restore re-enables a previously canceled subscription.
func (c *PaymentMethods) CancelStarsSubscriptionByBot(ctx context.Context, userID int64, chargeID string, restore bool) error {
	c.Log.Debugf("CancelStarsSubscriptionByBot user_id=%d restore=%v", userID, restore)
	user, err := resolveUserID(ctx, c.Client, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}
	_, err = c.Raw().PaymentsBotCancelStarsSubscription(ctx, &tg.PaymentsBotCancelStarsSubscriptionRequest{
		Restore:  restore,
		UserID:   user,
		ChargeID: chargeID,
	})
	if err != nil {
		return fmt.Errorf("cancel stars subscription by bot: %w", err)
	}
	return nil
}

// GetStarsTransactionsByID fetches specific Stars transactions of the logged-in
// user by identifier.
func (c *PaymentMethods) GetStarsTransactionsByID(ctx context.Context, ids []*tg.InputStarsTransaction) ([]*tg.StarsTransaction, error) {
	c.Log.Debugf("GetStarsTransactionsByID n=%d", len(ids))
	result, err := c.Raw().PaymentsGetStarsTransactionsByID(ctx, &tg.PaymentsGetStarsTransactionsByIDRequest{
		Peer: &tg.InputPeerSelf{},
		ID:   ids,
	})
	if err != nil {
		return nil, fmt.Errorf("get stars transactions by id: %w", err)
	}
	return result.History, nil
}

// GetStarsRevenueStats returns the Stars revenue statistics of a chat owned by
// the logged-in user. dark selects the dark chart theme.
func (c *PaymentMethods) GetStarsRevenueStats(ctx context.Context, chatID int64, dark bool) (*tg.PaymentsStarsRevenueStats, error) {
	c.Log.Debugf("GetStarsRevenueStats chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().PaymentsGetStarsRevenueStats(ctx, &tg.PaymentsGetStarsRevenueStatsRequest{
		Dark: dark,
		Peer: peer,
	})
	if err != nil {
		return nil, fmt.Errorf("get stars revenue stats: %w", err)
	}
	return result, nil
}

// WithdrawStarsRevenue opens a withdrawal URL for the Stars balance of a chat
// owned by the logged-in user. password is the cloud password, required when
// one is set.
func (c *PaymentMethods) WithdrawStarsRevenue(ctx context.Context, chatID int64, amount int64, password string) (string, error) {
	c.Log.Debugf("WithdrawStarsRevenue chat_id=%d amount=%d", chatID, amount)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return "", fmt.Errorf("resolve peer: %w", err)
	}
	req := &tg.PaymentsGetStarsRevenueWithdrawalURLRequest{
		Peer:   peer,
		Amount: amount,
	}
	if password != "" {
		srp, err := c.computeSRP(ctx, password)
		if err != nil {
			return "", fmt.Errorf("compute SRP: %w", err)
		}
		req.Password = srp
	}
	result, err := c.Raw().PaymentsGetStarsRevenueWithdrawalURL(ctx, req)
	if err != nil {
		return "", fmt.Errorf("withdraw stars revenue: %w", err)
	}
	return result.URL, nil
}

// GetStarsRevenueAdsAccountURL returns the ad-platform account URL connected
// to a channel's Stars revenue.
func (c *PaymentMethods) GetStarsRevenueAdsAccountURL(ctx context.Context, chatID int64) (string, error) {
	c.Log.Debugf("GetStarsRevenueAdsAccountURL chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return "", fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().PaymentsGetStarsRevenueAdsAccountURL(ctx, &tg.PaymentsGetStarsRevenueAdsAccountURLRequest{
		Peer: peer,
	})
	if err != nil {
		return "", fmt.Errorf("get stars revenue ads account url: %w", err)
	}
	return result.URL, nil
}

// GetPremiumGiftCodeOptions returns purchasable Premium-gift-code packages
// (for gifting or boosting boostPeer).
func (c *PaymentMethods) GetPremiumGiftCodeOptions(ctx context.Context, boostPeer int64) ([]*tg.PremiumGiftCodeOption, error) {
	c.Log.Debugf("GetPremiumGiftCodeOptions boost_peer=%d", boostPeer)
	req := &tg.PaymentsGetPremiumGiftCodeOptionsRequest{}
	if boostPeer != 0 {
		peer, err := resolvePeer(ctx, c.Client, boostPeer)
		if err != nil {
			return nil, fmt.Errorf("resolve boost peer: %w", err)
		}
		req.BoostPeer = peer
	}
	result, err := c.Raw().PaymentsGetPremiumGiftCodeOptions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get premium gift code options: %w", err)
	}
	options, err := unwrapStarsVector[*tg.PremiumGiftCodeOption](result, "premium gift code options")
	if err != nil {
		return nil, fmt.Errorf("get premium gift code options: %w", err)
	}
	return options, nil
}

// CheckGiftCode inspects a gift-code slug without redeeming it.
func (c *PaymentMethods) CheckGiftCode(ctx context.Context, slug string) (*tg.PaymentsCheckedGiftCode, error) {
	c.Log.Debugf("CheckGiftCode slug=%s", slug)
	result, err := c.Raw().PaymentsCheckGiftCode(ctx, &tg.PaymentsCheckGiftCodeRequest{
		Slug: slug,
	})
	if err != nil {
		return nil, fmt.Errorf("check gift code: %w", err)
	}
	return result, nil
}

// ApplyGiftCode redeems a gift code (Telegram Premium or Stars) for the
// logged-in user.
func (c *PaymentMethods) ApplyGiftCode(ctx context.Context, slug string) error {
	c.Log.Debugf("ApplyGiftCode slug=%s", slug)
	_, err := c.Raw().PaymentsApplyGiftCode(ctx, &tg.PaymentsApplyGiftCodeRequest{
		Slug: slug,
	})
	if err != nil {
		return fmt.Errorf("apply gift code: %w", err)
	}
	return nil
}

// GetGiveawayInfo returns participation or result information about a
// giveaway message.
func (c *PaymentMethods) GetGiveawayInfo(ctx context.Context, chatID int64, messageID int32) (tg.GiveawayInfoClass, error) {
	c.Log.Debugf("GetGiveawayInfo chat_id=%d msg=%d", chatID, messageID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().PaymentsGetGiveawayInfo(ctx, &tg.PaymentsGetGiveawayInfoRequest{
		Peer:  peer,
		MsgID: messageID,
	})
	if err != nil {
		return nil, fmt.Errorf("get giveaway info: %w", err)
	}
	return result, nil
}

// LaunchPrepaidGiveaway launches a previously prepaid giveaway in a chat,
// using the giveaway identifier from the purchase receipt.
func (c *PaymentMethods) LaunchPrepaidGiveaway(ctx context.Context, chatID int64, giveawayID int64, purpose tg.InputStorePaymentPurposeClass) error {
	c.Log.Debugf("LaunchPrepaidGiveaway chat_id=%d giveaway=%d", chatID, giveawayID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	_, err = c.Raw().PaymentsLaunchPrepaidGiveaway(ctx, &tg.PaymentsLaunchPrepaidGiveawayRequest{
		Peer:       peer,
		GiveawayID: giveawayID,
		Purpose:    purpose,
	})
	if err != nil {
		return fmt.Errorf("launch prepaid giveaway: %w", err)
	}
	return nil
}

// GetPaymentReceipt fetches the receipt for a paid invoice message.
func (c *PaymentMethods) GetPaymentReceipt(ctx context.Context, chatID int64, messageID int32) (tg.PaymentReceiptClass, error) {
	c.Log.Debugf("GetPaymentReceipt chat_id=%d msg=%d", chatID, messageID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	result, err := c.Raw().PaymentsGetPaymentReceipt(ctx, &tg.PaymentsGetPaymentReceiptRequest{
		Peer:  peer,
		MsgID: messageID,
	})
	if err != nil {
		return nil, fmt.Errorf("get payment receipt: %w", err)
	}
	return result, nil
}

// ValidateRequestedInfo validates shipping information for an invoice and
// returns the available shipping options; save stores the info for reuse.
func (c *PaymentMethods) ValidateRequestedInfo(ctx context.Context, invoice tg.InputInvoiceClass, info *tg.PaymentRequestedInfo, save bool) (*tg.PaymentsValidatedRequestedInfo, error) {
	c.Log.Debugf("ValidateRequestedInfo save=%v", save)
	result, err := c.Raw().PaymentsValidateRequestedInfo(ctx, &tg.PaymentsValidateRequestedInfoRequest{
		Save:    save,
		Invoice: invoice,
		Info:    info,
	})
	if err != nil {
		return nil, fmt.Errorf("validate requested info: %w", err)
	}
	return result, nil
}

// GetSavedPaymentInfo returns the saved shipping info and whether payment
// credentials are stored.
func (c *PaymentMethods) GetSavedPaymentInfo(ctx context.Context) (*tg.PaymentsSavedInfo, error) {
	c.Log.Debug("GetSavedPaymentInfo")
	result, err := c.Raw().PaymentsGetSavedInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("get saved payment info: %w", err)
	}
	return result, nil
}

// ClearSavedPaymentInfo deletes the saved shipping info and/or payment
// credentials.
func (c *PaymentMethods) ClearSavedPaymentInfo(ctx context.Context, info, credentials bool) error {
	c.Log.Debugf("ClearSavedPaymentInfo info=%v credentials=%v", info, credentials)
	_, err := c.Raw().PaymentsClearSavedInfo(ctx, &tg.PaymentsClearSavedInfoRequest{
		Info:        info,
		Credentials: credentials,
	})
	if err != nil {
		return fmt.Errorf("clear saved payment info: %w", err)
	}
	return nil
}

// CanPurchaseFromStore checks whether a store purchase (Premium, Stars,
// gifts) is available for the given purpose before opening the store flow.
func (c *PaymentMethods) CanPurchaseFromStore(ctx context.Context, purpose tg.InputStorePaymentPurposeClass) (bool, error) {
	c.Log.Debug("CanPurchaseFromStore")
	ok, err := c.Raw().PaymentsCanPurchaseStore(ctx, &tg.PaymentsCanPurchaseStoreRequest{
		Purpose: purpose,
	})
	if err != nil {
		return false, fmt.Errorf("can purchase from store: %w", err)
	}
	return ok, nil
}

// AssignAppStoreTransaction applies an App Store in-app purchase receipt to
// its purpose.
func (c *PaymentMethods) AssignAppStoreTransaction(ctx context.Context, receipt []byte, purpose tg.InputStorePaymentPurposeClass) error {
	c.Log.Debugf("AssignAppStoreTransaction purpose=%T", purpose)
	_, err := c.Raw().PaymentsAssignAppStoreTransaction(ctx, &tg.PaymentsAssignAppStoreTransactionRequest{
		Receipt: receipt,
		Purpose: purpose,
	})
	if err != nil {
		return fmt.Errorf("assign app store transaction: %w", err)
	}
	return nil
}

// AssignPlayMarketTransaction applies a Google Play in-app purchase receipt
// (a JSON payload) to its purpose.
func (c *PaymentMethods) AssignPlayMarketTransaction(ctx context.Context, receiptJSON string, purpose tg.InputStorePaymentPurposeClass) error {
	c.Log.Debugf("AssignPlayMarketTransaction purpose=%T", purpose)
	_, err := c.Raw().PaymentsAssignPlayMarketTransaction(ctx, &tg.PaymentsAssignPlayMarketTransactionRequest{
		Receipt: &tg.DataJSON{Data: receiptJSON},
		Purpose: purpose,
	})
	if err != nil {
		return fmt.Errorf("assign play market transaction: %w", err)
	}
	return nil
}
