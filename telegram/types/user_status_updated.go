package types

// UserStatusUpdated represents a real-time notification that a user's online
// presence status has changed. Delivered as an update to subscribed clients.
type UserStatusUpdated struct {
	// UserID is the Telegram user ID whose status changed.
	UserID int64
	// Status is the new presence state (online, offline, recently, etc.).
	Status UserStatus
}
