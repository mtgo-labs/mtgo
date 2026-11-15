package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func sendReplyOpt(replyTo int32, opts ...*params.SendMessage) *params.SendMessage {
	opt := params.GetOptDef(&params.SendMessage{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return opt
}

func (c *Client) BoundReplyText(ctx context.Context, chatID int64, text string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMessage(ctx, chatID, text, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswer(ctx context.Context, chatID int64, text string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMessage(ctx, chatID, text, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyPhoto(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendPhoto) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendPhoto{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendPhoto(ctx, chatID, file, caption, opt)
}

func (c *Client) BoundAnswerPhoto(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendPhoto) (*types.Message, error) {
	return c.SendPhoto(ctx, chatID, file, caption, opts...)
}

func (c *Client) BoundReplyAudio(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendAudio) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendAudio{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendAudio(ctx, chatID, file, caption, opt)
}

func (c *Client) BoundAnswerAudio(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAudio) (*types.Message, error) {
	return c.SendAudio(ctx, chatID, file, caption, opts...)
}

func (c *Client) BoundReplyDocument(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendDocument) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendDocument{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendDocument(ctx, chatID, file, caption, opt)
}

func (c *Client) BoundAnswerDocument(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*types.Message, error) {
	return c.SendDocument(ctx, chatID, file, caption, opts...)
}

func (c *Client) BoundReplyAnimation(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendAnimation) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendAnimation{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendAnimation(ctx, chatID, file, caption, opt)
}

func (c *Client) BoundAnswerAnimation(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAnimation) (*types.Message, error) {
	return c.SendAnimation(ctx, chatID, file, caption, opts...)
}

func (c *Client) BoundReplyVoice(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendVoice) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendVoice{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendVoice(ctx, chatID, file, caption, opt)
}

func (c *Client) BoundAnswerVoice(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVoice) (*types.Message, error) {
	return c.SendVoice(ctx, chatID, file, caption, opts...)
}

func (c *Client) BoundReplyVideoNote(ctx context.Context, chatID int64, file *InputFile, replyTo int32, opts ...*params.SendVideoNote) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendVideoNote{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendVideoNote(ctx, chatID, file, opt)
}

func (c *Client) BoundAnswerVideoNote(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendVideoNote) (*types.Message, error) {
	return c.SendVideoNote(ctx, chatID, file, opts...)
}

func (c *Client) BoundReplySticker(ctx context.Context, chatID int64, file *InputFile, replyTo int32, opts ...*params.SendSticker) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendSticker{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendSticker(ctx, chatID, file, opt)
}

func (c *Client) BoundAnswerSticker(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendSticker) (*types.Message, error) {
	return c.SendSticker(ctx, chatID, file, opts...)
}

func (c *Client) BoundReplyVideo(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendVideo) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendVideo{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendVideo(ctx, chatID, file, caption, opt)
}

func (c *Client) BoundAnswerVideo(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVideo) (*types.Message, error) {
	return c.SendVideo(ctx, chatID, file, caption, opts...)
}

func (c *Client) BoundReplyCachedMedia(ctx context.Context, chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendDocument) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendDocument{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendDocument(ctx, chatID, file, caption, opt)
}

func (c *Client) BoundAnswerCachedMedia(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*types.Message, error) {
	return c.SendDocument(ctx, chatID, file, caption, opts...)
}

func (c *Client) BoundReplyPaidMedia(ctx context.Context, chatID int64, starsAmount int64, media []tg.InputMediaClass, caption string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	paid := &tg.InputMediaPaidMedia{
		StarsAmount:   starsAmount,
		ExtendedMedia: media,
	}
	return c.SendMedia(ctx, chatID, paid, caption, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerPaidMedia(ctx context.Context, chatID int64, starsAmount int64, media []tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	paid := &tg.InputMediaPaidMedia{
		StarsAmount:   starsAmount,
		ExtendedMedia: media,
	}
	return c.SendMedia(ctx, chatID, paid, caption, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyMediaGroup(ctx context.Context, chatID int64, media []tg.InputMediaClass, replyTo int32, opts ...*params.SendMessage) ([]*types.Message, error) {
	opt := sendReplyOpt(replyTo, opts...)
	items := make([]*tg.InputSingleMedia, len(media))
	for i, m := range media {
		items[i] = &tg.InputSingleMedia{
			Media:    m,
			RandomID: c.RandomID(),
		}
	}
	return c.SendMediaGroup(ctx, chatID, items, opt)
}

func (c *Client) BoundAnswerMediaGroup(ctx context.Context, chatID int64, media []tg.InputMediaClass, opts ...*params.SendMessage) ([]*types.Message, error) {
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	items := make([]*tg.InputSingleMedia, len(media))
	for i, m := range media {
		items[i] = &tg.InputSingleMedia{
			Media:    m,
			RandomID: c.RandomID(),
		}
	}
	return c.SendMediaGroup(ctx, chatID, items, opt)
}

func (c *Client) BoundReplyContact(ctx context.Context, chatID int64, phone, firstName, lastName string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendContact(ctx, chatID, phone, firstName, lastName, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerContact(ctx context.Context, chatID int64, phone, firstName, lastName string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendContact(ctx, chatID, phone, firstName, lastName, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyLocation(ctx context.Context, chatID int64, lat, lng float64, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendLocation(ctx, chatID, lat, lng, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerLocation(ctx context.Context, chatID int64, lat, lng float64, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendLocation(ctx, chatID, lat, lng, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyVenue(ctx context.Context, chatID int64, lat, lng float64, title, address string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendVenue(ctx, chatID, lat, lng, title, address, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerVenue(ctx context.Context, chatID int64, lat, lng float64, title, address string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendVenue(ctx, chatID, lat, lng, title, address, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyPoll(ctx context.Context, chatID int64, question string, options []string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendPoll(ctx, chatID, question, options, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerPoll(ctx context.Context, chatID int64, question string, options []string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendPoll(ctx, chatID, question, options, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyDice(ctx context.Context, chatID int64, emoji string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	emoticon := "\U0001F3B2"
	if emoji != "" {
		emoticon = emoji
	}
	media := &tg.InputMediaDice{Emoticon: emoticon}
	return c.SendMedia(ctx, chatID, media, "", sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerDice(ctx context.Context, chatID int64, emoji string, opts ...*params.SendMessage) (*types.Message, error) {
	emoticon := "\U0001F3B2"
	if emoji != "" {
		emoticon = emoji
	}
	media := &tg.InputMediaDice{Emoticon: emoticon}
	return c.SendMedia(ctx, chatID, media, "", params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyGame(ctx context.Context, chatID int64, gameShortName string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendGame(ctx, chatID, gameShortName, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerGame(ctx context.Context, chatID int64, gameShortName string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendGame(ctx, chatID, gameShortName, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyInvoice(ctx context.Context, chatID int64, invoice *tg.InputMediaInvoice, caption string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendInvoice(ctx, chatID, invoice, caption, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerInvoice(ctx context.Context, chatID int64, invoice *tg.InputMediaInvoice, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendInvoice(ctx, chatID, invoice, caption, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyChatAction(ctx context.Context, chatID int64, action tg.SendMessageActionClass) error {
	return c.SendChatAction(ctx, chatID, action)
}

func (c *Client) BoundReplyInlineBotResult(ctx context.Context, chatID int64, queryID int64, resultID string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	opt := sendReplyOpt(replyTo, opts...)
	inlineOpt := &SendInlineBotResultOption{
		ReplyTo:      int64(opt.ReplyToMessageID),
		Silent:       opt.DisableNotification || opt.Silent,
		ScheduleDate: opt.ScheduleDate,
		ClearDraft:   opt.ClearDraft,
	}
	return c.SendInlineBotResult(ctx, chatID, queryID, resultID, inlineOpt)
}

func (c *Client) BoundAnswerInlineBotResult(ctx context.Context, chatID int64, queryID int64, resultID string, opts ...*params.SendMessage) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	inlineOpt := &SendInlineBotResultOption{
		Silent:       opt.DisableNotification || opt.Silent,
		ScheduleDate: opt.ScheduleDate,
		ClearDraft:   opt.ClearDraft,
	}
	return c.SendInlineBotResult(ctx, chatID, queryID, resultID, inlineOpt)
}

func (c *Client) BoundForwardMediaGroup(ctx context.Context, chatID int64, fromChatID int64, msgIDs []int32, opts ...*params.ForwardMessages) ([]*types.Message, error) {
	opt := params.GetOptDef(&params.ForwardMessages{}, opts...)
	return c.ForwardMediaGroup(ctx, chatID, fromChatID, msgIDs, opt)
}

func (c *Client) BoundClick(ctx context.Context, chatID int64, msgID int32, data []byte) (*tg.MessagesBotCallbackAnswer, error) {
	peer, err := resolvePeer(ctx, c, chatID)
	if err != nil {
		return nil, err
	}
	return c.Raw().MessagesGetBotCallbackAnswer(ctx, &tg.MessagesGetBotCallbackAnswerRequest{
		Flags: func() tg.Fields {
			var f tg.Fields
			if len(data) > 0 {
				f.Set(0)
			}
			return f
		}(),
		Peer:  peer,
		MsgID: msgID,
		Data:  data,
	})
}

func (c *Client) BoundPay(ctx context.Context, chatID int64, msgID int32) (tg.PaymentResultClass, error) {
	form, err := c.GetPaymentForm(ctx, chatID, msgID)
	if err != nil {
		return nil, err
	}
	var formID int64
	if form != nil {
		formID = form.ID
	}
	creds := &tg.InputPaymentCredentials{
		Data: &tg.DataJSON{Data: "{}"},
	}
	return c.SendPaymentForm(ctx, formID, chatID, msgID, creds)
}

func (c *Client) BoundView(ctx context.Context, chatID int64, msgID int32) error {
	return c.ReadHistory(ctx, chatID, msgID)
}

func (c *Client) BoundEditLiveLocation(ctx context.Context, chatID int64, msgID int32, lat, lng float64) (*types.Message, error) {
	peer, err := resolvePeer(ctx, c, chatID)
	if err != nil {
		return nil, err
	}
	result, err := c.Raw().MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
		Peer: peer,
		ID:   msgID,
		Media: &tg.InputMediaGeoLive{
			GeoPoint: &tg.InputGeoPoint{Lat: lat, Long: lng},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("edit message: %w", err)
	}
	return extractSingleMessage(result, c)
}

func (c *Client) BoundStopLiveLocation(ctx context.Context, chatID int64, msgID int32) (*types.Message, error) {
	peer, err := resolvePeer(ctx, c, chatID)
	if err != nil {
		return nil, err
	}
	result, err := c.Raw().MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
		Peer: peer,
		ID:   msgID,
		Media: &tg.InputMediaGeoLive{
			Flags:   1 << 0,
			Stopped: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("edit message: %w", err)
	}
	return extractSingleMessage(result, c)
}

func (c *Client) BoundAcceptGiftPurchaseOffer(ctx context.Context, chatID int64, msgID int32) (*types.Message, error) {
	result, err := c.Raw().PaymentsResolveStarGiftOffer(ctx, &tg.PaymentsResolveStarGiftOfferRequest{
		OfferMsgID: msgID,
	})
	if err != nil {
		return nil, fmt.Errorf("payments resolve star gift offer: %w", err)
	}
	return extractSingleMessage(result, c)
}

func (c *Client) BoundRejectGiftPurchaseOffer(ctx context.Context, chatID int64, msgID int32) (*types.Message, error) {
	result, err := c.Raw().PaymentsResolveStarGiftOffer(ctx, &tg.PaymentsResolveStarGiftOfferRequest{
		Flags:      1 << 0,
		Decline:    true,
		OfferMsgID: msgID,
	})
	if err != nil {
		return nil, fmt.Errorf("payments resolve star gift offer: %w", err)
	}
	return extractSingleMessage(result, c)
}

func (c *Client) BoundSummarize(ctx context.Context, chatID int64, msgID int32) (*types.Message, error) {
	peer, err := resolvePeer(ctx, c, chatID)
	if err != nil {
		return nil, err
	}
	_, err = c.Raw().MessagesGetExtendedMedia(ctx, &tg.MessagesGetExtendedMediaRequest{
		Peer: peer,
		ID:   []int32{msgID},
	})
	if err != nil {
		return nil, fmt.Errorf("get extended media: %w", err)
	}
	return nil, nil
}

func (c *Client) BoundReplyChecklist(ctx context.Context, chatID int64, checklist *tg.InputMediaTodo, replyTo int32, opts ...*params.SendChecklist) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendChecklist{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendMedia(ctx, chatID, checklist, "", opt.ToSendMsg())
}

func (c *Client) BoundEditChecklist(ctx context.Context, chatID int64, msgID int32, media tg.InputMediaClass, opts ...*params.EditMessage) (*types.Message, error) {
	return c.EditMessageMedia(ctx, chatID, msgID, media, opts...)
}
