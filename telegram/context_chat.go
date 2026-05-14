package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// GetChat retrieves full information about the chat associated with the current context.
// The chat is resolved from the update that triggered the handler (message, callback, etc.).
//
// Returns:
//   - *types.Chat: the chat information
//   - error: non-nil if the chat ID cannot be resolved or the request fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    chat, _ := ctx.GetChat()
//	    log.Printf("Chat title: %s", chat.Title)
//	})
func (c *Context) GetChat() (*types.Chat, error) {
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	return c.Client.GetChat(c.Ctx, chatID)
}

// LeaveChat removes the current user or bot from the chat associated with the context.
// The bot will no longer receive updates from this chat.
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the leave operation fails
func (c *Context) LeaveChat() error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.LeaveChat(c.Ctx, chatID)
}

// Ban removes and permanently bans a user from the chat. The banned user cannot rejoin
// unless unbanned. Requires appropriate admin rights in the chat.
//
// Parameters:
//   - userID: the Telegram user ID to ban
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the ban fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    if ctx.Message != nil {
//	        ctx.Ban(spammerID)
//	    }
//	})
func (c *Context) Ban(userID int64) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.BanChatMember(c.Ctx, chatID, userID)
}

// Unban removes a previously applied ban on a user, allowing them to rejoin the chat.
// Does nothing if the user was not banned.
//
// Parameters:
//   - userID: the Telegram user ID to unban
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the unban fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Unban(previouslyBannedUserID)
//	})
func (c *Context) Unban(userID int64) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.UnbanChatMember(c.Ctx, chatID, userID)
}

// Restrict applies specific restrictions to a user in the chat, such as forbidding them
// from sending messages or media. Requires appropriate admin rights.
//
// Parameters:
//   - userID: the Telegram user ID to restrict
//   - rights: the set of actions the user is banned from performing
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the restriction fails
func (c *Context) Restrict(userID int64, rights *tg.ChatBannedRights) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.RestrictChatMember(c.Ctx, chatID, userID, rights)
}

// Promote grants admin privileges to a user in the chat. The rights parameter controls
// which specific admin actions the user is allowed to perform.
//
// Parameters:
//   - userID: the Telegram user ID to promote
//   - rights: the admin rights to grant
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the promotion fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    rights := &tg.ChatAdminRights{DeleteMessages: true, BanUsers: true}
//	    ctx.Promote(targetUserID, rights)
//	})
func (c *Context) Promote(userID int64, rights *tg.ChatAdminRights) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.PromoteChatMember(c.Ctx, chatID, userID, rights)
}

// SetAdministratorTitle sets the custom title displayed for an administrator in the chat.
// This is visible in the member list and only applies to admins who are not the chat owner.
//
// Parameters:
//   - userID: the Telegram user ID of the administrator
//   - title: the custom title to display (e.g. "Moderator", "Support")
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the user is not an admin, or the request fails
func (c *Context) SetAdministratorTitle(userID int64, title string) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SetAdministratorTitle(c.Ctx, chatID, userID, title)
}

// GetMember retrieves information about a specific chat member, including their role,
// permissions, and join date.
//
// Parameters:
//   - userID: the Telegram user ID to look up
//
// Returns:
//   - *types.ChatMember: the member information
//   - error: non-nil if the chat ID cannot be resolved or the member cannot be found
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    member, _ := ctx.GetMember(ctx.Message.FromID)
//	    log.Printf("Member role: %s", member.Status)
//	})
func (c *Context) GetMember(userID int64) (*types.ChatMember, error) {
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	return c.Client.GetChatMember(c.Ctx, chatID, userID)
}

// GetMembers retrieves a paginated list of chat members. Use limit and offset to control
// pagination; typical limit values range from 0 to 200.
//
// Parameters:
//   - limit: maximum number of members to return
//   - offset: number of members to skip before returning results
//
// Returns:
//   - []*types.ChatMember: slice of chat member information
//   - error: non-nil if the chat ID cannot be resolved or the request fails
func (c *Context) GetMembers(limit, offset int) ([]*types.ChatMember, error) {
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	return c.Client.GetChatMembers(c.Ctx, chatID, limit, offset)
}

// SetTitle changes the title of the chat. Only works for groups, supergroups, and channels.
//
// Parameters:
//   - title: the new chat title
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the update fails
func (c *Context) SetTitle(title string) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SetChatTitle(c.Ctx, chatID, title)
}

// SetDescription updates the description (about text) of the chat.
//
// Parameters:
//   - about: the new description text
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the update fails
func (c *Context) SetDescription(about string) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SetChatDescription(c.Ctx, chatID, about)
}

// SetPhoto updates the chat photo. Only works for groups, supergroups, and channels
// where the caller has the appropriate admin rights.
//
// Parameters:
//   - photo: the new chat photo as an [tg.InputChatPhotoClass]
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the upload fails
func (c *Context) SetPhoto(photo tg.InputChatPhotoClass) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SetChatPhoto(c.Ctx, chatID, photo)
}

// DeleteChatPhoto removes the chat photo. Only applicable to groups, supergroups, and
// channels where the caller has the appropriate admin rights.
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the deletion fails
func (c *Context) DeleteChatPhoto() error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.DeleteChatPhoto(c.Ctx, chatID)
}

// SetTTL sets the Time-To-Live (auto-delete) period for messages in the chat. A value of
// 0 disables auto-deletion.
//
// Parameters:
//   - ttl: time in seconds after which messages are automatically deleted
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the TTL update fails
func (c *Context) SetTTL(ttl int) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SetChatTTL(c.Ctx, chatID, ttl)
}

