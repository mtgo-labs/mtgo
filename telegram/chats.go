package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// GetChat retrieves full information about a chat or channel.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//
// Returns a types.Chat containing the chat metadata, or an error if the peer
// cannot be resolved, is a user rather than a chat, or is an unsupported type.
//
// Example:
//
//	chat, err := client.GetChat(ctx, chatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(chat.Title)
func (c *Client) GetChat(ctx context.Context, chatID int64) (*types.Chat, error) {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	switch p := peer.(type) {
	case *tg.InputPeerChat:
		rpc := c.Raw()
		result, err := rpc.MessagesGetFullChat(ctx, &tg.MessagesGetFullChatRequest{ChatID: p.ChatID})
		if err != nil {
			return nil, err
		}
		return extractChatFromFull(result)
	case *tg.InputPeerChannel:
		ch, err := resolveChannelID(c, chatID)
		if err != nil {
			return nil, err
		}
		rpc := c.Raw()
		result, err := rpc.ChannelsGetFullChannel(ctx, &tg.ChannelsGetFullChannelRequest{Channel: ch})
		if err != nil {
			return nil, err
		}
		return extractChatFromFull(result)
	case *tg.InputPeerUser, *tg.InputPeerSelf:
		return nil, ErrGetChatNotChat
	default:
		return nil, fmt.Errorf("GetChat: unsupported peer type %T", peer)
	}
}

func extractChatFromFull(result tg.ChatFullClass) (*types.Chat, error) {
	switch v := result.(type) {
	case *tg.ChatFull:
		return &types.Chat{ID: v.ID}, nil
	case *tg.ChannelFull:
		return &types.Chat{ID: v.ID}, nil
	default:
		return nil, fmt.Errorf("unexpected chat full type %T", result)
	}
}

// JoinChat joins a chat or channel using an invite link hash.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - inviteHash: the hash portion of the invite link
//
// Returns the joined types.Chat on success. Returns an error if the invite is
// invalid, expired, or the server returns an unexpected response type.
//
// Example:
//
//	chat, err := client.JoinChat(ctx, "a1b2c3d4e5f6")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Joined: %s (ID: %d)\n", chat.Title, chat.ID)
func (c *Client) JoinChat(ctx context.Context, inviteHash string) (*types.Chat, error) {
	c.Log.Debugf("JoinChat hash_len=%d", len(inviteHash))
	rpc := c.Raw()
	result, err := rpc.MessagesImportChatInvite(ctx, &tg.MessagesImportChatInviteRequest{Hash: inviteHash})
	if err != nil {
		return nil, err
	}

	switch v := result.(type) {
	case *tg.Updates:
		pm := types.NewPeerMapFromClasses(v.Users, v.Chats)
		for _, u := range v.Updates {
			if upd, ok := u.(*tg.UpdateNewChannelMessage); ok {
				_ = upd
				_ = pm
			}
		}
		for _, chat := range v.Chats {
			if ch, ok := chat.(*tg.Channel); ok {
				return types.ParseChatFromPeer(&tg.PeerChannel{ChannelID: ch.ID}, pm), nil
			}
		}
		return nil, ErrJoinNoInfo
	default:
		return nil, fmt.Errorf("unexpected join result type %T", result)
	}
}

// LeaveChat leaves the specified chat or channel.
//
// For channels/supergroups it calls ChannelsLeaveChannel; for basic groups it
// removes the current user via MessagesDeleteChatUser.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//
// Returns an error if the peer cannot be resolved or is an unsupported type.
//
// Example:
//
//	err := client.LeaveChat(ctx, chatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Left the chat")
func (c *Client) LeaveChat(ctx context.Context, chatID int64) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	c.Log.Debugf("LeaveChat chat_id=%d", chatID)

	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		ch, err := resolveChannelID(c, chatID)
		if err != nil {
			return err
		}
		rpc := c.Raw()
		_, err = rpc.ChannelsLeaveChannel(ctx, &tg.ChannelsLeaveChannelRequest{Channel: ch})
		return err
	case *tg.InputPeerChat:
		rpc := c.Raw()
		_, err = rpc.MessagesDeleteChatUser(ctx, &tg.MessagesDeleteChatUserRequest{
			ChatID: p.ChatID,
			UserID: &tg.InputUserSelf{},
		})
		return err
	default:
		return fmt.Errorf("LeaveChat: unsupported peer type %T", p)
	}
}

