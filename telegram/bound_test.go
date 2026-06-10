package telegram

import (
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestBoundArchive(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	mock.setResult(tg.FoldersEditPeerFoldersTypeID, &tg.Updates{})

	if err := c.BoundArchive(10); err != nil {
		t.Fatalf("BoundArchive() error: %v", err)
	}

	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}

	req, ok := mock.lastCall().(*tg.FoldersEditPeerFoldersRequest)
	if !ok {
		t.Fatalf("expected FoldersEditPeerFoldersRequest, got %T", mock.lastCall())
	}

	if len(req.FolderPeers) != 1 {
		t.Fatalf("expected 1 folder peer, got %d", len(req.FolderPeers))
	}

	if req.FolderPeers[0].FolderID != 1 {
		t.Errorf("FolderID = %d, want 1", req.FolderPeers[0].FolderID)
	}

	peer, ok := req.FolderPeers[0].Peer.(*tg.InputPeerChannel)
	if !ok {
		t.Fatalf("expected InputPeerChannel, got %T", req.FolderPeers[0].Peer)
	}
	if peer.ChannelID != 10 || peer.AccessHash != 20 {
		t.Errorf("peer = (%d, %d), want (10, 20)", peer.ChannelID, peer.AccessHash)
	}
}

func TestBoundArchivePeerNotFound(t *testing.T) {
	c, _ := newClientWithBotRPCMock(t)

	err := c.BoundArchive(999)
	if err == nil {
		t.Fatal("expected error for unknown peer")
	}
	if !errors.Is(err, ErrPeerNotFound) {
		t.Errorf("error = %v, want ErrPeerNotFound", err)
	}
}

func TestBoundSetTitle(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	mock.setResult(tg.ChannelsEditTitleTypeID, &tg.Updates{})

	if err := c.BoundSetTitle(10, "New Title"); err != nil {
		t.Fatalf("BoundSetTitle() error: %v", err)
	}

	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}

	req, ok := mock.lastCall().(*tg.ChannelsEditTitleRequest)
	if !ok {
		t.Fatalf("expected ChannelsEditTitleRequest, got %T", mock.lastCall())
	}
	if req.Title != "New Title" {
		t.Errorf("Title = %q, want %q", req.Title, "New Title")
	}

	ch, ok := req.Channel.(*tg.InputChannel)
	if !ok {
		t.Fatalf("expected InputChannel, got %T", req.Channel)
	}
	if ch.ChannelID != 10 || ch.AccessHash != 20 {
		t.Errorf("channel = (%d, %d), want (10, 20)", ch.ChannelID, ch.AccessHash)
	}
}

func TestBoundSetTitleChatPeer(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(50, &tg.InputPeerChat{ChatID: 50})
	mock.setResult(tg.MessagesEditChatTitleTypeID, &tg.Updates{})

	if err := c.BoundSetTitle(50, "Chat Title"); err != nil {
		t.Fatalf("BoundSetTitle() error: %v", err)
	}

	req, ok := mock.lastCall().(*tg.MessagesEditChatTitleRequest)
	if !ok {
		t.Fatalf("expected MessagesEditChatTitleRequest, got %T", mock.lastCall())
	}
	if req.ChatID != 50 {
		t.Errorf("ChatID = %d, want 50", req.ChatID)
	}
	if req.Title != "Chat Title" {
		t.Errorf("Title = %q, want %q", req.Title, "Chat Title")
	}
}

func TestBoundSetTitlePeerNotFound(t *testing.T) {
	c, _ := newClientWithBotRPCMock(t)

	err := c.BoundSetTitle(999, "test")
	if err == nil {
		t.Fatal("expected error for unknown peer")
	}
	if !errors.Is(err, ErrPeerNotFound) {
		t.Errorf("error = %v, want ErrPeerNotFound", err)
	}
}

func TestBoundSetDescription(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	if err := c.BoundSetDescription(10, "New description"); err != nil {
		t.Fatalf("BoundSetDescription() error: %v", err)
	}

	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}

	req, ok := mock.lastCall().(*tg.MessagesEditChatAboutRequest)
	if !ok {
		t.Fatalf("expected MessagesEditChatAboutRequest, got %T", mock.lastCall())
	}
	if req.About != "New description" {
		t.Errorf("About = %q, want %q", req.About, "New description")
	}

	peer, ok := req.Peer.(*tg.InputPeerChannel)
	if !ok {
		t.Fatalf("expected InputPeerChannel, got %T", req.Peer)
	}
	if peer.ChannelID != 10 {
		t.Errorf("ChannelID = %d, want 10", peer.ChannelID)
	}
}

