package types

import "github.com/mtgo-labs/mtgo/tg"

// ChatEvent represents a single event in the chat admin event log. Each event
// records an administrative action (title change, member promotion, settings
// update, etc.) performed in the chat.
type ChatEvent struct {
	// ID is the unique event identifier.
	ID int64
	// Date is the Unix timestamp when the event occurred.
	Date int32
	// UserID is the ID of the admin who performed the action.
	UserID int64
	// Action describes the type and details of the event.
	Action ChatEventAction
}

// ParseChatEvent converts an MTProto ChannelAdminLogEvent to a ChatEvent.
// Returns nil if raw is nil.
func ParseChatEvent(raw *tg.ChannelAdminLogEvent) *ChatEvent {
	if raw == nil {
		return nil
	}
	return &ChatEvent{
		ID:     raw.ID,
		Date:   raw.Date,
		UserID: raw.UserID,
		Action: ChatEventAction("event"),
	}
}