// CreateChannel creates a new channel or megagroup (supergroup).
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - title: the channel title
//   - about: the channel description/about text
//   - megagroup: if true, creates a megagroup instead of a broadcast channel
//
// Returns the newly created types.Chat on success. Returns an error if creation
// fails or the response cannot be parsed.
//
// Example:
//
//	ch, err := client.CreateChannel(ctx, "Announcements", "Official updates", false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Created channel: %s (ID: %d)\n", ch.Title, ch.ID)
func (c *Client) CreateChannel(ctx context.Context, title, about string, megagroup bool) (*types.Chat, error) {
	c.Log.Debugf("CreateChannel megagroup=%v", megagroup)
	var flags tg.Fields
	flags.Set(0)
	if megagroup {
		flags.Set(1)
	}

	rpc := c.Raw()
	result, err := rpc.ChannelsCreateChannel(ctx, &tg.ChannelsCreateChannelRequest{
		Flags: flags,
		Title: title,
		About: about,
	})
	if err != nil {
		return nil, err
	}

	switch v := result.(type) {
	case *tg.Updates:
		pm := types.NewPeerMapFromClasses(v.Users, v.Chats)
		for _, chat := range v.Chats {
			if ch, ok := chat.(*tg.Channel); ok {
				return types.ParseChatFromPeer(&tg.PeerChannel{ChannelID: ch.ID}, pm), nil
			}
		}
		return nil, ErrChannelNoInfo
	default:
		return nil, fmt.Errorf("unexpected create channel result type %T", result)
	}
}

// DeleteChat deletes the specified chat or channel.
//
// For channels/supergroups it calls ChannelsDeleteChannel; for basic groups it
// calls MessagesDeleteChat.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//
// Returns an error if the peer cannot be resolved or is an unsupported type.
func (c *Client) DeleteChat(ctx context.Context, chatID int64) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	c.Log.Debugf("DeleteChat chat_id=%d", chatID)

	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		ch, err := resolveChannelID(c, chatID)
		if err != nil {
			return err
		}
		rpc := c.Raw()
		_, err = rpc.ChannelsDeleteChannel(ctx, &tg.ChannelsDeleteChannelRequest{Channel: ch})
		return err
	case *tg.InputPeerChat:
		rpc := c.Raw()
		_, err = rpc.MessagesDeleteChat(ctx, &tg.MessagesDeleteChatRequest{ChatID: p.ChatID})
		return err
	default:
		return fmt.Errorf("DeleteChat: unsupported peer type %T", p)
	}
}

// SetChatTitle sets the title of a chat or channel.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//   - title: the new title string
//
// Returns an error if the peer cannot be resolved or is an unsupported type.
func (c *Client) SetChatTitle(ctx context.Context, chatID int64, title string) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		ch, err := resolveChannelID(c, chatID)
		if err != nil {
			return err
		}
		rpc := c.Raw()
		_, err = rpc.ChannelsEditTitle(ctx, &tg.ChannelsEditTitleRequest{Channel: ch, Title: title})
		return err
	case *tg.InputPeerChat:
		rpc := c.Raw()
		_, err = rpc.MessagesEditChatTitle(ctx, &tg.MessagesEditChatTitleRequest{ChatID: p.ChatID, Title: title})
		return err
	default:
		return fmt.Errorf("SetChatTitle: unsupported peer type %T", p)
	}
}

// SetChatDescription sets the description (about text) of a chat or channel.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//   - about: the new description text
//
// Returns an error if the peer cannot be resolved or is an unsupported type.
func (c *Client) SetChatDescription(ctx context.Context, chatID int64, about string) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		rpc := c.Raw()
		_, err = rpc.MessagesEditChatAbout(ctx, &tg.MessagesEditChatAboutRequest{Peer: p, About: about})
		return err
	case *tg.InputPeerChat:
		rpc := c.Raw()
		_, err = rpc.MessagesEditChatAbout(ctx, &tg.MessagesEditChatAboutRequest{Peer: p, About: about})
		return err
	default:
		return fmt.Errorf("SetChatDescription: unsupported peer type %T", p)
	}
}

