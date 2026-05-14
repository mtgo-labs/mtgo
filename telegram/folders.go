package telegram

import (
	"context"
	"fmt"

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
func (c *Client) ArchiveChat(ctx context.Context, chatID int64) error {
	c.Log.Debugf("ArchiveChat chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
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
func (c *Client) UnarchiveChat(ctx context.Context, chatID int64) error {
	c.Log.Debugf("UnarchiveChat chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
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
