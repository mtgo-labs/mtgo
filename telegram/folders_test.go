package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func folderFiltersResult(filters ...tg.DialogFilterClass) *tg.MessagesDialogFilters {
	return &tg.MessagesDialogFilters{Filters: filters}
}

func TestGetFolders_ParsesAndSkipsSystemEntries(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesGetDialogFiltersTypeID, folderFiltersResult(
		&tg.DialogFilterDefault{},
		&tg.DialogFilter{ID: 2, Title: &tg.TextWithEntities{Text: "Work"}, Emoticon: "💼"},
		&tg.DialogFilter{ID: 5, Title: &tg.TextWithEntities{Text: "Personal"}},
	))

	folders, err := c.GetFolders(context.Background())
	if err != nil {
		t.Fatalf("GetFolders() error: %v", err)
	}
	if len(folders) != 2 {
		t.Fatalf("GetFolders() returned %d folders, want 2 (system entries skipped)", len(folders))
	}
	if folders[0].ID != 2 || folders[0].Name != "Work" || folders[0].Icon != "💼" {
		t.Errorf("folders[0] = %+v, want folder 2 Work", folders[0])
	}
}

func TestGetFolders_BinderWired(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesGetDialogFiltersTypeID, folderFiltersResult(
		&tg.DialogFilter{ID: 7, Title: &tg.TextWithEntities{Text: "News"}},
	))
	inv.setResult(tg.MessagesUpdateDialogFilterTypeID, &tg.BoolTrue{})

	folders, err := c.GetFolders(context.Background())
	if err != nil {
		t.Fatalf("GetFolders() error: %v", err)
	}
	if len(folders) != 1 {
		t.Fatalf("GetFolders() returned %d folders, want 1", len(folders))
	}
	if err := folders[0].DeleteContext(context.Background()); err != nil {
		t.Fatalf("bound DeleteContext() error: %v (binder not wired?)", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesUpdateDialogFilterRequest)
	if !ok {
		t.Fatalf("expected MessagesUpdateDialogFilterRequest, got %T", inv.lastCall())
	}
	if req.ID != 7 || req.Filter != nil {
		t.Errorf("delete request = %#v, want ID 7 with nil filter", req)
	}
}

