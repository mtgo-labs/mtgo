package types

// ChatBoostUpdated represents a boost applied to a chat. Boosts increase a
// channel's or group's feature levels (e.g. more story posting quota, custom
// colors).
type ChatBoostUpdated struct {
	// ChatID is the ID of the chat that received the boost.
	ChatID int64
	// UserID is the ID of the user who applied the boost.
	UserID int64
	// BoostID is the unique identifier of this boost instance.
	BoostID string
	// Date is the Unix timestamp when the boost was applied.
	Date int32
	// Expires is the Unix timestamp when the boost expires.
	Expires int32
	// Multiplier is the number of boost slots used (users with Premium can apply
	// multiple boosts).
	Multiplier int
	// Stars is the number of Telegram Stars contributed by this boost, or 0 for
	// regular boosts.
	Stars int64
}
