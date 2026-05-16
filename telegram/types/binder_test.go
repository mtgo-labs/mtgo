package types

import (
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	tl "github.com/mtgo-labs/mtgo/tg"
)

type mockBinder struct {
	sendChatID    int64
	sendText      string
	sendReplyTo   int32
	deletedChatID int64
	deletedMsgIDs []int32
	pinnedChatID  int64
	pinnedMsgID   int32
	answerID      int64
	answerText    string
	answerAlert   bool
	mediaCaption  string
	mediaReplyTo  int32
	editMedia     tl.InputMediaClass
	err           error
}

func (m *mockBinder) BoundSend(chatID int64, text string, replyTo int32, opts ...*params.SendMessage) (*Message, error) {
	m.sendChatID = chatID
	m.sendText = text
	m.sendReplyTo = replyTo
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 99, ChatID: chatID}, nil
}

func (m *mockBinder) BoundSendMedia(chatID int64, media tl.InputMediaClass, caption string, replyTo int32, opts ...*params.SendMessage) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	m.mediaReplyTo = replyTo
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundForward(chatID int64, fromChatID int64, msgID int32, opts ...*params.ForwardMessages) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) BoundCopy(chatID int64, fromChatID int64, msgID int32, opts ...*params.CopyMessage) (int64, error) {
	return 0, nil
}

func (m *mockBinder) BoundEdit(chatID int64, msgID int32, text string, opts ...*params.EditMessage) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) BoundEditInline(inlineMessageID tl.InputBotInlineMessageIDClass, text string, opts ...*params.EditMessage) (bool, error) {
	return true, nil
}

func (m *mockBinder) BoundEditCaption(chatID int64, msgID int32, caption string, opts ...*params.EditMessage) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) BoundEditInlineCaption(inlineMessageID tl.InputBotInlineMessageIDClass, caption string, opts ...*params.EditMessage) (bool, error) {
	return true, nil
}

func (m *mockBinder) BoundEditMedia(chatID int64, msgID int32, media tl.InputMediaClass) (*Message, error) {
	m.editMedia = media
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: msgID, ChatID: chatID}, nil
}

func (m *mockBinder) BoundEditInlineMedia(inlineMessageID tl.InputBotInlineMessageIDClass, media tl.InputMediaClass) (bool, error) {
	return true, nil
}

func (m *mockBinder) BoundEditReplyMarkup(chatID int64, msgID int32, markup tl.ReplyMarkupClass) (*Message, error) {
	return nil, nil
}

func (m *mockBinder) BoundEditInlineReplyMarkup(inlineMessageID tl.InputBotInlineMessageIDClass, markup tl.ReplyMarkupClass) (bool, error) {
	return true, nil
}

func (m *mockBinder) BoundDelete(chatID int64, msgIDs []int32, opts ...*params.DeleteMessages) (int, error) {
	m.deletedChatID = chatID
	m.deletedMsgIDs = msgIDs
	if m.err != nil {
		return 0, m.err
	}
	return len(msgIDs), nil
}

func (m *mockBinder) BoundReact(chatID int64, msgID int32, opts ...*params.React) error {
	return nil
}

func (m *mockBinder) BoundPin(chatID int64, msgID int32, opts ...*params.PinMessage) error {
	m.pinnedChatID = chatID
	m.pinnedMsgID = msgID
	return m.err
}

func (m *mockBinder) BoundUnpin(chatID int64, msgID int32, opts ...*params.PinMessage) error {
	return nil
}

func (m *mockBinder) BoundRead(chatID int64, msgID int32) error {
	return nil
}

func (m *mockBinder) BoundAnswerCallback(queryID int64, opts ...*params.AnswerCallback) error {
	o := params.GetOptDef(&params.AnswerCallback{}, opts...)
	m.answerID = queryID
	m.answerText = o.Text
	m.answerAlert = o.ShowAlert
	return m.err
}

func (m *mockBinder) BoundDownload(chatID int64, msgID int32, opts ...*params.Download) ([]byte, error) {
	return nil, nil
}

func (m *mockBinder) BoundDownloadTo(chatID int64, msgID int32, fileName string, opts ...*params.Download) (string, error) {
	return "", nil
}

func (m *mockBinder) BoundSendContact(chatID int64, phone, firstName, lastName string, replyTo int32, opts ...*params.SendContact) (*Message, error) {
	m.sendChatID = chatID
	m.sendReplyTo = replyTo
	return &Message{ID: 101, ChatID: chatID}, nil
}

