package types

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// Chat represents a Telegram chat, which may be a private conversation, bot,
// group, supergroup, or channel. When created by a Client, it carries a
// ChatBinder so that Archive, SetTitle, BanMember, and other convenience methods
// can operate directly against the API.
//
// Example:
//
//	chats, err := client.GetDialogs(ctx, 0, 10)
//	for _, c := range chats {
//	    fmt.Printf("%s (%s) — %d members\n", c.Title, c.Type, c.MembersCount)
//	}
type Chat struct {
	// ID is the unique chat identifier. Negative for groups and channels, positive
	// for users (private chats).
	ID int64
	// Type classifies the chat as private, bot, group, supergroup, channel, or
	// forum.
	Type ChatType
	// IsVerified is true when the chat (channel or bot) is verified by Telegram.
	IsVerified bool
	// IsScam is true when the chat has been flagged as a scam by Telegram.
	IsScam bool
	// IsRestricted is true when the chat is geo-restricted or otherwise restricted
	// by Telegram.
	IsRestricted bool
	// IsCreator is true when the current user is the creator/owner of the chat.
	IsCreator bool
	// IsForum is true when the supergroup has forum (topics) enabled.
	IsForum bool
	// Title is the display title of the chat (empty for private users).
	Title string
	// FirstName is the first name of the user (only set for private chats and
	// bots).
	FirstName string
	// LastName is the last name of the user (only set for private chats and bots).
	LastName string
	// Username is the public username of the chat, without the "@" prefix.
	Username string
	// Photo contains the chat's profile photo variants.
	Photo *ChatPhoto
	// Description is the chat's about text (groups, supergroups, channels).
	Description string
	// Permissions describes the default actions non-admin members can perform.
	Permissions *ChatPermissions
	// AdminRights lists the administrative privileges of the current user in this
	// chat.
	AdminRights *ChatAdminRights
	// BannedRights lists the restrictions applied to the current user in this
	// chat.
	BannedRights *ChatBannedRights
	// MembersCount is the number of members in the chat, or 0 when not available.
	MembersCount int
	// AccessHash is required to access channels or users that the client has not
	// cached locally.
	AccessHash int64
	binder     ChatBinder
}

// ChatPreview contains a minimal subset of chat information used for link
// previews and search results where full chat details are not needed.
type ChatPreview struct {
	// ID is the unique chat identifier.
	ID int64
	// Type classifies the chat.
	Type ChatType
	// Title is the display title of the chat.
	Title string
	// Username is the public username, without the "@" prefix.
	Username string
	// Photo contains the chat's profile photo.
	Photo *ChatPhoto
	// MembersCount is the number of members in the chat.
	MembersCount int
}

// String returns a human-readable identifier for the chat, preferring username,
// then title, then first/last name, and finally a fallback "chat_<id>" format.
func (c *Chat) String() string {
	if c.Username != "" {
		return c.Username
	}
	if c.Title != "" {
		return c.Title
	}
	name := c.FirstName
	if c.LastName != "" {
		name += " " + c.LastName
	}
	if name == "" {
		return fmt.Sprintf("chat_%d", c.ID)
	}
	return name
}

// ParseChatFromUser converts an MTProto User to a Chat.
// Returns nil if raw is nil.
//
// Example:
//
//	chat := types.ParseChatFromUser(userTL)
//	fmt.Println(chat.Username)
func ParseChatFromUser(raw tg.UserClass) *Chat {
	if raw == nil {
		return nil
	}
	switch r := raw.(type) {
	case *tg.UserEmpty:
		return &Chat{ID: r.ID, Type: ChatTypePrivate}
	case *tg.User:
		chatType := ChatTypePrivate
		if r.Bot {
			chatType = ChatTypeBot
		}
		c := &Chat{
			ID:         r.ID,
			Type:       chatType,
			IsVerified: r.Verified,
			IsScam:     r.Scam,
			Photo:      parseUserProfilePhoto(r.Photo),
		}
		if r.FirstName != "" {
			c.FirstName = r.FirstName
		}
		if r.LastName != "" {
			c.LastName = r.LastName
		}
		if r.Username != "" {
			c.Username = r.Username
		}
		if r.AccessHash != 0 {
			c.AccessHash = r.AccessHash
		}
		if r.Deleted {
			c.FirstName = "Deleted Account"
			c.LastName = ""
			c.Username = ""
		}
		return c
	}
	return nil
}

