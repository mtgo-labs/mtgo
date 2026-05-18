package telegram

import (
	"context"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg/e2e"
)

// Context represents the environment surrounding a single Telegram update event.
// It carries the originating update, the client that received it, any parsed
// high-level event payloads, and lifecycle flags. Handler functions receive a
// Context so they can inspect the update, call client methods, and control
// middleware chain propagation.
//
// Example:
//
//	func myHandler(ctx *telegram.Context) {
//	    if ctx.Message != nil {
//	        fmt.Printf("message from chat %d: %s\n", ctx.Message.ChatID, ctx.Message.Text)
//	    }
//	}
type Context struct {
	// Ctx is the standard Go context propagated through the handler chain.
	// Use it for deadlines, cancellation, and request-scoped values.
	Ctx context.Context

	// Client is the MTProto client that received the update.
	// Use it to make API calls in response to the event.
	Client *Client

	// Update is the raw update object dispatched from the Telegram update
	// loop. It contains the full Users and Chat maps for peer resolution.
	// May be nil if the Context was created outside of an update handler.
	Update *Update

	// PluginData stores per-request plugin state. Plugins use this to attach
	// data (like locale, translator) to the current update context.
	PluginData map[string]interface{}

	// Message is populated when the update contains a new incoming message
	// from a user or group. Nil for all other update types.
	Message *types.Message

	// EditedMessage is populated when the update contains a message that was
	// edited by its sender. Nil for all other update types.
	EditedMessage *types.Message

	// BusinessMessage is populated when the update contains a new incoming
	// message sent through a business connection on behalf of a business.
	// Nil for all other update types.
	BusinessMessage *types.Message

	// EditedBusinessMessage is populated when the update contains a message
	// that was edited through a business connection. Nil for all other
	// update types.
	EditedBusinessMessage *types.Message

	// DeletedMessages is populated when one or more messages were deleted in
	// a chat. Nil for all other update types.
	DeletedMessages *types.DeletedMessages

	// DeletedBusinessMessages is populated when messages were deleted that
	// were originally sent through a business connection. Nil for all other
	// update types.
	DeletedBusinessMessages *types.DeletedMessages

	// CallbackQuery is populated when a user presses an inline callback
	// button. Use it to answer the query and identify the originating chat
	// and message. Nil for all other update types.
	CallbackQuery *types.CallbackQuery

	// InlineQuery is populated when a user types a query in an inline mode
	// input field. Use it to return inline results. Nil for all other update
	// types.
	InlineQuery *types.InlineQuery

	// ChosenInlineResult is populated when a user selects a result from an
	// inline query. Useful for tracking which inline result was picked. Nil
	// for all other update types.
	ChosenInlineResult *types.ChosenInlineResult

	// UserStatus is populated when a user's online status changes (going
	// online, offline, or updating their last-seen timestamp). Nil for all
	// other update types.
	UserStatus *types.UserStatusUpdated

	// ChatMember is populated when a chat member's status is updated (e.g.
	// joined, left, kicked, promoted). Nil for all other update types.
	ChatMember *types.ChatMemberUpdated

	// MessageReaction is populated when a user adds or changes a reaction on
	// a message. Nil for all other update types.
	MessageReaction *types.MessageReactions

	// MessageReactionCount is populated when the aggregate reaction count on
	// a message changes (e.g. anonymous reactions). Nil for all other update
	// types.
	MessageReactionCount *types.MessageReactions

	// Poll is populated when a poll state is updated (new vote or poll
	// closure). Nil for all other update types.
	Poll *types.PollUpdate

	// BusinessConnection is populated when a business connection is
	// established or its settings change. Nil for all other update types.
	BusinessConnection *types.BusinessConnection

	// Story is populated when a story is posted or edited. Nil for all other
	// update types.
	Story *types.Story

	// ChatBoost is populated when a boost is applied or removed from a chat.
	// Nil for all other update types.
	ChatBoost *types.ChatBoostUpdated

	// ChatJoinRequest is populated when a user requests to join a chat that
	// requires approval. Nil for all other update types.
	ChatJoinRequest *types.ChatJoinRequest

	// PreCheckoutQuery is populated during a payment flow when the user
	// confirms checkout. Respond with answerPreCheckoutQuery to approve or
	// reject. Nil for all other update types.
	PreCheckoutQuery *types.PreCheckoutQuery

	// ShippingQuery is populated during a payment flow when the user
	// selects a shipping option. Respond with answerShippingQuery. Nil for
	// all other update types.
	ShippingQuery *types.ShippingQuery

	// PurchasedPaidMedia is populated when a user purchases paid media in a
	// chat. Nil for all other update types.
	PurchasedPaidMedia *types.PurchasedPaidMedia

	// ManagedBot is populated when a managed bot's connection state or
	// settings are updated. Nil for all other update types.
	ManagedBot *types.ManagedBotUpdated

	// SecretChat is populated when an encryption update is received
	// (new secret chat request, accepted, or discarded). Nil for all other
	// update types.
	SecretChat *SecretChat

	// SecretMessage is populated when a decrypted secret chat message is
	// received. Nil for all other update types.
	SecretMessage *e2e.DecryptedMessageLayer

	// Error is set when the update loop encounters an error during
	// processing. Handlers can inspect this to perform error logging or
	// recovery before the update is discarded.
	Error error

	// Stopped indicates whether handler chain propagation has been halted.
	// Set to true by StopPropagation; middleware should check this before
	// continuing to the next handler.
	Stopped bool

	// Connected is set to true when the client has successfully connected to
	// the Telegram server. Use it in connection-lifecycle handlers to
	// trigger initialization logic that should run on reconnect.
	Connected bool

	// Disconnected is set to true when the client loses its connection to
	// the Telegram server. Use it in connection-lifecycle handlers to pause
	// outgoing work or notify monitoring.
	Disconnected bool

	// Started is set to true once the client has completed its initial setup
	// (e.g. first connection established). Use it to run one-time bootstrap
	// logic in a start handler.
	Started bool
}