func (m *mockBinder) BoundSendLocation(chatID int64, lat, lng float64, replyTo int32, opts ...*params.SendLocation) (*Message, error) {
	m.sendChatID = chatID
	m.sendReplyTo = replyTo
	return &Message{ID: 102, ChatID: chatID}, nil
}

func (m *mockBinder) BoundSendVenue(chatID int64, lat, lng float64, title, address string, replyTo int32, opts ...*params.SendVenue) (*Message, error) {
	m.sendChatID = chatID
	m.sendReplyTo = replyTo
	return &Message{ID: 103, ChatID: chatID}, nil
}

func (m *mockBinder) BoundSendPoll(chatID int64, question string, options []string, replyTo int32, opts ...*params.SendPoll) (*Message, error) {
	m.sendChatID = chatID
	m.sendReplyTo = replyTo
	return &Message{ID: 104, ChatID: chatID}, nil
}

func (m *mockBinder) BoundSendDice(chatID int64, emoji string, replyTo int32, opts ...*params.SendDice) (*Message, error) {
	m.sendChatID = chatID
	m.sendReplyTo = replyTo
	return &Message{ID: 105, ChatID: chatID}, nil
}

func (m *mockBinder) BoundSendGame(chatID int64, gameShortName string, replyTo int32, opts ...*params.SendGame) (*Message, error) {
	m.sendChatID = chatID
	m.sendReplyTo = replyTo
	return &Message{ID: 106, ChatID: chatID}, nil
}

func (m *mockBinder) BoundSendMediaGroup(chatID int64, media []tl.InputMediaClass, replyTo int32, opts ...*params.SendMediaGroup) ([]*Message, error) {
	m.sendChatID = chatID
	m.sendReplyTo = replyTo
	return []*Message{{ID: 107, ChatID: chatID}}, nil
}

func (m *mockBinder) BoundSendChatAction(chatID int64, action tl.SendMessageActionClass) error {
	return nil
}

func (m *mockBinder) BoundSendInlineBotResult(chatID int64, queryID int64, resultID string, replyTo int32, opts ...*params.SendInlineBotResult) (*Message, error) {
	m.sendChatID = chatID
	m.sendReplyTo = replyTo
	return &Message{ID: 108, ChatID: chatID}, nil
}

func (m *mockBinder) BoundVote(chatID int64, msgID int32, options [][]byte) error {
	return nil
}

func (m *mockBinder) BoundRetractVote(chatID int64, msgID int32) error {
	return nil
}

func (m *mockBinder) BoundGetMediaGroup(chatID int64, msgID int32) ([]*Message, error) {
	return []*Message{{ID: msgID, ChatID: chatID}}, nil
}

func (m *mockBinder) BoundCopyMediaGroup(chatID int64, fromChatID int64, msgID int32) ([]*Message, error) {
	return []*Message{{ID: 109, ChatID: chatID}}, nil
}

func (m *mockBinder) BoundStub(method string) error {
	return errors.New("stub: " + method)
}

func (m *mockBinder) BoundAnswerInline(queryID int64, results []tl.InputBotInlineResultClass, opts ...*params.InlineQuery) error {
	return m.err
}

func (m *mockBinder) BoundBlock(userID int64) error {
	return m.err
}

func (m *mockBinder) BoundUnblock(userID int64) error {
	return m.err
}

func (m *mockBinder) BoundGetCommonChats(userID int64, limit int) ([]*Chat, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []*Chat{{ID: userID}}, nil
}

func (m *mockBinder) BoundArchiveUser(chatID int64) error {
	return m.err
}

func (m *mockBinder) BoundUnarchiveUser(chatID int64) error {
	return m.err
}

func (m *mockBinder) BoundAnswerPreCheckout(queryID int64, opts ...*params.AnswerPreCheckout) error {
	return m.err
}

func (m *mockBinder) BoundAnswerShipping(queryID int64, opts ...*params.AnswerShipping) error {
	return m.err
}

func (m *mockBinder) BoundApproveJoinRequest(chatID int64, userID int64) error {
	return m.err
}

func (m *mockBinder) BoundDeclineJoinRequest(chatID int64, userID int64) error {
	return m.err
}

func (m *mockBinder) BoundStoryReply(peerID int64, storyID int32, text string, opts ...*params.SendMessage) (*Message, error) {
	return &Message{ID: 200}, m.err
}

func (m *mockBinder) BoundStoryReplyMedia(peerID int64, storyID int32, media tl.InputMediaClass, caption string, opts ...*params.SendMessage) (*Message, error) {
	return &Message{ID: 201}, m.err
}

func (m *mockBinder) BoundStoryForward(fromChatID int64, storyID int32, chatID int64, opts ...*params.StoryForward) (*Message, error) {
	return &Message{ID: 202}, m.err
}

