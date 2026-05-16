package types

import (
	"errors"
	"fmt"
	"time"
)

// Web App init data validation errors.
//
// These errors are returned by ValidateWebAppInitData and related functions
// when verifying the authenticity and freshness of Telegram Web App init data.
//
// Example:
//
//	data, err := types.ParseWebAppInitData(initDataStr)
//	if err != nil {
//		if errors.Is(err, types.ErrWebAppDataInvalid) {
//			log.Println("init data is malformed")
//		}
//	}
var (
	// ErrWebAppDataInvalid is returned when the init data string cannot be
	// parsed or is missing required fields.
	ErrWebAppDataInvalid = errors.New("webapp: invalid init data")
	// ErrWebAppDataOutdated is returned when the init data's auth_date is
	// older than the maximum allowed age.
	ErrWebAppDataOutdated = errors.New("webapp: init data is outdated")
	// ErrWebAppDataMismatch is returned when the HMAC-SHA256 hash computed
	// from the init data does not match the hash provided in the query string.
	ErrWebAppDataMismatch = errors.New("webapp: hash mismatch")
)

// WebAppInitUser holds the user information embedded in a Telegram Web App init data payload.
type WebAppInitUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	IsPremium    bool   `json:"is_premium"`
	PhotoURL     string `json:"photo_url"`
}

// WebAppInitData represents the parsed init data sent by a Telegram Web App,
// including the query ID, user info, auth date, hash, and raw key-value pairs.
//
// Example:
//
//	data, _ := types.ParseWebAppInitData(initDataStr)
//	fmt.Printf("User: %s (ID: %d), auth_date: %s\n", data.User.FirstName, data.User.ID, data.AuthDate)
type WebAppInitData struct {
	QueryID      string
	User         WebAppInitUser
	AuthDate     time.Time
	Hash         string
	StartParam   string
	ChatInstance string
	ChatType     string
	Raw          map[string]string
}

func (d *WebAppInitData) IsOutdated(maxAge time.Duration) bool {
	return time.Since(d.AuthDate) > maxAge
}

// ValidateWebAppInitDataAge checks that the init data's auth_date is within
// the specified maximum age. Returns ErrWebAppDataOutdated if the data is too old.
//
// Example:
//
//	if err := types.ValidateWebAppInitDataAge(data, 5*time.Minute); err != nil {
//	    log.Println("Init data expired:", err)
//	}
func ValidateWebAppInitDataAge(data *WebAppInitData, maxAge time.Duration) error {
	if time.Since(data.AuthDate) > maxAge {
		return fmt.Errorf("%w: auth_date %d, max age %s", ErrWebAppDataOutdated, data.AuthDate.Unix(), maxAge)
	}
	return nil
}
