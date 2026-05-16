package types

import (
	"errors"
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// User represents a Telegram user with all fields from both the base User and
// UserFull TL objects. Base fields are populated by ParseUser; Full fields are
// merged in by EnrichUserFull.
//
// When created by a Client, the User carries a Binder enabling bound methods
// like Block, Unblock, and GetCommonChats.
type User struct {
	ID int64

	IsSelf                   bool
	IsContact                bool
	IsMutualContact          bool
	IsDeleted                bool
	IsBot                    bool
	IsVerified               bool
	IsRestricted             bool
	IsScam                   bool
	IsFake                   bool
	IsPremium                bool
	IsSupport                bool
	IsMin                    bool
	IsContactRequirePremium  bool
	IsCloseFriend            bool
	IsStoriesHidden          bool
	IsStoriesUnavailable     bool
	IsAddedToAttachmentMenu  bool
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
	UsesUnofficialApp        bool

	CanBeEdited                bool
	CanJoinGroups              bool
	CanReadAllGroupMessages    bool
	CanConnectToBusiness       bool
	CanManageBots              bool
	HasMainWebApp              bool
	HasTopics                  bool
	AllowsUsersToCreateTopics  bool
	InlineNeedLocation         bool
	SupportsGuestQueries       bool
	CanPinMessage              bool
	CanScheduleMessages        bool
	CanSendVoiceMessages       bool
	CanViewRevenue             bool
	BotCanManageEmojiStatus    bool
	DisplayGiftsButton         bool
	CanBeAddedToAttachmentMenu bool

	FirstName              string
	LastName               string
	Username               string
	Phone                  string
	Language               string
	Bio                    string
	PrivateForwardName     string
	InlineQueryPlaceholder string

	Status UserStatus
	Photo  *ChatPhoto
	Raw    *tg.User

	EmojiStatus     *EmojiStatus
	ReplyColor      *ChatColor
	ProfileColor    *ChatColor
	Restrictions    []*Restriction
	Usernames       []*Username
	BotVerification *BotVerification
	Birthday        *Birthday

	BotInfoVersion       int32
	BotActiveUsers       int32
	AccessHash           int64
	BotVerificationIcon  int64
	PaidMessageStarCount int64
	CommonChats          int32
	FolderID             int32
	MessageAutoDelete    int32
	GiftCount            int32
	PinnedMessageID      int32
	PersonalChannelID    int64
	PersonalChannelMsg   int32

	binder Binder
}

// ErrNoUserBinder is returned when a bound convenience method (Archive,
// Unarchive, etc.) is called on a User that was not created by a client.
//
// Example:
//
//	err := user.Archive()
//	if errors.Is(err, types.ErrNoUserBinder) {
//		log.Println("user was not created by a client")
//	}
var ErrNoUserBinder = errors.New("types: bound methods not available (user was not created by a client)")

// SetBinder injects the Binder that backs bound convenience methods on this User.
func (u *User) SetBinder(b Binder) {
	u.binder = b
}

func (u *User) Archive() error {
	if u.binder == nil {
		return ErrNoUserBinder
	}
	return u.binder.BoundArchiveUser(u.ID)
}

func (u *User) Unarchive() error {
	if u.binder == nil {
		return ErrNoUserBinder
	}
	return u.binder.BoundUnarchiveUser(u.ID)
}

func (u *User) Block() error {
	if u.binder == nil {
		return ErrNoUserBinder
	}
	return u.binder.BoundBlock(u.ID)
}

func (u *User) Unblock() error {
	if u.binder == nil {
		return ErrNoUserBinder
	}
	return u.binder.BoundUnblock(u.ID)
}

func (u *User) GetCommonChats(limit int) ([]*Chat, error) {
	if u.binder == nil {
		return nil, ErrNoUserBinder
	}
	return u.binder.BoundGetCommonChats(u.ID, limit)
}

// String returns the user's username, full name, or a fallback "user_<id>" string.
func (u *User) String() string {
	if u.Username != "" {
		return u.Username
	}
	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}
	if name == "" {
		return fmt.Sprintf("user_%d", u.ID)
	}
	return name
}

// MentionName returns "@username" if set, otherwise falls back to String().
func (u *User) MentionName() string {
	if u.Username != "" {
		return "@" + u.Username
	}
	return u.String()
}

// ParseUser converts an MTProto UserClass into a User. Returns nil if raw is nil.
func ParseUser(raw tg.UserClass) *User {
	if raw == nil {
		return nil
	}
	switch r := raw.(type) {
	case *tg.UserEmpty:
		return &User{ID: r.ID}
	case *tg.User:
		return parseUserTL(r)
	}
	return nil
}

