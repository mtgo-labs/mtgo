package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSetGameScore(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	mock.setResult(tg.MessagesSetGameScoreTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewMessage{
				Message: &tg.Message{
					ID: 55,
				},
			},
		},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	msg, err := c.SetGameScore(context.Background(), int64(10), 55, 42, 100, false, false)
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
	if req.Score != 100 {
		t.Errorf("Score = %d, want 100", req.Score)
	}
	if req.ID != 55 {
		t.Errorf("ID = %d, want 55", req.ID)
	}
}

func TestSetGameScore_WithForce(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	mock.setResult(tg.MessagesSetGameScoreTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewMessage{
				Message: &tg.Message{
					ID: 55,
				},
			},
		},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	msg, err := c.SetGameScore(context.Background(), int64(10), 55, 42, 200, true, false)
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
	if !req.Force {
		t.Error("Force should be true")
	}
	if req.Flags&(1<<1) == 0 {
		t.Error("Force flag bit not set")
	}
}

func TestGetGameHighScores(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	mock.setResult(tg.MessagesGetGameHighScoresTypeID, &tg.MessagesHighScores{
		Scores: []*tg.HighScore{
			{Pos: 1, UserID: 42, Score: 500},
			{Pos: 2, UserID: 99, Score: 300},
		},
	})

	scores, err := c.GetGameHighScores(context.Background(), int64(10), 55, 42)
	if err != nil {
		t.Fatalf("GetGameHighScores() error: %v", err)
	}
	if len(scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(scores))
	}
	if scores[0].Score != 500 {
		t.Errorf("Score[0] = %d, want 500", scores[0].Score)
	}
	if scores[1].Pos != 2 {
		t.Errorf("Pos[1] = %d, want 2", scores[1].Pos)
	}
}

func TestSetGameScore_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	mock.setError(tg.MessagesSetGameScoreTypeID, errors.New("rpc error"))

	_, err := c.SetGameScore(context.Background(), int64(10), 55, 42, 100, false, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetGameHighScores_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})

	mock.setError(tg.MessagesGetGameHighScoresTypeID, errors.New("rpc error"))

	_, err := c.GetGameHighScores(context.Background(), int64(10), 55, 42)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
