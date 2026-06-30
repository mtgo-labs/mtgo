package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

func (c *Client) BoundGetFolders() ([]tg.DialogFilterClass, error) {
	result, err := c.Raw().MessagesGetDialogFilters(context.Background())
	if err != nil {
		return nil, err
	}
	return result.Filters, nil
}

func (c *Client) BoundCreateFolder(filter *tg.DialogFilter) error {
	ctx := context.Background()
	_, err := c.Raw().MessagesUpdateDialogFilter(ctx, &tg.MessagesUpdateDialogFilterRequest{
		Flags:  1 << 0,
		ID:     filter.ID,
		Filter: filter,
	})
	return err
}

func (c *Client) BoundEditFolder(filter *tg.DialogFilter) error {
	ctx := context.Background()
	_, err := c.Raw().MessagesUpdateDialogFilter(ctx, &tg.MessagesUpdateDialogFilterRequest{
		Flags:  1 << 0,
		ID:     filter.ID,
		Filter: filter,
	})
	return err
}

func (c *Client) BoundDeleteFolder(folderID int32) error {
	ctx := context.Background()
	_, err := c.Raw().MessagesUpdateDialogFilter(ctx, &tg.MessagesUpdateDialogFilterRequest{
		ID: folderID,
	})
	return err
}

func (c *Client) BoundReorderFolders(order []int32) error {
	ctx := context.Background()
	_, err := c.Raw().MessagesUpdateDialogFiltersOrder(ctx, &tg.MessagesUpdateDialogFiltersOrderRequest{
		Order: order,
	})
	return err
}

func (c *Client) BoundIncludeChat(folderID int32, chatID int64) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	folders, err := c.BoundGetFolders()
	if err != nil {
		return err
	}
	for _, f := range folders {
		if df, ok := f.(*tg.DialogFilter); ok && df.ID == folderID {
			df.IncludePeers = append(df.IncludePeers, peer)
			return c.BoundEditFolder(df)
		}
	}
	return fmt.Errorf("%w: %d", ErrFolderNotFound, folderID)
}

func (c *Client) BoundExcludeChat(folderID int32, chatID int64) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	folders, err := c.BoundGetFolders()
	if err != nil {
		return err
	}
	for _, f := range folders {
		if df, ok := f.(*tg.DialogFilter); ok && df.ID == folderID {
			df.ExcludePeers = append(df.ExcludePeers, peer)
			return c.BoundEditFolder(df)
		}
	}
	return fmt.Errorf("%w: %d", ErrFolderNotFound, folderID)
}

func (c *Client) BoundUpdateFolderColor(folderID int32, color int32) error {
	folders, err := c.BoundGetFolders()
	if err != nil {
		return err
	}
	for _, f := range folders {
		if df, ok := f.(*tg.DialogFilter); ok && df.ID == folderID {
			df.Color = color
			return c.BoundEditFolder(df)
		}
	}
	return fmt.Errorf("%w: %d", ErrFolderNotFound, folderID)
}

func (c *Client) BoundPinChatInFolder(folderID int32, chatID int64) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	folders, err := c.BoundGetFolders()
	if err != nil {
		return err
	}
	for _, f := range folders {
		if df, ok := f.(*tg.DialogFilter); ok && df.ID == folderID {
			df.PinnedPeers = append(df.PinnedPeers, peer)
			return c.BoundEditFolder(df)
		}
	}
	return fmt.Errorf("%w: %d", ErrFolderNotFound, folderID)
}

func (c *Client) BoundRemoveChatFromFolder(folderID int32, chatID int64) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	folders, err := c.BoundGetFolders()
	if err != nil {
		return err
	}
	for _, f := range folders {
		if df, ok := f.(*tg.DialogFilter); ok && df.ID == folderID {
			df.PinnedPeers = removePeerFromSlice(df.PinnedPeers, peer)
			df.IncludePeers = removePeerFromSlice(df.IncludePeers, peer)
			df.ExcludePeers = removePeerFromSlice(df.ExcludePeers, peer)
			return c.BoundEditFolder(df)
		}
	}
	return fmt.Errorf("%w: %d", ErrFolderNotFound, folderID)
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
