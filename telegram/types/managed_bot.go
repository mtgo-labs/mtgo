package types

// ManagedBotUpdated is sent when a managed bot's configuration changes,
// such as when the bot token is rotated or the bot is reassigned.
type ManagedBotUpdated struct {
	// BotID is the Telegram user ID of the managed bot.
	BotID int64
	// OwnerID is the Telegram user ID of the bot's owner or manager.
	OwnerID int64
	// TokenUpdated indicates whether the bot's API token was rotated as part of this update.
	TokenUpdated bool
}
