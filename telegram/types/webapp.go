package types

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrWebAppDataInvalid  = errors.New("webapp: invalid init data")
	ErrWebAppDataOutdated = errors.New("webapp: init data is outdated")
	ErrWebAppDataMismatch = errors.New("webapp: hash mismatch")
)

type WebAppInitUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	IsPremium    bool   `json:"is_premium"`
	PhotoURL     string `json:"photo_url"`
}

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

func ValidateWebAppInitDataAge(data *WebAppInitData, maxAge time.Duration) error {
	if time.Since(data.AuthDate) > maxAge {
		return fmt.Errorf("%w: auth_date %d, max age %s", ErrWebAppDataOutdated, data.AuthDate.Unix(), maxAge)
	}
	return nil
}
