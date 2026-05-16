package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tg/e2e"
)

// Update represents a single event dispatched from the Telegram update loop to
// registered handlers. Only one event-specific field is populated per update;
// all others are nil. Lifecycle fields (Connected, Disconnected, Started,
// Stopped) are bool and default to false.
type Update struct {
	// Users holds user objects referenced by the current update, keyed by user ID.
	Users map[int64]*types.User
	// Chats holds chat objects referenced by the current update, keyed by chat ID.
	Chats map[int64]*types.Chat
	// Message is populated when a new text/media message is received.
	Message *types.Message
	// EditedMessage is populated when an existing message is edited by its author.
	EditedMessage *types.Message
	// BusinessMessage is populated for incoming messages on a business connection.
	BusinessMessage *types.Message
	// EditedBusinessMessage is populated when a message on a business connection is edited.
	EditedBusinessMessage *types.Message
	// DeletedMessages is populated when one or more messages are deleted from a chat.
	DeletedMessages *types.DeletedMessages
	// DeletedBusinessMessages is populated when messages are deleted on a business connection.
	DeletedBusinessMessages *types.DeletedMessages
	// CallbackQuery is populated when a user presses an inline callback button.
	CallbackQuery *types.CallbackQuery
	// InlineQuery is populated when a user types a query in an inline mode request.
	InlineQuery *types.InlineQuery
	// ChosenInlineResult is populated when a user selects a result from an inline query.
	ChosenInlineResult *types.ChosenInlineResult
	// UserStatus is populated when a contact's online/offline status changes.
	UserStatus *types.UserStatusUpdated
	// ChatMember is populated when a member's status in a chat changes (e.g. joined, kicked).
	ChatMember *types.ChatMemberUpdated
	// MessageReaction is populated when a user adds or changes a reaction on a message.
	MessageReaction *types.MessageReactions
	// MessageReactionCount is populated when the anonymous reaction count on a message changes.
	MessageReactionCount *types.MessageReactions
	// Poll is populated when a poll's state is updated (new votes, closed, etc.).
	Poll *types.PollUpdate
	// BusinessConnection is populated when a business connection is created or updated.
	BusinessConnection *types.BusinessConnection
	// Story is populated when a story event is received (new, deleted, or updated).
	Story *types.Story
	// ChatBoost is populated when a chat receives or loses a boost.
	ChatBoost *types.ChatBoostUpdated
	// ChatJoinRequest is populated when a user requests to join a chat.
	ChatJoinRequest *types.ChatJoinRequest
	// PreCheckoutQuery is populated during payment flows before the user confirms checkout.
	PreCheckoutQuery *types.PreCheckoutQuery
	// ShippingQuery is populated during payment flows to request shipping options.
	ShippingQuery *types.ShippingQuery
	// PurchasedPaidMedia is populated when a user purchases paid media content.
	PurchasedPaidMedia *types.PurchasedPaidMedia
	// ManagedBot is populated when a managed bot's status changes.
	ManagedBot *types.ManagedBotUpdated
	// Error is populated when the update loop encounters an unrecoverable error.
	Error error
	// Connected is true when the client has successfully established a connection
	// to the Telegram server.
	Connected bool
	// Disconnected is true when the client's connection to the server has been lost.
	Disconnected bool
	// Started is true when the client has completed initialization and the update
	// loop is running.
	Started bool
	// Stopped is true when the client has been fully shut down and the update loop
	// has exited.
	Stopped bool
	// Raw contains the original TL object from Telegram for updates that do not map
	// to a typed field above. Use this for advanced or unsupported update types.
	Raw tg.TLObject
	// SecretChat is populated for UpdateEncryption and UpdateNewEncryptedMessage
	// updates. It carries the secret chat state machine for the relevant chat.
	SecretChat *SecretChat
	// SecretMessage is populated when a new encrypted message is received in a
	// secret chat. It contains the decrypted message layer.
	SecretMessage *e2e.DecryptedMessageLayer
}

func (u *Update) reset() {
	*u = Update{}
}