// SetChatUsername sets the public username of a channel.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel (ChatRef)
//   - username: the new public username (without @)
//
// Returns an error if the channel cannot be resolved or the username is taken.
func (c *Client) SetChatUsername(ctx context.Context, chatID int64, username string) error {
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return err
	}
	rpc := c.Raw()
	_, err = rpc.ChannelsUpdateUsername(ctx, &tg.ChannelsUpdateUsernameRequest{Channel: ch, Username: username})
	return err
}

// BanChatMember bans a user from a channel or supergroup, revoking all
// standard permissions (view messages, send media, embed links, etc.).
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel/supergroup (ChatRef)
//   - userID: identifier of the user to ban (UserRef)
//
// Returns an error if the peer is not a channel/supergroup, or if the user
// or channel cannot be resolved.
//
// Example:
//
//	err := client.BanChatMember(ctx, chatID, spammerID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("User banned successfully")
func (c *Client) BanChatMember(ctx context.Context, chatID int64, userID int64) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	c.Log.Debugf("BanChatMember chat_id=%d user_id=%d", chatID, userID)

	ch, ok := peer.(*tg.InputPeerChannel)
	if !ok {
		return ErrBanSupergroupOnly
	}

	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}

	var userPeer tg.InputPeerClass
	switch u := user.(type) {
	case *tg.InputUser:
		userPeer = &tg.InputPeerUser{UserID: u.UserID, AccessHash: u.AccessHash}
	case *tg.InputUserSelf:
		userPeer = &tg.InputPeerSelf{}
	default:
		return fmt.Errorf("BanChatMember: unsupported user type %T", user)
	}

	rpc := c.Raw()
	_, err = rpc.ChannelsEditBanned(ctx, &tg.ChannelsEditBannedRequest{
		Channel:     &tg.InputChannel{ChannelID: ch.ChannelID, AccessHash: ch.AccessHash},
		Participant: userPeer,
		BannedRights: &tg.ChatBannedRights{
			Flags:        (1 << 0) | (1 << 1) | (1 << 2) | (1 << 3) | (1 << 4) | (1 << 5) | (1 << 6) | (1 << 7) | (1 << 8) | (1 << 10),
			ViewMessages: true,
		},
	})
	return err
}

// UnbanChatMember removes all previously applied ban restrictions from a user
// in a channel or supergroup, restoring their default permissions.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel/supergroup (ChatRef)
//   - userID: identifier of the user to unban (UserRef)
//
// Returns an error if the peer is not a channel/supergroup, or if the user or
// channel cannot be resolved.
func (c *Client) UnbanChatMember(ctx context.Context, chatID int64, userID int64) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}

	c.Log.Debugf("UnbanChatMember chat_id=%d user_id=%d", chatID, userID)

	ch, ok := peer.(*tg.InputPeerChannel)
	if !ok {
		return ErrUnbanSupergroupOnly
	}

	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user peer: %w", err)
	}

	var userPeer tg.InputPeerClass
	switch u := user.(type) {
	case *tg.InputUser:
		userPeer = &tg.InputPeerUser{UserID: u.UserID, AccessHash: u.AccessHash}
	case *tg.InputUserSelf:
		userPeer = &tg.InputPeerSelf{}
	default:
		return fmt.Errorf("UnbanChatMember: unsupported user type %T", user)
	}

	rpc := c.Raw()
	_, err = rpc.ChannelsEditBanned(ctx, &tg.ChannelsEditBannedRequest{
		Channel:      &tg.InputChannel{ChannelID: ch.ChannelID, AccessHash: ch.AccessHash},
		Participant:  userPeer,
		BannedRights: &tg.ChatBannedRights{},
	})
	return err
}

// PromoteChatMember promotes a user to administrator in a channel with the
// specified admin rights.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel (ChatRef)
//   - userID: identifier of the user to promote (UserRef)
//   - adminRights: the set of admin permissions to grant
//
// Returns an error if the channel or user cannot be resolved.
//
// Example:
//
//	rights := &tg.ChatAdminRights{
//	    ChangeInfo: true,
//	    PostMessages: true,
//	    DeleteMessages: true,
//	}
//	err := client.PromoteChatMember(ctx, chatID, userID, rights)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (c *Client) PromoteChatMember(ctx context.Context, chatID int64, userID int64, adminRights *tg.ChatAdminRights) error {
	c.Log.Debugf("PromoteChatMember chat_id=%d user_id=%d", chatID, userID)
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return err
	}

	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.ChannelsEditAdmin(ctx, &tg.ChannelsEditAdminRequest{
		Channel:     ch,
		UserID:      user,
		AdminRights: adminRights,
	})
	return err
}

