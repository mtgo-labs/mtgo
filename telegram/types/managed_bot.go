package types

// ManagedBotUpdated is sent when a managed bot's configuration changes,
// such as when the bot token is rotated or the bot is reassigned.
type ManagedBotUpdated struct {
	User         *User
	Bot          *User
	BotID        int64
	OwnerID      int64
	TokenUpdated bool
}
