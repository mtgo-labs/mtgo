package types

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestParseChatFolderInviteLinkInfo_NewFolder(t *testing.T) {
	raw := &tg.ChatlistsChatlistInvite{
		Title:    &tg.TextWithEntities{Text: "Work"},
		Emoticon: "folder",
		Peers: []tg.PeerClass{
			&tg.PeerChat{ChatID: 10},
			&tg.PeerChannel{ChannelID: 20},
		},
		Chats: []tg.ChatClass{
			&tg.Chat{ID: 10, Title: "Team"},
			&tg.Channel{ID: 20, Title: "Announcements", Broadcast: true},
		},
	}

	info := ParseChatFolderInviteLinkInfo(raw)
	if info == nil {
		t.Fatal("ParseChatFolderInviteLinkInfo returned nil")
	}
	if info.ChatFolderInfo == nil {
		t.Fatal("ChatFolderInfo is nil")
	}
	if info.ChatFolderInfo.ID != nil {
		t.Fatalf("ChatFolderInfo.ID = %v, want nil", *info.ChatFolderInfo.ID)
	}
	if info.ChatFolderInfo.Title != "Work" {
		t.Fatalf("ChatFolderInfo.Title = %q, want Work", info.ChatFolderInfo.Title)
	}
	if info.ChatFolderInfo.Emoticon != "folder" {
		t.Fatalf("ChatFolderInfo.Emoticon = %q, want folder", info.ChatFolderInfo.Emoticon)
	}
	if len(info.MissingChats) != 2 {
		t.Fatalf("MissingChats len = %d, want 2", len(info.MissingChats))
	}
	if len(info.AddedChats) != 0 {
		t.Fatalf("AddedChats len = %d, want 0", len(info.AddedChats))
	}
}

func TestParseChatFolderInviteLinkInfo_ExistingFolder(t *testing.T) {
	raw := &tg.ChatlistsChatlistInviteAlready{
		FilterID: 7,
		MissingPeers: []tg.PeerClass{
			&tg.PeerUser{UserID: 1},
		},
		AlreadyPeers: []tg.PeerClass{
			&tg.PeerChat{ChatID: 2},
		},
		Users: []tg.UserClass{
			&tg.User{ID: 1, FirstName: "Ada"},
		},
		Chats: []tg.ChatClass{
			&tg.Chat{ID: 2, Title: "Group"},
		},
	}

	info := ParseChatFolderInviteLinkInfo(raw)
	if info == nil {
		t.Fatal("ParseChatFolderInviteLinkInfo returned nil")
	}
	if info.ChatFolderInfo == nil || info.ChatFolderInfo.ID == nil {
		t.Fatal("ChatFolderInfo.ID is nil")
	}
	if *info.ChatFolderInfo.ID != 7 {
		t.Fatalf("ChatFolderInfo.ID = %d, want 7", *info.ChatFolderInfo.ID)
	}
	if len(info.MissingChats) != 1 || info.MissingChats[0].ID != 1 {
		t.Fatalf("MissingChats = %+v, want user chat 1", info.MissingChats)
	}
	if len(info.AddedChats) != 1 || info.AddedChats[0].ID != -2 {
		t.Fatalf("AddedChats = %+v, want group chat -2", info.AddedChats)
	}
}

func TestParseChatFolderInviteLinkInfo_Nil(t *testing.T) {
	if info := ParseChatFolderInviteLinkInfo(nil); info != nil {
		t.Fatalf("ParseChatFolderInviteLinkInfo(nil) = %+v, want nil", info)
	}
}
