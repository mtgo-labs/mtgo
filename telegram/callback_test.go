package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestAnswerCallbackQuery_Basic(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.AnswerCallbackQuery(context.Background(), 12345, "", false, "", 0)
	if err != nil {
		t.Fatalf("AnswerCallbackQuery() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.MessagesSetBotCallbackAnswerRequest)
	if !ok {
		t.Fatalf("expected MessagesSetBotCallbackAnswerRequest, got %T", mock.lastCall())
	}
	if req.QueryID != 12345 {
		t.Errorf("QueryID = %d, want 12345", req.QueryID)
	}
}

func TestAnswerCallbackQuery_WithText(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.AnswerCallbackQuery(context.Background(), 12345, "Hello!", false, "", 0)
	if err != nil {
		t.Fatalf("AnswerCallbackQuery() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.MessagesSetBotCallbackAnswerRequest)
	if !ok {
		t.Fatalf("expected MessagesSetBotCallbackAnswerRequest, got %T", mock.lastCall())
	}
	if req.Message != "Hello!" {
		t.Errorf("Message = %v, want %q", req.Message, "Hello!")
	}
}

func TestAnswerCallbackQuery_WithAlert(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.AnswerCallbackQuery(context.Background(), 12345, "Alert!", true, "", 0)
	if err != nil {
		t.Fatalf("AnswerCallbackQuery() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.MessagesSetBotCallbackAnswerRequest)
	if !ok {
		t.Fatalf("expected MessagesSetBotCallbackAnswerRequest, got %T", mock.lastCall())
	}
	if !req.Alert {
		t.Error("Alert should be true")
	}
}

func TestAnswerCallbackQuery_WithUrl(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.AnswerCallbackQuery(context.Background(), 12345, "", false, "https://example.com", 60)
	if err != nil {
		t.Fatalf("AnswerCallbackQuery() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.MessagesSetBotCallbackAnswerRequest)
	if !ok {
		t.Fatalf("expected MessagesSetBotCallbackAnswerRequest, got %T", mock.lastCall())
	}
	if req.URL != "https://example.com" {
		t.Errorf("Url = %v, want %q", req.URL, "https://example.com")
	}
	if req.CacheTime != 60 {
		t.Errorf("CacheTime = %d, want 60", req.CacheTime)
	}
}

func TestAnswerCallbackQuery_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setError(tg.MessagesSetBotCallbackAnswerTypeID, errors.New("rpc error"))

	err := c.AnswerCallbackQuery(context.Background(), 12345, "", false, "", 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAnswerWebAppQuery(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setResult(tg.MessagesSendWebViewResultMessageTypeID, &tg.WebViewMessageSent{})

	result, err := c.AnswerWebAppQuery(context.Background(), "AAQF5eY", &tg.InputBotInlineResultGame{
		ID:        "game1",
		ShortName: "mygame",
	})
	if err != nil {
		t.Fatalf("AnswerWebAppQuery() error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	req, ok := mock.lastCall().(*tg.MessagesSendWebViewResultMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesSendWebViewResultMessageRequest, got %T", mock.lastCall())
	}
	if req.BotQueryID != "AAQF5eY" {
		t.Errorf("BotQueryID = %q, want %q", req.BotQueryID, "AAQF5eY")
	}
}

func TestAnswerWebAppQuery_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setError(tg.MessagesSendWebViewResultMessageTypeID, errors.New("rpc error"))

	_, err := c.AnswerWebAppQuery(context.Background(), "AAQF5eY", &tg.InputBotInlineResultGame{
		ID:        "game1",
		ShortName: "mygame",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRequestCallbackAnswer(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	mock.setResult(tg.MessagesGetBotCallbackAnswerTypeID, &tg.MessagesBotCallbackAnswer{
		Flags:   0x01,
		Message: "callback response",
	})

	answer, err := c.RequestCallbackAnswer(context.Background(), 10, 55, []byte("data"))
	if err != nil {
		t.Fatalf("RequestCallbackAnswer() error: %v", err)
	}
	if answer.Message != "callback response" {
		t.Errorf("Message = %v, want %q", answer.Message, "callback response")
	}
	req, ok := mock.lastCall().(*tg.MessagesGetBotCallbackAnswerRequest)
	if !ok {
		t.Fatalf("expected MessagesGetBotCallbackAnswerRequest, got %T", mock.lastCall())
	}
	if req.MsgID != 55 {
		t.Errorf("MsgID = %d, want 55", req.MsgID)
	}
}

func TestRequestCallbackAnswer_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	mock.setError(tg.MessagesGetBotCallbackAnswerTypeID, errors.New("rpc error"))

	_, err := c.RequestCallbackAnswer(context.Background(), 10, 55, []byte("data"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
