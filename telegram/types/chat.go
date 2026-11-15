package types

import (
	"context"
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/internal/peerid"
	"github.com/mtgo-labs/mtgo/tg"
)

// Chat represents any Telegram chat (private, bot, group, supergroup, channel, or forum).
// It merges fields from the base TL object (User/Chat/Channel) and the corresponding Full
// object (UserFull/ChatFull/ChannelFull) into a single flat struct following a single flat-struct model.
//
// Base fields are populated by ParseChatFromUser, ParseChatFromChat, or ParseChatFromPeer.
// Full fields are merged in by EnrichChatFull.
type Chat struct {
	ID   int64
	Type ChatType
	Raw  any

	IsVerified               bool
	IsScam                   bool
	IsFake                   bool
	IsRestricted             bool
	IsCreator                bool
	IsLeft                   bool
	IsAdmin                  bool
	IsForum                  bool
	IsMin                    bool
	IsPublic                 bool
	IsDeactivated            bool
	IsCallActive             bool
	IsCallNotEmpty           bool
	IsSlowMode               bool
	IsNoforwards             bool
	IsJoinToSend             bool
	IsJoinRequest            bool
	IsGigagroup              bool
	IsMonoforum              bool
	IsDirectMessages         bool
	IsStoriesHidden          bool
	IsStoriesUnavailable     bool
	IsSupport                bool
	IsBusinessBot            bool
	IsBanned                 bool
	IsMembersHidden          bool
	IsPreview                bool
	IsContactRequirePremium  bool
	HasForumTabs             bool
	HasLink                  bool
	HasGeo                   bool
	SignMessages             bool
	ShowMessageSenderName    bool
	HasProtectedContent      bool
	HasAutoTranslation       bool
	HasVisibleHistory        bool
	HasAggressiveAntiSpam    bool
	HasDirectMessagesGroup   bool
	IsPaidReactionsAvailable bool

	IsBlocked                bool
	IsPhoneCallsAvailable    bool
	IsPhoneCallsPrivate      bool
	IsVideoCallsAvailable    bool
	IsWallpaperOverridden    bool
	IsTranslationsDisabled   bool
	IsPinnedStoriesAvailable bool
	IsBlockedMyStoriesFrom   bool
	IsReadDatesAvailable     bool
	IsAdsEnabled             bool
	IsAdsRestricted          bool
	IsPaidMessagesAvailable  bool
	UsesUnofficialApp        bool
	CanPinMessage            bool
	CanScheduleMessages      bool
	CanSendVoiceMessages     bool
	CanViewRevenue           bool
	CanViewStarsRevenue      bool
	CanViewStats             bool
	CanViewParticipants      bool
	CanDeleteChannel         bool
	CanSetUsername           bool
	CanSetStickerSet         bool
	CanSetLocation           bool
	CanSendPaidMedia         bool
	CanSendGift              bool
	BotCanManageEmojiStatus  bool
	DisplayGiftsButton       bool

	Title                  string
	FirstName              string
	LastName               string
	Username               string
	Description            string
	Bio                    string
	Phone                  string
	Language               string
	PrivateForwardName     string
	InviteLink             string
	StickerSetName         string
	CustomEmojiStickerSet  string
	ThemeEmoticon          string
	InlineQueryPlaceholder string

	Photo           *ChatPhoto
	PersonalPhoto   *ChatPhoto
	PublicPhoto     *ChatPhoto
	Restrictions    []*Restriction
	Usernames       []*Username
	EmojiStatus     *EmojiStatus
	ReplyColor      *ChatColor
	ProfileColor    *ChatColor
	Birthday        *Birthday
	BotVerification *BotVerification

	AccessHash            int64
	MembersCount          int
	Date                  int32
	Level                 int32
	SubscriptionUntilDate time.Time
	BotVerificationIcon   int64
	PaidMessageStarCount  int64

	CommonChats          int32
	FolderID             int32
	MessageAutoDelete    int32
	GiftCount            int32
	PinnedMessageID      int32
	PersonalChannelID    int64
	PersonalChannelMsg   int32
	LinkedChatID         int64
	DirectMessagesChatID int64
	MigratedFromChatID   int64
	MigratedFromMaxID    int32
	UnreadCount          int32
	ReadInboxMaxID       int32
	ReadOutboxMaxID      int32
	OnlineCount          int32
	AdminsCount          int32
	KickedCount          int32
	BannedCount          int32
	AvailableMinID       int32
	SlowModeDelay        int32
	SlowmodeNextSendDate time.Time
	JoinRequestsCount    int32
	BoostsApplied        int32
	BoostsUnrestrict     int32
	ReactionsLimit       int32
	StatsDC              int32
	AdminRights          *ChatAdminRights
	BannedRights         *ChatBannedRights
	Permissions          *ChatPermissions
	BotGroupRights       *ChatAdminRights
	BotBroadcastRights   *ChatAdminRights
	AvailableReactions   tg.ChatReactionsClass

	VerificationStatus     *VerificationStatus
	Stories                []*Story
	ChatBackground         *ChatBackground
	PinnedMessage          *Message
	Members                []*User
	PersonalChannel        *Chat
	PersonalChannelMessage *Message
	ParentChat             *Chat
	LinkedChat             *Chat
	SendAsChat             *Chat

	BusinessAwayMessage     *BusinessMessage
	BusinessGreetingMessage *BusinessMessage
	BusinessWorkHours       *BusinessWorkingHours
	BusinessLocation        *LocationMedia
	BusinessIntro           *BusinessIntro

	IsJoinByRequest       bool
	IsViewForumAsMessages bool
	BannedUntilDate       time.Time
	CanManageBots         bool

	MainProfileTab    ProfileTab
	FirstProfileAudio *DocumentMedia
	Rating            *UserRating
	PendingRating     *UserRating
	PendingRatingDate time.Time
	Settings          *ChatSettings

	ChannelAdminRights *ChatAdministratorRights
	ChatAdminRights    *ChatAdministratorRights
	Theme              string
	AcceptedGiftTypes  *AcceptedGiftTypes
	Note               *FormattedText

	binder ChatBinder
}

