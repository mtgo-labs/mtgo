package types

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestParseServiceActionRequestedPeer(t *testing.T) {
	action := &tg.MessageActionRequestedPeer{
		ButtonID: 1,
		Peers: []tg.PeerClass{
			&tg.PeerChannel{ChannelID: 100},
			&tg.PeerUser{UserID: 42},
		},
	}

	svc := parseServiceAction(action)
	if svc.Type != ServiceActionRequestedPeer {
		t.Fatalf("type = %d, want ServiceActionRequestedPeer", svc.Type)
	}
	if svc.RequestedPeers == nil {
		t.Fatal("expected RequestedPeers")
	}
	if svc.RequestedPeers.ButtonID != 1 {
		t.Errorf("buttonID = %d, want 1", svc.RequestedPeers.ButtonID)
	}
	if len(svc.RequestedPeers.ChatIDs) != 1 || svc.RequestedPeers.ChatIDs[0] != -1000000000100 {
		t.Errorf("chatIDs = %v, want [-1000000000100]", svc.RequestedPeers.ChatIDs)
	}
	if len(svc.RequestedPeers.UserIDs) != 1 || svc.RequestedPeers.UserIDs[0] != 42 {
		t.Errorf("userIDs = %v, want [42]", svc.RequestedPeers.UserIDs)
	}
}

func TestParseServiceActionRequestedPeerSentMe(t *testing.T) {
	action := &tg.MessageActionRequestedPeerSentMe{
		ButtonID: 3,
		Peers: []tg.RequestedPeerClass{
			&tg.RequestedPeerUser{UserID: 99},
			&tg.RequestedPeerChat{ChatID: 200},
			&tg.RequestedPeerChannel{ChannelID: 1798673537},
		},
	}

	svc := parseServiceAction(action)
	if svc.Type != ServiceActionRequestedPeer {
		t.Fatalf("type = %d, want ServiceActionRequestedPeer", svc.Type)
	}
	if len(svc.RequestedPeers.UserIDs) != 1 || svc.RequestedPeers.UserIDs[0] != 99 {
		t.Errorf("userIDs = %v, want [99]", svc.RequestedPeers.UserIDs)
	}
	wantChatIDs := []int64{-200, -1001798673537}
	if len(svc.RequestedPeers.ChatIDs) != len(wantChatIDs) {
		t.Fatalf("chatIDs = %v, want %v", svc.RequestedPeers.ChatIDs, wantChatIDs)
	}
	for i, want := range wantChatIDs {
		if svc.RequestedPeers.ChatIDs[i] != want {
			t.Errorf("chatIDs[%d] = %d, want %d", i, svc.RequestedPeers.ChatIDs[i], want)
		}
	}
}

func TestParseServiceActionUnknown(t *testing.T) {
	svc := parseServiceAction(&tg.MessageActionEmpty{})
	if svc.Type != ServiceActionUnknown {
		t.Errorf("type = %d, want ServiceActionUnknown", svc.Type)
	}
}

func TestParseServiceActionGroupCreate(t *testing.T) {
	svc := parseServiceAction(&tg.MessageActionChatCreate{})
	if svc.Type != ServiceActionGroupCreate {
		t.Errorf("type = %d, want ServiceActionGroupCreate", svc.Type)
	}
}