// SetPermissions updates the default chat permissions for all members. Only affects
// non-admin members and only works in groups and supergroups.
//
// Parameters:
//   - permissions: the set of actions that are banned by default for all members
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the update fails
func (c *Context) SetPermissions(permissions *tg.ChatBannedRights) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SetChatPermissions(c.Ctx, chatID, permissions)
}

// ExportInviteLink generates and returns a new primary invite link for the chat. Any
// previously generated primary link is revoked.
//
// Returns:
//   - string: the new invite link
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the export fails
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    link, _ := ctx.ExportInviteLink()
//	    ctx.Reply("Join here: " + link)
//	})
func (c *Context) ExportInviteLink() (string, error) {
	chatID, err := c.chatID()
	if err != nil {
		return "", err
	}
	return c.Client.ExportChatInviteLink(c.Ctx, chatID)
}

// Archive moves the chat to the archived chats folder in the user's chat list.
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the archive operation fails
func (c *Context) Archive() error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.ArchiveChat(c.Ctx, chatID)
}

// Unarchive moves the chat out of the archived chats folder back to the main chat list.
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the unarchive operation fails
func (c *Context) Unarchive() error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.UnarchiveChat(c.Ctx, chatID)
}

// MarkUnread toggles the unread status of the chat in the chat list without reading
// the actual messages.
//
// Parameters:
//   - unread: if true, marks the chat as unread; if false, clears the unread mark
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the mark operation fails
func (c *Context) MarkUnread(unread bool) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.MarkChatUnread(c.Ctx, chatID, unread)
}

// SetProtectedContent enables or disables content protection on the chat. When enabled,
// users cannot forward messages or save media from the chat.
//
// Parameters:
//   - enabled: true to enable content protection, false to disable
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the update fails
func (c *Context) SetProtectedContent(enabled bool) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SetProtectedContent(c.Ctx, chatID, enabled)
}

// UnpinAllMessages removes all pinned messages from the chat.
//
// Returns:
//   - int: the number of messages that were unpinned
//   - error: non-nil if the chat ID cannot be resolved or the unpin operation fails
func (c *Context) UnpinAllMessages() (int, error) {
	chatID, err := c.chatID()
	if err != nil {
		return 0, err
	}
	return c.Client.UnpinAllMessages(c.Ctx, chatID)
}

// Mute disables all notifications for the chat. The chat will not produce sound or
// visual alerts for new messages.
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the mute operation fails
func (c *Context) Mute() error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.MuteChat(c.Ctx, chatID)
}

// Unmute re-enables notifications for the chat that were previously disabled by [Context.Mute].
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved or the unmute operation fails
func (c *Context) Unmute() error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.UnmuteChat(c.Ctx, chatID)
}

// AddMembers adds one or more users to the chat. Each user is added individually; if any
// add fails, the method returns immediately with the error.
//
// Parameters:
//   - userIDs: slice of Telegram user IDs to add to the chat
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, any user cannot be added, or privacy settings block the add
func (c *Context) AddMembers(userIDs []int64) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	for _, uid := range userIDs {
		if err := c.Client.AddChatMember(c.Ctx, chatID, uid); err != nil {
			return err
		}
	}
	return nil
}

// SetSlowMode configures the slow mode delay between consecutive messages in the chat.
// A value of 0 disables slow mode.
//
// Parameters:
//   - seconds: minimum number of seconds a user must wait between messages (0 to disable)
//
// Returns:
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the update fails
func (c *Context) SetSlowMode(seconds int) error {
	chatID, err := c.chatID()
	if err != nil {
		return err
	}
	return c.Client.SetSlowMode(c.Ctx, chatID, seconds)
}

// GetChatEventLog retrieves the audit log of actions taken in the chat. Only available
// for supergroups and channels where the caller has admin rights.
//
// Parameters:
//   - query: search string to filter events by (empty string returns all events)
//   - limit: maximum number of events to return
//
// Returns:
//   - []*types.ChatEvent: slice of chat event entries
//   - error: non-nil if the chat ID cannot be resolved, the caller lacks permissions, or the request fails
func (c *Context) GetChatEventLog(query string, limit int) ([]*types.ChatEvent, error) {
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	return c.Client.GetChatEventLog(c.Ctx, chatID, query, limit)
}

// SearchMessages searches for messages within the chat that match the given query.
//
// Parameters:
//   - query: the search string
//   - opts: optional [SearchMessagesOption] parameters for filtering and pagination
//
// Returns:
//   - []*types.Message: slice of matching messages
//   - error: non-nil if the chat ID cannot be resolved or the search fails
func (c *Context) SearchMessages(query string, opts ...*SearchMessagesOption) ([]*types.Message, error) {
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	return c.Client.SearchMessages(c.Ctx, chatID, query, opts...)
}

// GetHistory retrieves recent messages from the chat history. Use offsetID to paginate
// by starting from messages older than the given ID.
//
// Parameters:
//   - limit: maximum number of messages to return
//   - offsetID: message ID to start from; use 0 to retrieve the most recent messages
//
// Returns:
//   - []*types.Message: slice of messages from the chat history
//   - error: non-nil if the chat ID cannot be resolved or the request fails
func (c *Context) GetHistory(limit int, offsetID int32) ([]*types.Message, error) {
	chatID, err := c.chatID()
	if err != nil {
		return nil, err
	}
	return c.Client.GetChatHistory(c.Ctx, chatID, limit, offsetID)
}

// GetCommonChats retrieves the list of chats that the given user shares with the current
// user or bot.
//
// Parameters:
//   - userID: the Telegram user ID to find common chats with
//   - limit: maximum number of chats to return
//
// Returns:
//   - []*types.Chat: slice of common chats
//   - error: non-nil if the request fails
func (c *Context) GetCommonChats(userID int64, limit int) ([]*types.Chat, error) {
	return c.Client.GetCommonChats(c.Ctx, userID, limit)
}
