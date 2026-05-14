package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestPeerToInputPeer(t *testing.T) {
	users := []tg.UserClass{
		&tg.User{ID: 123, AccessHash: 999},
	}
	chats := []tg.ChatClass{
		&tg.Chat{ID: 456},
	}

	tests := []struct {
		name    string
		peer    tg.PeerClass
		wantErr bool
	}{
		{
			name: "peer_user",
			peer: &tg.PeerUser{UserID: 123},
		},
		{
			name: "peer_chat",
			peer: &tg.PeerChat{ChatID: 456},
		},
		{
			name:    "peer_channel_missing",
			peer:    &tg.PeerChannel{ChannelID: 789},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PeerToInputPeer(tt.peer, users, chats)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			switch tt.name {
			case "peer_user":
				ipu, ok := result.(*tg.InputPeerUser)
				if !ok {
					t.Fatalf("expected *InputPeerUser, got %T", result)
				}
				if ipu.UserID != 123 || ipu.AccessHash != 999 {
					t.Fatalf("wrong user peer: %+v", ipu)
				}
			case "peer_chat":
				ipc, ok := result.(*tg.InputPeerChat)
				if !ok {
					t.Fatalf("expected *InputPeerChat, got %T", result)
				}
				if ipc.ChatID != 456 {
					t.Fatalf("wrong chat peer: %+v", ipc)
				}
			}
		})
	}
}

func TestPeerToInputPeerSelf(t *testing.T) {
	result, err := PeerToInputPeer(&tg.PeerUser{UserID: 0}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.(*tg.InputPeerSelf); !ok {
		t.Fatalf("expected *InputPeerSelf, got %T", result)
	}
}

func TestPeerToInputPeerChannel(t *testing.T) {
	chats := []tg.ChatClass{
		&tg.Channel{ID: 789, AccessHash: 1111},
	}
	result, err := PeerToInputPeer(&tg.PeerChannel{ChannelID: 789}, nil, chats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ch, ok := result.(*tg.InputPeerChannel)
	if !ok {
		t.Fatalf("expected *InputPeerChannel, got %T", result)
	}
	if ch.ChannelID != 789 || ch.AccessHash != 1111 {
		t.Fatalf("wrong channel peer: %+v", ch)
	}
}