// ChatPreview represents a partial view of a chat obtained from an invite link,
// without requiring the user to join.
type ChatPreview struct {
	ID           int64
	Type         ChatType
	Title        string
	Username     string
	Photo        *ChatPhoto
	MembersCount int
	IsVerified   bool
	IsScam       bool
	IsFake       bool
}

// SetBinder injects the ChatBinder that backs bound convenience methods on this Chat.
func (c *Chat) SetBinder(b ChatBinder) {
	c.binder = b
}

// String returns the chat's title, full name (for private/bot), or a fallback "chat_<id>" string.
func (c *Chat) String() string {
	if c.Type == ChatTypePrivate || c.Type == ChatTypeBot {
		name := c.FirstName
		if c.LastName != "" {
			name += " " + c.LastName
		}
		if name != "" {
			return name
		}
	}
	if c.Title != "" {
		return c.Title
	}
	return fmt.Sprintf("chat_%d", c.ID)
}

// MentionName returns "@username" if set, otherwise falls back to String().
func (c *Chat) MentionName() string {
	if c.Username != "" {
		return "@" + c.Username
	}
	return c.String()
}

// FullName returns the display name for the chat: full name for private/bot chats,
// title for groups/channels.
func (c *Chat) FullName() string {
	if c.Type == ChatTypePrivate || c.Type == ChatTypeBot {
		if c.LastName != "" {
			return c.FirstName + " " + c.LastName
		}
		return c.FirstName
	}
	return c.Title
}

func (c *Chat) requireBinder() error {
	if c.binder == nil {
		return ErrNoChatBinder
	}
	return nil
}

