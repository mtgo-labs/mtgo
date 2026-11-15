package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/internal/peerid"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// ArchiveChat moves the specified chat to the archived chats folder (folder ID
// 1). Archived chats are hidden from the main chat list.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat to archive
//
// Returns an error if the peer cannot be resolved or the folder update fails.
//
// Example:
//
//	ctx := context.Background()
//	err := client.ArchiveChat(ctx, chatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *ChatMethods) ArchiveChat(ctx context.Context, chatID int64) error {
	c.Log.Debugf("ArchiveChat chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.FoldersEditPeerFolders(ctx, &tg.FoldersEditPeerFoldersRequest{
		FolderPeers: []*tg.InputFolderPeer{
			{Peer: peer, FolderID: 1},
		},
	})
	return err
}

// UnarchiveChat moves the specified chat from the archived folder back to the
// main chat list (folder ID 0).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat to unarchive
//
// Returns an error if the peer cannot be resolved or the folder update fails.
//
// Example:
//
//	ctx := context.Background()
//	err := client.UnarchiveChat(ctx, chatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *ChatMethods) UnarchiveChat(ctx context.Context, chatID int64) error {
	c.Log.Debugf("UnarchiveChat chat_id=%d", chatID)
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.FoldersEditPeerFolders(ctx, &tg.FoldersEditPeerFoldersRequest{
		FolderPeers: []*tg.InputFolderPeer{
			{Peer: peer, FolderID: 0},
		},
	})
	return err
}

// GetFolders returns the user's chat folders. Each folder carries bound
// management methods. Special system entries (the default All-Chats folder and
// shareable chatlist filters) are not part of the manageable set and are
// skipped.
//
// Example:
//
//	folders, err := client.GetFolders(ctx)
//	for _, f := range folders {
//	    fmt.Println(f.ID, f.Name)
//	}
func (c *FolderMethods) GetFolders(ctx context.Context) ([]*types.Folder, error) {
	result, err := c.Raw().MessagesGetDialogFilters(ctx)
	if err != nil {
		return nil, fmt.Errorf("get folders: %w", err)
	}
	folders := make([]*types.Folder, 0, len(result.Filters))
	for _, f := range result.Filters {
		df, ok := f.(*tg.DialogFilter)
		if !ok {
			continue
		}
		folder := types.ParseFolder(df)
		folder.SetBinder(c)
		folders = append(folders, folder)
	}
	return folders, nil
}

// findDialogFilter fetches the current folder set and returns the raw filter
// with the given ID.
func (c *FolderMethods) findDialogFilter(ctx context.Context, folderID int32) (*tg.DialogFilter, error) {
	result, err := c.Raw().MessagesGetDialogFilters(ctx)
	if err != nil {
		return nil, fmt.Errorf("get folders: %w", err)
	}
	for _, f := range result.Filters {
		if df, ok := f.(*tg.DialogFilter); ok && df.ID == folderID {
			return df, nil
		}
	}
	return nil, fmt.Errorf("%w: %d", ErrFolderNotFound, folderID)
}

// CreateFolder creates a new chat folder from the given filter definition.
func (c *FolderMethods) CreateFolder(ctx context.Context, filter *tg.DialogFilter) error {
	_, err := c.Raw().MessagesUpdateDialogFilter(ctx, &tg.MessagesUpdateDialogFilterRequest{
		Flags:  1 << 0,
		ID:     filter.ID,
		Filter: filter,
	})
	return err
}

// EditFolder saves changes to an existing chat folder.
func (c *FolderMethods) EditFolder(ctx context.Context, filter *tg.DialogFilter) error {
	_, err := c.Raw().MessagesUpdateDialogFilter(ctx, &tg.MessagesUpdateDialogFilterRequest{
		Flags:  1 << 0,
		ID:     filter.ID,
		Filter: filter,
	})
	return err
}

// DeleteFolder removes a chat folder by ID. Chats inside the folder are not
// deleted; they simply return to the default view.
func (c *FolderMethods) DeleteFolder(ctx context.Context, folderID int32) error {
	_, err := c.Raw().MessagesUpdateDialogFilter(ctx, &tg.MessagesUpdateDialogFilterRequest{
		ID: folderID,
	})
	return err
}