func TestBoundSetDescriptionChatPeer(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(50, &tg.InputPeerChat{ChatID: 50})

	if err := c.BoundSetDescription(50, "Chat desc"); err != nil {
		t.Fatalf("BoundSetDescription() error: %v", err)
	}

	req, ok := mock.lastCall().(*tg.MessagesEditChatAboutRequest)
	if !ok {
		t.Fatalf("expected MessagesEditChatAboutRequest, got %T", mock.lastCall())
	}
	if req.About != "Chat desc" {
		t.Errorf("About = %q, want %q", req.About, "Chat desc")
	}
}

func TestBoundBanMember(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.CachePeer(42, &tg.InputPeerUser{UserID: 42, AccessHash: 100})
	mock.setResult(tg.ChannelsEditBannedTypeID, &tg.Updates{})

	if err := c.BoundBanMember(10, 42); err != nil {
		t.Fatalf("BoundBanMember() error: %v", err)
	}

	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}

	req, ok := mock.lastCall().(*tg.ChannelsEditBannedRequest)
	if !ok {
		t.Fatalf("expected ChannelsEditBannedRequest, got %T", mock.lastCall())
	}

	ch, ok := req.Channel.(*tg.InputChannel)
	if !ok {
		t.Fatalf("expected InputChannel, got %T", req.Channel)
	}
	if ch.ChannelID != 10 || ch.AccessHash != 20 {
		t.Errorf("channel = (%d, %d), want (10, 20)", ch.ChannelID, ch.AccessHash)
	}

	if req.BannedRights == nil {
		t.Fatal("BannedRights should not be nil")
	}
	if !req.BannedRights.ViewMessages {
		t.Error("ViewMessages should be true for a full ban")
	}
}

func TestBoundBanMemberNonChannelPeer(t *testing.T) {
	c, _ := newClientWithBotRPCMock(t)
	c.CachePeer(50, &tg.InputPeerChat{ChatID: 50})

	err := c.BoundBanMember(50, 42)
	if err == nil {
		t.Fatal("expected error for non-channel peer")
	}
	if !errors.Is(err, ErrBanSupergroupOnly) {
		t.Errorf("error = %v, want ErrBanSupergroupOnly", err)
	}
}

func TestBoundBanMemberPeerNotFound(t *testing.T) {
	c, _ := newClientWithBotRPCMock(t)

	err := c.BoundBanMember(999, 42)
	if err == nil {
		t.Fatal("expected error for unknown peer")
	}
	if !errors.Is(err, ErrPeerNotFound) {
		t.Errorf("error = %v, want ErrPeerNotFound", err)
	}
}

func TestBoundJoinChat(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.MessagesImportChatInviteTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{},
		Users:   []tg.UserClass{},
		Chats: []tg.ChatClass{
			&tg.Channel{ID: 100, Title: "Test Channel"},
		},
	})

	chat, err := c.BoundJoinChat(0, "a1b2c3d4e5f6g7h8i9j0")
	if err != nil {
		t.Fatalf("BoundJoinChat() error: %v", err)
	}
	if chat == nil {
		t.Fatal("expected non-nil chat")
	}

	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}

	req, ok := mock.lastCall().(*tg.MessagesImportChatInviteRequest)
	if !ok {
		t.Fatalf("expected MessagesImportChatInviteRequest, got %T", mock.lastCall())
	}
	if req.Hash != "a1b2c3d4e5f6g7h8i9j0" {
		t.Errorf("Hash = %q, want %q", req.Hash, "a1b2c3d4e5f6g7h8i9j0")
	}
}

func TestBoundJoinChatURL(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.MessagesImportChatInviteTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{},
		Users:   []tg.UserClass{},
		Chats: []tg.ChatClass{
			&tg.Channel{ID: 200, Title: "Private Channel"},
		},
	})

	chat, err := c.BoundJoinChat(0, "https://t.me/+tPknWlMV9eU3NThk")
	if err != nil {
		t.Fatalf("BoundJoinChat() error: %v", err)
	}
	if chat == nil {
		t.Fatal("expected non-nil chat")
	}

	req, ok := mock.lastCall().(*tg.MessagesImportChatInviteRequest)
	if !ok {
		t.Fatalf("expected MessagesImportChatInviteRequest, got %T", mock.lastCall())
	}
	if req.Hash != "tPknWlMV9eU3NThk" {
		t.Errorf("Hash = %q, want %q", req.Hash, "tPknWlMV9eU3NThk")
	}
}

