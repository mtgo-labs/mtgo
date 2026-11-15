package types

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

// mockBinder implements the binder interfaces with recording for the flows the
// tests observe.
type mockBinder struct {
	sendChatID           int64
	sendText             string
	sendReplyTo          int32
	deletedChatID        int64
	deletedMsgIDs        []int32
	pinnedChatID         int64
	pinnedMsgID          int32
	mediaCaption         string
	mediaReplyTo         int32
	editMedia            tg.InputMediaClass
	answerID             int64
	answerText           string
	answerAlert          bool
	clickedData          []byte
	giftOfferDeclined    bool
	widgetPublicData     string
	widgetChatData       string
	answeredWidgetID     int64
	answeredWidgetResult string
}

func firstReplyTo(opts ...*params.SendMessage) int32 {
	for _, o := range opts {
		if o != nil {
			return o.ReplyToMessageID
		}
	}
	return 0
}

func firstAnimReplyTo(opts ...*params.SendAnimation) int32 {
	for _, o := range opts {
		if o != nil {
			return o.ReplyToMessageID
		}
	}
	return 0
}

func firstPhotoReplyTo(opts ...*params.SendPhoto) int32 {
	for _, o := range opts {
		if o != nil {
			return o.ReplyToMessageID
		}
	}
	return 0
}

func (m *mockBinder) ResetSession(ctx context.Context, hash int64) error {
	return nil
}

func (m *mockBinder) AnswerCallbackQuery(ctx context.Context, callbackQueryID int64, text string, showAlert bool, url string, cacheTime int) error {
	m.answerID, m.answerText, m.answerAlert = callbackQueryID, text, showAlert
	return nil
}

func (m *mockBinder) DeleteMessages(ctx context.Context, chatID int64, messageIDs []int32, opts ...*params.DeleteMessages) (int, error) {
	m.deletedChatID, m.deletedMsgIDs = chatID, messageIDs
	return len(messageIDs), nil
}

func (m *mockBinder) EditInlineCaption(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, caption string, opts ...*params.EditInlineOpts) (bool, error) {
	return false, nil
}

func (m *mockBinder) EditInlineMedia(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, media tg.InputMediaClass, opts ...*params.EditInlineOpts) (bool, error) {
	return false, nil
}

func (m *mockBinder) EditInlineReplyMarkup(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, replyMarkup tg.ReplyMarkupClass) (bool, error) {
	return false, nil
}

func (m *mockBinder) EditInlineRichMessage(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, rm tg.InputRichMessageClass, opts ...*params.EditInlineOpts) (bool, error) {
	return false, nil
}

func (m *mockBinder) EditInlineText(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, text string, opts ...*params.EditInlineOpts) (bool, error) {
	return false, nil
}

func (m *mockBinder) EditMessageCaption(ctx context.Context, chatID int64, messageID int32, caption string, opts ...*params.EditMessage) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) EditMessageMedia(ctx context.Context, chatID int64, messageID int32, media tg.InputMediaClass, opts ...*params.EditMessage) (*Message, error) {
	m.editMedia = media
	return &Message{ID: 99}, nil
}

func (m *mockBinder) EditMessageReplyMarkup(ctx context.Context, chatID int64, messageID int32, replyMarkup tg.ReplyMarkupClass) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) EditMessageText(ctx context.Context, chatID int64, messageID int32, text string, opts ...*params.EditMessage) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendMessage(ctx context.Context, chatID int64, text string, opts ...*params.SendMessage) (*Message, error) {
	m.sendChatID, m.sendText, m.sendReplyTo = chatID, text, firstReplyTo(opts...)
	return &Message{ID: 99}, nil
}

func (m *mockBinder) AddChatMember(ctx context.Context, chatID int64, userID int64) error {
	return nil
}

func (m *mockBinder) ArchiveChat(ctx context.Context, chatID int64) error {
	return nil
}

func (m *mockBinder) BanChatMember(ctx context.Context, chatID int64, userID int64) error {
	return nil
}

func (m *mockBinder) DeleteChatPhoto(ctx context.Context, chatID int64) error {
	return nil
}

func (m *mockBinder) ExportChatInviteLink(ctx context.Context, chatID int64) (string, error) {
	return "", nil
}

func (m *mockBinder) GetChat(ctx context.Context, chatID int64) (*Chat, error) {
	return nil, nil
}

