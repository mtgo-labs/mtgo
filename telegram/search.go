package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// SearchResult holds the users and chats returned by a contact or peer search.
//
// Example:
//
//	result, err := client.SearchContacts(ctx, "john", 5)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, u := range result.Users {
//	    fmt.Printf("- %s (@%s)\n", u.FirstName, u.Username)
//	}
type SearchResult struct {
	// Users contains the user objects matching the search query.
	Users []*types.User
	// Chats contains the chat and channel objects matching the search query.
	Chats []*types.Chat
}

// SearchContacts searches the user's Telegram contacts and global user database
// for the given query string. It returns matching users and chats.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - query: search string (name, username, or phone number fragment)
//   - limit: maximum number of results to return (defaults to 10 if <= 0)
//
// Returns a SearchResult containing matched users and chats, or an error if
// the client is not connected or the search RPC fails.
//
// Example:
//
//	ctx := context.Background()
//	result, err := client.SearchContacts(ctx, "alice", 10)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Found %d users, %d chats\n", len(result.Users), len(result.Chats))
func (c *Client) SearchContacts(ctx context.Context, query string, limit int) (*SearchResult, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.Log.Debug("SearchContacts")

	if limit <= 0 {
		limit = 10
	}

	rpc := c.Raw()
	found, err := rpc.ContactsSearch(ctx, &tg.ContactsSearchRequest{
		Q:     query,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}

	result := &SearchResult{
		Users: make([]*types.User, 0, len(found.Users)),
		Chats: make([]*types.Chat, 0, len(found.Chats)),
	}

	for _, u := range found.Users {
		result.Users = append(result.Users, types.ParseUser(u))
	}

	for _, ch := range found.Chats {
		if chat := types.ParseChatFromChat(ch); chat != nil {
			result.Chats = append(result.Chats, chat)
		}
	}

	return result, nil
}

// GetUser retrieves full information about a single user by their ID.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - userID: identifier of the user to retrieve
//
// Returns the User object with full profile information, or an error if the
// client is not connected, the user cannot be resolved, or the RPC call fails.
//
// Example:
//
//	ctx := context.Background()
//	user, err := client.GetUser(ctx, 12345678)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("User: %s %s\n", user.FirstName, user.LastName)
func (c *Client) GetUser(ctx context.Context, userID int64) (*types.User, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	c.Log.Debugf("GetUser user_id=%d", userID)

	user, err := resolveUserID(c, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.UsersGetFullUser(ctx, &tg.UsersGetFullUserRequest{ID: user})
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	switch v := result.(type) {
	case *tg.UsersUserFull:
		if len(v.Users) > 0 {
			return types.ParseUser(v.Users[0]), nil
		}
		return nil, fmt.Errorf("no user in response")
	default:
		return nil, fmt.Errorf("unexpected result type %T", result)
	}
}