func (c *Chat) Archive() error {
	return c.ArchiveContext(context.Background())
}

func (c *Chat) ArchiveContext(ctx context.Context) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.ArchiveChat(ctx, c.ID)
}

func (c *Chat) Unarchive() error {
	return c.UnarchiveContext(context.Background())
}

func (c *Chat) UnarchiveContext(ctx context.Context) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.UnarchiveChat(ctx, c.ID)
}

func (c *Chat) SetTitle(title string) error {
	return c.SetTitleContext(context.Background(), title)
}

func (c *Chat) SetTitleContext(ctx context.Context, title string) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.SetChatTitle(ctx, c.ID, title)
}

func (c *Chat) SetDescription(description string) error {
	return c.SetDescriptionContext(context.Background(), description)
}

func (c *Chat) SetDescriptionContext(ctx context.Context, description string) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.SetChatDescription(ctx, c.ID, description)
}

func (c *Chat) SetPhoto(photo tg.InputChatPhotoClass) error {
	return c.SetPhotoContext(context.Background(), photo)
}

func (c *Chat) SetPhotoContext(ctx context.Context, photo tg.InputChatPhotoClass) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.SetChatPhoto(ctx, c.ID, photo)
}

func (c *Chat) DeletePhoto() error {
	return c.DeletePhotoContext(context.Background())
}

func (c *Chat) DeletePhotoContext(ctx context.Context) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.DeleteChatPhoto(ctx, c.ID)
}

func (c *Chat) SetUsername(username string) error {
	return c.SetUsernameContext(context.Background(), username)
}

func (c *Chat) SetUsernameContext(ctx context.Context, username string) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.SetChatUsername(ctx, c.ID, username)
}

func (c *Chat) BanMember(userID int64) error {
	return c.BanMemberContext(context.Background(), userID)
}

func (c *Chat) BanMemberContext(ctx context.Context, userID int64) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.BanChatMember(ctx, c.ID, userID)
}

func (c *Chat) UnbanMember(userID int64) error {
	return c.UnbanMemberContext(context.Background(), userID)
}

func (c *Chat) UnbanMemberContext(ctx context.Context, userID int64) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.UnbanChatMember(ctx, c.ID, userID)
}

func (c *Chat) RestrictMember(userID int64, bannedRights *tg.ChatBannedRights) error {
	return c.RestrictMemberContext(context.Background(), userID, bannedRights)
}

func (c *Chat) RestrictMemberContext(ctx context.Context, userID int64, bannedRights *tg.ChatBannedRights) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.RestrictChatMember(ctx, c.ID, userID, bannedRights)
}

func (c *Chat) PromoteMember(userID int64, adminRights *tg.ChatAdminRights) error {
	return c.PromoteMemberContext(context.Background(), userID, adminRights)
}

func (c *Chat) PromoteMemberContext(ctx context.Context, userID int64, adminRights *tg.ChatAdminRights) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.PromoteChatMember(ctx, c.ID, userID, adminRights)
}

func (c *Chat) Join(link string) (*Chat, error) {
	return c.JoinContext(context.Background(), link)
}

func (c *Chat) JoinContext(ctx context.Context, link string) (*Chat, error) {
	if err := c.requireBinder(); err != nil {
		return nil, err
	}
	return c.binder.JoinChat(ctx, link)
}

func (c *Chat) Leave() error {
	return c.LeaveContext(context.Background())
}

func (c *Chat) LeaveContext(ctx context.Context) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.LeaveChat(ctx, c.ID)
}

func (c *Chat) ExportInviteLink() (string, error) {
	return c.ExportInviteLinkContext(context.Background())
}

func (c *Chat) ExportInviteLinkContext(ctx context.Context) (string, error) {
	if err := c.requireBinder(); err != nil {
		return "", err
	}
	return c.binder.ExportChatInviteLink(ctx, c.ID)
}