func TestCreateFolder_SetsFilterFlag(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesUpdateDialogFilterTypeID, &tg.BoolTrue{})

	filter := &tg.DialogFilter{ID: 9, Title: &tg.TextWithEntities{Text: "New"}}
	if err := c.CreateFolder(context.Background(), filter); err != nil {
		t.Fatalf("CreateFolder() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesUpdateDialogFilterRequest)
	if !ok {
		t.Fatalf("expected MessagesUpdateDialogFilterRequest, got %T", inv.lastCall())
	}
	if req.Flags&(1<<0) == 0 {
		t.Error("create must set the filter-present flag")
	}
	if req.ID != 9 || req.Filter != filter {
		t.Errorf("request = %#v, want ID 9 carrying the filter", req)
	}
}

func TestDeleteFolder_OmitsFilter(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesUpdateDialogFilterTypeID, &tg.BoolTrue{})

	if err := c.DeleteFolder(context.Background(), 3); err != nil {
		t.Fatalf("DeleteFolder() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesUpdateDialogFilterRequest)
	if !ok {
		t.Fatalf("expected MessagesUpdateDialogFilterRequest, got %T", inv.lastCall())
	}
	if req.ID != 3 || req.Filter != nil {
		t.Errorf("request = %#v, want ID 3 with nil filter", req)
	}
}

func TestReorderFolders_PassesOrder(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesUpdateDialogFiltersOrderTypeID, &tg.BoolTrue{})

	if err := c.ReorderFolders(context.Background(), []int32{5, 2, 9}); err != nil {
		t.Fatalf("ReorderFolders() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesUpdateDialogFiltersOrderRequest)
	if !ok {
		t.Fatalf("expected MessagesUpdateDialogFiltersOrderRequest, got %T", inv.lastCall())
	}
	if len(req.Order) != 3 || req.Order[0] != 5 {
		t.Errorf("order = %v, want [5 2 9]", req.Order)
	}
}

func TestIncludeChat_AppendsAndSaves(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesGetDialogFiltersTypeID, folderFiltersResult(
		&tg.DialogFilter{ID: 2, Title: &tg.TextWithEntities{Text: "Work"}},
	))
	inv.setResult(tg.MessagesUpdateDialogFilterTypeID, &tg.BoolTrue{})

	if err := c.IncludeChat(context.Background(), 2, 10); err != nil {
		t.Fatalf("IncludeChat() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesUpdateDialogFilterRequest)
	if !ok {
		t.Fatalf("expected MessagesUpdateDialogFilterRequest, got %T", inv.lastCall())
	}
	saved, ok := req.Filter.(*tg.DialogFilter)
	if !ok {
		t.Fatalf("filter = %T, want *tg.DialogFilter", req.Filter)
	}
	if len(saved.IncludePeers) != 1 {
		t.Fatalf("include peers = %v, want one entry", saved.IncludePeers)
	}
	if p, ok := saved.IncludePeers[0].(*tg.InputPeerUser); !ok || p.UserID != 10 {
		t.Errorf("include peer = %#v, want user 10", saved.IncludePeers[0])
	}
}

func TestIncludeChat_FolderNotFound(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesGetDialogFiltersTypeID, folderFiltersResult())

	err := c.IncludeChat(context.Background(), 42, 10)
	if !errors.Is(err, ErrFolderNotFound) {
		t.Fatalf("IncludeChat() error = %v, want ErrFolderNotFound", err)
	}
}

func TestRemoveChatFromFolder_RemovesAllReferences(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesGetDialogFiltersTypeID, folderFiltersResult(
		&tg.DialogFilter{
			ID:           2,
			Title:        &tg.TextWithEntities{Text: "Work"},
			PinnedPeers:  []tg.InputPeerClass{&tg.InputPeerUser{UserID: 10, AccessHash: 20}},
			IncludePeers: []tg.InputPeerClass{&tg.InputPeerUser{UserID: 10, AccessHash: 20}, &tg.InputPeerUser{UserID: 11, AccessHash: 21}},
			ExcludePeers: []tg.InputPeerClass{&tg.InputPeerUser{UserID: 10, AccessHash: 20}},
		},
	))
	inv.setResult(tg.MessagesUpdateDialogFilterTypeID, &tg.BoolTrue{})

	if err := c.RemoveChatFromFolder(context.Background(), 2, 10); err != nil {
		t.Fatalf("RemoveChatFromFolder() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesUpdateDialogFilterRequest)
	if !ok {
		t.Fatalf("expected MessagesUpdateDialogFilterRequest, got %T", inv.lastCall())
	}
	f, ok := req.Filter.(*tg.DialogFilter)
	if !ok {
		t.Fatalf("filter = %T, want *tg.DialogFilter", req.Filter)
	}
	if len(f.PinnedPeers) != 0 {
		t.Errorf("pinned peers = %v, want empty", f.PinnedPeers)
	}
	if len(f.IncludePeers) != 1 {
		t.Errorf("include peers = %v, want only user 11", f.IncludePeers)
	}
	if len(f.ExcludePeers) != 0 {
		t.Errorf("exclude peers = %v, want empty", f.ExcludePeers)
	}
}

func TestGetFolderInviteLinks(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.ChatlistsGetExportedInvitesTypeID, &tg.ChatlistsExportedInvites{
		Invites: []tg.ExportedChatlistInviteClass{
			&tg.ExportedChatlistInvite{
				Title: "Team",
				URL:   "https://t.me/addlist/abc",
				Peers: []tg.PeerClass{&tg.PeerChannel{ChannelID: 100}},
			},
		},
	})

	links, err := c.GetFolderInviteLinks(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetFolderInviteLinks() error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1", len(links))
	}
	if links[0].InviteLink != "https://t.me/addlist/abc" || links[0].Name != "Team" {
		t.Errorf("link = %+v, want Team invite", links[0])
	}
	if len(links[0].ChatIDs) != 1 || links[0].ChatIDs[0] != -1000000000100 {
		t.Errorf("chat IDs = %v, want [-1000000000100]", links[0].ChatIDs)
	}
	req, ok := inv.lastCall().(*tg.ChatlistsGetExportedInvitesRequest)
	if !ok {
		t.Fatalf("expected ChatlistsGetExportedInvitesRequest, got %T", inv.lastCall())
	}
	if req.Chatlist == nil || req.Chatlist.FilterID != 2 {
		t.Errorf("chatlist = %#v, want folder 2", req.Chatlist)
	}
}

func TestCreateFolderInviteLink(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(100, &tg.InputPeerChannel{ChannelID: 100, AccessHash: 5})
	inv.setResult(tg.ChatlistsExportChatlistInviteTypeID, &tg.ChatlistsExportedChatlistInvite{
		Filter: &tg.DialogFilterChatlist{ID: 2},
		Invite: &tg.ExportedChatlistInvite{
			Title: "Team",
			URL:   "https://t.me/addlist/xyz",
			Peers: []tg.PeerClass{&tg.PeerChannel{ChannelID: 100}},
		},
	})

	link, err := c.CreateFolderInviteLink(context.Background(), 2, "Team", []int64{-1000000000100})
	if err != nil {
		t.Fatalf("CreateFolderInviteLink() error: %v", err)
	}
	if link == nil || link.InviteLink != "https://t.me/addlist/xyz" {
		t.Fatalf("link = %+v, want parsed invite", link)
	}
	req, ok := inv.lastCall().(*tg.ChatlistsExportChatlistInviteRequest)
	if !ok {
		t.Fatalf("expected ChatlistsExportChatlistInviteRequest, got %T", inv.lastCall())
	}
	if req.Title != "Team" || len(req.Peers) != 1 {
		t.Errorf("request = %#v, want title Team with one peer", req)
	}
}

func TestCheckFolderInviteLink(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.ChatlistsCheckChatlistInviteTypeID, &tg.ChatlistsChatlistInvite{
		Title: &tg.TextWithEntities{Text: "Team"},
	})

	invite, err := c.CheckFolderInviteLink(context.Background(), "abc")
	if err != nil {
		t.Fatalf("CheckFolderInviteLink() error: %v", err)
	}
	if _, ok := invite.(*tg.ChatlistsChatlistInvite); !ok {
		t.Fatalf("invite = %T, want ChatlistsChatlistInvite", invite)
	}
	req, ok := inv.lastCall().(*tg.ChatlistsCheckChatlistInviteRequest)
	if !ok || req.Slug != "abc" {
		t.Fatalf("request = %#v, want slug abc", inv.lastCall())
	}
}

func TestJoinFolderInviteLink(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(100, &tg.InputPeerChannel{ChannelID: 100, AccessHash: 5})
	inv.setResult(tg.ChatlistsJoinChatlistInviteTypeID, &tg.Updates{})

	if err := c.JoinFolderInviteLink(context.Background(), "abc", []int64{-1000000000100}); err != nil {
		t.Fatalf("JoinFolderInviteLink() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.ChatlistsJoinChatlistInviteRequest)
	if !ok {
		t.Fatalf("expected ChatlistsJoinChatlistInviteRequest, got %T", inv.lastCall())
	}
	if req.Slug != "abc" || len(req.Peers) != 1 {
		t.Errorf("request = %#v, want slug abc with one peer", req)
	}
}

func TestGetFolderUpdates(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.ChatlistsGetChatlistUpdatesTypeID, &tg.ChatlistsChatlistUpdates{
		MissingPeers: []tg.PeerClass{&tg.PeerChannel{ChannelID: 100}},
	})

	updates, err := c.GetFolderUpdates(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetFolderUpdates() error: %v", err)
	}
	if len(updates.MissingPeers) != 1 {
		t.Fatalf("missing peers = %v, want one", updates.MissingPeers)
	}
}

func TestLeaveFolder(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.ChatlistsLeaveChatlistTypeID, &tg.Updates{})

	if err := c.LeaveFolder(context.Background(), 2, nil); err != nil {
		t.Fatalf("LeaveFolder() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.ChatlistsLeaveChatlistRequest)
	if !ok {
		t.Fatalf("expected ChatlistsLeaveChatlistRequest, got %T", inv.lastCall())
	}
	if req.Chatlist == nil || req.Chatlist.FilterID != 2 {
		t.Errorf("chatlist = %#v, want folder 2", req.Chatlist)
	}
}
