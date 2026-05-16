package types

import (
	"testing"

	tl "github.com/mtgo-labs/mtgo/tg"
)

func TestGetPeerID_NilPeer(t *testing.T) {
	got := GetPeerID(nil)
	if got != 0 {
		t.Errorf("GetPeerID(nil) = %d, want 0", got)
	}
}

func TestGetPeerID_User(t *testing.T) {
	got := GetPeerID(&tl.PeerUser{UserID: 42})
	if got != 42 {
		t.Errorf("GetPeerID(PeerUser{42}) = %d, want 42", got)
	}
}

func TestGetPeerID_Chat(t *testing.T) {
	got := GetPeerID(&tl.PeerChat{ChatID: 100})
	if got != -100 {
		t.Errorf("GetPeerID(PeerChat{100}) = %d, want -100", got)
	}
}

func TestGetPeerID_Channel(t *testing.T) {
	got := GetPeerID(&tl.PeerChannel{ChannelID: 500})
	if got != -500 {
		t.Errorf("GetPeerID(PeerChannel{500}) = %d, want -500", got)
	}
}

func TestNewPeerMap(t *testing.T) {
	users := []*tl.User{
		{ID: 1},
		{ID: 2},
	}
	chats := []*tl.Chat{
		{ID: 100},
	}
	channels := []*tl.Channel{
		{ID: 200},
	}
	pm := NewPeerMap(users, chats, channels)
	if len(pm.Users) != 2 {
		t.Errorf("len(Users) = %d, want 2", len(pm.Users))
	}
	if len(pm.Chats) != 1 {
		t.Errorf("len(Chats) = %d, want 1", len(pm.Chats))
	}
	if len(pm.Channels) != 1 {
		t.Errorf("len(Channels) = %d, want 1", len(pm.Channels))
	}
	if _, ok := pm.Users[1]; !ok {
		t.Error("Users[1] missing")
	}
	if _, ok := pm.Chats[100]; !ok {
		t.Error("Chats[100] missing")
	}
	if _, ok := pm.Channels[200]; !ok {
		t.Error("Channels[200] missing")
	}
}

func TestNewPeerMap_Empty(t *testing.T) {
	pm := NewPeerMap(nil, nil, nil)
	if pm.Users == nil || pm.Chats == nil || pm.Channels == nil {
		t.Error("expected non-nil maps")
	}
}

func TestNewPeerMap_NilEntries(t *testing.T) {
	users := []*tl.User{nil, {ID: 5}}
	pm := NewPeerMap(users, nil, nil)
	if len(pm.Users) != 1 {
		t.Errorf("len(Users) = %d, want 1 (nil entry skipped)", len(pm.Users))
	}
}