func (c *Chat) GetMember(userID int64) (*ChatMember, error) {
	return c.GetMemberContext(context.Background(), userID)
}

func (c *Chat) GetMemberContext(ctx context.Context, userID int64) (*ChatMember, error) {
	if err := c.requireBinder(); err != nil {
		return nil, err
	}
	return c.binder.GetChatMember(ctx, c.ID, userID)
}

func (c *Chat) GetMembers(limit int, offset int) ([]*ChatMember, error) {
	return c.GetMembersContext(context.Background(), limit, offset)
}

func (c *Chat) GetMembersContext(ctx context.Context, limit int, offset int) ([]*ChatMember, error) {
	if err := c.requireBinder(); err != nil {
		return nil, err
	}
	return c.binder.GetChatMembers(ctx, c.ID, limit, offset)
}

func (c *Chat) AddMembers(userID int64) error {
	return c.AddMembersContext(context.Background(), userID)
}

func (c *Chat) AddMembersContext(ctx context.Context, userID int64) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.AddChatMember(ctx, c.ID, userID)
}

func (c *Chat) MarkUnread(unread bool) error {
	return c.MarkUnreadContext(context.Background(), unread)
}

func (c *Chat) MarkUnreadContext(ctx context.Context, unread bool) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.MarkChatUnread(ctx, c.ID, unread)
}

func (c *Chat) SetProtectedContent(enabled bool) error {
	return c.SetProtectedContentContext(context.Background(), enabled)
}

func (c *Chat) SetProtectedContentContext(ctx context.Context, enabled bool) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.SetProtectedContent(ctx, c.ID, enabled)
}

func (c *Chat) SetTTL(ttl int) error {
	return c.SetTTLContext(context.Background(), ttl)
}

func (c *Chat) SetTTLContext(ctx context.Context, ttl int) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.SetChatTTL(ctx, c.ID, ttl)
}

func (c *Chat) SetPermissions(permissions *tg.ChatBannedRights) error {
	return c.SetPermissionsContext(context.Background(), permissions)
}

func (c *Chat) SetPermissionsContext(ctx context.Context, permissions *tg.ChatBannedRights) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.SetChatPermissions(ctx, c.ID, permissions)
}

func (c *Chat) SetAdminTitle(userID int64, title string) error {
	return c.SetAdminTitleContext(context.Background(), userID, title)
}

func (c *Chat) SetAdminTitleContext(ctx context.Context, userID int64, title string) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.SetAdministratorTitle(ctx, c.ID, userID, title)
}

func (c *Chat) SetSlowMode(seconds int) error {
	return c.SetSlowModeContext(context.Background(), seconds)
}

func (c *Chat) SetSlowModeContext(ctx context.Context, seconds int) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.SetSlowMode(ctx, c.ID, seconds)
}

func (c *Chat) Mute() error {
	return c.MuteContext(context.Background())
}

func (c *Chat) MuteContext(ctx context.Context) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.MuteChat(ctx, c.ID)
}

func (c *Chat) Unmute() error {
	return c.UnmuteContext(context.Background())
}

func (c *Chat) UnmuteContext(ctx context.Context) error {
	if err := c.requireBinder(); err != nil {
		return err
	}
	return c.binder.UnmuteChat(ctx, c.ID)
}

func (c *Chat) UnpinAll() (int, error) {
	return c.UnpinAllContext(context.Background())
}

func (c *Chat) UnpinAllContext(ctx context.Context) (int, error) {
	if err := c.requireBinder(); err != nil {
		return 0, err
	}
	return c.binder.UnpinAllMessages(ctx, c.ID)
}

func (c *Chat) UnpinAllMessages() (int, error) {
	return c.UnpinAll()
}

func (c *Chat) GetChat() (*Chat, error) {
	return c.GetChatContext(context.Background())
}