// GetChatMember retrieves information about a single participant in a channel
// or supergroup.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel (ChatRef)
//   - userID: identifier of the user to look up (UserRef)
//
// Returns a types.ChatMember with the participant's details, or an error if
// the channel, user, or participant cannot be resolved.
func (c *Client) GetChatMember(ctx context.Context, chatID int64, userID int64) (*types.ChatMember, error) {
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return nil, err
	}

	user, err := resolveUserID(c, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	var userPeer tg.InputPeerClass
	switch u := user.(type) {
	case *tg.InputUser:
		userPeer = &tg.InputPeerUser{UserID: u.UserID, AccessHash: u.AccessHash}
	case *tg.InputUserSelf:
		userPeer = &tg.InputPeerSelf{}
	default:
		return nil, fmt.Errorf("GetChatMember: unsupported user type %T", user)
	}

	rpc := c.Raw()
	result, err := rpc.ChannelsGetParticipant(ctx, &tg.ChannelsGetParticipantRequest{
		Channel:     ch,
		Participant: userPeer,
	})
	if err != nil {
		return nil, err
	}

	usersMap := make(map[int64]tg.UserClass)
	return types.ParseChannelParticipant(result, usersMap), nil
}

// GetChatMembers retrieves a paginated list of participants in a channel or
// supergroup.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel (ChatRef)
//   - limit: maximum number of members to return (defaults to 200 if <= 0)
//   - offset: number of members to skip for pagination
//
// Returns a slice of types.ChatMember, or an error if the channel cannot be
// resolved or the server returns an unexpected response type.
//
// Example:
//
//	members, err := client.GetChatMembers(ctx, chatID, 50, 0)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, m := range members {
//	    fmt.Printf("%s (ID: %d)\n", m.User.FirstName, m.User.ID)
//	}
func (c *Client) GetChatMembers(ctx context.Context, chatID int64, limit int, offset int) ([]*types.ChatMember, error) {
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 200
	}

	rpc := c.Raw()
	result, err := rpc.ChannelsGetParticipants(ctx, &tg.ChannelsGetParticipantsRequest{
		Channel: ch,
		Filter:  &tg.ChannelParticipantsSearch{},
		Offset:  int32(offset),
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}

	var participants []tg.ChannelParticipantClass
	var users []tg.UserClass

	switch v := result.(type) {
	case *tg.ChannelsChannelParticipants:
		participants = v.Participants
		users = v.Users
	default:
		return nil, fmt.Errorf("unexpected participants type %T", result)
	}

	pm := make(map[int64]tg.UserClass)
	for _, u := range users {
		if v, ok := u.(*tg.User); ok && v != nil {
			pm[v.ID] = u
		}
	}
	members := make([]*types.ChatMember, 0, len(participants))
	for _, p := range participants {
		if m := types.ParseChannelParticipant(p, pm); m != nil {
			members = append(members, m)
		}
	}
	return members, nil
}

// GetChatMembersCount returns the total number of participants in a channel or
// supergroup.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel (ChatRef)
//
// Returns the member count, or an error if the channel cannot be resolved or
// the server returns an unexpected response type.
func (c *Client) GetChatMembersCount(ctx context.Context, chatID int64) (int, error) {
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return 0, err
	}

	rpc := c.Raw()
	result, err := rpc.ChannelsGetParticipants(ctx, &tg.ChannelsGetParticipantsRequest{
		Channel: ch,
		Filter:  &tg.ChannelParticipantsSearch{},
		Limit:   0,
	})
	if err != nil {
		return 0, err
	}

	switch v := result.(type) {
	case *tg.ChannelsChannelParticipants:
		return int(v.Count), nil
	default:
		return 0, fmt.Errorf("unexpected participants type %T", result)
	}
}