func TestBoundJoinChatByUsername(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.ContactsResolveUsernameTypeID, &tg.ContactsResolvedPeer{
		Peer: &tg.PeerChannel{ChannelID: 300},
		Chats: []tg.ChatClass{
			&tg.Channel{ID: 300, Title: "mtgo labs", AccessHash: 12345},
		},
		Users: []tg.UserClass{},
	})
	mock.setResult(tg.ChannelsJoinChannelTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{},
		Users:   []tg.UserClass{},
		Chats: []tg.ChatClass{
			&tg.Channel{ID: 300, Title: "mtgo labs"},
		},
	})

	chat, err := c.BoundJoinChat(0, "mtgo_labs")
	if err != nil {
		t.Fatalf("BoundJoinChat() error: %v", err)
	}
	if chat == nil {
		t.Fatal("expected non-nil chat")
	}
	if mock.callCount() != 2 {
		t.Fatalf("expected 2 RPC calls, got %d", mock.callCount())
	}

	resolveReq, ok := mock.getCall(0).(*tg.ContactsResolveUsernameRequest)
	if !ok {
		t.Fatalf("expected ContactsResolveUsernameRequest, got %T", mock.getCall(0))
	}
	if resolveReq.Username != "mtgo_labs" {
		t.Errorf("Username = %q, want %q", resolveReq.Username, "mtgo_labs")
	}

	joinReq, ok := mock.getCall(1).(*tg.ChannelsJoinChannelRequest)
	if !ok {
		t.Fatalf("expected ChannelsJoinChannelRequest, got %T", mock.getCall(1))
	}
	if ch, ok := joinReq.Channel.(*tg.InputChannel); ok {
		if ch.ChannelID != 300 {
			t.Errorf("ChannelID = %d, want %d", ch.ChannelID, 300)
		}
		if ch.AccessHash != 12345 {
			t.Errorf("AccessHash = %d, want %d", ch.AccessHash, 12345)
		}
	} else {
		t.Fatalf("expected *tg.InputChannel, got %T", joinReq.Channel)
	}
}

func TestBoundJoinChatURLUsername(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.ContactsResolveUsernameTypeID, &tg.ContactsResolvedPeer{
		Peer: &tg.PeerChannel{ChannelID: 400},
		Chats: []tg.ChatClass{
			&tg.Channel{ID: 400, Title: "mtgo labs", AccessHash: 99},
		},
		Users: []tg.UserClass{},
	})
	mock.setResult(tg.ChannelsJoinChannelTypeID, &tg.Updates{
		Updates: []tg.UpdateClass{},
		Users:   []tg.UserClass{},
		Chats: []tg.ChatClass{
			&tg.Channel{ID: 400, Title: "mtgo labs"},
		},
	})

	chat, err := c.BoundJoinChat(0, "https://t.me/mtgo_labs")
	if err != nil {
		t.Fatalf("BoundJoinChat() error: %v", err)
	}
	if chat == nil {
		t.Fatal("expected non-nil chat")
	}

	resolveReq, ok := mock.getCall(0).(*tg.ContactsResolveUsernameRequest)
	if !ok {
		t.Fatalf("expected ContactsResolveUsernameRequest, got %T", mock.getCall(0))
	}
	if resolveReq.Username != "mtgo_labs" {
		t.Errorf("Username = %q, want %q", resolveReq.Username, "mtgo_labs")
	}
}

func TestBoundJoinChatEmptyUsername(t *testing.T) {
	c, _ := newClientWithBotRPCMock(t)

	_, err := c.BoundJoinChat(0, "")
	if err == nil {
		t.Fatal("expected error for empty username")
	}
	if !errors.Is(err, ErrJoinRequiresInvite) {
		t.Errorf("error = %v, want ErrJoinRequiresInvite", err)
	}
}