func parseUserTL(raw *tg.User) *User {
	u := &User{
		ID:                         raw.ID,
		IsSelf:                     raw.Self,
		IsContact:                  raw.Contact,
		IsMutualContact:            raw.MutualContact,
		IsDeleted:                  raw.Deleted,
		IsBot:                      raw.Bot,
		IsVerified:                 raw.Verified,
		IsRestricted:               raw.Restricted,
		IsScam:                     raw.Scam,
		IsFake:                     raw.Fake,
		IsPremium:                  raw.Premium,
		IsSupport:                  raw.Support,
		IsMin:                      raw.Min,
		IsContactRequirePremium:    raw.ContactRequirePremium,
		IsCloseFriend:              raw.CloseFriend,
		IsStoriesHidden:            raw.StoriesHidden,
		IsStoriesUnavailable:       raw.StoriesUnavailable,
		IsAddedToAttachmentMenu:    raw.AttachMenuEnabled,
		CanBeEdited:                raw.BotCanEdit,
		CanJoinGroups:              !raw.BotNochats,
		CanReadAllGroupMessages:    raw.BotChatHistory,
		CanConnectToBusiness:       raw.BotBusiness,
		CanManageBots:              raw.BotCanManageBots,
		HasMainWebApp:              raw.BotHasMainApp,
		HasTopics:                  raw.BotForumView,
		AllowsUsersToCreateTopics:  raw.BotForumCanManageTopics,
		InlineNeedLocation:         raw.BotInlineGeo,
		SupportsGuestQueries:       raw.BotGuestchat,
		CanBeAddedToAttachmentMenu: raw.BotAttachMenu,
		Photo:                      parseUserProfilePhoto(raw.Photo),
		EmojiStatus:                ParseEmojiStatus(raw.EmojiStatus),
		ReplyColor:                 ParseChatColorFromPeer(raw.Color),
		ProfileColor:               ParseChatColorFromPeer(raw.ProfileColor),
		Restrictions:               parseRestrictions(raw.RestrictionReason),
		Usernames:                  parseUsernames(raw.Usernames),
		BotInfoVersion:             raw.BotInfoVersion,
		BotActiveUsers:             raw.BotActiveUsers,
		AccessHash:                 raw.AccessHash,
		BotVerificationIcon:        raw.BotVerificationIcon,
		PaidMessageStarCount:       raw.SendPaidMessagesStars,
		Raw:                        raw,
	}
	if raw.FirstName != "" {
		u.FirstName = raw.FirstName
	}
	if raw.LastName != "" {
		u.LastName = raw.LastName
	}
	if raw.Username != "" {
		u.Username = raw.Username
	}
	if raw.Phone != "" {
		u.Phone = raw.Phone
	}
	if raw.LangCode != "" {
		u.Language = raw.LangCode
	}
	if raw.BotInlinePlaceholder != "" {
		u.InlineQueryPlaceholder = raw.BotInlinePlaceholder
	}
	u.Status = parseUserStatus(raw.Status)
	return u
}

// EnrichUserFull merges fields from a UserFull TL object into an existing User.
// The User must already be populated by ParseUser. Does nothing if either argument is nil.
func EnrichUserFull(u *User, full *tg.UserFull) {
	if u == nil || full == nil {
		return
	}

	u.IsBlocked = full.Blocked
	u.IsPhoneCallsAvailable = full.PhoneCallsAvailable
	u.IsPhoneCallsPrivate = full.PhoneCallsPrivate
	u.IsVideoCallsAvailable = full.VideoCallsAvailable
	u.IsWallpaperOverridden = full.WallpaperOverridden
	u.IsTranslationsDisabled = full.TranslationsDisabled
	u.IsPinnedStoriesAvailable = full.StoriesPinnedAvailable
	u.IsBlockedMyStoriesFrom = full.BlockedMyStoriesFrom
	u.IsReadDatesAvailable = !full.ReadDatesPrivate
	u.IsAdsEnabled = full.SponsoredEnabled
	u.CanPinMessage = full.CanPinMessage
	u.CanScheduleMessages = full.HasScheduled
	u.CanSendVoiceMessages = !full.VoiceMessagesForbidden
	u.CanViewRevenue = full.CanViewRevenue
	u.BotCanManageEmojiStatus = full.BotCanManageEmojiStatus
	u.DisplayGiftsButton = full.DisplayGiftsButton
	u.UsesUnofficialApp = full.UnofficialSecurityRisk

	if full.About != "" {
		u.Bio = full.About
	}
	if full.PrivateForwardName != "" {
		u.PrivateForwardName = full.PrivateForwardName
	}

	u.CommonChats = full.CommonChatsCount
	u.FolderID = full.FolderID
	u.MessageAutoDelete = full.TTLPeriod
	u.GiftCount = full.StargiftsCount
	u.PinnedMessageID = full.PinnedMsgID
	u.PersonalChannelID = full.PersonalChannelID
	u.PersonalChannelMsg = full.PersonalChannelMessage

	if full.Birthday != nil {
		u.Birthday = ParseBirthday(full.Birthday)
	}
	if full.BotVerification != nil {
		u.BotVerification = &BotVerification{
			CustomEmojiID: fmt.Sprintf("%d", full.BotVerification.Icon),
			Description:   full.BotVerification.Description,
		}
	}
	if full.SendPaidMessagesStars != 0 {
		u.PaidMessageStarCount = full.SendPaidMessagesStars
	}
}