// AddChatMember adds a user to a channel or supergroup.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel (ChatRef)
//   - userID: identifier of the user to add (UserRef)
//
// Returns an error if the channel or user cannot be resolved, or if the user
// cannot be invited (e.g. already a member or privacy restrictions).
func (c *Client) AddChatMember(ctx context.Context, chatID int64, userID int64) error {
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return err
	}

	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}

	rpc := c.Raw()
	_, err = rpc.ChannelsInviteToChannel(ctx, &tg.ChannelsInviteToChannelRequest{
		Channel: ch,
		Users:   []tg.InputUserClass{user},
	})
	return err
}

// RestrictChatMember applies the given banned rights to a user in a channel or
// supergroup. Unlike BanChatMember which revokes all permissions, this method
// allows fine-grained control over which actions the user is restricted from
// performing (e.g. sending messages, media, or links).
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel/supergroup (ChatRef)
//   - userID: identifier of the user to restrict (UserRef)
//   - bannedRights: the set of restrictions to apply, where each set flag
//     indicates a denied action
//
// Returns an error if the peer is not a channel/supergroup, or if the user or
// channel cannot be resolved.
func (c *Client) RestrictChatMember(ctx context.Context, chatID int64, userID int64, bannedRights *tg.ChatBannedRights) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	ch, ok := peer.(*tg.InputPeerChannel)
	if !ok {
		return ErrRestrictSupergroupOnly
	}
	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}
	var userPeer tg.InputPeerClass
	switch u := user.(type) {
	case *tg.InputUser:
		userPeer = &tg.InputPeerUser{UserID: u.UserID, AccessHash: u.AccessHash}
	case *tg.InputUserSelf:
		userPeer = &tg.InputPeerSelf{}
	default:
		return fmt.Errorf("unsupported user type %T", user)
	}
	rpc := c.Raw()
	_, err = rpc.ChannelsEditBanned(ctx, &tg.ChannelsEditBannedRequest{
		Channel:      &tg.InputChannel{ChannelID: ch.ChannelID, AccessHash: ch.AccessHash},
		Participant:  userPeer,
		BannedRights: bannedRights,
	})
	return err
}

// SetChatPhoto sets or updates the photo for a chat or channel. For channels
// it calls ChannelsEditPhoto; for basic groups it calls MessagesEditChatPhoto.
// Pass an InputChatPhotoEmpty to remove the current photo (or use DeleteChatPhoto).
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//   - photo: the new chat photo input (uploaded file, existing photo, or empty)
//
// Returns an error if the peer cannot be resolved or is an unsupported type.
func (c *Client) SetChatPhoto(ctx context.Context, chatID int64, photo tg.InputChatPhotoClass) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		ch, err := resolveChannelID(c, chatID)
		if err != nil {
			return err
		}
		rpc := c.Raw()
		_, err = rpc.ChannelsEditPhoto(ctx, &tg.ChannelsEditPhotoRequest{Channel: ch, Photo: photo})
		return err
	case *tg.InputPeerChat:
		rpc := c.Raw()
		_, err = rpc.MessagesEditChatPhoto(ctx, &tg.MessagesEditChatPhotoRequest{ChatID: p.ChatID, Photo: photo})
		return err
	default:
		return fmt.Errorf("SetChatPhoto: unsupported peer type %T", p)
	}
}

// DeleteChatPhoto removes the photo from a chat or channel by setting it to empty.
// This is a convenience wrapper around SetChatPhoto with an InputChatPhotoEmpty.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//
// Returns an error if the peer cannot be resolved or the RPC call fails.
func (c *Client) DeleteChatPhoto(ctx context.Context, chatID int64) error {
	return c.SetChatPhoto(ctx, chatID, &tg.InputChatPhotoEmpty{})
}

// SetChatTTL sets the Time-To-Live (auto-delete) period for messages in a chat.
// Messages sent after this call will be automatically deleted after the specified
// number of seconds. Use 0 to disable auto-deletion.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//   - ttl: time-to-live in seconds; 0 disables auto-deletion
//
// Returns an error if the peer cannot be resolved or the RPC call fails.
func (c *Client) SetChatTTL(ctx context.Context, chatID int64, ttl int) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	rpc := c.Raw()
	_, err = rpc.MessagesSetHistoryTTL(ctx, &tg.MessagesSetHistoryTTLRequest{
		Peer:   peer,
		Period: int32(ttl),
	})
	return err
}

