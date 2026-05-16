package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// GetUsers retrieves full user information for each of the provided user IDs.
// Users that cannot be resolved are silently skipped.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - userIDs: slice of user identifiers (UserRef) to look up
//
// Returns a slice of types.User for successfully resolved users. Returns an
// error only if a user ID cannot be resolved at the input stage; individual
// fetch failures are skipped.
//
// Example:
//
//	users, err := client.GetUsers(ctx, []int64{123456, 789012})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, u := range users {
//	    fmt.Printf("%s %s (ID: %d)\n", u.FirstName, u.LastName, u.ID)
//	}
func (c *Client) GetUsers(ctx context.Context, userIDs []int64) ([]*types.User, error) {
	c.Log.Debugf("GetUsers count=%d", len(userIDs))
	inputs := make([]tg.InputUserClass, len(userIDs))
	for i, id := range userIDs {
		u, err := resolveUserID(c, id)
		if err != nil {
			return nil, fmt.Errorf("resolve user %v: %w", id, err)
		}
		inputs[i] = u
	}

	users := make([]*types.User, 0, len(inputs))
	for _, u := range inputs {
		result, err := c.Raw().UsersGetFullUser(ctx, &tg.UsersGetFullUserRequest{ID: u})
		if err != nil {
			continue
		}
		var userClass tg.UserClass
		switch v := result.(type) {
		case *tg.UsersUserFull:
			if len(v.Users) > 0 {
				userClass = v.Users[0]
			}
		default:
			continue
		}
		if userClass != nil {
			users = append(users, types.ParseUser(userClass))
		}
	}
	return users, nil
}

// GetMe retrieves the currently authenticated user's full information.
// The result is also cached on the client via SetMe.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//
// Returns the authenticated types.User, or an error if the client is not
// connected or the server response is unexpected.
func (c *Client) GetMe(ctx context.Context) (*types.User, error) {
	if !c.IsConnected() {
		return nil, ErrNotConnected
	}

	c.Log.Debug("GetMe")
	c.mu.RLock()
	me := c.me
	c.mu.RUnlock()
	if me != nil {
		return me, nil
	}

	result, err := c.Raw().UsersGetFullUser(ctx, &tg.UsersGetFullUserRequest{
		ID: &tg.InputUserSelf{},
	})
	if err != nil {
		c.Log.Warnf("GetMe failed err=%v", err)
		return nil, err
	}

	uf, ok := result.(*tg.UsersUserFull)
	if !ok {
		return nil, fmt.Errorf("GetMe: unexpected type %T", result)
	}
	if len(uf.Users) > 0 {
		user := types.ParseUser(uf.Users[0])
		c.SetMe(user)
		c.saveMeToStorage(user)
		return user, nil
	}
	if uf.FullUser != nil {
		if full, ok := uf.FullUser.(*tg.UserFull); ok {
			user := &types.User{
				ID:    full.ID,
				IsBot: true,
			}
			c.SetMe(user)
			return user, nil
		}
	}
	return nil, ErrNoUserInResponse
}

// GetCommonChats retrieves the list of chats (groups, channels, supergroups)
// that the current user shares with the specified user.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - userID: identifier of the target user (UserRef)
//   - limit: maximum number of common chats to return (defaults to 100 if <= 0)
//
// Returns a slice of types.Chat entries representing the shared chats, or an
// error if the user cannot be resolved or the RPC call fails.
//
// Example:
//
//	chats, err := client.GetCommonChats(ctx, userID, 20)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, ch := range chats {
//	    fmt.Println(ch.Title)
//	}
func (c *Client) GetCommonChats(ctx context.Context, userID int64, limit int) ([]*types.Chat, error) {
	c.Log.Debugf("GetCommonChats user_id=%d", userID)
	user, err := resolveUserID(c, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}
	if limit <= 0 {
		limit = 100
	}
	rpc := c.Raw()
	result, err := rpc.MessagesGetCommonChats(ctx, &tg.MessagesGetCommonChatsRequest{
		UserID: user,
		MaxID:  0,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	chats := make([]*types.Chat, 0)
	switch v := result.(type) {
	case *tg.MessagesChats:
		for _, ch := range v.Chats {
			if parsed := types.ParseChatFromChat(ch); parsed != nil {
				chats = append(chats, parsed)
			}
		}
	}
	return chats, nil
}

// UpdateProfile updates the current user's first name, last name, and bio.
// Only non-empty fields are included in the update request; pass an empty
// string for any field that should remain unchanged.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - firstName: the new first name (empty to skip)
//   - lastName: the new last name (empty to skip)
//   - bio: the new bio/about text (empty to skip)
//
// Returns the updated types.User on success, or an error if the RPC call fails.
//
// Example:
//
//	updated, err := client.UpdateProfile(ctx, "Alice", "", "Full-stack developer")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Updated profile: %s\n", updated.FirstName)
func (c *Client) UpdateProfile(ctx context.Context, firstName, lastName, bio string) (*types.User, error) {
	c.Log.Debug("UpdateProfile")
	var fn, ln, b string
	if firstName != "" {
		fn = firstName
	}
	if lastName != "" {
		ln = lastName
	}
	if bio != "" {
		b = bio
	}
	rpc := c.Raw()
	result, err := rpc.AccountUpdateProfile(ctx, &tg.AccountUpdateProfileRequest{
		FirstName: fn,
		LastName:  ln,
		About:     b,
	})
	if err != nil {
		return nil, err
	}
	return types.ParseUser(result), nil
}

func (c *Client) UpdateStatus(ctx context.Context, offline bool) error {
	c.Log.Debugf("UpdateStatus offline=%v", offline)
	rpc := c.Raw()
	_, err := rpc.AccountUpdateStatus(ctx, &tg.AccountUpdateStatusRequest{
		Offline: offline,
	})
	return err
}
