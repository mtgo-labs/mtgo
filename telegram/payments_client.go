package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// GetPaymentFormOption holds optional parameters for GetPaymentForm.
type GetPaymentFormOption struct {
	ThemeParams *string
}

// SendPaymentFormOption holds optional parameters for SendPaymentForm.
type SendPaymentFormOption struct {
	RequestedInfoID  string
	ShippingOptionID string
	TipAmount        *int64
}

// GetPaymentForm retrieves the payment form for an invoice message in the specified chat.
// The payment form contains the invoice details, available payment methods, and form ID
// needed for SendPaymentForm.
//
// Example:
//
//	form, err := client.GetPaymentForm(ctx, chatID, msgID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Form ID: %d, title: %s\n", form.GetFormID(), form.GetTitle())
func (c *Client) GetPaymentForm(ctx context.Context, chatID int64, messageID int32, opts ...*GetPaymentFormOption) (*types.PaymentForm, error) {
	c.Log.Debugf("GetPaymentForm chat_id=%d msg_id=%d", chatID, messageID)
	opt := params.GetOptDef(&GetPaymentFormOption{}, opts...)

	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	req := &tg.PaymentsGetPaymentFormRequest{
		Invoice: &tg.InputInvoiceMessage{Peer: peer, MsgID: messageID},
	}
	if opt.ThemeParams != nil {
		req.ThemeParams = &tg.DataJSON{Data: *opt.ThemeParams}
	}

	rpc := c.Raw()
	raw, err := rpc.PaymentsGetPaymentForm(ctx, req)
	if err != nil {
		return nil, err
	}
	return types.ParsePaymentForm(raw), nil
}

// SendPaymentForm submits payment credentials for a previously retrieved payment form.
// The formID comes from GetPaymentForm. creds contains the payment method (e.g. saved
// credentials, new card, or Telegram Stars). Optional parameters include shipping option
// and tip amount.
//
// Example:
//
//	creds := &tg.InputPaymentCredentialsSaved{ID: "card_hash"}
//	result, err := client.SendPaymentForm(ctx, formID, chatID, msgID, creds)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) SendPaymentForm(ctx context.Context, formID int64, chatID int64, messageID int32, creds tg.InputPaymentCredentialsClass, opts ...*SendPaymentFormOption) (tg.PaymentResultClass, error) {
	c.Log.Debugf("SendPaymentForm form_id=%d chat_id=%d msg_id=%d", formID, chatID, messageID)
	if creds == nil {
		return nil, ErrPaymentsCredentialsRequired
	}
	opt := params.GetOptDef(&SendPaymentFormOption{}, opts...)

	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	req := &tg.PaymentsSendPaymentFormRequest{
		FormID:      formID,
		Invoice:     &tg.InputInvoiceMessage{Peer: peer, MsgID: messageID},
		Credentials: creds,
	}
	if opt.RequestedInfoID != "" {
		req.RequestedInfoID = opt.RequestedInfoID
	}
	if opt.ShippingOptionID != "" {
		req.ShippingOptionID = opt.ShippingOptionID
	}
	if opt.TipAmount != nil {
		req.TipAmount = *opt.TipAmount
	}

	rpc := c.Raw()
	return rpc.PaymentsSendPaymentForm(ctx, req)
}

// GetStarsBalance returns the Telegram Stars balance for the specified peer.
// Use the bot's own chatID to check the bot's Stars balance.
//
// Example:
//
//	balance, err := client.GetStarsBalance(ctx, botChatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Stars balance: %d\n", balance)
func (c *Client) GetStarsBalance(ctx context.Context, chatID int64) (int64, error) {
	c.Log.Debugf("GetStarsBalance chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return 0, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.PaymentsGetStarsStatus(ctx, &tg.PaymentsGetStarsStatusRequest{
		Peer: peer,
	})
	if err != nil {
		return 0, err
	}
	if result.Balance == nil {
		return 0, nil
	}
	if amt, ok := result.Balance.(*tg.StarsAmount); ok {
		return amt.Amount, nil
	}
	return 0, nil
}