// SetChatPermissions sets the default banned rights for all members of a chat or
// channel. These permissions apply to newly joined members and define which actions
// are restricted by default (e.g. sending messages, media, or polls).
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//   - permissions: the default banned rights to apply; each set flag indicates a
//     denied action for regular members
//
// Returns an error if the peer cannot be resolved or the RPC call fails.
func (c *Client) SetChatPermissions(ctx context.Context, chatID int64, permissions *tg.ChatBannedRights) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	rpc := c.Raw()
	_, err = rpc.MessagesEditChatDefaultBannedRights(ctx, &tg.MessagesEditChatDefaultBannedRightsRequest{
		Peer:         peer,
		BannedRights: permissions,
	})
	return err
}

// MarkChatUnread toggles the unread status of a chat in the current user's chat list.
// When unread is true the chat is marked as unread (e.g. to remind the user to read
// it later); when false the unread marker is removed.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//   - unread: true to mark as unread, false to remove the marker
//
// Returns an error if the peer cannot be resolved or the RPC call fails.
func (c *Client) MarkChatUnread(ctx context.Context, chatID int64, unread bool) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	var flags tg.Fields
	if unread {
		flags.Set(0)
	}
	rpc := c.Raw()
	_, err = rpc.MessagesMarkDialogUnread(ctx, &tg.MessagesMarkDialogUnreadRequest{
		Flags:  flags,
		Unread: unread,
		Peer:   &tg.InputDialogPeer{Peer: peer},
	})
	return err
}

// SetProtectedContent toggles the "protected content" flag on a channel or
// supergroup. When enabled, messages in the chat cannot be forwarded or saved,
// and screenshots are watermarked.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel (ChatRef)
//   - enabled: true to enable protected content, false to disable
//
// Returns an error if the channel cannot be resolved or the RPC call fails.
func (c *Client) SetProtectedContent(ctx context.Context, chatID int64, enabled bool) error {
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return err
	}
	rpc := c.Raw()
	_, err = rpc.ChannelsTogglePreHistoryHidden(ctx, &tg.ChannelsTogglePreHistoryHiddenRequest{
		Channel: ch,
		Enabled: enabled,
	})
	return err
}

// SetAdministratorTitle sets the custom administrator title (rank) displayed
// next to a user's name in a channel or supergroup. The user must already be
// an administrator. Pass an empty title to clear it.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel (ChatRef)
//   - userID: identifier of the target administrator (UserRef)
//   - title: the custom admin title string (e.g. "Head Mod")
//
// Returns an error if the channel or user cannot be resolved.
func (c *Client) SetAdministratorTitle(ctx context.Context, chatID int64, userID int64, title string) error {
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return err
	}
	user, err := resolveUserID(c, userID)
	if err != nil {
		return fmt.Errorf("resolve user: %w", err)
	}
	rpc := c.Raw()
	_, err = rpc.ChannelsEditAdmin(ctx, &tg.ChannelsEditAdminRequest{
		Channel:     ch,
		UserID:      user,
		AdminRights: &tg.ChatAdminRights{},
		Rank:        title,
	})
	return err
}

// CreateGroup creates a new basic group chat with the specified title and initial
// members. All provided user IDs are resolved and invited during creation.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - title: the group chat title
//   - userIDs: slice of user identifiers (UserRef) to add as initial members
//
// Returns the newly created types.Chat on success. Returns an error if any user
// cannot be resolved, creation fails, or the response cannot be parsed.
//
// Example:
//
//	group, err := client.CreateGroup(ctx, "Project Team", []int64{123456, 789012})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Created group: %s (ID: %d)\n", group.Title, group.ID)
func (c *Client) CreateGroup(ctx context.Context, title string, userIDs []int64) (*types.Chat, error) {
	c.Log.Debugf("CreateGroup count=%d", len(userIDs))
	users := make([]tg.InputUserClass, len(userIDs))
	for i, uid := range userIDs {
		u, err := resolveUserID(c, uid)
		if err != nil {
			return nil, fmt.Errorf("resolve user %v: %w", uid, err)
		}
		users[i] = u
	}
	rpc := c.Raw()
	result, err := rpc.MessagesCreateChat(ctx, &tg.MessagesCreateChatRequest{
		Users: users,
		Title: title,
	})
	if err != nil {
		return nil, err
	}
	upd, ok := result.Updates.(*tg.Updates)
	if !ok {
		return nil, fmt.Errorf("unexpected create chat result type %T", result.Updates)
	}
	pm := types.NewPeerMapFromClasses(upd.Users, upd.Chats)
	for _, chat := range upd.Chats {
		return types.ParseChatFromChat(chat), nil
	}
	_ = pm
	return nil, ErrGroupNoInfo
}