func (c *Chat) GetChatContext(ctx context.Context) (*Chat, error) {
	if err := c.requireBinder(); err != nil {
		return nil, err
	}
	return c.binder.GetChat(ctx, c.ID)
}

func (c *Chat) GetEventLog(query string, limit int) ([]*ChatEvent, error) {
	return c.GetEventLogContext(context.Background(), query, limit)
}

func (c *Chat) GetEventLogContext(ctx context.Context, query string, limit int) ([]*ChatEvent, error) {
	if err := c.requireBinder(); err != nil {
		return nil, err
	}
	return c.binder.GetChatEventLog(ctx, c.ID, query, limit)
}

// DC returns the data center ID where the chat's photo is stored, or 0 if no photo.
func (c *Chat) DC() int {
	if c.Photo == nil {
		return 0
	}
	return int(c.Photo.DcID)
}

// ParseChatFromUser creates a Chat from a TL UserClass (private chat or bot).
// Returns nil if raw is nil.
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
			ID:                      r.ID,
			Type:                    chatType,
			IsVerified:              r.Verified,
			IsScam:                  r.Scam,
			IsFake:                  r.Fake,
			IsRestricted:            r.Restricted,
			IsSupport:               r.Support,
			IsStoriesHidden:         r.StoriesHidden,
			IsStoriesUnavailable:    r.StoriesUnavailable,
			IsContactRequirePremium: r.ContactRequirePremium,
			IsBusinessBot:           r.BotBusiness,
			Photo:                   parseUserProfilePhoto(r.Photo),
			EmojiStatus:             ParseEmojiStatus(r.EmojiStatus),
			ReplyColor:              ParseChatColorFromPeer(r.Color),
			ProfileColor:            ParseChatColorFromPeer(r.ProfileColor),
			Restrictions:            parseRestrictions(r.RestrictionReason),
			Usernames:               parseUsernames(r.Usernames),
			BotVerificationIcon:     r.BotVerificationIcon,
			PaidMessageStarCount:    r.SendPaidMessagesStars,
			Date:                    0,
			Raw:                     r,
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
		if r.LangCode != "" {
			c.Language = r.LangCode
		}
		if r.Phone != "" {
			c.Phone = r.Phone
		}
		if r.BotInlinePlaceholder != "" {
			c.InlineQueryPlaceholder = r.BotInlinePlaceholder
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

// ParseChatFromChat creates a Chat from a TL ChatClass (group, channel, supergroup, or forum).
// Returns nil if raw is nil.
func ParseChatFromChat(raw tg.ChatClass) *Chat {
	if raw == nil {
		return nil
	}
	switch r := raw.(type) {
	case *tg.ChatEmpty:
		return nil
	case *tg.Chat:
		c := &Chat{
			ID:                  -r.ID,
			Type:                ChatTypeGroup,
			Title:               r.Title,
			IsCreator:           r.Creator,
			IsLeft:              r.Left,
			IsAdmin:             r.AdminRights != nil,
			IsDeactivated:       r.Deactivated,
			IsCallActive:        r.CallActive,
			IsCallNotEmpty:      r.CallNotEmpty,
			IsNoforwards:        r.Noforwards,
			HasProtectedContent: r.Noforwards,
			Photo:               parseChatPhoto(r.Photo),
			Permissions:         ParseChatPermissions(r.DefaultBannedRights),
			AdminRights:         ParseChatAdminRights(r.AdminRights),
			Date:                r.Date,
			Raw:                 r,
		}
		if r.ParticipantsCount != 0 {
			c.MembersCount = int(r.ParticipantsCount)
		}
		return c
	case *tg.ChatForbidden:
		return &Chat{
			ID:       peerid.MarkChannel(r.ID),
			Type:     ChatTypeGroup,
			Title:    r.Title,
			IsBanned: true,
			Raw:      r,
		}
	case *tg.Channel:
		chatType := ChatTypeChannel
		if r.Megagroup {
			chatType = ChatTypeSupergroup
		}
		if r.Forum {
			chatType = ChatTypeForum
		}
		c := &Chat{
			ID:                     peerid.MarkChannel(r.ID),
			Type:                   chatType,
			Title:                  r.Title,
			IsVerified:             r.Verified,
			IsScam:                 r.Scam,
			IsFake:                 r.Fake,
			IsRestricted:           r.Restricted,
			IsCreator:              r.Creator,
			IsLeft:                 r.Left,
			IsAdmin:                r.AdminRights != nil,
			IsForum:                r.Forum,
			IsMin:                  r.Min,
			IsPublic:               r.HasLink,
			IsGigagroup:            r.Gigagroup,
			IsMonoforum:            r.Monoforum,
			IsCallActive:           r.CallActive,
			IsCallNotEmpty:         r.CallNotEmpty,
			IsSlowMode:             r.SlowmodeEnabled,
			IsNoforwards:           r.Noforwards,
			IsJoinToSend:           r.JoinToSend,
			IsJoinRequest:          r.JoinRequest,
			IsStoriesHidden:        r.StoriesHidden,
			IsStoriesUnavailable:   r.StoriesUnavailable,
			SignMessages:           r.Signatures,
			ShowMessageSenderName:  r.SignatureProfiles,
			HasProtectedContent:    r.Noforwards,
			HasAutoTranslation:     r.Autotranslation,
			HasDirectMessagesGroup: r.BroadcastMessagesAllowed,
			HasForumTabs:           r.ForumTabs,
			Photo:                  parseChatPhoto(r.Photo),
			Permissions:            ParseChatPermissions(r.DefaultBannedRights),
			AdminRights:            ParseChatAdminRights(r.AdminRights),
			BannedRights:           ParseChatBannedRights(r.BannedRights),
			Restrictions:           parseRestrictions(r.RestrictionReason),
			Usernames:              parseUsernames(r.Usernames),
			EmojiStatus:            ParseEmojiStatus(r.EmojiStatus),
			ReplyColor:             ParseChatColorFromPeer(r.Color),
			ProfileColor:           ParseChatColorFromPeer(r.ProfileColor),
			Date:                   r.Date,
			Level:                  r.Level,
			SubscriptionUntilDate:  time.Unix(int64(r.SubscriptionUntilDate), 0),
			BotVerificationIcon:    r.BotVerificationIcon,
			PaidMessageStarCount:   r.SendPaidMessagesStars,
			Raw:                    r,
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
			ID:       peerid.MarkChannel(r.ID),
			Type:     chatType,
			Title:    r.Title,
			IsBanned: true,
			Raw:      r,
		}
	}
	return nil
}

// ParseChatFromPeer resolves a PeerClass using the PeerMap and returns a Chat.
// Falls back to a minimal Chat with just ID and Type if the peer is not found in the map.
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
		return &Chat{ID: peerid.MarkChannel(p.ChannelID), Type: ChatTypeChannel}
	}
	return nil
}

