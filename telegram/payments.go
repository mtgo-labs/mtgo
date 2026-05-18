package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) SendInvoice(ctx context.Context, chatID int64, invoice *tg.InputMediaInvoice, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendInvoice chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
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
		return nil, err
	}
	return extractSingleMessage(result, c)
}

func (c *Client) CreateInvoiceLink(ctx context.Context, invoice *tg.InputMediaInvoice) (string, error) {
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
