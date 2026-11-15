package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestEditLiveLocation_Success(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesEditMessageTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateEditMessage{
				Message: &tg.Message{
					ID:      55,
					PeerID:  &tg.PeerUser{UserID: 10},
					Message: "loc",
				},
			},
		},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	msg, err := c.EditLiveLocation(context.Background(), 10, 55, 52.5, 13.4)
	if err != nil {
		t.Fatalf("EditLiveLocation() error: %v", err)
	}
	if msg == nil || msg.ID != 55 {
		t.Fatalf("EditLiveLocation() = %#v, want message 55", msg)
	}
	req, ok := inv.lastCall().(*tg.MessagesEditMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesEditMessageRequest, got %T", inv.lastCall())
	}
	geo, ok := req.Media.(*tg.InputMediaGeoLive)
	if !ok {
		t.Fatalf("expected InputMediaGeoLive, got %T", req.Media)
	}
	if geo.Stopped {
		t.Error("EditLiveLocation must not set the stopped flag")
	}
}

func TestStopLiveLocation_Success(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesEditMessageTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateEditMessage{
				Message: &tg.Message{ID: 55, PeerID: &tg.PeerUser{UserID: 10}},
			},
		},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	msg, err := c.StopLiveLocation(context.Background(), 10, 55)
	if err != nil {
		t.Fatalf("StopLiveLocation() error: %v", err)
	}
	if msg == nil || msg.ID != 55 {
		t.Fatalf("StopLiveLocation() = %#v, want message 55", msg)
	}
	req, ok := inv.lastCall().(*tg.MessagesEditMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesEditMessageRequest, got %T", inv.lastCall())
	}
	geo, ok := req.Media.(*tg.InputMediaGeoLive)
	if !ok || !geo.Stopped {
		t.Fatalf("expected stopped InputMediaGeoLive, got %#v", req.Media)
	}
}

func TestSummarizeMessage_Success(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesSummarizeTextTypeID, &tg.TextWithEntities{Text: "summary"})

	text, err := c.SummarizeMessage(context.Background(), 10, 55)
	if err != nil {
		t.Fatalf("SummarizeMessage() error: %v", err)
	}
	if text == nil || text.Text != "summary" {
		t.Fatalf("SummarizeMessage() = %#v, want summary text", text)
	}
	req, ok := inv.lastCall().(*tg.MessagesSummarizeTextRequest)
	if !ok {
		t.Fatalf("expected MessagesSummarizeTextRequest, got %T", inv.lastCall())
	}
	if req.ID != 55 {
		t.Errorf("request ID = %d, want 55", req.ID)
	}
}

func TestSendScreenshotNotification_Success(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesSendScreenshotNotificationTypeID, &tg.Updates{})

	if err := c.SendScreenshotNotification(context.Background(), 10, 0); err != nil {
		t.Fatalf("SendScreenshotNotification() error: %v", err)
	}
	if _, ok := inv.lastCall().(*tg.MessagesSendScreenshotNotificationRequest); !ok {
		t.Fatalf("expected MessagesSendScreenshotNotificationRequest, got %T", inv.lastCall())
	}
}

func TestGetScheduledMessages_Success(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesGetScheduledMessagesTypeID, &tg.MessagesMessages{
		Messages: []tg.MessageClass{
			&tg.Message{ID: 7, PeerID: &tg.PeerUser{UserID: 10}, Message: "scheduled"},
		},
		Chats: []tg.ChatClass{},
		Users: []tg.UserClass{&tg.User{ID: 10, Username: "u", FirstName: "U"}},
	})

	msgs, err := c.GetScheduledMessages(context.Background(), 10, nil)
	if err != nil {
		t.Fatalf("GetScheduledMessages() error: %v", err)
	}
	if len(msgs) != 1 || msgs[0].ID != 7 {
		t.Fatalf("GetScheduledMessages() = %#v, want one message 7", msgs)
	}
}

func TestTranslateText_Success(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesTranslateTextTypeID, &tg.MessagesTranslateResult{
		Result: []*tg.TextWithEntities{{Text: "hallo"}},
	})

	texts, err := c.TranslateText(context.Background(), []*tg.TextWithEntities{{Text: "hello"}}, "de")
	if err != nil {
		t.Fatalf("TranslateText() error: %v", err)
	}
	if len(texts) != 1 || texts[0].Text != "hallo" {
		t.Fatalf("TranslateText() = %#v, want one translated text", texts)
	}
	req, ok := inv.lastCall().(*tg.MessagesTranslateTextRequest)
	if !ok {
		t.Fatalf("expected MessagesTranslateTextRequest, got %T", inv.lastCall())
	}
	if req.ToLang != "de" || len(req.Text) != 1 {
		t.Errorf("unexpected request: %#v", req)
	}
}