// ParseChatPreview converts a TL ChatInvite into a ChatPreview showing the chat's
// public metadata without requiring the caller to join. Returns nil if chatInvite is nil.
//
// Example:
//
//	preview := types.ParseChatPreview(invite)
//	fmt.Printf("Preview: %s (%d members, type=%s)\n", preview.Title, preview.MembersCount, preview.Type)
func ParseChatPreview(chatInvite *tg.ChatInvite) *ChatPreview {
	if chatInvite == nil {
		return nil
	}
	c := &ChatPreview{
		Title:      chatInvite.Title,
		IsVerified: chatInvite.Verified,
		IsScam:     chatInvite.Scam,
		IsFake:     chatInvite.Fake,
	}
	if chatInvite.Channel {
		c.Type = ChatTypeChannel
		if chatInvite.Megagroup {
			c.Type = ChatTypeSupergroup
		}
	} else {
		c.Type = ChatTypeGroup
	}
	if chatInvite.ParticipantsCount != 0 {
		c.MembersCount = int(chatInvite.ParticipantsCount)
	}
	return c
}

// EnrichChatFull merges fields from a Full TL object (ChatFull, ChannelFull, or UserFull)
// into an existing Chat. Does nothing if either argument is nil.
func EnrichChatFull(c *Chat, full any) {
	if c == nil || full == nil {
		return
	}
	switch f := full.(type) {
	case *tg.ChatFull:
		enrichChatFromChatFull(c, f)
	case *tg.ChannelFull:
		enrichChatFromChannelFull(c, f)
	case *tg.UserFull:
		enrichChatFromUserFull(c, f)
	}
}

