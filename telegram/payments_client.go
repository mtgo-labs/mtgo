package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
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
func (c *Client) GetPaymentForm(ctx context.Context, chatID int64, messageID int32, opts ...*GetPaymentFormOption) (tg.PaymentFormClass, error) {
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
	return rpc.PaymentsGetPaymentForm(ctx, req)
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