// AnswerPreCheckoutQuery responds to a pre-checkout query, approving or rejecting the payment.
// When ok is true the payment proceeds; when false, errorMessage is shown to the user.
// Call this inside a PreCheckoutQuery handler.
//
// Example:
//
//	client.OnPreCheckoutQuery(func(ctx *telegram.Context) {
//	    query := ctx.PreCheckoutQuery
//	    if err := client.AnswerPreCheckoutQuery(ctx.Ctx, query.ID, true, ""); err != nil {
//	        log.Printf("pre-checkout error: %v", err)
//	    }
//	})
func (c *Client) AnswerPreCheckoutQuery(ctx context.Context, queryID int64, ok bool, errorMessage string) error {
	c.Log.Debugf("AnswerPreCheckoutQuery query_id=%d ok=%v", queryID, ok)
	rpc := c.Raw()

	req := &tg.MessagesSetBotPrecheckoutResultsRequest{
		Success: ok,
		QueryID: queryID,
	}
	if !ok && errorMessage != "" {
		req.Error = errorMessage
	}

	_, err := rpc.MessagesSetBotPrecheckoutResults(ctx, req)
	return err
}

// AnswerShippingQuery responds to a shipping query with available shipping options.
// When ok is true, provide the available shipping options; when false, the payment
// is rejected. Call this inside a ShippingQuery handler.
//
// Example:
//
//	client.OnShippingQuery(func(ctx *telegram.Context) {
//	    query := ctx.ShippingQuery
//	    options := []*tg.ShippingOption{
//	        {ID: "express", Title: "Express", Prices: []tg.LabeledPrice{{Label: "Shipping", Amount: 500}}},
//	    }
//	    client.AnswerShippingQuery(ctx.Ctx, query.ID, true, options)
//	})
func (c *Client) AnswerShippingQuery(ctx context.Context, queryID int64, ok bool, options []*tg.ShippingOption) error {
	c.Log.Debugf("AnswerShippingQuery query_id=%d ok=%v", queryID, ok)
	rpc := c.Raw()

	req := &tg.MessagesSetBotShippingResultsRequest{
		QueryID:         queryID,
		ShippingOptions: options,
	}

	_, err := rpc.MessagesSetBotShippingResults(ctx, req)
	return err
}

// GetStarsTransactions retrieves a paginated list of Telegram Stars transactions
// for the specified peer (user, bot, or channel). Use the inbound/outbound flags
// to filter direction and offset for pagination.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the peer whose transactions to retrieve (0 for self)
//   - inbound: include incoming transactions
//   - outbound: include outgoing transactions
//   - offset: pagination cursor (empty string for first page)
//   - limit: maximum number of transactions to return (defaults to 100 if <= 0)
//
// Returns a *PaymentsStarsStatus containing the transaction list and balance on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
func (c *Client) GetStarsTransactions(ctx context.Context, chatID int64, inbound, outbound bool, offset string, limit int32, opts ...*params.GetStarsTransactionsOption) (*types.StarsStatus, error) {
	c.Log.Debugf("GetStarsTransactions chat_id=%d inbound=%v outbound=%v limit=%d", chatID, inbound, outbound, limit)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	if limit <= 0 {
		limit = 100
	}

	opt := params.GetOptDef(&params.GetStarsTransactionsOption{}, opts...)

	rpc := c.Raw()
	raw, err := rpc.PaymentsGetStarsTransactions(ctx, &tg.PaymentsGetStarsTransactionsRequest{
		Inbound:        inbound,
		Outbound:       outbound,
		Ascending:      opt.Ascending,
		SubscriptionID: opt.SubscriptionID,
		Ton:            opt.Ton,
		Peer:           peer,
		Offset:         offset,
		Limit:          limit,
	})
	if err != nil {
		return nil, err
	}
	return types.ParseStarsStatus(raw), nil
}

// RefundStarsCharge refunds a Telegram Stars charge. Bots can use this to refund
// a payment made by a user. The charge_id is obtained from the original payment
// update or transaction list.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - userID: the user who made the original payment
//   - chargeID: the unique charge identifier to refund
//
// Returns an error if:
//   - the user cannot be resolved
//   - the charge_id is invalid or already refunded
//   - the RPC call fails
func (c *Client) RefundStarsCharge(ctx context.Context, userID int64, chargeID string) error {
	c.Log.Debugf("RefundStarsCharge user_id=%d charge_id=%s", userID, chargeID)
	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.PaymentsRefundStarsCharge(ctx, &tg.PaymentsRefundStarsChargeRequest{
		UserID:   user,
		ChargeID: chargeID,
	})
	return err
}