func enrichChatFromChatFull(c *Chat, f *tg.ChatFull) {
	if f.About != "" {
		c.Description = f.About
	}
	c.CanSetUsername = f.CanSetUsername
	c.CanScheduleMessages = f.HasScheduled
	c.IsTranslationsDisabled = f.TranslationsDisabled
	c.FolderID = f.FolderID
	c.MessageAutoDelete = f.TTLPeriod
	c.PinnedMessageID = f.PinnedMsgID
	c.ThemeEmoticon = f.ThemeEmoticon
	c.JoinRequestsCount = f.RequestsPending
	c.AvailableReactions = f.AvailableReactions
	c.ReactionsLimit = f.ReactionsLimit
	if f.ExportedInvite != nil {
		if ei, ok := f.ExportedInvite.(*tg.ChatInviteExported); ok {
			c.InviteLink = ei.Link
		}
	}
}

func enrichChatFromChannelFull(c *Chat, f *tg.ChannelFull) {
	if f.About != "" {
		c.Description = f.About
	}
	c.CanViewParticipants = f.CanViewParticipants
	c.CanSetUsername = f.CanSetUsername
	c.CanSetStickerSet = f.CanSetStickers
	c.HasVisibleHistory = f.HiddenPrehistory
	c.CanSetLocation = f.CanSetLocation
	c.CanScheduleMessages = f.HasScheduled
	c.CanViewStats = f.CanViewStats
	c.IsBlocked = f.Blocked
	c.CanDeleteChannel = f.CanDeleteChannel
	c.HasAggressiveAntiSpam = f.Antispam
	c.IsMembersHidden = f.ParticipantsHidden
	c.IsTranslationsDisabled = f.TranslationsDisabled
	c.IsPinnedStoriesAvailable = f.StoriesPinnedAvailable
	c.IsAdsRestricted = f.RestrictedSponsored
	c.CanViewRevenue = f.CanViewRevenue
	c.CanSendPaidMedia = f.PaidMediaAllowed
	c.CanViewStarsRevenue = f.CanViewStarsRevenue
	c.IsPaidReactionsAvailable = f.PaidReactionsAvailable
	c.CanSendGift = f.StargiftsAvailable
	c.IsPaidMessagesAvailable = f.PaidMessagesAvailable
	c.FolderID = f.FolderID
	c.MessageAutoDelete = f.TTLPeriod
	c.PinnedMessageID = f.PinnedMsgID
	c.MigratedFromChatID = f.MigratedFromChatID
	c.MigratedFromMaxID = f.MigratedFromMaxID
	c.AvailableMinID = f.AvailableMinID
	c.LinkedChatID = f.LinkedChatID
	c.SlowModeDelay = f.SlowmodeSeconds
	c.SlowmodeNextSendDate = time.Unix(int64(f.SlowmodeNextSendDate), 0)
	c.StatsDC = f.StatsDC
	c.ThemeEmoticon = f.ThemeEmoticon
	c.JoinRequestsCount = f.RequestsPending
	c.AvailableReactions = f.AvailableReactions
	c.ReactionsLimit = f.ReactionsLimit
	c.BoostsApplied = f.BoostsApplied
	c.BoostsUnrestrict = f.BoostsUnrestrict
	c.GiftCount = f.StargiftsCount
	c.PaidMessageStarCount = f.SendPaidMessagesStars
	c.UnreadCount = f.UnreadCount
	c.ReadInboxMaxID = f.ReadInboxMaxID
	c.ReadOutboxMaxID = f.ReadOutboxMaxID
	c.OnlineCount = f.OnlineCount
	c.AdminsCount = f.AdminsCount
	c.KickedCount = f.KickedCount
	c.BannedCount = f.BannedCount
	if f.ParticipantsCount != 0 {
		c.MembersCount = int(f.ParticipantsCount)
	}
	if f.BotVerification != nil {
		c.BotVerification = &BotVerification{
			CustomEmojiID: fmt.Sprintf("%d", f.BotVerification.Icon),
			Description:   textWithEntitiesText(f.BotVerification.Description),
		}
	}
	if f.ExportedInvite != nil {
		if ei, ok := f.ExportedInvite.(*tg.ChatInviteExported); ok {
			c.InviteLink = ei.Link
		}
	}
	if f.Stickerset != nil {
		if ss, ok := f.Stickerset.(*tg.StickerSet); ok {
			c.StickerSetName = ss.ShortName
		}
	}
	if f.Emojiset != nil {
		if es, ok := f.Emojiset.(*tg.StickerSet); ok {
			c.CustomEmojiStickerSet = es.ShortName
		}
	}
}