func parseUserStatus(status tg.UserStatusClass) UserStatus {
	if status == nil {
		return UserStatusLongAgo
	}
	switch s := status.(type) {
	case *tg.UserStatusEmpty:
		return UserStatusLongAgo
	case *tg.UserStatusOnline:
		if time.Now().Unix() < int64(s.Expires) {
			return UserStatusOnline
		}
		return UserStatusOffline
	case *tg.UserStatusOffline:
		diff := time.Since(time.Unix(int64(s.WasOnline), 0))
		switch {
		case diff < 7*24*time.Hour:
			return UserStatusRecently
		case diff < 30*24*time.Hour:
			return UserStatusLastWeek
		case diff < 365*24*time.Hour:
			return UserStatusLastMonth
		default:
			return UserStatusLongAgo
		}
	case *tg.UserStatusRecently:
		return UserStatusRecently
	case *tg.UserStatusLastWeek:
		return UserStatusLastWeek
	case *tg.UserStatusLastMonth:
		return UserStatusLastMonth
	}
	return UserStatusLongAgo
}

// FullName returns "FirstName LastName" or just "FirstName" if no last name is set.
func (u *User) FullName() string {
	if u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	return u.FirstName
}

// Mention returns an inline mention string for the user in HTML or Markdown format.
// Defaults to HTML. Pass "md" or "markdown" as the optional style argument for Markdown.
func (u *User) Mention(style ...string) string {
	s := "html"
	if len(style) > 0 {
		s = style[0]
	}
	name := u.FullName()
	if name == "" {
		name = "Deleted Account"
	}
	switch s {
	case "md", "markdown":
		return "[" + name + "](tg://user?id=" + fmt.Sprintf("%d", u.ID) + ")"
	default:
		return "<a href=\"tg://user?id=" + fmt.Sprintf("%d", u.ID) + "\">" + name + "</a>"
	}
}

// DC returns the data center ID where the user's profile photo is stored, or 0 if no photo.
func (u *User) DC() int {
	if u.Photo == nil {
		return 0
	}
	return int(u.Photo.DcID)
}

// Verification returns a composite VerificationStatus for the user.
func (u *User) Verification() *VerificationStatus {
	return NewVerificationStatus(u.IsVerified, u.IsScam, u.IsFake, u.BotVerificationIcon)
}

// Birthday represents a user's birthday date. Used for profile display and
// birthday notifications. Presence on a User or Chat indicates the birthday is
// set; a zero Year means the user chose not to disclose it.
type Birthday struct {
	Day   int32
	Month int32
	Year  int32
}

// Username represents a Telegram username with its activation and editability state.
type Username struct {
	Username string
	Active   bool
	Editable bool
}

// ParseUsername converts an MTProto Username into a Username.
// Returns nil if raw is nil.
func ParseUsername(raw *tg.Username) *Username {
	if raw == nil {
		return nil
	}
	return &Username{
		Username: raw.Username,
		Active:   raw.Active,
		Editable: raw.Editable,
	}
}

func parseUsernames(raw []*tg.Username) []*Username {
	if raw == nil {
		return nil
	}
	out := make([]*Username, 0, len(raw))
	for _, u := range raw {
		if v := ParseUsername(u); v != nil {
			out = append(out, v)
		}
	}
	return out
}

// ParseBirthday converts an MTProto Birthday into a Birthday.
// Returns nil if raw is nil.
func ParseBirthday(raw *tg.Birthday) *Birthday {
	if raw == nil {
		return nil
	}
	b := &Birthday{
		Day:   raw.Day,
		Month: raw.Month,
	}
	if raw.Year != 0 {
		b.Year = raw.Year
	}
	return b
}