func (m *mockBinder) GetChatEventLog(ctx context.Context, chatID int64, query string, limit int) ([]*ChatEvent, error) {
	return nil, nil
}

func (m *mockBinder) GetChatMember(ctx context.Context, chatID int64, userID int64) (*ChatMember, error) {
	return nil, nil
}

func (m *mockBinder) GetChatMembers(ctx context.Context, chatID int64, limit int, offset int) ([]*ChatMember, error) {
	return nil, nil
}

func (m *mockBinder) JoinChat(ctx context.Context, link string) (*Chat, error) {
	return nil, nil
}

func (m *mockBinder) LeaveChat(ctx context.Context, chatID int64) error {
	return nil
}

func (m *mockBinder) MarkChatUnread(ctx context.Context, chatID int64, unread bool) error {
	return nil
}

func (m *mockBinder) MuteChat(ctx context.Context, chatID int64) error {
	return nil
}

func (m *mockBinder) PromoteChatMember(ctx context.Context, chatID int64, userID int64, adminRights *tg.ChatAdminRights) error {
	return nil
}

func (m *mockBinder) RestrictChatMember(ctx context.Context, chatID int64, userID int64, bannedRights *tg.ChatBannedRights) error {
	return nil
}

func (m *mockBinder) SetAdministratorTitle(ctx context.Context, chatID int64, userID int64, title string) error {
	return nil
}

func (m *mockBinder) SetChatDescription(ctx context.Context, chatID int64, about string) error {
	return nil
}

func (m *mockBinder) SetChatPermissions(ctx context.Context, chatID int64, permissions *tg.ChatBannedRights) error {
	return nil
}

func (m *mockBinder) SetChatPhoto(ctx context.Context, chatID int64, photo tg.InputChatPhotoClass) error {
	return nil
}

func (m *mockBinder) SetChatTTL(ctx context.Context, chatID int64, ttl int) error {
	return nil
}

func (m *mockBinder) SetChatTitle(ctx context.Context, chatID int64, title string) error {
	return nil
}

func (m *mockBinder) SetChatUsername(ctx context.Context, chatID int64, username string) error {
	return nil
}

func (m *mockBinder) SetProtectedContent(ctx context.Context, chatID int64, enabled bool) error {
	return nil
}

func (m *mockBinder) SetSlowMode(ctx context.Context, chatID int64, seconds int) error {
	return nil
}

func (m *mockBinder) UnarchiveChat(ctx context.Context, chatID int64) error {
	return nil
}

func (m *mockBinder) UnbanChatMember(ctx context.Context, chatID int64, userID int64) error {
	return nil
}

func (m *mockBinder) UnmuteChat(ctx context.Context, chatID int64) error {
	return nil
}

func (m *mockBinder) UnpinAllMessages(ctx context.Context, chatID int64) (int, error) {
	return 0, nil
}

func (m *mockBinder) ApproveChatJoinRequest(ctx context.Context, chatID int64, userID int64) error {
	return nil
}

func (m *mockBinder) DeclineChatJoinRequest(ctx context.Context, chatID int64, userID int64) error {
	return nil
}

func (m *mockBinder) ConvertGift(ctx context.Context, gift tg.InputSavedStarGiftClass) error {
	return nil
}

func (m *mockBinder) HideGift(ctx context.Context, gift tg.InputSavedStarGiftClass) error {
	return nil
}

func (m *mockBinder) ShowGift(ctx context.Context, gift tg.InputSavedStarGiftClass) error {
	return nil
}

func (m *mockBinder) TransferGift(ctx context.Context, gift tg.InputSavedStarGiftClass, recipientID int64) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) UpgradeGift(ctx context.Context, gift tg.InputSavedStarGiftClass, keepOriginalDetails bool) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) AnswerInlineQuery(ctx context.Context, queryID int64, results []tg.InputBotInlineResultClass, opts ...*params.AnswerInlineQueryOption) error {
	return nil
}

func (m *mockBinder) CopyMediaGroup(ctx context.Context, chatID int64, fromChatID int64, groupedID int64) ([]*Message, error) {
	return nil, nil
}

func (m *mockBinder) CopyMessage(ctx context.Context, chatID int64, fromChatID int64, messageID int32, opts ...*params.CopyMessage) (int64, error) {
	return 0, nil
}

func (m *mockBinder) DownloadMedia(ctx context.Context, media Media, thumbSize string, opts *params.Download) ([]byte, error) {
	return nil, nil
}

