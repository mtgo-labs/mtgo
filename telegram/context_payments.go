package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// GetPaymentForm retrieves the payment form for a product invoice message. Use this
// to display payment details and collect payment credentials from the user.
//
// Parameters:
//   - chatID: the chat ID where the invoice message was sent
//   - messageID: the message ID of the invoice
//   - opts: optional [GetPaymentFormOption] parameters for additional configuration
//
// Returns:
//   - tg.PaymentFormClass: the payment form containing product and payment details
//   - error: non-nil if the context has no client or the request fails
//
// Example:
//
//	form, err := ctx.GetPaymentForm(chatID, invoiceMsgID)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Context) GetPaymentForm(chatID int64, messageID int32, opts ...*GetPaymentFormOption) (*types.PaymentForm, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.GetPaymentForm(c.Ctx, chatID, messageID, opts...)
}

// SendPaymentForm submits completed payment credentials to finalize a payment. This is
// the second step after retrieving the payment form via [Context.GetPaymentForm].
//
// Parameters:
//   - formID: the payment form ID obtained from GetPaymentForm
//   - chatID: the chat ID where the invoice message was sent
//   - messageID: the message ID of the invoice
//   - creds: the payment credentials provided by the user
//   - opts: optional [SendPaymentFormOption] parameters for additional configuration
//
// Returns:
//   - tg.PaymentResultClass: the payment result (success confirmation or payment URL)
//   - error: non-nil if the context has no client, the credentials are invalid, or the payment fails
//
// Example:
//
//	result, err := ctx.SendPaymentForm(formID, chatID, msgID, creds)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Context) SendPaymentForm(formID int64, chatID int64, messageID int32, creds tg.InputPaymentCredentialsClass, opts ...*SendPaymentFormOption) (tg.PaymentResultClass, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.SendPaymentForm(c.Ctx, formID, chatID, messageID, creds, opts...)
}

// GetStarsBalance retrieves the current Telegram Stars balance for the specified chat
// (typically a bot chat).
//
// Parameters:
//   - chatID: the chat ID (bot ID) to check the Stars balance for
//
// Returns:
//   - int64: the current Stars balance
//   - error: non-nil if the context has no client or the request fails
//
// Example:
//
//	balance, err := ctx.GetStarsBalance(botChatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Stars balance: %d", balance)
func (c *Context) GetStarsBalance(chatID int64) (int64, error) {
	if c.Client == nil {
		return 0, ErrContextNoClient
	}
	return c.Client.GetStarsBalance(c.Ctx, chatID)
}

// SendGift sends a Telegram gift to the specified user. Gifts are special items that can
// be displayed on the user's profile.
//
// Parameters:
//   - userID: the Telegram user ID of the gift recipient
//   - giftID: the identifier of the gift to send
//   - message: an accompanying text message for the gift
//
// Returns:
//   - error: non-nil if the context has no client, the gift cannot be sent, or the user cannot receive gifts
//
// Example:
//
//	err := ctx.SendGift(recipientID, giftID, "Happy birthday!")
//	if err != nil {
//	    log.Printf("failed to send gift: %v", err)
//	}
func (c *Context) SendGift(userID int64, giftID int64, message string, opts ...*params.GiftSend) (*types.Message, error) {
	if c.Client == nil {
		return nil, ErrContextNoClient
	}
	return c.Client.SendGift(c.Ctx, userID, giftID, message, opts...)
}