func TestBoundLeaveChat(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	mock.setResult(tg.ChannelsLeaveChannelTypeID, &tg.Updates{})

	if err := c.BoundLeaveChat(10); err != nil {
		t.Fatalf("BoundLeaveChat() error: %v", err)
	}

	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}

	req, ok := mock.lastCall().(*tg.ChannelsLeaveChannelRequest)
	if !ok {
		t.Fatalf("expected ChannelsLeaveChannelRequest, got %T", mock.lastCall())
	}

	ch, ok := req.Channel.(*tg.InputChannel)
	if !ok {
		t.Fatalf("expected InputChannel, got %T", req.Channel)
	}
	if ch.ChannelID != 10 || ch.AccessHash != 20 {
		t.Errorf("channel = (%d, %d), want (10, 20)", ch.ChannelID, ch.AccessHash)
	}
}

func TestBoundLeaveChatPeer(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	c.CachePeer(50, &tg.InputPeerChat{ChatID: 50})
	mock.setResult(tg.MessagesDeleteChatUserTypeID, &tg.Updates{})

	if err := c.BoundLeaveChat(50); err != nil {
		t.Fatalf("BoundLeaveChat() error: %v", err)
	}

	req, ok := mock.lastCall().(*tg.MessagesDeleteChatUserRequest)
	if !ok {
		t.Fatalf("expected MessagesDeleteChatUserRequest, got %T", mock.lastCall())
	}
	if req.ChatID != 50 {
		t.Errorf("ChatID = %d, want 50", req.ChatID)
	}
}

func TestBoundLeaveChatPeerNotFound(t *testing.T) {
	c, _ := newClientWithBotRPCMock(t)

	err := c.BoundLeaveChat(999)
	if err == nil {
		t.Fatal("expected error for unknown peer")
	}
	if !errors.Is(err, ErrPeerNotFound) {
		t.Errorf("error = %v, want ErrPeerNotFound", err)
	}
}

func TestBoundAddToGifs(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	doc := &tg.InputDocument{ID: 123, AccessHash: 456}
	if err := c.BoundAddToGifs(doc); err != nil {
		t.Fatalf("BoundAddToGifs() error: %v", err)
	}

	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}

	req, ok := mock.lastCall().(*tg.MessagesSaveGIFRequest)
	if !ok {
		t.Fatalf("expected MessagesSaveGIFRequest, got %T", mock.lastCall())
	}
	if req.Unsave {
		t.Error("Unsave should be false for add")
	}

	id, ok := req.ID.(*tg.InputDocument)
	if !ok {
		t.Fatalf("expected InputDocument, got %T", req.ID)
	}
	if id.ID != 123 || id.AccessHash != 456 {
		t.Errorf("doc = (%d, %d), want (123, 456)", id.ID, id.AccessHash)
	}
}

func TestBoundRemoveFromGifs(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	doc := &tg.InputDocument{ID: 789, AccessHash: 101}
	if err := c.BoundRemoveFromGifs(doc); err != nil {
		t.Fatalf("BoundRemoveFromGifs() error: %v", err)
	}

	req, ok := mock.lastCall().(*tg.MessagesSaveGIFRequest)
	if !ok {
		t.Fatalf("expected MessagesSaveGIFRequest, got %T", mock.lastCall())
	}
	if !req.Unsave {
		t.Error("Unsave should be true for remove")
	}
}

func TestBoundGetSavedGifs(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	mock.setResult(tg.MessagesGetSavedGifsTypeID, &tg.MessagesSavedGifs{
		Gifs: []tg.DocumentClass{},
	})

	result, err := c.BoundGetSavedGifs(0)
	if err != nil {
		t.Fatalf("BoundGetSavedGifs() error: %v", err)
	}

	saved, ok := result.(*tg.MessagesSavedGifs)
	if !ok {
		t.Fatalf("expected *tg.MessagesSavedGifs, got %T", result)
	}
	if len(saved.Gifs) != 0 {
		t.Errorf("expected 0 gifs, got %d", len(saved.Gifs))
	}

	if mock.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", mock.callCount())
	}

	req, ok := mock.lastCall().(*tg.MessagesGetSavedGifsRequest)
	if !ok {
		t.Fatalf("expected MessagesGetSavedGifsRequest, got %T", mock.lastCall())
	}
	if req.Hash != 0 {
		t.Errorf("Hash = %d, want 0", req.Hash)
	}
}

func TestBoundGetSavedGifsRPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)
	rpcErr := errors.New("rpc error")
	mock.setError(tg.MessagesGetSavedGifsTypeID, rpcErr)

	_, err := c.BoundGetSavedGifs(0)
	if err == nil {
		t.Fatal("expected error from RPC")
	}
}
