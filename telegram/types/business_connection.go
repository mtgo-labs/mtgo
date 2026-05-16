package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// BusinessConnection represents an active connection between a Telegram business
// account and a bot.
type BusinessConnection struct {
	ID        string
	User      *User
	DC        int32
	Date      time.Time
	IsEnabled bool
	Rights    *BusinessBotRights
}

// ParseBusinessConnection converts a TL BotBusinessConnection into a BusinessConnection.
// Returns nil if raw is nil.
//
// Example:
//
//	conn := types.ParseBusinessConnection(rawConn, user)
//	fmt.Printf("Connection %s enabled=%v DC=%d\n", conn.ID, conn.IsEnabled, conn.DC)
func ParseBusinessConnection(raw *tg.BotBusinessConnection, u *User) *BusinessConnection {
	if raw == nil {
		return nil
	}
	return &BusinessConnection{
		ID:        raw.ConnectionID,
		User:      u,
		DC:        raw.DCID,
		Date:      time.Unix(int64(raw.Date), 0),
		IsEnabled: !raw.Disabled,
		Rights:    ParseBusinessBotRights(raw.Rights),
	}
}