// ParseChatFromChat converts an MTProto Chat or Channel to a Chat.
// Handles basic groups, supergroups, and channels, including forbidden variants.
// Returns nil if raw is nil or represents an empty chat.
//
// Example:
//
//	chat := types.ParseChatFromChat(channelTL)
//	fmt.Printf("Channel: %s (ID: %d)\n", chat.Title, chat.ID)
func ParseChatFromChat(raw tg.ChatClass) *Chat {
	if raw == nil {
		return nil
	}
	switch r := raw.(type) {
	case *tg.ChatEmpty:
		return nil
	case *tg.Chat:
		return &Chat{
			ID:           -r.ID,
			Type:         ChatTypeGroup,
			Title:        r.Title,
			IsCreator:    r.Creator,
			Permissions:  ParseChatPermissions(r.DefaultBannedRights),
			MembersCount: int(r.ParticipantsCount),
			Photo:        parseChatPhoto(r.Photo),
			AdminRights:  ParseChatAdminRights(r.AdminRights),
		}
	case *tg.ChatForbidden:
		return &Chat{
			ID:    -r.ID,
			Type:  ChatTypeGroup,
			Title: r.Title,
		}
	case *tg.Channel:
		chatType := ChatTypeChannel
		if r.Megagroup {
			chatType = ChatTypeSupergroup
		}
		c := &Chat{
			ID:           -r.ID,
			Type:         chatType,
			Title:        r.Title,
			IsVerified:   r.Verified,
			IsRestricted: r.Restricted,
			IsCreator:    r.Creator,
			IsForum:      r.Forum,
			Photo:        parseChatPhoto(r.Photo),
			Permissions:  ParseChatPermissions(r.DefaultBannedRights),
			AdminRights:  ParseChatAdminRights(r.AdminRights),
			BannedRights: ParseChatBannedRights(r.BannedRights),
		}
		if r.Username != "" {
			c.Username = r.Username
		}
		if r.AccessHash != 0 {
			c.AccessHash = r.AccessHash
		}
		if r.ParticipantsCount != 0 {
			c.MembersCount = int(r.ParticipantsCount)
		}
		return c
	case *tg.ChannelForbidden:
		chatType := ChatTypeChannel
		if r.Megagroup {
			chatType = ChatTypeSupergroup
		}
		return &Chat{
			ID:         -r.ID,
			Type:       chatType,
			Title:      r.Title,
			AccessHash: r.AccessHash,
		}
	}
	return nil
}

// ParseChatFromPeer resolves a Peer using the provided PeerMap and converts
// the result to a Chat. Falls back to a minimal Chat with just ID and Type
// when the peer is not found in the map. Returns nil if either peer or pm is nil.
func ParseChatFromPeer(peer tg.PeerClass, pm *PeerMap) *Chat {
	if peer == nil || pm == nil {
		return nil
	}
	switch p := peer.(type) {
	case *tg.PeerUser:
		if u, ok := pm.Users[p.UserID]; ok {
			return ParseChatFromUser(u)
		}
		return &Chat{ID: p.UserID, Type: ChatTypePrivate}
	case *tg.PeerChat:
		if c, ok := pm.Chats[p.ChatID]; ok {
			return ParseChatFromChat(c)
		}
		return &Chat{ID: -p.ChatID, Type: ChatTypeGroup}
	case *tg.PeerChannel:
		if ch, ok := pm.Channels[p.ChannelID]; ok {
			return ParseChatFromChat(ch)
		}
		return &Chat{ID: -p.ChannelID, Type: ChatTypeChannel}
	}
	return nil
}

// Archive moves the chat to the archived folder.
// Returns ErrNoChatBinder if the chat was not created by a client.
//
// Example:
//
//	err := chat.Archive()
func (c *Chat) Archive() error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundArchive(c.ID)
}

// Unarchive moves the chat back to the main chat list.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) Unarchive() error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundUnarchive(c.ID)
}

// SetTitle changes the chat's display title.
// Returns ErrNoChatBinder if the chat was not created by a client.
//
// Example:
//
//	err := chat.SetTitle("New Group Name")
func (c *Chat) SetTitle(title string) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundSetTitle(c.ID, title)
}

// SetDescription updates the chat's about/description text.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) SetDescription(description string) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundSetDescription(c.ID, description)
}

// SetPhoto changes the chat's profile photo.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) SetPhoto(photo tg.InputChatPhotoClass) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundSetPhoto(c.ID, photo)
}

// DeletePhoto removes the chat's profile photo.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) DeletePhoto() error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundDeletePhoto(c.ID)
}

// SetUsername changes the public username of the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) SetUsername(username string) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundSetUsername(c.ID, username)
}

