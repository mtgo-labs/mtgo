package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *PaymentMethods) SendInvoice(ctx context.Context, chatID int64, invoice *tg.InputMediaInvoice, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendInvoice chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	opt := params.GetOptDef(&params.SendMessage{}, opts...)

	var flags tg.Fields
	if opt.Silent || opt.DisableNotification {
		flags.Set(5)
	}
	if opt.Background {
		flags.Set(6)
	}
	if opt.NoForwards {
		flags.Set(14)
	}

	var replyTo tg.InputReplyToClass
	if opt.ReplyTo != nil {
		flags.Set(0)
		replyTo = opt.ReplyTo
	} else if opt.ReplyToMessageID != 0 {
		flags.Set(0)
		replyTo = &tg.InputReplyToMessage{ReplyToMsgID: opt.ReplyToMessageID}
	}
	if opt.ReplyMarkup != nil {
		flags.Set(2)
	}
	if caption != "" {
		flags.Set(11)
	}

	req := &tg.MessagesSendMediaRequest{
		Flags:       flags,
		Silent:      opt.Silent || opt.DisableNotification,
		Background:  opt.Background,
		Noforwards:  opt.NoForwards,
		Peer:        peer,
		ReplyTo:     replyTo,
		Media:       invoice,
		RandomID:    c.RandomID(),
		Message:     caption,
		ReplyMarkup: opt.ReplyMarkup,
	}
	if opt.ScheduleDate != nil {
		req.ScheduleDate = *opt.ScheduleDate
	}

	rpc := c.Raw()
	result, err := rpc.MessagesSendMedia(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("send media: %w", err)
	}
	return extractSingleMessage(result, c)
}

func (c *PaymentMethods) CreateInvoiceLink(ctx context.Context, invoice *tg.InputMediaInvoice) (string, error) {
	c.Log.Debug("CreateInvoiceLink")
	rpc := c.Raw()
	result, err := rpc.PaymentsExportInvoice(ctx, &tg.PaymentsExportInvoiceRequest{
		InvoiceMedia: invoice,
	})
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

// PaymentVerificationNeededError reports that the provider requires
// additional verification (for example 3-D Secure) before the payment can
// complete. Open URL in a browser and retry the payment afterwards.
type PaymentVerificationNeededError struct {
	// URL is the verification page to open for the user.
	URL string
}

func (e *PaymentVerificationNeededError) Error() string {
	return "payment verification needed: open " + e.URL
}

// PayMessage runs the interactive checkout for an invoice message: fetch the
// payment form, validate the caller's order information when supplied, and
// submit the chosen credential. It returns the receipt message on success.
//
// A *PaymentVerificationNeededError means the provider wants additional
// confirmation (3-D Secure): complete it at the error's URL and call again.
//
// Example:
//
//	receipt, err := client.PayMessage(ctx, chatID, invoiceID, &params.Pay{
//	    CardData: params.StripeCardData(token),
//	})
func (c *PaymentMethods) PayMessage(ctx context.Context, chatID int64, messageID int32, opts ...*params.Pay) (*types.Message, error) {
	c.Log.Debugf("PayMessage chat_id=%d msg=%d", chatID, messageID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	invoice := &tg.InputInvoiceMessage{Peer: peer, MsgID: messageID}

	rawForm, err := c.Raw().PaymentsGetPaymentForm(ctx, &tg.PaymentsGetPaymentFormRequest{
		Invoice: invoice,
	})
	if err != nil {
		return nil, fmt.Errorf("get payment form: %w", err)
	}
	form, ok := rawForm.(*tg.PaymentsPaymentForm)
	if !ok {
		return nil, fmt.Errorf("get payment form: unexpected result %T", rawForm)
	}

	opt := getOptDef(&params.Pay{}, opts...)
	req := &tg.PaymentsSendPaymentFormRequest{
		FormID:           form.FormID,
		Invoice:          invoice,
		ShippingOptionID: opt.ShippingOptionID,
		TipAmount:        opt.TipAmount,
	}

	if opt.RequestedInfo != nil {
		validated, err := c.ValidateRequestedInfo(ctx, invoice, opt.RequestedInfo, opt.SaveInfo)
		if err != nil {
			return nil, fmt.Errorf("validate requested info: %w", err)
		}
		req.RequestedInfoID = validated.ID
	}

	switch {
	case opt.SavedCredentialID != "":
		req.Credentials = &tg.InputPaymentCredentialsSaved{
			ID:          opt.SavedCredentialID,
			TmpPassword: opt.TmpPassword,
		}
	case opt.CardData != nil:
		req.Credentials = &tg.InputPaymentCredentials{
			Save: opt.SaveCard,
			Data: opt.CardData,
		}
	default:
		req.Credentials = &tg.InputPaymentCredentials{
			Data: &tg.DataJSON{Data: "{}"},
		}
	}

	result, err := c.Raw().PaymentsSendPaymentForm(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("send payment form: %w", err)
	}
	switch v := result.(type) {
	case *tg.PaymentsPaymentResult:
		if msg, err := extractSingleMessage(v.Updates, c); err == nil && msg != nil {
			return msg, nil
		}
		return nil, fmt.Errorf("payment sent: receipt message not found in updates")
	case *tg.PaymentsPaymentVerificationNeeded:
		return nil, &PaymentVerificationNeededError{URL: v.URL}
	default:
		return nil, fmt.Errorf("send payment form: unexpected result %T", result)
	}
}