// ReorderFolders changes the order of chat folders. order lists folder IDs in
// their new display order.
func (c *FolderMethods) ReorderFolders(ctx context.Context, order []int32) error {
	_, err := c.Raw().MessagesUpdateDialogFiltersOrder(ctx, &tg.MessagesUpdateDialogFiltersOrderRequest{
		Order: order,
	})
	return err
}

// IncludeChat adds a chat to a folder's explicit include list, so the folder
// always shows it regardless of the folder's rules.
func (c *FolderMethods) IncludeChat(ctx context.Context, folderID int32, chatID int64) error {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	df, err := c.findDialogFilter(ctx, folderID)
	if err != nil {
		return err
	}
	df.IncludePeers = append(df.IncludePeers, peer)
	return c.EditFolder(ctx, df)
}

// ExcludeChat adds a chat to a folder's exclusion list, so the folder never
// shows it even when the folder's rules would match.
func (c *FolderMethods) ExcludeChat(ctx context.Context, folderID int32, chatID int64) error {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	df, err := c.findDialogFilter(ctx, folderID)
	if err != nil {
		return err
	}
	df.ExcludePeers = append(df.ExcludePeers, peer)
	return c.EditFolder(ctx, df)
}

// UpdateFolderColor sets the accent color of a folder.
func (c *FolderMethods) UpdateFolderColor(ctx context.Context, folderID int32, color int32) error {
	df, err := c.findDialogFilter(ctx, folderID)
	if err != nil {
		return err
	}
	df.Color = color
	return c.EditFolder(ctx, df)
}

// PinChatInFolder pins a chat inside a folder, keeping it at the top of the
// folder's chat list.
func (c *FolderMethods) PinChatInFolder(ctx context.Context, folderID int32, chatID int64) error {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	df, err := c.findDialogFilter(ctx, folderID)
	if err != nil {
		return err
	}
	df.PinnedPeers = append(df.PinnedPeers, peer)
	return c.EditFolder(ctx, df)
}

