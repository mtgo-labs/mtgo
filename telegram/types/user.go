package types

import (
	"errors"
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// User represents a Telegram user with their profile information and flags.
// Constructed from MTProto TL user objects via ParseUser.
//
// Example:
//
//	user := types.ParseUser(userTL)
//	fmt.Printf("%s (@%s) — Premium: %v\n", user.FirstName, user.Username, user.IsPremium)
type User struct {
	// ID is the unique Telegram user identifier.
	ID int64
	// IsSelf is true when this user represents the currently logged-in account.
	IsSelf bool
	// IsContact is true when the user is in the current user's contact list.
	IsContact bool
	// IsMutualContact is true when both users have each other in their contact
	// lists.
	IsMutualContact bool
	// IsDeleted is true when the user's account has been deleted.
	IsDeleted bool
	// IsBot is true when this user is a bot.
	IsBot bool
	// IsVerified is true when the user has a verified badge.
	IsVerified bool
	// IsRestricted is true when the user is restricted by Telegram (e.g. spam
	// reports).
	IsRestricted bool
	// IsScam is true when the user has been flagged as a scam.
	IsScam bool
	// IsFake is true when the user is a fake user created by Telegram for
	// internal purposes.
	IsFake bool
	// IsPremium is true when the user has Telegram Premium.
	IsPremium bool
	// IsSupport is true when the user is an official Telegram support account.
	IsSupport bool
	// FirstName is the user's first name.
	FirstName string
	// LastName is the user's last name, which may be empty.
	LastName string
	// Username is the user's public username without the "@" prefix, or empty.
	Username string
	// Phone is the user's phone number (visible only to mutual contacts), or
	// empty.
	Phone string
	// Language is the IETF language tag of the user's client, or empty.
	Language string
	// Status indicates the user's current online/offline presence.
	Status UserStatus
	// Photo contains the user's profile photo variants, or nil if no photo is
	// set.
	Photo *ChatPhoto
	// BotInfoVersion is the version number of the bot's info, incremented each
	// time the bot updates its profile. 0 for non-bots.
	BotInfoVersion int32
	// AccessHash is needed to access the user when they are not cached locally.
	AccessHash int64
	binder     Binder
}

var ErrNoUserBinder = errors.New("types: bound methods not available (user was not created by a client)")

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

// String returns a human-readable identifier for the user. It prefers the
// username, falls back to "FirstName LastName", and finally to "user_<ID>".
//
// Example:
//
//	fmt.Printf("Sender: %s\n", user.String())
//	// Output: Sender: john_doe
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

// MentionName returns the user's mention handle. Prefers "@username" and
// falls back to the display name returned by String.
//
// Example:
//
//	fmt.Println(user.MentionName())
//	// Output: @john_doe
func (u *User) MentionName() string {
	if u.Username != "" {
		return "@" + u.Username
	}
	return u.String()
}

// ParseUser converts a TL user object into a User. Returns nil if raw is nil.
//
// Example:
//
//	user := types.ParseUser(userTL)
//	if user != nil && user.IsBot {
//	    fmt.Println("This is a bot:", user.FirstName)
//	}
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
		ID:              raw.ID,
		IsSelf:          raw.Self,
		IsContact:       raw.Contact,
		IsMutualContact: raw.MutualContact,
		IsDeleted:       raw.Deleted,
		IsBot:           raw.Bot,
		IsVerified:      raw.Verified,
		IsRestricted:    raw.Restricted,
		IsScam:          raw.Scam,
		IsFake:          raw.Fake,
		IsPremium:       raw.Premium,
		IsSupport:       raw.Support,
		Photo:           parseUserProfilePhoto(raw.Photo),
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
	if raw.BotInfoVersion != 0 {
		u.BotInfoVersion = raw.BotInfoVersion
	}
	if raw.AccessHash != 0 {
		u.AccessHash = raw.AccessHash
	}
	u.Status = parseUserStatus(raw.Status)
	return u
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