func (m *mockBinder) BoundStoryRead(peerID int64, storyID int32) error {
	return m.err
}

func (m *mockBinder) BoundStoryDelete(peerID int64, storyID int32) error {
	return m.err
}

func (m *mockBinder) BoundStoryEditCaption(peerID int64, storyID int32, opts ...*params.EditCaption) (*Story, error) {
	return nil, m.err
}

func (m *mockBinder) BoundStoryEditMedia(peerID int64, storyID int32, media tl.InputMediaClass) (*Story, error) {
	return nil, m.err
}

func (m *mockBinder) BoundStoryEditPrivacy(peerID int64, storyID int32, opts ...*params.EditPrivacy) (*Story, error) {
	return nil, m.err
}

func (m *mockBinder) BoundStoryReact(peerID int64, storyID int32, opts ...*params.React) error {
	return m.err
}

func (m *mockBinder) BoundStoryDownload(peerID int64, storyID int32, opts ...*params.Download) ([]byte, error) {
	return nil, m.err
}

func (m *mockBinder) BoundReplyChecklist(chatID int64, checklist *tl.InputMediaTodo, replyTo int32, opts ...*params.SendChecklist) (*Message, error) {
	m.sendChatID = chatID
	m.sendReplyTo = replyTo
	return &Message{ID: 110, ChatID: chatID}, m.err
}

func (m *mockBinder) BoundReplyPhoto(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendPhoto) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	m.mediaReplyTo = replyTo
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundAnswerPhoto(chatID int64, file *InputFile, caption string, opts ...*params.SendPhoto) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundReplyAudio(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendAudio) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	m.mediaReplyTo = replyTo
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundAnswerAudio(chatID int64, file *InputFile, caption string, opts ...*params.SendAudio) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundReplyVideo(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendVideo) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	m.mediaReplyTo = replyTo
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundAnswerVideo(chatID int64, file *InputFile, caption string, opts ...*params.SendVideo) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundReplyDocument(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendDocument) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	m.mediaReplyTo = replyTo
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundAnswerDocument(chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundReplyAnimation(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendAnimation) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	m.mediaReplyTo = replyTo
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundAnswerAnimation(chatID int64, file *InputFile, caption string, opts ...*params.SendAnimation) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundReplyVoice(chatID int64, file *InputFile, caption string, replyTo int32, opts ...*params.SendVoice) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	m.mediaReplyTo = replyTo
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundAnswerVoice(chatID int64, file *InputFile, caption string, opts ...*params.SendVoice) (*Message, error) {
	m.sendChatID = chatID
	m.mediaCaption = caption
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundReplyVideoNote(chatID int64, file *InputFile, replyTo int32, opts ...*params.SendVideoNote) (*Message, error) {
	m.sendChatID = chatID
	m.mediaReplyTo = replyTo
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundAnswerVideoNote(chatID int64, file *InputFile, opts ...*params.SendVideoNote) (*Message, error) {
	m.sendChatID = chatID
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundReplySticker(chatID int64, file *InputFile, replyTo int32, opts ...*params.SendSticker) (*Message, error) {
	m.sendChatID = chatID
	m.mediaReplyTo = replyTo
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
}

func (m *mockBinder) BoundAnswerSticker(chatID int64, file *InputFile, opts ...*params.SendSticker) (*Message, error) {
	m.sendChatID = chatID
	if m.err != nil {
		return nil, m.err
	}
	return &Message{ID: 100, ChatID: chatID}, nil
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
	m := &Message{ID: 1, ChatID: 42, binder: b}
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
	m := &Message{ID: 1, ChatID: 42, binder: b}
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
	m := &Message{ID: 5, ChatID: 10, binder: b}
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
	m := &Message{ID: 7, ChatID: 20, binder: b}
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
	m := &Message{ID: 3, ChatID: 50, binder: b}
	media := &tl.InputMediaDocument{ID: &tl.InputDocument{ID: 100, AccessHash: 200}}
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
	m := &Message{ID: 3, ChatID: 50, binder: b}
	media := &tl.InputMediaPhoto{ID: &tl.InputPhoto{ID: 10, AccessHash: 20}}
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
	m := &Message{ID: 5, ChatID: 30, binder: b}
	media := &tl.InputMediaDocument{ID: &tl.InputDocument{ID: 999}}
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
	m := &Message{ID: 1, ChatID: 40, binder: b}
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
	m := &Message{ID: 2, ChatID: 40, binder: b}
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
	q := &CallbackQuery{ID: 100, binder: b}
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
	q := &CallbackQuery{ID: 200, binder: b}
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
