package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// Birthday represents a user's birthday date. Used for profile display and
// birthday notifications.
type Birthday struct {
	// Day is the day of the month (1–31).
	Day int32
	// Month is the month of the year (1–12).
	Month int32
	// Year is the birth year, or zero if the user chose not to disclose it.
	Year int32
	// Has indicates whether the user has set a birthday on their profile.
	Has bool
}

// UsernameInfo represents a Telegram username with its activation and editability state.
// Users may have multiple usernames (a primary active one and inactive alternatives).
type UsernameInfo struct {
	// Username is the Telegram @username without the @ prefix.
	Username string
	// Active indicates whether this username is currently active and resolvable.
	Active bool
	// Editable indicates whether the user can change this username.
	Editable bool
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
		Has:   true,
	}
	if raw.Year != 0 {
		b.Year = raw.Year
	}
	return b
}

// ParseUsername converts an MTProto Username into a UsernameInfo.
// Returns nil if raw is nil.
func ParseUsername(raw *tg.Username) *UsernameInfo {
	if raw == nil {
		return nil
	}
	return &UsernameInfo{
		Username: raw.Username,
		Active:   raw.Active,
		Editable: raw.Editable,
	}
}