func (m *mockBinder) ForwardMessages(ctx context.Context, chatID int64, fromChatID int64, messageIDs []int32, opts ...*params.ForwardMessages) ([]*Message, error) {
	return nil, nil
}

func (m *mockBinder) GetMediaGroup(ctx context.Context, chatID int64, messageID int32) ([]*Message, error) {
	return nil, nil
}

func (m *mockBinder) GetMessages(ctx context.Context, chatID int64, messageIDs []int32) ([]*Message, error) {
	return nil, nil
}

func (m *mockBinder) PinMessage(ctx context.Context, chatID int64, messageID int32, opts ...*params.PinMessage) (*Message, error) {
	m.pinnedChatID, m.pinnedMsgID = chatID, messageID
	return nil, nil
}

func (m *mockBinder) RandomID() int64 {
	return 0
}

func (m *mockBinder) ReadChatHistory(ctx context.Context, chatID int64, maxID int32) error {
	return nil
}

func (m *mockBinder) RetractVote(ctx context.Context, chatID int64, messageID int32) error {
	return nil
}

func (m *mockBinder) SendAnimation(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAnimation) (*Message, error) {
	m.mediaCaption, m.mediaReplyTo = caption, firstAnimReplyTo(opts...)
	return &Message{ID: 99}, nil
}

func (m *mockBinder) SendAudio(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAudio) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendChatAction(ctx context.Context, chatID int64, action tg.SendMessageActionClass) error {
	return nil
}

func (m *mockBinder) SendContact(ctx context.Context, chatID int64, phoneNumber, firstName, lastName string, opts ...*params.SendContact) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendDocument(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendGame(ctx context.Context, chatID int64, gameShortName string, opts ...*params.SendMessage) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendInlineBotResult(ctx context.Context, chatID int64, queryID int64, resultID string, opts ...*params.SendInlineBotResultOption) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendLocation(ctx context.Context, chatID int64, lat, lng float64, opts ...*params.SendLocation) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendMedia(ctx context.Context, chatID int64, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error) {
	m.sendChatID, m.mediaCaption, m.mediaReplyTo = chatID, caption, firstReplyTo(opts...)
	return &Message{ID: 100}, nil
}

func (m *mockBinder) SendMediaGroup(ctx context.Context, chatID int64, items []*tg.InputSingleMedia, opts ...*params.SendMessage) ([]*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendPhoto(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendPhoto) (*Message, error) {
	m.mediaCaption, m.mediaReplyTo = caption, firstPhotoReplyTo(opts...)
	return &Message{ID: 99}, nil
}

func (m *mockBinder) SendPoll(ctx context.Context, chatID int64, question string, options []string, opts ...*params.SendPoll) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendReaction(ctx context.Context, chatID int64, messageID int32, reactions []Reaction, opts ...*params.SendReactionOption) error {
	return nil
}

func (m *mockBinder) SendRichMessage(ctx context.Context, chatID int64, rm tg.InputRichMessageClass, opts ...*params.SendMessage) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendSticker(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendSticker) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendVenue(ctx context.Context, chatID int64, lat, lng float64, title, address string, opts ...*params.SendVenue) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendVideo(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVideo) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendVideoNote(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendVideoNote) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SendVoice(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVoice) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) SummarizeMessage(ctx context.Context, chatID int64, messageID int32) (*tg.TextWithEntities, error) {
	return nil, nil
}

func (m *mockBinder) UnpinMessage(ctx context.Context, chatID int64, messageID int32) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) VotePoll(ctx context.Context, chatID int64, messageID int32, options [][]byte) error {
	return nil
}

func (m *mockBinder) AnswerPreCheckoutQuery(ctx context.Context, queryID int64, ok bool, errorMessage string) error {
	return nil
}

func (m *mockBinder) AnswerShippingQuery(ctx context.Context, queryID int64, ok bool, options []*tg.ShippingOption) error {
	return nil
}

func (m *mockBinder) DeleteStories(ctx context.Context, chatID int64, storyIDs []int32) error {
	return nil
}

func (m *mockBinder) EditStoryCaption(ctx context.Context, chatID int64, storyID int32, caption string) (*Story, error) {
	return nil, nil
}

func (m *mockBinder) EditStoryMedia(ctx context.Context, chatID int64, storyID int32, media tg.InputMediaClass) (*Story, error) {
	return nil, nil
}

func (m *mockBinder) EditStoryPrivacy(ctx context.Context, peerID int64, storyID int32, opts ...*params.EditPrivacy) (*Story, error) {
	return nil, nil
}

