package telegram

import (
	"context"

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

func (c *Client) BoundReplyText(chatID int64, text string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMessage(context.Background(), chatID, text, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswer(chatID int64, text string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendMessage(context.Background(), chatID, text, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyPhoto(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendPhoto) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendPhoto{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendPhoto(context.Background(), chatID, file, caption, opt)
}

func (c *Client) BoundAnswerPhoto(chatID int64, file *InputFile, caption string, opts ...*params.SendPhoto) (*types.Message, error) {
	return c.SendPhoto(context.Background(), chatID, file, caption, opts...)
}

func (c *Client) BoundReplyAudio(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendAudio) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendAudio{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendAudio(context.Background(), chatID, file, caption, opt)
}

func (c *Client) BoundAnswerAudio(chatID int64, file *InputFile, caption string, opts ...*params.SendAudio) (*types.Message, error) {
	return c.SendAudio(context.Background(), chatID, file, caption, opts...)
}

func (c *Client) BoundReplyDocument(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendDocument) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendDocument{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendDocument(context.Background(), chatID, file, caption, opt)
}

func (c *Client) BoundAnswerDocument(chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*types.Message, error) {
	return c.SendDocument(context.Background(), chatID, file, caption, opts...)
}

func (c *Client) BoundReplyAnimation(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendAnimation) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendAnimation{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendAnimation(context.Background(), chatID, file, caption, opt)
}

func (c *Client) BoundAnswerAnimation(chatID int64, file *InputFile, caption string, opts ...*params.SendAnimation) (*types.Message, error) {
	return c.SendAnimation(context.Background(), chatID, file, caption, opts...)
}

func (c *Client) BoundReplyVoice(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendVoice) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendVoice{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendVoice(context.Background(), chatID, file, caption, opt)
}

func (c *Client) BoundAnswerVoice(chatID int64, file *InputFile, caption string, opts ...*params.SendVoice) (*types.Message, error) {
	return c.SendVoice(context.Background(), chatID, file, caption, opts...)
}

func (c *Client) BoundReplyVideoNote(chatID int64, file *InputFile, replyTo int32, opts ...*params.SendVideoNote) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendVideoNote{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendVideoNote(context.Background(), chatID, file, opt)
}

func (c *Client) BoundAnswerVideoNote(chatID int64, file *InputFile, opts ...*params.SendVideoNote) (*types.Message, error) {
	return c.SendVideoNote(context.Background(), chatID, file, opts...)
}

func (c *Client) BoundReplySticker(chatID int64, file *InputFile, replyTo int32, opts ...*params.SendSticker) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendSticker{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendSticker(context.Background(), chatID, file, opt)
}

func (c *Client) BoundAnswerSticker(chatID int64, file *InputFile, opts ...*params.SendSticker) (*types.Message, error) {
	return c.SendSticker(context.Background(), chatID, file, opts...)
}

func (c *Client) BoundReplyVideo(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendVideo) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendVideo{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendVideo(context.Background(), chatID, file, caption, opt)
}

func (c *Client) BoundAnswerVideo(chatID int64, file *InputFile, caption string, opts ...*params.SendVideo) (*types.Message, error) {
	return c.SendVideo(context.Background(), chatID, file, caption, opts...)
}

func (c *Client) BoundReplyCachedMedia(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendDocument) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendDocument{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendDocument(context.Background(), chatID, file, caption, opt)
}

func (c *Client) BoundAnswerCachedMedia(chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*types.Message, error) {
	return c.SendDocument(context.Background(), chatID, file, caption, opts...)
}

func (c *Client) BoundReplyPaidMedia(chatID int64, starsAmount int64, media []tg.InputMediaClass, caption string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	paid := &tg.InputMediaPaidMedia{
		StarsAmount:   starsAmount,
		ExtendedMedia: media,
	}
	return c.SendMedia(context.Background(), chatID, paid, caption, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerPaidMedia(chatID int64, starsAmount int64, media []tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	paid := &tg.InputMediaPaidMedia{
		StarsAmount:   starsAmount,
		ExtendedMedia: media,
	}
	return c.SendMedia(context.Background(), chatID, paid, caption, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyMediaGroup(chatID int64, media []tg.InputMediaClass, replyTo int32, opts ...*params.SendMessage) ([]*types.Message, error) {
	opt := sendReplyOpt(replyTo, opts...)
	items := make([]*tg.InputSingleMedia, len(media))
	for i, m := range media {
		items[i] = &tg.InputSingleMedia{
			Media:    m,
			RandomID: c.RandomID(),
		}
	}
	return c.SendMediaGroup(context.Background(), chatID, items, opt)
}

func (c *Client) BoundAnswerMediaGroup(chatID int64, media []tg.InputMediaClass, opts ...*params.SendMessage) ([]*types.Message, error) {
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	items := make([]*tg.InputSingleMedia, len(media))
	for i, m := range media {
		items[i] = &tg.InputSingleMedia{
			Media:    m,
			RandomID: c.RandomID(),
		}
	}
	return c.SendMediaGroup(context.Background(), chatID, items, opt)
}

func (c *Client) BoundReplyContact(chatID int64, phone, firstName, lastName string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendContact(context.Background(), chatID, phone, firstName, lastName, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerContact(chatID int64, phone, firstName, lastName string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendContact(context.Background(), chatID, phone, firstName, lastName, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyLocation(chatID int64, lat, lng float64, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendLocation(context.Background(), chatID, lat, lng, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerLocation(chatID int64, lat, lng float64, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendLocation(context.Background(), chatID, lat, lng, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyVenue(chatID int64, lat, lng float64, title, address string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendVenue(context.Background(), chatID, lat, lng, title, address, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerVenue(chatID int64, lat, lng float64, title, address string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendVenue(context.Background(), chatID, lat, lng, title, address, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyPoll(chatID int64, question string, options []string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendPoll(context.Background(), chatID, question, options, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerPoll(chatID int64, question string, options []string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendPoll(context.Background(), chatID, question, options, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyDice(chatID int64, emoji string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	emoticon := "\U0001F3B2"
	if emoji != "" {
		emoticon = emoji
	}
	media := &tg.InputMediaDice{Emoticon: emoticon}
	return c.SendMedia(context.Background(), chatID, media, "", sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerDice(chatID int64, emoji string, opts ...*params.SendMessage) (*types.Message, error) {
	emoticon := "\U0001F3B2"
	if emoji != "" {
		emoticon = emoji
	}
	media := &tg.InputMediaDice{Emoticon: emoticon}
	return c.SendMedia(context.Background(), chatID, media, "", params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyGame(chatID int64, gameShortName string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendGame(context.Background(), chatID, gameShortName, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerGame(chatID int64, gameShortName string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendGame(context.Background(), chatID, gameShortName, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyInvoice(chatID int64, invoice *tg.InputMediaInvoice, caption string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendInvoice(context.Background(), chatID, invoice, caption, sendReplyOpt(replyTo, opts...))
}

func (c *Client) BoundAnswerInvoice(chatID int64, invoice *tg.InputMediaInvoice, caption string, opts ...*params.SendMessage) (*types.Message, error) {
	return c.SendInvoice(context.Background(), chatID, invoice, caption, params.GetOptDef(&params.SendMessage{}, opts...))
}

func (c *Client) BoundReplyChatAction(chatID int64, action tg.SendMessageActionClass) error {
	return c.SendChatAction(context.Background(), chatID, action)
}

func (c *Client) BoundReplyInlineBotResult(chatID int64, queryID int64, resultID string, replyTo int32, opts ...*params.SendMessage) (*types.Message, error) {
	opt := sendReplyOpt(replyTo, opts...)
	inlineOpt := &SendInlineBotResultOption{
		ReplyTo:      int64(opt.ReplyToMessageID),
		Silent:       opt.DisableNotification || opt.Silent,
		ScheduleDate: opt.ScheduleDate,
		ClearDraft:   opt.ClearDraft,
	}
	return c.SendInlineBotResult(context.Background(), chatID, queryID, resultID, inlineOpt)
}

func (c *Client) BoundAnswerInlineBotResult(chatID int64, queryID int64, resultID string, opts ...*params.SendMessage) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	inlineOpt := &SendInlineBotResultOption{
		Silent:       opt.DisableNotification || opt.Silent,
		ScheduleDate: opt.ScheduleDate,
		ClearDraft:   opt.ClearDraft,
	}
	return c.SendInlineBotResult(context.Background(), chatID, queryID, resultID, inlineOpt)
}

func (c *Client) BoundForwardMediaGroup(chatID int64, fromChatID int64, msgIDs []int32, opts ...*params.ForwardMessages) ([]*types.Message, error) {
	opt := params.GetOptDef(&params.ForwardMessages{}, opts...)
	return c.ForwardMediaGroup(context.Background(), chatID, fromChatID, msgIDs, opt)
}

func (c *Client) BoundClick(chatID int64, msgID int32, data []byte) (*tg.MessagesBotCallbackAnswer, error) {
	ctx := context.Background()
	peer, err := resolvePeer(c, chatID)
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

func (c *Client) BoundPay(chatID int64, msgID int32) (tg.PaymentResultClass, error) {
	ctx := context.Background()
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

func (c *Client) BoundView(chatID int64, msgID int32) error {
	ctx := context.Background()
	return c.ReadHistory(ctx, chatID, msgID)
}

func (c *Client) BoundEditLiveLocation(chatID int64, msgID int32, lat, lng float64) (*types.Message, error) {
	ctx := context.Background()
	peer, err := resolvePeer(c, chatID)
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
		return nil, err
	}
	return extractSingleMessage(result, c)
}

func (c *Client) BoundStopLiveLocation(chatID int64, msgID int32) (*types.Message, error) {
	ctx := context.Background()
	peer, err := resolvePeer(c, chatID)
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
		return nil, err
	}
	return extractSingleMessage(result, c)
}

func (c *Client) BoundAcceptGiftPurchaseOffer(chatID int64, msgID int32) (*types.Message, error) {
	ctx := context.Background()
	result, err := c.Raw().PaymentsResolveStarGiftOffer(ctx, &tg.PaymentsResolveStarGiftOfferRequest{
		OfferMsgID: msgID,
	})
	if err != nil {
		return nil, err
	}
	return extractSingleMessage(result, c)
}

func (c *Client) BoundRejectGiftPurchaseOffer(chatID int64, msgID int32) (*types.Message, error) {
	ctx := context.Background()
	result, err := c.Raw().PaymentsResolveStarGiftOffer(ctx, &tg.PaymentsResolveStarGiftOfferRequest{
		Flags:      1 << 0,
		Decline:    true,
		OfferMsgID: msgID,
	})
	if err != nil {
		return nil, err
	}
	return extractSingleMessage(result, c)
}

func (c *Client) BoundSummarize(chatID int64, msgID int32) (*types.Message, error) {
	ctx := context.Background()
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, err
	}
	_, err = c.Raw().MessagesGetExtendedMedia(ctx, &tg.MessagesGetExtendedMediaRequest{
		Peer: peer,
		ID:   []int32{msgID},
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *Client) BoundReplyChecklist(chatID int64, checklist *tg.InputMediaTodo, replyTo int32, opts ...*params.SendChecklist) (*types.Message, error) {
	opt := params.GetOptDef(&params.SendChecklist{ReplyToMessageID: replyTo}, opts...)
	if opt.ReplyToMessageID == 0 {
		opt.ReplyToMessageID = replyTo
	}
	return c.SendMedia(context.Background(), chatID, checklist, "", opt.ToSendMsg())
}

func (c *Client) BoundEditChecklist(chatID int64, msgID int32, media tg.InputMediaClass, opts ...*params.EditMessage) (*types.Message, error) {
	return c.EditMessageMedia(context.Background(), chatID, msgID, media, opts...)
}