// BanMember bans a user from the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
//
// Example:
//
//	err := chat.BanMember(spammerID)
func (c *Chat) BanMember(userID int64) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundBanMember(c.ID, userID)
}

// UnbanMember removes a ban, allowing the user to rejoin the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) UnbanMember(userID int64) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundUnbanMember(c.ID, userID)
}

// RestrictMember applies the given banned rights to restrict a user's
// capabilities in the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) RestrictMember(userID int64, bannedRights *tg.ChatBannedRights) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundRestrictMember(c.ID, userID, bannedRights)
}

// PromoteMember grants the given admin rights to a user in the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) PromoteMember(userID int64, adminRights *tg.ChatAdminRights) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundPromoteMember(c.ID, userID, adminRights)
}

// Join makes the current user join the chat using its username.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) Join() (*Chat, error) {
	if c.binder == nil {
		return nil, ErrNoChatBinder
	}
	return c.binder.BoundJoinChat(c.ID, c.Username)
}

// Leave removes the current user from the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
//
// Example:
//
//	err := chat.Leave()
func (c *Chat) Leave() error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundLeaveChat(c.ID)
}

// ExportInviteLink generates a new primary invite link for the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) ExportInviteLink() (string, error) {
	if c.binder == nil {
		return "", ErrNoChatBinder
	}
	return c.binder.BoundExportInviteLink(c.ID)
}

// GetMember retrieves a single chat member by their user ID.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) GetMember(userID int64) (*ChatMember, error) {
	if c.binder == nil {
		return nil, ErrNoChatBinder
	}
	return c.binder.BoundGetMember(c.ID, userID)
}

// GetMembers retrieves a page of chat members with the given limit and offset.
// Returns ErrNoChatBinder if the chat was not created by a client.
//
// Example:
//
//	members, err := chat.GetMembers(50, 0)
//	for _, m := range members {
//	    fmt.Printf("%s — %s\n", m.User.String(), m.Status)
//	}
func (c *Chat) GetMembers(limit int, offset int) ([]*ChatMember, error) {
	if c.binder == nil {
		return nil, ErrNoChatBinder
	}
	return c.binder.BoundGetMembers(c.ID, limit, offset)
}

// AddMembers adds a user to the chat by their user ID.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) AddMembers(userID int64) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundAddMembers(c.ID, userID)
}

// MarkUnread toggles the unread marker on the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) MarkUnread(unread bool) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundMarkUnread(c.ID, unread)
}

// SetProtectedContent enables or disables content protection (no-forward
// restriction) on the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) SetProtectedContent(enabled bool) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundSetProtectedContent(c.ID, enabled)
}

// SetTTL sets the auto-delete TTL (Time-To-Live) for messages in the chat, in
// seconds. Pass 0 to disable auto-deletion.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) SetTTL(ttl int) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundSetTTL(c.ID, ttl)
}

// SetPermissions applies the given default permissions for non-admin members.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) SetPermissions(permissions *tg.ChatBannedRights) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundSetPermissions(c.ID, permissions)
}

// SetAdminTitle assigns a custom title to an admin in a supergroup.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) SetAdminTitle(userID int64, title string) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundSetAdminTitle(c.ID, userID, title)
}

// SetSlowMode enables slow mode with the given cooldown in seconds. Pass 0 to
// disable.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) SetSlowMode(seconds int) error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundSetSlowMode(c.ID, seconds)
}

// Mute disables notifications for the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) Mute() error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundMute(c.ID)
}

// Unmute re-enables notifications for the chat.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) Unmute() error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return c.binder.BoundUnmute(c.ID)
}

// UnpinAll removes all pinned messages from the chat and returns the count of
// unpinned messages.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) UnpinAll() (int, error) {
	if c.binder == nil {
		return 0, ErrNoChatBinder
	}
	return c.binder.BoundUnpinAll(c.ID)
}

// GetChat fetches the latest full chat information from the server.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) GetChat() (*Chat, error) {
	if c.binder == nil {
		return nil, ErrNoChatBinder
	}
	return c.binder.BoundGetChat(c.ID)
}

// GetEventLog retrieves recent admin log events matching the query, up to limit
// entries.
// Returns ErrNoChatBinder if the chat was not created by a client.
func (c *Chat) GetEventLog(query string, limit int) ([]*ChatEvent, error) {
	if c.binder == nil {
		return nil, ErrNoChatBinder
	}
	return c.binder.BoundGetEventLog(c.ID, query, limit)
}