func (m *mockBinder) ForwardStory(ctx context.Context, targetChatID int64, sourceChatID int64, storyID int32) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) GetStories(ctx context.Context, userID int64, storyIDs []int32) ([]*Story, error) {
	return nil, nil
}

func (m *mockBinder) ReadStories(ctx context.Context, peerID int64, maxStoryID int32) error {
	return nil
}

func (m *mockBinder) SendStoryReaction(ctx context.Context, peerID int64, storyID int32, opts ...*params.React) error {
	return nil
}

func (m *mockBinder) BlockUser(ctx context.Context, userID int64) error {
	return nil
}

func (m *mockBinder) GetCommonChats(ctx context.Context, userID int64, limit int) ([]*Chat, error) {
	return nil, nil
}

func (m *mockBinder) UnblockUser(ctx context.Context, userID int64) error {
	return nil
}

func TestMessage_Reply_NoBinder(t *testing.T) {
	m := &Message{ID: 1, ChatID: 42}
	_, err := m.Reply("hello")
	if !errors.Is(err, ErrNoBinder) {
		t.Fatalf("expected ErrNoBinder, got %v", err)
	}
}

func TestMessage_Reply_WithBinder(t *testing.T) {
	b := &mockBinder{}
	m := &Message{ID: 1, ChatID: 42}
	m.SetBinder(b)
	reply, err := m.Reply("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.sendChatID != 42 {
		t.Errorf("sendChatID = %d, want 42", b.sendChatID)
	}
	if b.sendText != "hello" {
		t.Errorf("sendText = %q, want %q", b.sendText, "hello")
	}
	if b.sendReplyTo != 1 {
		t.Errorf("sendReplyTo = %d, want 1", b.sendReplyTo)
	}
	if reply.ID != 99 {
		t.Errorf("reply ID = %d, want 99", reply.ID)
	}
}

func TestMessage_Send_NoReplyTo(t *testing.T) {
	b := &mockBinder{}
	m := &Message{ID: 1, ChatID: 42}
	m.SetBinder(b)
	_, err := m.Send("hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.sendReplyTo != 0 {
		t.Errorf("sendReplyTo = %d, want 0", b.sendReplyTo)
	}
}

func TestMessage_Delete(t *testing.T) {
	b := &mockBinder{}
	m := &Message{ID: 5, ChatID: 10}
	m.SetBinder(b)
	n, err := m.Delete()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted %d, want 1", n)
	}
	if b.deletedChatID != 10 {
		t.Errorf("deletedChatID = %d, want 10", b.deletedChatID)
	}
	if len(b.deletedMsgIDs) != 1 || b.deletedMsgIDs[0] != 5 {
		t.Errorf("deletedMsgIDs = %v, want [5]", b.deletedMsgIDs)
	}
}

func TestMessage_Pin(t *testing.T) {
	b := &mockBinder{}
	m := &Message{ID: 7, ChatID: 20}
	m.SetBinder(b)
	err := m.Pin()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.pinnedChatID != 20 || b.pinnedMsgID != 7 {
		t.Errorf("pin(%d, %d), want (20, 7)", b.pinnedChatID, b.pinnedMsgID)
	}
}

func TestMessage_ReplyMedia(t *testing.T) {
	b := &mockBinder{}
	m := &Message{ID: 3, ChatID: 50}
	m.SetBinder(b)
	media := &tg.InputMediaDocument{ID: &tg.InputDocument{ID: 100, AccessHash: 200}}
	reply, err := m.ReplyMedia(media, "caption")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.sendChatID != 50 {
		t.Errorf("sendChatID = %d, want 50", b.sendChatID)
	}
	if b.mediaCaption != "caption" {
		t.Errorf("caption = %q, want %q", b.mediaCaption, "caption")
	}
	if b.mediaReplyTo != 3 {
		t.Errorf("replyTo = %d, want 3", b.mediaReplyTo)
	}
	if reply.ID != 100 {
		t.Errorf("reply ID = %d, want 100", reply.ID)
	}
}

func TestMessage_SendMedia(t *testing.T) {
	b := &mockBinder{}
	m := &Message{ID: 3, ChatID: 50}
	m.SetBinder(b)
	media := &tg.InputMediaPhoto{ID: &tg.InputPhoto{ID: 10, AccessHash: 20}}
	_, err := m.SendMedia(media, "photo caption")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.mediaReplyTo != 0 {
		t.Errorf("replyTo = %d, want 0 (no reply)", b.mediaReplyTo)
	}
}

