package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// InviteLinkOption configures optional parameters when creating or editing a
// chat invite link.
type InviteLinkOption struct {
	// ExpireDate is the Unix timestamp when the invite link expires. nil means
	// the link never expires.
	ExpireDate *int32
	// UsageLimit is the maximum number of times the link can be used. nil means
	// unlimited uses.
	UsageLimit *int32
	// Title is a human-readable name for the invite link.
	Title *string
}

// GetChatInviteLink retrieves an existing exported invite link for the
// specified chat.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat or channel
//   - link: the invite link string to look up
//
// Returns the ChatInviteLink object or an error if the peer cannot be resolved
// or the link does not exist.
//
// Example:
//
//	link, err := client.GetChatInviteLink(ctx, chatID, "https://t.me/+AbCdEfGh")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Invite link: %s\n", link.InviteLink)
func (c *Client) GetChatInviteLink(ctx context.Context, chatID int64, link string) (*types.ChatInviteLink, error) {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetExportedChatInvite(ctx, &tg.MessagesGetExportedChatInviteRequest{
		Peer: peer,
		Link: link,
	})
	if err != nil {
		return nil, err
	}
	return extractInviteLink(result)
}

// CreateChatInviteLink generates a new invite link for the specified chat. Use
// opts to set an expiration date, usage limit, or title.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat or channel
//   - opts: optional [InviteLinkOption] configuration
//
// Returns the newly created ChatInviteLink or an error if the peer cannot be
// resolved or the creation fails.
//
// Example:
//
//	link, err := client.CreateChatInviteLink(ctx, chatID, &telegram.InviteLinkOption{
//	    Title:      "Team Invite",
//	    UsageLimit: ptr.Int32(10),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Created: %s\n", link.InviteLink)
func (c *Client) CreateChatInviteLink(ctx context.Context, chatID int64, opts ...*InviteLinkOption) (*types.ChatInviteLink, error) {
	c.Log.Debugf("CreateChatInviteLink chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	opt := getOptDef(&InviteLinkOption{}, opts...)
	var expireDate int32
	if opt.ExpireDate != nil {
		expireDate = *opt.ExpireDate
	}
	var usageLimit int32
	if opt.UsageLimit != nil {
		usageLimit = *opt.UsageLimit
	}
	var title string
	if opt.Title != nil {
		title = *opt.Title
	}

	rpc := c.Raw()
	result, err := rpc.MessagesExportChatInvite(ctx, &tg.MessagesExportChatInviteRequest{
		Peer:       peer,
		ExpireDate: expireDate,
		UsageLimit: usageLimit,
		Title:      title,
	})
	if err != nil {
		return nil, err
	}
	return extractInviteLink(result)
}

// EditChatInviteLink modifies an existing invite link's properties (expiration,
// usage limit, title).
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat or channel
//   - link: the invite link string to edit
//   - opts: optional [InviteLinkOption] with updated values
//
// Returns the modified ChatInviteLink or an error if the peer cannot be
// resolved or the edit fails.
//
// Example:
//
//	updated, err := client.EditChatInviteLink(ctx, chatID, "https://t.me/+AbCdEfGh", &telegram.InviteLinkOption{
//	    UsageLimit: ptr.Int32(50),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Updated: %s\n", updated.InviteLink)
func (c *Client) EditChatInviteLink(ctx context.Context, chatID int64, link string, opts ...*InviteLinkOption) (*types.ChatInviteLink, error) {
	c.Log.Debugf("EditChatInviteLink chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	opt := getOptDef(&InviteLinkOption{}, opts...)
	var expireDate int32
	if opt.ExpireDate != nil {
		expireDate = *opt.ExpireDate
	}
	var usageLimit int32
	if opt.UsageLimit != nil {
		usageLimit = *opt.UsageLimit
	}
	var title string
	if opt.Title != nil {
		title = *opt.Title
	}

	rpc := c.Raw()
	result, err := rpc.MessagesEditExportedChatInvite(ctx, &tg.MessagesEditExportedChatInviteRequest{
		Peer:       peer,
		Link:       link,
		ExpireDate: expireDate,
		UsageLimit: usageLimit,
		Title:      title,
	})
	if err != nil {
		return nil, err
	}
	return extractInviteLink(result)
}

// RevokeChatInviteLink revokes an existing invite link, preventing new users
// from joining with it.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat or channel
//   - link: the invite link string to revoke
//
// Returns the revoked ChatInviteLink or an error if the peer cannot be resolved
// or the revocation fails.
//
// Example:
//
//	revoked, err := client.RevokeChatInviteLink(ctx, chatID, "https://t.me/+AbCdEfGh")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Revoked: %v\n", revoked.Revoked)
func (c *Client) RevokeChatInviteLink(ctx context.Context, chatID int64, link string) (*types.ChatInviteLink, error) {
	c.Log.Debugf("RevokeChatInviteLink chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesEditExportedChatInvite(ctx, &tg.MessagesEditExportedChatInviteRequest{
		Revoked: true,
		Peer:    peer,
		Link:    link,
	})
	if err != nil {
		return nil, err
	}
	return extractInviteLink(result)
}

// ExportChatInviteLink creates a new primary invite link for the specified chat
// and returns the link string. This is a convenience wrapper around
// [Client.CreateChatInviteLink].
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat or channel
//
// Returns the invite link string or an error if creation fails.
//
// Example:
//
//	linkStr, err := client.ExportChatInviteLink(ctx, chatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Share this link: %s\n", linkStr)
func (c *Client) ExportChatInviteLink(ctx context.Context, chatID int64) (string, error) {
	link, err := c.CreateChatInviteLink(ctx, chatID)
	if err != nil {
		return "", err
	}
	return link.InviteLink, nil
}

func extractInviteLink(result tg.ExportedChatInviteClass) (*types.ChatInviteLink, error) {
	switch v := result.(type) {
	case *tg.ChatInviteExported:
		users := make(map[int64]tg.UserClass)
		link := types.ParseChatInviteLink(v, users)
		return link, nil
	default:
		return nil, fmt.Errorf("unexpected invite link type %T", result)
	}
}

// GetChatInviteLinkJoiners retrieves the list of users who joined the chat via
// the specified invite link.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat or channel
//   - link: the invite link to query joiners for
//   - limit: maximum number of joiners to return (defaults to 50 if <= 0)
//
// Returns a slice of ChatInviteLinkJoiner objects or an error if the peer
// cannot be resolved or the RPC call fails.
func (c *Client) GetChatInviteLinkJoiners(ctx context.Context, chatID int64, link string, limit int) ([]*types.ChatInviteLinkJoiner, error) {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	if limit <= 0 {
		limit = 50
	}
	rpc := c.Raw()
	result, err := rpc.MessagesGetChatInviteImporters(ctx, &tg.MessagesGetChatInviteImportersRequest{
		Peer:  peer,
		Link:  link,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	joiners := make([]*types.ChatInviteLinkJoiner, 0, len(result.Importers))
	for _, imp := range result.Importers {
		joiners = append(joiners, types.ParseChatInviteImporter(imp))
	}
	return joiners, nil
}

// GetChatAdminInviteLinks retrieves invite links created by a specific admin in
// the specified chat.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat or channel
//   - adminID: user ID of the admin whose links to retrieve
//   - limit: maximum number of links to return (defaults to 50 if <= 0)
//
// Returns a slice of ChatInviteLink objects or an error if the peer or admin
// cannot be resolved or the RPC call fails.
func (c *Client) GetChatAdminInviteLinks(ctx context.Context, chatID int64, adminID int64, limit int) ([]*types.ChatInviteLink, error) {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	if limit <= 0 {
		limit = 50
	}
	user, err := resolveUserID(c, adminID)
	if err != nil {
		return nil, fmt.Errorf("resolve admin: %w", err)
	}
	rpc := c.Raw()
	result, err := rpc.MessagesGetExportedChatInvites(ctx, &tg.MessagesGetExportedChatInvitesRequest{
		Peer:    peer,
		AdminID: user,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}
	links := make([]*types.ChatInviteLink, 0)
	users := make(map[int64]tg.UserClass)
	for _, u := range result.Users {
		if v, ok := u.(*tg.User); ok {
			users[v.ID] = u
		}
	}
	for _, inv := range result.Invites {
		if exported, ok := inv.(*tg.ChatInviteExported); ok {
			links = append(links, types.ParseChatInviteLink(exported, users))
		}
	}
	return links, nil
}

// DeleteChatInviteLink permanently deletes an invite link.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat or channel
//   - link: the invite link string to delete
//
// Returns an error if the peer cannot be resolved or the deletion fails.
func (c *Client) DeleteChatInviteLink(ctx context.Context, chatID int64, link string) error {
	c.Log.Debugf("DeleteChatInviteLink chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	rpc := c.Raw()
	_, err = rpc.MessagesDeleteExportedChatInvite(ctx, &tg.MessagesDeleteExportedChatInviteRequest{
		Peer: peer,
		Link: link,
	})
	return err
}

// ApproveChatJoinRequest approves a pending join request from a user for a chat
// that requires admin approval.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat
//   - userID: identifier of the user whose request to approve
//
// Returns an error if the peer or user cannot be resolved or the approval fails.
func (c *Client) ApproveChatJoinRequest(ctx context.Context, chatID int64, userID int64) error {
	c.Log.Debugf("ApproveChatJoinRequest chat_id=%d user_id=%d", chatID, userID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}
	rpc := c.Raw()
	_, err = rpc.MessagesHideChatJoinRequest(ctx, &tg.MessagesHideChatJoinRequestRequest{
		Approved: true,
		Peer:     peer,
		UserID:   user,
	})
	return err
}

// DeclineChatJoinRequest declines (rejects) a pending join request from a user.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: identifier of the chat
//   - userID: identifier of the user whose request to decline
//
// Returns an error if the peer or user cannot be resolved or the rejection fails.
func (c *Client) DeclineChatJoinRequest(ctx context.Context, chatID int64, userID int64) error {
	c.Log.Debugf("DeclineChatJoinRequest chat_id=%d user_id=%d", chatID, userID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}
	rpc := c.Raw()
	_, err = rpc.MessagesHideChatJoinRequest(ctx, &tg.MessagesHideChatJoinRequestRequest{
		Approved: false,
		Peer:     peer,
		UserID:   user,
	})
	return err
}
