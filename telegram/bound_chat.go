package telegram

import (
	"context"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// BoundArchive moves the specified chat to the archived chats folder. This is a
// bound-method convenience wrapper around [Client.ArchiveChat].
//
// Parameters:
//   - chatID: identifier of the chat to archive
//
// Returns an error if the peer cannot be resolved or the archive operation fails.
func (c *Client) BoundArchive(chatID int64) error {
	return c.ArchiveChat(context.Background(), chatID)
}

// BoundUnarchive moves the specified chat back to the main chat list from the
// archived folder. This is a bound-method convenience wrapper around
// [Client.UnarchiveChat].
//
// Parameters:
//   - chatID: identifier of the chat to unarchive
//
// Returns an error if the peer cannot be resolved or the unarchive operation fails.
func (c *Client) BoundUnarchive(chatID int64) error {
	return c.UnarchiveChat(context.Background(), chatID)
}

// BoundSetTitle changes the title of a group or channel. The caller must have
// appropriate admin permissions. This is a bound-method convenience wrapper
// around [Client.SetChatTitle].
//
// Parameters:
//   - chatID: identifier of the chat whose title to change
//   - title: new title string
//
// Returns an error if the peer cannot be resolved or the title update fails.
func (c *Client) BoundSetTitle(chatID int64, title string) error {
	return c.SetChatTitle(context.Background(), chatID, title)
}

// BoundSetDescription changes the description (bio or about text) of a group or
// channel. This is a bound-method convenience wrapper around
// [Client.SetChatDescription].
//
// Parameters:
//   - chatID: identifier of the chat whose description to change
//   - description: new description text
//
// Returns an error if the peer cannot be resolved or the update fails.
func (c *Client) BoundSetDescription(chatID int64, description string) error {
	return c.SetChatDescription(context.Background(), chatID, description)
}

// BoundSetPhoto changes the profile photo of a group or channel. This is a
// bound-method convenience wrapper around [Client.SetChatPhoto].
//
// Parameters:
//   - chatID: identifier of the chat whose photo to change
//   - photo: input photo object (uploaded image or cropped area)
//
// Returns an error if the peer cannot be resolved or the photo update fails.
func (c *Client) BoundSetPhoto(chatID int64, photo tg.InputChatPhotoClass) error {
	return c.SetChatPhoto(context.Background(), chatID, photo)
}

// BoundDeletePhoto removes the profile photo of a group or channel, reverting
// to the default. This is a bound-method convenience wrapper around
// [Client.DeleteChatPhoto].
//
// Parameters:
//   - chatID: identifier of the chat whose photo to delete
//
// Returns an error if the peer cannot be resolved or the deletion fails.
func (c *Client) BoundDeletePhoto(chatID int64) error {
	return c.DeleteChatPhoto(context.Background(), chatID)
}

// BoundSetUsername changes the public username of a group, channel, or the
// current user. This is a bound-method convenience wrapper around
// [Client.SetChatUsername].
//
// Parameters:
//   - chatID: identifier of the chat whose username to change
//   - username: new public username (without @)
//
// Returns an error if the peer cannot be resolved or the username update fails.
func (c *Client) BoundSetUsername(chatID int64, username string) error {
	return c.SetChatUsername(context.Background(), chatID, username)
}

// BoundBanMember bans a user from a group or channel. The banned user cannot
// rejoin unless unbanned. This is a bound-method convenience wrapper around
// [Client.BanChatMember].
//
// Parameters:
//   - chatID: identifier of the group or channel
//   - userID: identifier of the user to ban
//
// Returns an error if the peer cannot be resolved or the ban fails.
func (c *Client) BoundBanMember(chatID int64, userID int64) error {
	return c.BanChatMember(context.Background(), chatID, userID)
}

// BoundUnbanMember removes a prior ban on a user, allowing them to rejoin the
// group or channel. This is a bound-method convenience wrapper around
// [Client.UnbanChatMember].
//
// Parameters:
//   - chatID: identifier of the group or channel
//   - userID: identifier of the user to unban
//
// Returns an error if the peer cannot be resolved or the unban fails.
func (c *Client) BoundUnbanMember(chatID int64, userID int64) error {
	return c.UnbanChatMember(context.Background(), chatID, userID)
}

// BoundRestrictMember applies restrictions to a user in a group, limiting what
// they can do (e.g. send messages, send media). This is a bound-method
// convenience wrapper around [Client.RestrictChatMember].
//
// Parameters:
//   - chatID: identifier of the group
//   - userID: identifier of the user to restrict
//   - bannedRights: set of actions to forbid
//
// Returns an error if the peer cannot be resolved or the restriction fails.
func (c *Client) BoundRestrictMember(chatID int64, userID int64, bannedRights *tg.ChatBannedRights) error {
	return c.RestrictChatMember(context.Background(), chatID, userID, bannedRights)
}

// BoundPromoteMember grants admin privileges to a user in a group or channel.
// This is a bound-method convenience wrapper around [Client.PromoteChatMember].
//
// Parameters:
//   - chatID: identifier of the group or channel
//   - userID: identifier of the user to promote
//   - adminRights: set of admin rights to grant
//
// Returns an error if the peer cannot be resolved or the promotion fails.
func (c *Client) BoundPromoteMember(chatID int64, userID int64, adminRights *tg.ChatAdminRights) error {
	return c.PromoteChatMember(context.Background(), chatID, userID, adminRights)
}

// BoundJoinChat joins a public chat or channel by its username. This is a
// bound-method convenience wrapper around [Client.JoinChat].
//
// Parameters:
//   - chatID: unused (reserved for future invite-hash based joins)
//   - username: public username of the chat or channel to join (without @)
//
// Returns the joined Chat object or an error if the username is empty or the
// join fails.
func (c *Client) BoundJoinChat(chatID int64, username string) (*types.Chat, error) {
	if username != "" {
		return c.JoinChat(context.Background(), username)
	}
	return nil, ErrJoinRequiresInvite
}

// BoundLeaveChat leaves the specified group or channel. This is a bound-method
// convenience wrapper around [Client.LeaveChat].
//
// Parameters:
//   - chatID: identifier of the chat to leave
//
// Returns an error if the peer cannot be resolved or the leave operation fails.
func (c *Client) BoundLeaveChat(chatID int64) error {
	return c.LeaveChat(context.Background(), chatID)
}

// BoundExportInviteLink creates a new invite link for the specified chat and
// returns it as a string. This is a bound-method convenience wrapper around
// [Client.ExportChatInviteLink].
//
// Parameters:
//   - chatID: identifier of the chat for which to generate an invite link
//
// Returns the invite link string or an error if generation fails.
func (c *Client) BoundExportInviteLink(chatID int64) (string, error) {
	return c.ExportChatInviteLink(context.Background(), chatID)
}

// BoundGetMember retrieves information about a single member of the specified
// chat. This is a bound-method convenience wrapper around
// [Client.GetChatMember].
//
// Parameters:
//   - chatID: identifier of the chat
//   - userID: identifier of the member to look up
//
// Returns the ChatMember information or an error if the lookup fails.
func (c *Client) BoundGetMember(chatID int64, userID int64) (*types.ChatMember, error) {
	return c.GetChatMember(context.Background(), chatID, userID)
}

// BoundGetMembers retrieves a paginated list of members in the specified chat.
// This is a bound-method convenience wrapper around [Client.GetChatMembers].
//
// Parameters:
//   - chatID: identifier of the chat
//   - limit: maximum number of members to return
//   - offset: number of members to skip (for pagination)
//
// Returns a slice of ChatMember objects or an error if the retrieval fails.
func (c *Client) BoundGetMembers(chatID int64, limit int, offset int) ([]*types.ChatMember, error) {
	return c.GetChatMembers(context.Background(), chatID, limit, offset)
}

// BoundAddMembers adds a user to the specified group or channel. The caller
// must have appropriate permissions. This is a bound-method convenience wrapper
// around [Client.AddChatMember].
//
// Parameters:
//   - chatID: identifier of the target chat
//   - userID: identifier of the user to add
//
// Returns an error if the peer cannot be resolved or the add operation fails.
func (c *Client) BoundAddMembers(chatID int64, userID int64) error {
	return c.AddChatMember(context.Background(), chatID, userID)
}

// BoundMarkUnread toggles the unread status of a chat in the chat list. When
// unread is true the chat is marked as unread even if there are no new
// messages. This is a bound-method convenience wrapper around
// [Client.MarkChatUnread].
//
// Parameters:
//   - chatID: identifier of the chat to mark
//   - unread: whether to mark as unread (true) or read (false)
//
// Returns an error if the operation fails.
func (c *Client) BoundMarkUnread(chatID int64, unread bool) error {
	return c.MarkChatUnread(context.Background(), chatID, unread)
}

// BoundSetProtectedContent enables or disables content protection (no-forward)
// for the specified chat. When enabled, users cannot forward messages from the
// chat. This is a bound-method convenience wrapper around
// [Client.SetProtectedContent].
//
// Parameters:
//   - chatID: identifier of the chat
//   - enabled: true to enable content protection, false to disable
//
// Returns an error if the operation fails.
func (c *Client) BoundSetProtectedContent(chatID int64, enabled bool) error {
	return c.SetProtectedContent(context.Background(), chatID, enabled)
}

// BoundSetTTL sets the Time-To-Live (auto-delete) period for messages in the
// specified chat. After the TTL expires, messages are automatically deleted.
// This is a bound-method convenience wrapper around [Client.SetChatTTL].
//
// Parameters:
//   - chatID: identifier of the chat
//   - ttl: auto-delete period in seconds (0 to disable)
//
// Returns an error if the operation fails.
func (c *Client) BoundSetTTL(chatID int64, ttl int) error {
	return c.SetChatTTL(context.Background(), chatID, ttl)
}

// BoundSetPermissions sets the default permissions for all members in a group.
// This is a bound-method convenience wrapper around [Client.SetChatPermissions].
//
// Parameters:
//   - chatID: identifier of the group
//   - permissions: set of banned rights defining what members cannot do
//
// Returns an error if the operation fails.
func (c *Client) BoundSetPermissions(chatID int64, permissions *tg.ChatBannedRights) error {
	return c.SetChatPermissions(context.Background(), chatID, permissions)
}

// BoundSetAdminTitle sets a custom title for an administrator in a supergroup
// or channel. This is a bound-method convenience wrapper around
// [Client.SetAdministratorTitle].
//
// Parameters:
//   - chatID: identifier of the supergroup or channel
//   - userID: identifier of the admin whose title to set
//   - title: custom title string (e.g. "Co-Founder")
//
// Returns an error if the operation fails.
func (c *Client) BoundSetAdminTitle(chatID int64, userID int64, title string) error {
	return c.SetAdministratorTitle(context.Background(), chatID, userID, title)
}

// BoundSetSlowMode sets the slow mode interval for a group. Members must wait
// the specified number of seconds between messages. This is a bound-method
// convenience wrapper around [Client.SetSlowMode].
//
// Parameters:
//   - chatID: identifier of the group
//   - seconds: slow mode interval in seconds (0 to disable)
//
// Returns an error if the operation fails.
func (c *Client) BoundSetSlowMode(chatID int64, seconds int) error {
	return c.SetSlowMode(context.Background(), chatID, seconds)
}

// BoundMute mutes notifications for the specified chat. This is a bound-method
// convenience wrapper around [Client.MuteChat].
//
// Parameters:
//   - chatID: identifier of the chat to mute
//
// Returns an error if the operation fails.
func (c *Client) BoundMute(chatID int64) error {
	return c.MuteChat(context.Background(), chatID)
}

// BoundUnmute unmutes notifications for the specified chat. This is a
// bound-method convenience wrapper around [Client.UnmuteChat].
//
// Parameters:
//   - chatID: identifier of the chat to unmute
//
// Returns an error if the operation fails.
func (c *Client) BoundUnmute(chatID int64) error {
	return c.UnmuteChat(context.Background(), chatID)
}

// BoundUnpinAll removes all pinned messages from the specified chat. This is a
// bound-method convenience wrapper around [Client.UnpinAllMessages].
//
// Parameters:
//   - chatID: identifier of the chat
//
// Returns the number of messages unpinned or an error if the operation fails.
func (c *Client) BoundUnpinAll(chatID int64) (int, error) {
	return c.UnpinAllMessages(context.Background(), chatID)
}

// BoundGetChat retrieves full information about the specified chat. This is a
// bound-method convenience wrapper around [Client.GetChat].
//
// Parameters:
//   - chatID: identifier of the chat to retrieve
//
// Returns the Chat object or an error if retrieval fails.
func (c *Client) BoundGetChat(chatID int64) (*types.Chat, error) {
	return c.GetChat(context.Background(), chatID)
}

// BoundGetEventLog retrieves the event log (admin log) for the specified chat.
// This is a bound-method convenience wrapper around [Client.GetChatEventLog].
//
// Parameters:
//   - chatID: identifier of the supergroup or channel
//   - query: search string to filter events (empty for all events)
//   - limit: maximum number of events to return
//
// Returns a slice of ChatEvent objects or an error if retrieval fails.
func (c *Client) BoundGetEventLog(chatID int64, query string, limit int) ([]*types.ChatEvent, error) {
	return c.GetChatEventLog(context.Background(), chatID, query, limit)
}