func TestMessage_EditMedia(t *testing.T) {
	b := &mockBinder{}
	m := &Message{ID: 5, ChatID: 30}
	m.SetBinder(b)
	media := &tg.InputMediaDocument{ID: &tg.InputDocument{ID: 999}}
	_, err := m.EditMedia(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.editMedia == nil {
		t.Fatal("editMedia should be set")
	}
}

func TestMessage_ReplyAnimation(t *testing.T) {
	b := &mockBinder{}
	m := &Message{ID: 1, ChatID: 40}
	m.SetBinder(b)
	reply, err := m.ReplyAnimation(nil, "anim caption")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.mediaReplyTo != 1 {
		t.Errorf("replyTo = %d, want 1", b.mediaReplyTo)
	}
	if b.mediaCaption != "anim caption" {
		t.Errorf("caption = %q, want %q", b.mediaCaption, "anim caption")
	}
	_ = reply
}

func TestMessage_ReplyPhoto(t *testing.T) {
	b := &mockBinder{}
	m := &Message{ID: 2, ChatID: 40}
	m.SetBinder(b)
	_, err := m.ReplyPhoto(nil, "photo caption")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.mediaCaption != "photo caption" {
		t.Errorf("caption = %q, want %q", b.mediaCaption, "photo caption")
	}
}

func TestCallbackQuery_Answer(t *testing.T) {
	b := &mockBinder{}
	q := &CallbackQuery{ID: 100}
	q.SetBinder(b)
	err := q.Answer("ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.answerID != 100 || b.answerText != "ok" || b.answerAlert {
		t.Errorf("answer(%d, %q, %v), want (100, %q, false)", b.answerID, b.answerText, b.answerAlert, "ok")
	}
}

func TestCallbackQuery_AnswerAlert(t *testing.T) {
	b := &mockBinder{}
	q := &CallbackQuery{ID: 200}
	q.SetBinder(b)
	err := q.AnswerAlert("warning!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !b.answerAlert {
		t.Error("expected showAlert = true")
	}
}

func TestCallbackQuery_NoBinder(t *testing.T) {
	q := &CallbackQuery{ID: 1}
	err := q.Answer("test")
	if !errors.Is(err, ErrNoBinder) {
		t.Fatalf("expected ErrNoBinder, got %v", err)
	}
}

func TestSetBinder(t *testing.T) {
	b := &mockBinder{}
	m := &Message{ID: 1}
	m.SetBinder(b)
	if m.binder == nil {
		t.Fatal("binder should be set")
	}
}

func (m *mockBinder) EditLiveLocation(ctx context.Context, chatID int64, messageID int32, lat, lng float64) (*Message, error) {
	return &Message{ID: messageID, ChatID: chatID}, nil
}

func (m *mockBinder) StopLiveLocation(ctx context.Context, chatID int64, messageID int32) (*Message, error) {
	return &Message{ID: messageID, ChatID: chatID}, nil
}

func (m *mockBinder) RequestCallbackAnswer(ctx context.Context, chatID int64, messageID int64, data []byte) (*tg.MessagesBotCallbackAnswer, error) {
	m.clickedData = data
	return &tg.MessagesBotCallbackAnswer{Message: "ok"}, nil
}

func (m *mockBinder) ResolveStarGiftOffer(ctx context.Context, offerMessageID int32, decline bool) (*Message, error) {
	m.giftOfferDeclined = decline
	return &Message{ID: offerMessageID}, nil
}

func (m *mockBinder) SendInvoice(ctx context.Context, chatID int64, invoice *tg.InputMediaInvoice, caption string, opts ...*params.SendMessage) (*Message, error) {
	m.mediaReplyTo = firstReplyTo(opts...)
	return &Message{ID: 1, ChatID: chatID}, nil
}

func (m *mockBinder) EditMessageWidgetData(ctx context.Context, chatID int64, messageID int32, publicData, chatData string) (*Message, error) {
	m.widgetPublicData, m.widgetChatData = publicData, chatData
	return &Message{ID: messageID, ChatID: chatID}, nil
}

func (m *mockBinder) AnswerWidgetQuery(ctx context.Context, queryID int64, result string) error {
	m.answeredWidgetID, m.answeredWidgetResult = queryID, result
	return nil
}
