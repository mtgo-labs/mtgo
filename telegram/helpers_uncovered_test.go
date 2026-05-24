package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestRemovePeerFromSlice(t *testing.T) {
	peers := []tg.InputPeerClass{
		&tg.InputPeerUser{UserID: 1, AccessHash: 10},
		&tg.InputPeerChat{ChatID: 2},
		&tg.InputPeerChannel{ChannelID: 3, AccessHash: 30},
	}

	result := removePeerFromSlice(peers, &tg.InputPeerChat{ChatID: 2})
	if len(result) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(result))
	}

	_, ok1 := result[0].(*tg.InputPeerUser)
	if !ok1 {
		t.Errorf("expected InputPeerUser at index 0, got %T", result[0])
	}
	_, ok2 := result[1].(*tg.InputPeerChannel)
	if !ok2 {
		t.Errorf("expected InputPeerChannel at index 1, got %T", result[1])
	}
}

func TestRemovePeerFromSlice_NotFound(t *testing.T) {
	peers := []tg.InputPeerClass{
		&tg.InputPeerUser{UserID: 1},
		&tg.InputPeerChat{ChatID: 2},
	}

	result := removePeerFromSlice(peers, &tg.InputPeerChannel{ChannelID: 99})
	if len(result) != 2 {
		t.Fatalf("expected 2 peers (unchanged), got %d", len(result))
	}
}

func TestRemovePeerFromSlice_Empty(t *testing.T) {
	peers := []tg.InputPeerClass{}
	result := removePeerFromSlice(peers, &tg.InputPeerChat{ChatID: 1})
	if len(result) != 0 {
		t.Fatalf("expected 0 peers, got %d", len(result))
	}
}

func TestRemovePeerFromSlice_AllMatching(t *testing.T) {
	peers := []tg.InputPeerClass{
		&tg.InputPeerChat{ChatID: 5},
		&tg.InputPeerChat{ChatID: 5},
	}
	result := removePeerFromSlice(peers, &tg.InputPeerChat{ChatID: 5})
	if len(result) != 0 {
		t.Fatalf("expected 0 peers, got %d", len(result))
	}
}

func TestPeersEqual_UserMatch(t *testing.T) {
	a := &tg.InputPeerUser{UserID: 42, AccessHash: 100}
	b := &tg.InputPeerUser{UserID: 42, AccessHash: 200}
	if !peersEqual(a, b) {
		t.Error("expected peers with same UserID to be equal")
	}
}

func TestPeersEqual_UserMismatch(t *testing.T) {
	a := &tg.InputPeerUser{UserID: 42}
	b := &tg.InputPeerUser{UserID: 43}
	if peersEqual(a, b) {
		t.Error("expected peers with different UserID to not be equal")
	}
}

func TestPeersEqual_ChatMatch(t *testing.T) {
	a := &tg.InputPeerChat{ChatID: 10}
	b := &tg.InputPeerChat{ChatID: 10}
	if !peersEqual(a, b) {
		t.Error("expected peers with same ChatID to be equal")
	}
}

func TestPeersEqual_ChatMismatch(t *testing.T) {
	a := &tg.InputPeerChat{ChatID: 10}
	b := &tg.InputPeerChat{ChatID: 11}
	if peersEqual(a, b) {
		t.Error("expected peers with different ChatID to not be equal")
	}
}

func TestPeersEqual_ChannelMatch(t *testing.T) {
	a := &tg.InputPeerChannel{ChannelID: 20, AccessHash: 30}
	b := &tg.InputPeerChannel{ChannelID: 20, AccessHash: 40}
	if !peersEqual(a, b) {
		t.Error("expected peers with same ChannelID to be equal")
	}
}

func TestPeersEqual_ChannelMismatch(t *testing.T) {
	a := &tg.InputPeerChannel{ChannelID: 20}
	b := &tg.InputPeerChannel{ChannelID: 21}
	if peersEqual(a, b) {
		t.Error("expected peers with different ChannelID to not be equal")
	}
}

func TestPeersEqual_DifferentTypes(t *testing.T) {
	a := &tg.InputPeerUser{UserID: 1}
	b := &tg.InputPeerChat{ChatID: 1}
	if peersEqual(a, b) {
		t.Error("expected different peer types to not be equal")
	}
}

func TestPeersEqual_UserVsChannel(t *testing.T) {
	a := &tg.InputPeerUser{UserID: 1}
	b := &tg.InputPeerChannel{ChannelID: 1}
	if peersEqual(a, b) {
		t.Error("expected user and channel to not be equal")
	}
}

func TestParseCommonChats_Chats(t *testing.T) {
	result := &tg.MessagesChats{
		Chats: []tg.ChatClass{
			&tg.Chat{ID: 10, Title: "Chat1"},
			&tg.ChatEmpty{ID: 11},
		},
	}

	chats := parseCommonChats(result)
	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
	if chats[0].Title != "Chat1" {
		t.Errorf("chat Title = %q, want %q", chats[0].Title, "Chat1")
	}
}

func TestParseCommonChats_ChatsSlice(t *testing.T) {
	result := &tg.MessagesChatsSlice{
		Chats: []tg.ChatClass{
			&tg.Channel{ID: 20, Title: "Channel1"},
		},
		Count: 1,
	}

	chats := parseCommonChats(result)
	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
	if chats[0].Title != "Channel1" {
		t.Errorf("chat Title = %q, want %q", chats[0].Title, "Channel1")
	}
}

func TestParseCommonChats_Empty(t *testing.T) {
	result := &tg.MessagesChats{
		Chats: []tg.ChatClass{},
	}

	chats := parseCommonChats(result)
	if len(chats) != 0 {
		t.Fatalf("expected 0 chats, got %d", len(chats))
	}
}