func TestTranslateMessageText_Success(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesTranslateTextTypeID, &tg.MessagesTranslateResult{
		Result: []*tg.TextWithEntities{{Text: "hallo"}},
	})

	texts, err := c.TranslateMessageText(context.Background(), 10, []int32{55}, "de")
	if err != nil {
		t.Fatalf("TranslateMessageText() error: %v", err)
	}
	if len(texts) != 1 || texts[0].Text != "hallo" {
		t.Fatalf("TranslateMessageText() = %#v", texts)
	}
	req, ok := inv.lastCall().(*tg.MessagesTranslateTextRequest)
	if !ok {
		t.Fatalf("expected MessagesTranslateTextRequest, got %T", inv.lastCall())
	}
	if req.Peer == nil || len(req.ID) != 1 {
		t.Errorf("unexpected request: %#v", req)
	}
}

func TestSearchPosts_Success(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.ChannelsSearchPostsTypeID, &tg.MessagesMessagesSlice{
		Count: 2,
		Messages: []tg.MessageClass{
			&tg.Message{ID: 1, PeerID: &tg.PeerChannel{ChannelID: 77}},
		},
		Chats: []tg.ChatClass{&tg.Channel{ID: 77, Title: "ch"}},
		Users: []tg.UserClass{},
	})

	res, err := c.SearchPosts(context.Background(), "go", "", 0, 0, 0, 10)
	if err != nil {
		t.Fatalf("SearchPosts() error: %v", err)
	}
	if res.Count != 2 || len(res.Messages) != 1 {
		t.Fatalf("SearchPosts() = %#v, want count 2 with one message", res)
	}
	req, ok := inv.lastCall().(*tg.ChannelsSearchPostsRequest)
	if !ok {
		t.Fatalf("expected ChannelsSearchPostsRequest, got %T", inv.lastCall())
	}
	if req.Query != "go" || req.Limit != 10 {
		t.Errorf("unexpected request: %#v", req)
	}
}

func TestStartBot_Success(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(20, &tg.InputPeerUser{UserID: 20, AccessHash: 1})
	c.CachePeer(30, &tg.InputPeerUser{UserID: 30, AccessHash: 2})
	inv.setResult(tg.MessagesStartBotTypeID, &tg.Updates{})

	if err := c.StartBot(context.Background(), 20, 30, "deep"); err != nil {
		t.Fatalf("StartBot() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesStartBotRequest)
	if !ok {
		t.Fatalf("expected MessagesStartBotRequest, got %T", inv.lastCall())
	}
	if req.StartParam != "deep" {
		t.Errorf("StartParam = %q, want deep", req.StartParam)
	}
}

func TestChecklistTasks_ToggleAndAppend(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesAppendTodoListTypeID, &tg.Updates{})
	inv.setResult(tg.MessagesToggleTodoCompletedTypeID, &tg.Updates{})

	if err := c.AddChecklistTasks(context.Background(), 10, 55, []*tg.TodoItem{{ID: 1, Title: &tg.TextWithEntities{Text: "task"}}}); err != nil {
		t.Fatalf("AddChecklistTasks() error: %v", err)
	}
	if err := c.MarkChecklistTasksAsDone(context.Background(), 10, 55, []int32{1}, nil); err != nil {
		t.Fatalf("MarkChecklistTasksAsDone() error: %v", err)
	}
	toggle, ok := inv.lastCall().(*tg.MessagesToggleTodoCompletedRequest)
	if !ok {
		t.Fatalf("expected MessagesToggleTodoCompletedRequest, got %T", inv.lastCall())
	}
	if len(toggle.Completed) != 1 || toggle.Completed[0] != 1 {
		t.Errorf("unexpected toggle request: %#v", toggle)
	}
}

func TestResolveStarGiftOffer_Accept(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.PaymentsResolveStarGiftOfferTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateEditMessage{
				Message: &tg.Message{ID: 55, PeerID: &tg.PeerUser{UserID: 10}},
			},
		},
		Chats: []tg.ChatClass{},
		Users: []tg.UserClass{},
	})

	msg, err := c.ResolveStarGiftOffer(context.Background(), 55, false)
	if err != nil {
		t.Fatalf("ResolveStarGiftOffer() error: %v", err)
	}
	if msg == nil || msg.ID != 55 {
		t.Fatalf("ResolveStarGiftOffer() = %#v, want message 55", msg)
	}
	req, ok := inv.lastCall().(*tg.PaymentsResolveStarGiftOfferRequest)
	if !ok {
		t.Fatalf("expected PaymentsResolveStarGiftOfferRequest, got %T", inv.lastCall())
	}
	if req.Decline {
		t.Error("accept path must not set Decline")
	}
}

func TestTranslateText_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setError(tg.MessagesTranslateTextTypeID, errors.New("rpc error"))

	if _, err := c.TranslateText(context.Background(), []*tg.TextWithEntities{{Text: "x"}}, "de"); err == nil {
		t.Fatal("expected error from RPC")
	}
}