// RemoveChatFromFolder removes every reference to a chat from a folder: pin,
// include, and exclude entries.
func (c *FolderMethods) RemoveChatFromFolder(ctx context.Context, folderID int32, chatID int64) error {
	peer, err := resolvePeer(ctx, c.Client, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	df, err := c.findDialogFilter(ctx, folderID)
	if err != nil {
		return err
	}
	df.PinnedPeers = removePeerFromSlice(df.PinnedPeers, peer)
	df.IncludePeers = removePeerFromSlice(df.IncludePeers, peer)
	df.ExcludePeers = removePeerFromSlice(df.ExcludePeers, peer)
	return c.EditFolder(ctx, df)
}

func removePeerFromSlice(peers []tg.InputPeerClass, target tg.InputPeerClass) []tg.InputPeerClass {
	result := make([]tg.InputPeerClass, 0, len(peers))
	for _, p := range peers {
		if !peersEqual(p, target) {
			result = append(result, p)
		}
	}
	return result
}

func peersEqual(a, b tg.InputPeerClass) bool {
	switch av := a.(type) {
	case *tg.InputPeerUser:
		if bv, ok := b.(*tg.InputPeerUser); ok {
			return av.UserID == bv.UserID
		}
	case *tg.InputPeerChat:
		if bv, ok := b.(*tg.InputPeerChat); ok {
			return av.ChatID == bv.ChatID
		}
	case *tg.InputPeerChannel:
		if bv, ok := b.(*tg.InputPeerChannel); ok {
			return av.ChannelID == bv.ChannelID
		}
	}
	return false
}

// folderChatlistInput builds the chatlist reference for a folder ID.
func folderChatlistInput(folderID int32) *tg.InputChatlistDialogFilter {
	return &tg.InputChatlistDialogFilter{FilterID: folderID}
}

// resolveFolderPeers resolves chat IDs into input peers for chatlist
// membership operations.
func resolveFolderPeers(ctx context.Context, c *Client, chatIDs []int64) ([]tg.InputPeerClass, error) {
	peers := make([]tg.InputPeerClass, 0, len(chatIDs))
	for _, id := range chatIDs {
		peer, err := resolvePeer(ctx, c, id)
		if err != nil {
			return nil, fmt.Errorf("resolve peer: %w", err)
		}
		peers = append(peers, peer)
	}
	return peers, nil
}

// GetFolderInviteLinks lists the invite links exported for a shareable folder
// (chatlist).
func (c *FolderMethods) GetFolderInviteLinks(ctx context.Context, folderID int32) ([]*types.FolderInviteLink, error) {
	result, err := c.Raw().ChatlistsGetExportedInvites(ctx, &tg.ChatlistsGetExportedInvitesRequest{
		Chatlist: folderChatlistInput(folderID),
	})
	if err != nil {
		return nil, fmt.Errorf("get folder invite links: %w", err)
	}
	links := make([]*types.FolderInviteLink, 0, len(result.Invites))
	for _, inv := range result.Invites {
		if e, ok := inv.(*tg.ExportedChatlistInvite); ok {
			links = append(links, types.ParseFolderInviteLink(e))
		}
	}
	return links, nil
}

// CreateFolderInviteLink exports a new invite link for a shareable folder.
// chatIDs lists the chats shared through the link.
func (c *FolderMethods) CreateFolderInviteLink(ctx context.Context, folderID int32, title string, chatIDs []int64) (*types.FolderInviteLink, error) {
	peers, err := resolveFolderPeers(ctx, c.Client, chatIDs)
	if err != nil {
		return nil, err
	}
	result, err := c.Raw().ChatlistsExportChatlistInvite(ctx, &tg.ChatlistsExportChatlistInviteRequest{
		Chatlist: folderChatlistInput(folderID),
		Title:    title,
		Peers:    peers,
	})
	if err != nil {
		return nil, fmt.Errorf("create folder invite link: %w", err)
	}
	if exported, ok := result.(*tg.ChatlistsExportedChatlistInvite); ok {
		if invite, ok := exported.Invite.(*tg.ExportedChatlistInvite); ok {
			return types.ParseFolderInviteLink(invite), nil
		}
	}
	return nil, nil
}

// EditFolderInviteLink changes the title or shared-chat set of an exported
// folder invite link, identified by its slug. Pass an empty title to keep the
// current one.
func (c *FolderMethods) EditFolderInviteLink(ctx context.Context, folderID int32, slug, title string, chatIDs []int64) (*types.FolderInviteLink, error) {
	peers, err := resolveFolderPeers(ctx, c.Client, chatIDs)
	if err != nil {
		return nil, err
	}
	req := &tg.ChatlistsEditExportedInviteRequest{
		Chatlist: folderChatlistInput(folderID),
		Slug:     slug,
	}
	if title != "" {
		req.Title = title
	}
	if len(peers) > 0 {
		req.Peers = peers
	}
	result, err := c.Raw().ChatlistsEditExportedInvite(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("edit folder invite link: %w", err)
	}
	if invite, ok := result.(*tg.ExportedChatlistInvite); ok {
		return types.ParseFolderInviteLink(invite), nil
	}
	return nil, nil
}

// DeleteFolderInviteLink revokes and deletes an exported folder invite link.
func (c *FolderMethods) DeleteFolderInviteLink(ctx context.Context, folderID int32, slug string) error {
	_, err := c.Raw().ChatlistsDeleteExportedInvite(ctx, &tg.ChatlistsDeleteExportedInviteRequest{
		Chatlist: folderChatlistInput(folderID),
		Slug:     slug,
	})
	if err != nil {
		return fmt.Errorf("delete folder invite link: %w", err)
	}
	return nil
}

// CheckFolderInviteLink inspects a folder invite link without joining it. The
// returned value describes the folder when the invite has not yet been used,
// or the chats already shared when it has.
func (c *FolderMethods) CheckFolderInviteLink(ctx context.Context, slug string) (tg.ChatlistInviteClass, error) {
	result, err := c.Raw().ChatlistsCheckChatlistInvite(ctx, &tg.ChatlistsCheckChatlistInviteRequest{
		Slug: slug,
	})
	if err != nil {
		return nil, fmt.Errorf("check folder invite link: %w", err)
	}
	return result, nil
}

// JoinFolderInviteLink joins a shareable folder via its invite link. chatIDs
// selects the chats to join from the folder.
func (c *FolderMethods) JoinFolderInviteLink(ctx context.Context, slug string, chatIDs []int64) error {
	peers, err := resolveFolderPeers(ctx, c.Client, chatIDs)
	if err != nil {
		return err
	}
	_, err = c.Raw().ChatlistsJoinChatlistInvite(ctx, &tg.ChatlistsJoinChatlistInviteRequest{
		Slug:  slug,
		Peers: peers,
	})
	if err != nil {
		return fmt.Errorf("join folder invite link: %w", err)
	}
	return nil
}

// GetFolderUpdates lists chats added to a shareable folder since the last
// join, so the user can accept or hide them.
func (c *FolderMethods) GetFolderUpdates(ctx context.Context, folderID int32) (*tg.ChatlistsChatlistUpdates, error) {
	result, err := c.Raw().ChatlistsGetChatlistUpdates(ctx, &tg.ChatlistsGetChatlistUpdatesRequest{
		Chatlist: folderChatlistInput(folderID),
	})
	if err != nil {
		return nil, fmt.Errorf("get folder updates: %w", err)
	}
	return result, nil
}

// JoinFolderUpdates accepts newly added chats of a shareable folder. chatIDs
// selects the new chats to join.
func (c *FolderMethods) JoinFolderUpdates(ctx context.Context, folderID int32, chatIDs []int64) error {
	peers, err := resolveFolderPeers(ctx, c.Client, chatIDs)
	if err != nil {
		return err
	}
	_, err = c.Raw().ChatlistsJoinChatlistUpdates(ctx, &tg.ChatlistsJoinChatlistUpdatesRequest{
		Chatlist: folderChatlistInput(folderID),
		Peers:    peers,
	})
	if err != nil {
		return fmt.Errorf("join folder updates: %w", err)
	}
	return nil
}

// HideFolderUpdates dismisses newly added chats of a shareable folder without
// joining them.
func (c *FolderMethods) HideFolderUpdates(ctx context.Context, folderID int32) error {
	_, err := c.Raw().ChatlistsHideChatlistUpdates(ctx, &tg.ChatlistsHideChatlistUpdatesRequest{
		Chatlist: folderChatlistInput(folderID),
	})
	if err != nil {
		return fmt.Errorf("hide folder updates: %w", err)
	}
	return nil
}

// LeaveFolder leaves a shareable folder. chatIDs optionally selects chats to
// leave as well.
func (c *FolderMethods) LeaveFolder(ctx context.Context, folderID int32, chatIDs []int64) error {
	peers, err := resolveFolderPeers(ctx, c.Client, chatIDs)
	if err != nil {
		return err
	}
	_, err = c.Raw().ChatlistsLeaveChatlist(ctx, &tg.ChatlistsLeaveChatlistRequest{
		Chatlist: folderChatlistInput(folderID),
		Peers:    peers,
	})
	if err != nil {
		return fmt.Errorf("leave folder: %w", err)
	}
	return nil
}

// GetLeaveFolderSuggestions lists chats the server suggests also leaving when
// leaving a shareable folder.
func (c *FolderMethods) GetLeaveFolderSuggestions(ctx context.Context, folderID int32) ([]int64, error) {
	raw, err := c.Raw().ChatlistsGetLeaveChatlistSuggestions(ctx, &tg.ChatlistsGetLeaveChatlistSuggestionsRequest{
		Chatlist: folderChatlistInput(folderID),
	})
	if err != nil {
		return nil, fmt.Errorf("get leave folder suggestions: %w", err)
	}
	vector, ok := raw.(*tg.GenericVector)
	if !ok {
		return nil, fmt.Errorf("get leave folder suggestions: unexpected result %T", raw)
	}
	ids := make([]int64, 0, len(vector.Items))
	for _, item := range vector.Items {
		if peer, ok := item.(tg.PeerClass); ok {
			ids = append(ids, peerid.MarkFromPeer(peer))
		}
	}
	return ids, nil
}
