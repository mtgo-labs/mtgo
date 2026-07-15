package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// AddContact adds a user to the contact list using the provided first name, last name, and
// phone number. If share is true, the user's phone number is shared with the contact.
// Returns an error if the user cannot be resolved or the RPC call fails.
//
// Example:
//
//	err := client.AddContact(ctx, userID, "Alice", "Smith", "+1234567890", true)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Contact added")
func (c *Client) AddContact(ctx context.Context, userID int64, firstName, lastName, phone string, share bool) error {
	c.Log.Debugf("AddContact user_id=%d", userID)
	_, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.ContactsImportContacts(ctx, &tg.ContactsImportContactsRequest{
		Contacts: []*tg.InputPhoneContact{
			{
				FirstName: firstName,
				LastName:  lastName,
				Phone:     phone,
			},
		},
	})
	return err
}

// DeleteContacts removes the specified users from the contact list. Each element of userIDs
// is resolved to an input user before deletion. Returns an error if any user cannot be
// resolved or the RPC call fails.
//
// Example:
//
//	err := client.DeleteContacts(ctx, []int64{123456, 789012})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Contacts deleted")
func (c *Client) DeleteContacts(ctx context.Context, userIDs []int64) error {
	c.Log.Debugf("DeleteContacts count=%d", len(userIDs))
	inputs := make([]tg.InputUserClass, len(userIDs))
	for i, id := range userIDs {
		u, err := resolveUserID(c, id)
		if err != nil {
			return fmt.Errorf("resolve user %v: %w", id, err)
		}
		inputs[i] = u
	}

	rpc := c.Raw()
	_, err := rpc.ContactsDeleteContacts(ctx, &tg.ContactsDeleteContactsRequest{ID: inputs})
	return err
}

// GetContacts retrieves the current user's contact list. The hash parameter is a
// checksum of the previously known contact list; if it matches the server-side hash
// an empty response may be returned. Returns the full contacts result on success.
//
// Example:
//
//	contacts, err := client.GetContacts(ctx, 0)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	_ = contacts
func (c *Client) GetContacts(ctx context.Context, hash int64) (tg.ContactsClass, error) {
	rpc := c.Raw()
	return rpc.ContactsGetContacts(ctx, &tg.ContactsGetContactsRequest{Hash: hash})
}

// BlockUser blocks the specified user from interacting with the current account.
// Returns an error if the peer cannot be resolved or the RPC call fails.
//
// Example:
//
//	err := client.BlockUser(ctx, spammerID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("User blocked")
func (c *Client) BlockUser(ctx context.Context, userID int64) error {
	c.Log.Debugf("BlockUser user_id=%d", userID)
	peer, err := resolvePeer(c, userID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.ContactsBlock(ctx, &tg.ContactsBlockRequest{ID: peer})
	return err
}

// UnblockUser removes the specified user from the blocked list, allowing them to
// interact with the current account again. Returns an error if the peer cannot be
// resolved or the RPC call fails.
func (c *Client) UnblockUser(ctx context.Context, userID int64) error {
	c.Log.Debugf("UnblockUser user_id=%d", userID)
	peer, err := resolvePeer(c, userID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.ContactsUnblock(ctx, &tg.ContactsUnblockRequest{ID: peer})
	return err
}

// GetBlocked retrieves a paginated list of blocked users. The offset parameter
// specifies the starting position, and limit controls the maximum number of entries
// returned. Returns the blocked peers result on success.
func (c *Client) GetBlocked(ctx context.Context, limit, offset int) (tg.BlockedClass, error) {
	rpc := c.Raw()
	return rpc.ContactsGetBlocked(ctx, &tg.ContactsGetBlockedRequest{
		Offset: int32(offset),
		Limit:  int32(limit),
	})
}
