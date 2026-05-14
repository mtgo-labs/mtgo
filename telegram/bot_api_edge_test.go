package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestAnswerInlineQuery_EmptyResults(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.AnswerInlineQuery(context.Background(), 12345, nil, nil)
	if err != nil {
		t.Fatalf("AnswerInlineQuery() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.MessagesSetInlineBotResultsRequest)
	if !ok {
		t.Fatalf("expected MessagesSetInlineBotResultsRequest, got %T", mock.lastCall())
	}
	if len(req.Results) != 0 {
		t.Errorf("Results length = %d, want 0", len(req.Results))
	}
}

func TestSetGameScore_ZeroScore(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	mock.setResult(tg.MessagesSetGameScoreTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewMessage{
				Message: &tg.Message{ID: 55},
			},
		},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	msg, err := c.SetGameScore(context.Background(), 10, 55, 42, 0, false, false)
	if err != nil {
		t.Fatalf("SetGameScore() error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	req, ok := mock.lastCall().(*tg.MessagesSetGameScoreRequest)
	if !ok {
		t.Fatalf("expected MessagesSetGameScoreRequest, got %T", mock.lastCall())
	}
	if req.Score != 0 {
		t.Errorf("Score = %d, want 0", req.Score)
	}
}

func TestGetGameHighScores_EmptyScores(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	mock.setResult(tg.MessagesGetGameHighScoresTypeID, &tg.MessagesHighScores{
		Scores: []*tg.HighScore{},
	})

	scores, err := c.GetGameHighScores(context.Background(), 10, 55, 42)
	if err != nil {
		t.Fatalf("GetGameHighScores() error: %v", err)
	}
	if len(scores) != 0 {
		t.Errorf("Scores length = %d, want 0", len(scores))
	}
}

func TestAnswerCallbackQuery_EmptyText(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.AnswerCallbackQuery(context.Background(), 12345, "", false, "", 0)
	if err != nil {
		t.Fatalf("AnswerCallbackQuery() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.MessagesSetBotCallbackAnswerRequest)
	if !ok {
		t.Fatalf("expected MessagesSetBotCallbackAnswerRequest, got %T", mock.lastCall())
	}
	if req.Message != "" {
		t.Errorf("Message = %q, want empty", req.Message)
	}
}

func TestSetBotCommands_EmptyCommands(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	err := c.SetBotCommands(context.Background(), &tg.BotCommandScopeDefault{}, "en", nil)
	if err != nil {
		t.Fatalf("SetBotCommands() error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.BotsSetBotCommandsRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotCommandsRequest, got %T", mock.lastCall())
	}
	if len(req.Commands) != 0 {
		t.Errorf("Commands length = %d, want 0", len(req.Commands))
	}
}

func TestSetBotCommands_MultipleScopes(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	commands := []*tg.BotCommand{{Command: "start", Description: "Start"}}

	err := c.SetBotCommands(context.Background(), &tg.BotCommandScopeDefault{}, "en", commands)
	if err != nil {
		t.Fatalf("SetBotCommands(Default) error: %v", err)
	}
	_, ok := mock.lastCall().(*tg.BotsSetBotCommandsRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotCommandsRequest, got %T", mock.lastCall())
	}

	err = c.SetBotCommands(context.Background(), &tg.BotCommandScopePeer{
		Peer: &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20},
	}, "en", commands)
	if err != nil {
		t.Fatalf("SetBotCommands(Peer) error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.BotsSetBotCommandsRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotCommandsRequest, got %T", mock.lastCall())
	}
	_, ok = req.Scope.(*tg.BotCommandScopePeer)
	if !ok {
		t.Errorf("Scope = %T, want BotCommandScopePeer", req.Scope)
	}
}

func TestMenuButtonTypes(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	err := c.SetChatMenuButton(context.Background(), 42, &tg.BotMenuButton{
		Text: "Open App",
		URL:  "https://example.com",
	})
	if err != nil {
		t.Fatalf("SetChatMenuButton(BotMenuButtonTL) error: %v", err)
	}
	req, ok := mock.lastCall().(*tg.BotsSetBotMenuButtonRequest)
	if !ok {
		t.Fatalf("expected BotsSetBotMenuButtonRequest, got %T", mock.lastCall())
	}
	btn, ok := req.Button.(*tg.BotMenuButton)
	if !ok {
		t.Fatalf("Button = %T, want BotMenuButtonTL", req.Button)
	}
	if btn.Text != "Open App" {
		t.Errorf("Text = %q, want %q", btn.Text, "Open App")
	}
	if btn.URL != "https://example.com" {
		t.Errorf("Url = %q, want %q", btn.URL, "https://example.com")
	}
}

func TestBotCommandScopePeerConstructors(t *testing.T) {
	scopes := []tg.BotCommandScopeClass{
		&tg.BotCommandScopeDefault{},
		&tg.BotCommandScopeUsers{},
		&tg.BotCommandScopeChats{},
		&tg.BotCommandScopeChatAdmins{},
		&tg.BotCommandScopePeer{Peer: &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20}},
		&tg.BotCommandScopePeerAdmins{Peer: &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20}},
		&tg.BotCommandScopePeerUser{
			Peer:   &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20},
			UserID: &tg.InputUser{UserID: 42, AccessHash: 100},
		},
	}
	for _, scope := range scopes {
		if scope == nil {
			t.Error("scope should not be nil")
		}
	}
}

func TestHighScore_Fields(t *testing.T) {
	hs := &tg.HighScore{Pos: 3, UserID: 42, Score: 250}
	if hs.Pos != 3 {
		t.Errorf("Pos = %d, want 3", hs.Pos)
	}
	if hs.UserID != 42 {
		t.Errorf("UserID = %d, want 42", hs.UserID)
	}
	if hs.Score != 250 {
		t.Errorf("Score = %d, want 250", hs.Score)
	}
}
