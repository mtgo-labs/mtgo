package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// BusinessConnection represents an active connection between a Telegram business
// account and a bot. Used to track which chats the bot can act on behalf of.
//
// Example:
//
//	bc := types.ParseBotBusinessConnection(rawConnection)
//	if bc != nil {
//	    fmt.Printf("Connection %s — canReply: %v\n", bc.ID, bc.CanReply)
//	}
type BusinessConnection struct {
	// ID is the unique identifier of the business connection.
	ID string
	// UserID is the Telegram user ID of the business account owner.
	UserID int64
	// DcID is the data center ID where the business connection is stored.
	DcID int32
	// Date is the Unix timestamp when the connection was established.
	Date int32
	// CanReply indicates whether the bot is allowed to send messages in this connection.
	CanReply bool
	// Disabled indicates whether the business connection has been deactivated.
	Disabled bool
	// Rights holds the granular permissions the business has granted to the bot.
	Rights *BusinessBotRights
}

// ParseBotBusinessConnection converts an MTProto BotBusinessConnection into a BusinessConnection.
// Returns nil if raw is nil.
func ParseBotBusinessConnection(raw *tg.BotBusinessConnection) *BusinessConnection {
	if raw == nil {
		return nil
	}
	bc := &BusinessConnection{
		ID:         raw.ConnectionID,
		UserID:     raw.UserID,
		DcID:       raw.DCID,
		Date:       raw.Date,
		Disabled:   raw.Disabled,
		CanReply:   raw.Rights != nil && raw.Rights.Reply,
		Rights:     ParseBusinessBotRights(raw.Rights),
	}
	return bc
}