// NewContext creates a new Context bound to the receiving client.
// If ctx is nil, context.Background is used as the base. Call this when
// you need a Context outside of the normal update dispatch path, for
// example in background goroutines or scheduled tasks.
//
// Example:
//
//	bgCtx := context.Background()
//	ctx := client.NewContext(bgCtx)
//	peer, err := ctx.Client.ResolvePeer(bgCtx, "@durov")
func (c *Client) NewContext(ctx context.Context) *Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Context{Client: c, Ctx: ctx}
}

// StopPropagation prevents subsequent handlers in the middleware chain from
// receiving this update. Call it when a handler has fully consumed the event
// and no further processing is needed (e.g. after sending a reply to a
// command that should not be handled by any other middleware).
//
// Example:
//
//	func startHandler(ctx *telegram.Context) {
//	    if ctx.Message != nil && ctx.Message.Text == "/start" {
//	        // send welcome message...
//	        ctx.StopPropagation()
//	    }
//	}
func (c *Context) StopPropagation() {
	c.Stopped = true
}

// ResolvePeer looks up a user or chat by its numeric ID in the current
// update's peer cache. Returns the matching User or Chat object on success,
// or nil when no peer with the given ID is available. Useful for resolving
// forward-from or reply-to senders without an additional API call.
func (c *Context) ResolvePeer(id int64) interface{} {
	if c.Update == nil {
		return nil
	}
	if u, ok := c.Update.Users[id]; ok {
		return u
	}
	if ch, ok := c.Update.Chats[id]; ok {
		return ch
	}
	return nil
}

func (c *Context) chatID() (int64, error) {
	if c.Message != nil && c.Message.ChatID != 0 {
		return c.Message.ChatID, nil
	}
	if c.EditedMessage != nil && c.EditedMessage.ChatID != 0 {
		return c.EditedMessage.ChatID, nil
	}
	if c.BusinessMessage != nil && c.BusinessMessage.ChatID != 0 {
		return c.BusinessMessage.ChatID, nil
	}
	if c.EditedBusinessMessage != nil && c.EditedBusinessMessage.ChatID != 0 {
		return c.EditedBusinessMessage.ChatID, nil
	}
	if c.CallbackQuery != nil && c.CallbackQuery.ChatID != 0 {
		return c.CallbackQuery.ChatID, nil
	}
	return 0, ErrContextNoChat
}

func (c *Context) messageID() (int32, error) {
	if c.Message != nil {
		return c.Message.ID, nil
	}
	if c.EditedMessage != nil {
		return c.EditedMessage.ID, nil
	}
	if c.BusinessMessage != nil {
		return c.BusinessMessage.ID, nil
	}
	if c.EditedBusinessMessage != nil {
		return c.EditedBusinessMessage.ID, nil
	}
	return 0, ErrContextNoMessage
}

// TranslatorFunc is the signature for a localization translation function.
// It accepts a translation key and optional format arguments, returning the
// localized string. Register a TranslatorFunc in PluginData to enable i18n
// in handler code via the Context.T helper method.
//
// Example:
//
//	translate := func(key string, args ...any) string {
//	    messages := map[string]string{"welcome": "Welcome, %s!"}
//	    return fmt.Sprintf(messages[key], args...)
//	}
//	ctx.PluginData["_t"] = telegram.TranslatorFunc(translate)
//	fmt.Println(ctx.T("welcome", "Alice")) // "Welcome, Alice!"
type TranslatorFunc = types.TranslatorFunc

func (c *Context) T(key string, args ...any) string {
	if c.PluginData != nil {
		if fn, ok := c.PluginData["_t"]; ok {
			if tf, ok := fn.(types.TranslatorFunc); ok {
				return tf(key, args...)
			}
		}
	}
	if c.Message != nil {
		return c.Message.T(key, args...)
	}
	return key
}

func (c *Context) Sender() *types.User {
	if c.Message != nil && c.Message.Sender != nil {
		return c.Message.Sender
	}
	if c.EditedMessage != nil && c.EditedMessage.Sender != nil {
		return c.EditedMessage.Sender
	}
	if c.CallbackQuery != nil {
		if c.Update != nil {
			if u, ok := c.Update.Users[c.CallbackQuery.UserID]; ok {
				return u
			}
		}
	}
	return nil
}