func enrichChatFromUserFull(c *Chat, f *tg.UserFull) {
	c.IsBlocked = f.Blocked
	c.IsPhoneCallsAvailable = f.PhoneCallsAvailable
	c.IsPhoneCallsPrivate = f.PhoneCallsPrivate
	c.IsVideoCallsAvailable = f.VideoCallsAvailable
	c.IsWallpaperOverridden = f.WallpaperOverridden
	c.IsTranslationsDisabled = f.TranslationsDisabled
	c.IsPinnedStoriesAvailable = f.StoriesPinnedAvailable
	c.IsBlockedMyStoriesFrom = f.BlockedMyStoriesFrom
	c.IsReadDatesAvailable = !f.ReadDatesPrivate
	c.IsAdsEnabled = f.SponsoredEnabled
	c.CanPinMessage = f.CanPinMessage
	c.CanScheduleMessages = f.HasScheduled
	c.CanSendVoiceMessages = !f.VoiceMessagesForbidden
	c.CanViewRevenue = f.CanViewRevenue
	c.BotCanManageEmojiStatus = f.BotCanManageEmojiStatus
	c.DisplayGiftsButton = f.DisplayGiftsButton
	c.UsesUnofficialApp = f.UnofficialSecurityRisk
	c.CommonChats = f.CommonChatsCount
	c.FolderID = f.FolderID
	c.MessageAutoDelete = f.TTLPeriod
	c.GiftCount = f.StargiftsCount
	c.PinnedMessageID = f.PinnedMsgID
	c.PersonalChannelID = f.PersonalChannelID
	c.PersonalChannelMsg = f.PersonalChannelMessage
	c.PaidMessageStarCount = f.SendPaidMessagesStars
	c.BotGroupRights = ParseChatAdminRights(f.BotGroupAdminRights)
	c.BotBroadcastRights = ParseChatAdminRights(f.BotBroadcastAdminRights)
	if f.About != "" {
		c.Bio = f.About
	}
	if f.PrivateForwardName != "" {
		c.PrivateForwardName = f.PrivateForwardName
	}
	if f.Birthday != nil {
		c.Birthday = ParseBirthday(f.Birthday)
	}
	if f.BotVerification != nil {
		c.BotVerification = &BotVerification{
			CustomEmojiID: fmt.Sprintf("%d", f.BotVerification.Icon),
			Description:   textWithEntitiesText(f.BotVerification.Description),
		}
	}
}