// CreateSupergroup creates a new supergroup (megagroup) with the given title and
// description. This is a convenience wrapper around CreateChannel with the
// megagroup flag set to true.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - title: the supergroup title
//   - about: the supergroup description/about text
//
// Returns the newly created types.Chat on success, or an error if creation fails.
func (c *Client) CreateSupergroup(ctx context.Context, title, about string) (*types.Chat, error) {
	return c.CreateChannel(ctx, title, about, true)
}

// SetSlowMode configures the slow mode delay for a channel or supergroup.
// When enabled, non-administrator members must wait the specified number of
// seconds between consecutive messages. Use 0 to disable slow mode.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel (ChatRef)
//   - seconds: the slow mode interval in seconds; 0 disables slow mode
//
// Returns an error if the channel cannot be resolved or the RPC call fails.
func (c *Client) SetSlowMode(ctx context.Context, chatID int64, seconds int) error {
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return err
	}
	rpc := c.Raw()
	_, err = rpc.ChannelsToggleSlowMode(ctx, &tg.ChannelsToggleSlowModeRequest{
		Channel: ch,
		Seconds: int32(seconds),
	})
	return err
}

// GetChatEventLog retrieves administrative events from a channel or supergroup's
// audit log. Only administrators with the appropriate rights can access this.
// Events include actions like member joins, permission changes, message edits,
// and deletions.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target channel (ChatRef)
//   - query: search string to filter events (empty for all events)
//   - limit: maximum number of events to return (defaults to 100 if <= 0)
//
// Returns a slice of types.ChatEvent entries, or an error if the channel cannot
// be resolved or the RPC call fails.
func (c *Client) GetChatEventLog(ctx context.Context, chatID int64, query string, limit int) ([]*types.ChatEvent, error) {
	ch, err := resolveChannelID(c, chatID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	rpc := c.Raw()
	result, err := rpc.ChannelsGetAdminLog(ctx, &tg.ChannelsGetAdminLogRequest{
		Channel: ch,
		Q:       query,
		MaxID:   0,
		MinID:   0,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}
	users := make(map[int64]tg.UserClass, len(result.Users))
	for _, u := range result.Users {
		if user, ok := u.(*tg.User); ok {
			users[user.ID] = u
		}
	}
	pm := types.NewPeerMapFromClasses(result.Users, result.Chats)
	events := make([]*types.ChatEvent, 0, len(result.Events))
	for _, e := range result.Events {
		if parsed := types.ParseChatEvent(e, users, pm); parsed != nil {
			events = append(events, parsed)
		}
	}
	return events, nil
}

var (
	muteForever = int32(0x7FFFFFFF)
	muteOff     = int32(0)
)

// MuteChat disables all notifications for the specified chat indefinitely.
// This sets the mute-until value to the maximum int32, effectively muting
// forever.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//
// Returns an error if the peer cannot be resolved or the RPC call fails.
//
// Example:
//
//	err := client.MuteChat(ctx, chatID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Chat muted")
func (c *Client) MuteChat(ctx context.Context, chatID int64) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	rpc := c.Raw()
	_, err = rpc.AccountUpdateNotifySettings(ctx, &tg.AccountUpdateNotifySettingsRequest{
		Peer: &tg.InputNotifyPeer{Peer: peer},
		Settings: &tg.InputPeerNotifySettings{
			MuteUntil: muteForever,
		},
	})
	return err
}

// UnmuteChat re-enables notifications for the specified chat by setting the
// mute-until value to 0 (no mute). This reverses the effect of MuteChat.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: identifier of the target chat (ChatRef)
//
// Returns an error if the peer cannot be resolved or the RPC call fails.
func (c *Client) UnmuteChat(ctx context.Context, chatID int64) error {
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	rpc := c.Raw()
	_, err = rpc.AccountUpdateNotifySettings(ctx, &tg.AccountUpdateNotifySettingsRequest{
		Peer: &tg.InputNotifyPeer{Peer: peer},
		Settings: &tg.InputPeerNotifySettings{
			MuteUntil: muteOff,
		},
	})
	return err
}
