package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestAnswerInlineQuery_Basic(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.AnswerInlineQuery(context.Background(), 12345, nil)
	if err != nil {
		t.Fatalf("AnswerInlineQuery() error: %v", err)
	}
	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}
	req, ok := mock.lastCall().(*tg.MessagesSetInlineBotResultsRequest)
	if !ok {
		t.Fatalf("expected MessagesSetInlineBotResultsRequest, got %T", mock.lastCall())
	}
	if req.QueryID != 12345 {
		t.Errorf("QueryID = %d, want 12345", req.QueryID)
	}
	if req.Gallery {
		t.Error("Gallery should be false")
	}
}

func TestAnswerInlineQuery_WithGallery(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.AnswerInlineQuery(
		context.Background(), 12345, nil,
		&AnswerInlineQueryOption{Gallery: true},
	)
	if err != nil {
		t.Fatalf("AnswerInlineQuery() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.MessagesSetInlineBotResultsRequest)
	if !ok {
		t.Fatalf("expected MessagesSetInlineBotResultsRequest, got %T", mock.lastCall())
	}
	if !req.Gallery {
		t.Error("Gallery should be true")
	}
}

func TestAnswerInlineQuery_WithNextOffset(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.AnswerInlineQuery(
		context.Background(), 12345, nil,
		&AnswerInlineQueryOption{NextOffset: "50"},
	)
	if err != nil {
		t.Fatalf("AnswerInlineQuery() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.MessagesSetInlineBotResultsRequest)
	if !ok {
		t.Fatalf("expected MessagesSetInlineBotResultsRequest, got %T", mock.lastCall())
	}
	if req.NextOffset != "50" {
		t.Errorf("NextOffset = %v, want %q", req.NextOffset, "50")
	}
}

func TestAnswerInlineQuery_WithSwitchPM(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.AnswerInlineQuery(
		context.Background(), 12345, nil,
		&AnswerInlineQueryOption{SwitchPM: &tg.InlineBotSwitchPm{Text: "Go", StartParam: "start"}},
	)
	if err != nil {
		t.Fatalf("AnswerInlineQuery() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.MessagesSetInlineBotResultsRequest)
	if !ok {
		t.Fatalf("expected MessagesSetInlineBotResultsRequest, got %T", mock.lastCall())
	}
	if req.SwitchPm == nil {
		t.Fatal("SwitchPm should not be nil")
	}
	if req.SwitchPm.Text != "Go" {
		t.Errorf("SwitchPm.Text = %q, want %q", req.SwitchPm.Text, "Go")
	}
	if req.SwitchPm.StartParam != "start" {
		t.Errorf("SwitchPm.StartParam = %q, want %q", req.SwitchPm.StartParam, "start")
	}
}

func TestAnswerInlineQuery_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setError(tg.MessagesSetInlineBotResultsTypeID, errors.New("rpc error"))

	err := c.AnswerInlineQuery(context.Background(), 12345, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAnswerGuestQuery(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setResult(tg.MessagesSetBotGuestChatResultTypeID, &tg.InputBotInlineMessageID{
		DCID:       4,
		ID:         123,
		AccessHash: 456,
	})

	result, err := c.AnswerGuestQuery(context.Background(), "98765", &tg.InputBotInlineResultGame{
		ID:        "game1",
		ShortName: "mygame",
	})
	if err != nil {
		t.Fatalf("AnswerGuestQuery() error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.InlineMessageID != "4:123:456" {
		t.Errorf("InlineMessageID = %q, want %q", result.InlineMessageID, "4:123:456")
	}
	req, ok := mock.lastCall().(*tg.MessagesSetBotGuestChatResultRequest)
	if !ok {
		t.Fatalf("expected MessagesSetBotGuestChatResultRequest, got %T", mock.lastCall())
	}
	if req.QueryID != 98765 {
		t.Errorf("QueryID = %d, want 98765", req.QueryID)
	}
}

func TestAnswerGuestQuery_InvalidQueryID(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	_, err := c.AnswerGuestQuery(context.Background(), "not-a-number", &tg.InputBotInlineResultGame{
		ID:        "game1",
		ShortName: "mygame",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.callCount() != 0 {
		t.Fatalf("expected no RPC calls, got %d", mock.callCount())
	}
}

func TestAnswerGuestQuery_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setError(tg.MessagesSetBotGuestChatResultTypeID, errors.New("rpc error"))

	_, err := c.AnswerGuestQuery(context.Background(), "98765", &tg.InputBotInlineResultGame{
		ID:        "game1",
		ShortName: "mygame",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetInlineBotResults(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.CachePeer(1, &tg.InputPeerUser{UserID: 1, AccessHash: 100})

	mock.setResult(tg.MessagesGetInlineBotResultsTypeID, &tg.MessagesBotResults{
		QueryID:   99999,
		CacheTime: 300,
	})

	results, err := c.GetInlineBotResults(context.Background(), 1, 10, "hello", "")
	if err != nil {
		t.Fatalf("GetInlineBotResults() error: %v", err)
	}
	if results.QueryID != 99999 {
		t.Errorf("QueryID = %d, want 99999", results.QueryID)
	}
	req, ok := mock.lastCall().(*tg.MessagesGetInlineBotResultsRequest)
	if !ok {
		t.Fatalf("expected MessagesGetInlineBotResultsRequest, got %T", mock.lastCall())
	}
	if req.Query != "hello" {
		t.Errorf("Query = %q, want %q", req.Query, "hello")
	}
}

func TestGetInlineBotResults_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.CachePeer(1, &tg.InputPeerUser{UserID: 1, AccessHash: 100})

	mock.setError(tg.MessagesGetInlineBotResultsTypeID, errors.New("rpc error"))

	_, err := c.GetInlineBotResults(context.Background(), 1, 10, "hello", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSendInlineBotResult(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	mock.setResult(tg.MessagesSendInlineBotResultTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewMessage{
				Message: &tg.Message{
					ID:     42,
					PeerID: &tg.PeerChannel{ChannelID: 10},
				},
			},
		},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	msg, err := c.SendInlineBotResult(context.Background(), 10, 12345, "result1")
	if err != nil {
		t.Fatalf("SendInlineBotResult() error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected message, got nil")
	}
	if msg.ID != 42 {
		t.Errorf("Message.ID = %d, want 42", msg.ID)
	}
	req, ok := mock.lastCall().(*tg.MessagesSendInlineBotResultRequest)
	if !ok {
		t.Fatalf("expected MessagesSendInlineBotResultRequest, got %T", mock.lastCall())
	}
	if req.QueryID != 12345 {
		t.Errorf("QueryID = %d, want 12345", req.QueryID)
	}
	if req.ID != "result1" {
		t.Errorf("ID = %q, want %q", req.ID, "result1")
	}
}
