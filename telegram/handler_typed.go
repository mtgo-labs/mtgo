package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// typedHandler is a generic handler implementation parameterized by the event
// payload type T. It provides the Check and Handle methods for all typed
// update handlers, eliminating ~2,500 lines of duplicated per-handler code.
//
// Each concrete handler (MessageHandler, CallbackQueryHandler, etc.) is a
// type alias for typedHandler[T] with T set to the appropriate event type.
type typedHandler[T any] struct {
	baseHandler
	callbackCtx    func(*Context)
	callbackClient func(*Client, T)
	callbackFull   func(*Context, T)
	callbackAll    func(*Context, *Client, T)

	// field accessors: isolate the only per-handler difference
	isUpdateNil func(*Update) bool      // true if the relevant Update field is nil
	fromCtx     func(*Context) T        // extract T from Context for Handle dispatch
	setCtx      func(*Context, *Update) // set T on a filter-check Context
}

// newTypedHandler builds a typedHandler[T] with the given field accessors and
// optional filters. The callback must be one of:
//   - func(*Context)
//   - func(*Client, T)
//   - func(*Context, T)
//   - func(*Context, *Client, T)
func newTypedHandler[T any](
	callback any,
	isUpdateNil func(*Update) bool,
	fromCtx func(*Context) T,
	setCtx func(*Context, *Update),
	filters []Filter,
) *typedHandler[T] {
	h := &typedHandler[T]{
		baseHandler: baseHandler{filters: mergeFilters(filters)},
		isUpdateNil: isUpdateNil,
		fromCtx:     fromCtx,
		setCtx:      setCtx,
	}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, T):
		h.callbackClient = fn
	case func(*Context, T):
		h.callbackFull = fn
	case func(*Context, *Client, T):
		h.callbackAll = fn
	}
	return h
}

// Check reports whether the incoming update contains a non-nil payload for
// this handler type and passes the configured filters.
func (h *typedHandler[T]) Check(update *Update) bool {
	if h.isUpdateNil(update) {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update}
	h.setCtx(ctx, update)
	return h.filters(ctx)
}

// Handle dispatches the update payload to whichever callback variant was
// provided at construction time. The all callback is preferred, followed by
// full, client, then context-only.
func (h *typedHandler[T]) Handle(ctx *Context) {
	v := h.fromCtx(ctx)
	switch {
	case h.callbackAll != nil:
		h.callbackAll(ctx, ctx.Client, v)
	case h.callbackFull != nil:
		h.callbackFull(ctx, v)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, v)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}

// ── Type aliases ───────────────────────────────────────────────────────────

// MessageHandler is a generic typedHandler parameterized by *types.Message.
// See NewMessageHandler for construction and callback signatures.
type MessageHandler = typedHandler[*types.Message]

// EditedMessageHandler is a generic typedHandler parameterized by *types.Message.
// See NewEditedMessageHandler for construction and callback signatures.
type EditedMessageHandler = typedHandler[*types.Message]

// BusinessMessageHandler is a generic typedHandler parameterized by *types.Message.
// See NewBusinessMessageHandler for construction and callback signatures.
type BusinessMessageHandler = typedHandler[*types.Message]

// EditedBusinessMessageHandler is a generic typedHandler parameterized by *types.Message.
// See NewEditedBusinessMessageHandler for construction and callback signatures.
type EditedBusinessMessageHandler = typedHandler[*types.Message]

// GuestMessageHandler is a generic typedHandler parameterized by *types.Message.
// See NewGuestMessageHandler for construction and callback signatures.
type GuestMessageHandler = typedHandler[*types.Message]

// CallbackQueryHandler is a generic typedHandler parameterized by *types.CallbackQuery.
// See NewCallbackQueryHandler for construction and callback signatures.
type CallbackQueryHandler = typedHandler[*types.CallbackQuery]

// InlineQueryHandler is a generic typedHandler parameterized by *types.InlineQuery.
// See NewInlineQueryHandler for construction and callback signatures.
type InlineQueryHandler = typedHandler[*types.InlineQuery]

// ChosenInlineResultHandler is a generic typedHandler parameterized by *types.ChosenInlineResult.
// See NewChosenInlineResultHandler for construction and callback signatures.
type ChosenInlineResultHandler = typedHandler[*types.ChosenInlineResult]

// UserStatusHandler is a generic typedHandler parameterized by *types.UserStatusUpdated.
// See NewUserStatusHandler for construction and callback signatures.
type UserStatusHandler = typedHandler[*types.UserStatusUpdated]

// ChatMemberHandler is a generic typedHandler parameterized by *types.ChatMemberUpdated.
// See NewChatMemberHandler for construction and callback signatures.
type ChatMemberHandler = typedHandler[*types.ChatMemberUpdated]

// MessageReactionHandler is a generic typedHandler parameterized by *types.MessageReactionUpdate.
// See NewMessageReactionHandler for construction and callback signatures.
type MessageReactionHandler = typedHandler[*types.MessageReactionUpdate]

// MessageReactionCountHandler is a generic typedHandler parameterized by *types.MessageReactionCountUpdate.
// See NewMessageReactionCountHandler for construction and callback signatures.
type MessageReactionCountHandler = typedHandler[*types.MessageReactionCountUpdate]

// PollHandler is a generic typedHandler parameterized by *types.PollUpdated.
// See NewPollHandler for construction and callback signatures.
type PollHandler = typedHandler[*types.PollUpdated]

// BusinessConnectionHandler is a generic typedHandler parameterized by *types.BusinessConnection.
// See NewBusinessConnectionHandler for construction and callback signatures.
type BusinessConnectionHandler = typedHandler[*types.BusinessConnection]

// StoryHandler is a generic typedHandler parameterized by *types.Story.
// See NewStoryHandler for construction and callback signatures.
type StoryHandler = typedHandler[*types.Story]

// ChatBoostHandler is a generic typedHandler parameterized by *types.ChatBoostUpdated.
// See NewChatBoostHandler for construction and callback signatures.
type ChatBoostHandler = typedHandler[*types.ChatBoostUpdated]

// ChatJoinRequestHandler is a generic typedHandler parameterized by *types.ChatJoinRequest.
// See NewChatJoinRequestHandler for construction and callback signatures.
type ChatJoinRequestHandler = typedHandler[*types.ChatJoinRequest]

// PreCheckoutQueryHandler is a generic typedHandler parameterized by *types.PreCheckoutQuery.
// See NewPreCheckoutQueryHandler for construction and callback signatures.
type PreCheckoutQueryHandler = typedHandler[*types.PreCheckoutQuery]

// ShippingQueryHandler is a generic typedHandler parameterized by *types.ShippingQuery.
// See NewShippingQueryHandler for construction and callback signatures.
type ShippingQueryHandler = typedHandler[*types.ShippingQuery]

// PurchasedPaidMediaHandler is a generic typedHandler parameterized by *types.PurchasedPaidMedia.
// See NewPurchasedPaidMediaHandler for construction and callback signatures.
type PurchasedPaidMediaHandler = typedHandler[*types.PurchasedPaidMedia]

// ManagedBotHandler is a generic typedHandler parameterized by *types.ManagedBotUpdated.
// See NewManagedBotHandler for construction and callback signatures.
type ManagedBotHandler = typedHandler[*types.ManagedBotUpdated]

// DeletedMessagesHandler is a generic typedHandler parameterized by *types.DeletedMessages.
// See NewDeletedMessagesHandler for construction and callback signatures.
type DeletedMessagesHandler = typedHandler[*types.DeletedMessages]

// DeletedBusinessMessagesHandler is a generic typedHandler parameterized by *types.DeletedMessages.
// See NewDeletedBusinessMessagesHandler for construction and callback signatures.
type DeletedBusinessMessagesHandler = typedHandler[*types.DeletedMessages]

// ── Constructors ────────────────────────────────────────────────────────────

// NewMessageHandler creates a handler for incoming new messages.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Message)
//   - func(*Context, *types.Message)
//   - func(*Context, *Client, *types.Message)
//
// Optional filters further restrict which updates trigger the handler.
func NewMessageHandler(callback any, filters ...Filter) *MessageHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.Message == nil },
		func(c *Context) *types.Message { return c.Message },
		func(c *Context, u *Update) { c.Message = u.Message },
		filters,
	)
}

// NewEditedMessageHandler creates a handler for edited message updates.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Message)
//   - func(*Context, *types.Message)
//   - func(*Context, *Client, *types.Message)
//
// Optional filters further restrict which updates trigger the handler.
func NewEditedMessageHandler(callback any, filters ...Filter) *EditedMessageHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.EditedMessage == nil },
		func(c *Context) *types.Message { return c.EditedMessage },
		func(c *Context, u *Update) { c.EditedMessage = u.EditedMessage },
		filters,
	)
}

// NewBusinessMessageHandler creates a handler for new business messages.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Message)
//   - func(*Context, *types.Message)
//   - func(*Context, *Client, *types.Message)
//
// Optional filters further restrict which updates trigger the handler.
func NewBusinessMessageHandler(callback any, filters ...Filter) *BusinessMessageHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.BusinessMessage == nil },
		func(c *Context) *types.Message { return c.BusinessMessage },
		func(c *Context, u *Update) { c.BusinessMessage = u.BusinessMessage },
		filters,
	)
}

// NewEditedBusinessMessageHandler creates a handler for edited business messages.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Message)
//   - func(*Context, *types.Message)
//   - func(*Context, *Client, *types.Message)
//
// Optional filters further restrict which updates trigger the handler.
func NewEditedBusinessMessageHandler(callback any, filters ...Filter) *EditedBusinessMessageHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.EditedBusinessMessage == nil },
		func(c *Context) *types.Message { return c.EditedBusinessMessage },
		func(c *Context, u *Update) { c.EditedBusinessMessage = u.EditedBusinessMessage },
		filters,
	)
}

// NewGuestMessageHandler creates a handler for messages from guest users
// pending approval in a Telegram Business chat. A GuestMessage filter is
// automatically prepended.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Message)
//   - func(*Context, *types.Message)
//   - func(*Context, *Client, *types.Message)
//
// Optional filters further restrict which updates trigger the handler.
func NewGuestMessageHandler(callback any, filters ...Filter) *GuestMessageHandler {
	allFilters := append([]Filter{GuestMessage}, filters...)
	return newTypedHandler(callback,
		func(u *Update) bool { return u.Message == nil },
		func(c *Context) *types.Message { return c.Message },
		func(c *Context, u *Update) { c.Message = u.Message },
		allFilters,
	)
}

// NewCallbackQueryHandler creates a handler for callback queries from inline
// keyboard button presses.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.CallbackQuery)
//   - func(*Context, *types.CallbackQuery)
//   - func(*Context, *Client, *types.CallbackQuery)
//
// Optional filters further restrict which updates trigger the handler.
func NewCallbackQueryHandler(callback any, filters ...Filter) *CallbackQueryHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.CallbackQuery == nil },
		func(c *Context) *types.CallbackQuery { return c.CallbackQuery },
		func(c *Context, u *Update) { c.CallbackQuery = u.CallbackQuery },
		filters,
	)
}

// NewInlineQueryHandler creates a handler for incoming inline queries.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.InlineQuery)
//   - func(*Context, *types.InlineQuery)
//   - func(*Context, *Client, *types.InlineQuery)
//
// Optional filters further restrict which updates trigger the handler.
func NewInlineQueryHandler(callback any, filters ...Filter) *InlineQueryHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.InlineQuery == nil },
		func(c *Context) *types.InlineQuery { return c.InlineQuery },
		func(c *Context, u *Update) { c.InlineQuery = u.InlineQuery },
		filters,
	)
}

// NewChosenInlineResultHandler creates a handler for chosen inline result updates.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ChosenInlineResult)
//   - func(*Context, *types.ChosenInlineResult)
//   - func(*Context, *Client, *types.ChosenInlineResult)
//
// Optional filters further restrict which updates trigger the handler.
func NewChosenInlineResultHandler(callback any, filters ...Filter) *ChosenInlineResultHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.ChosenInlineResult == nil },
		func(c *Context) *types.ChosenInlineResult { return c.ChosenInlineResult },
		func(c *Context, u *Update) { c.ChosenInlineResult = u.ChosenInlineResult },
		filters,
	)
}

// NewUserStatusHandler creates a handler for user online/offline status changes.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.UserStatusUpdated)
//   - func(*Context, *types.UserStatusUpdated)
//   - func(*Context, *Client, *types.UserStatusUpdated)
//
// Optional filters further restrict which updates trigger the handler.
func NewUserStatusHandler(callback any, filters ...Filter) *UserStatusHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.UserStatus == nil },
		func(c *Context) *types.UserStatusUpdated { return c.UserStatus },
		func(c *Context, u *Update) { c.UserStatus = u.UserStatus },
		filters,
	)
}

// NewChatMemberHandler creates a handler for chat member status changes.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ChatMemberUpdated)
//   - func(*Context, *types.ChatMemberUpdated)
//   - func(*Context, *Client, *types.ChatMemberUpdated)
//
// Optional filters further restrict which updates trigger the handler.
func NewChatMemberHandler(callback any, filters ...Filter) *ChatMemberHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.ChatMember == nil },
		func(c *Context) *types.ChatMemberUpdated { return c.ChatMember },
		func(c *Context, u *Update) { c.ChatMember = u.ChatMember },
		filters,
	)
}

// NewMessageReactionHandler creates a handler for message reaction updates.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.MessageReactionUpdate)
//   - func(*Context, *types.MessageReactionUpdate)
//   - func(*Context, *Client, *types.MessageReactionUpdate)
//
// Optional filters further restrict which updates trigger the handler.
func NewMessageReactionHandler(callback any, filters ...Filter) *MessageReactionHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.MessageReaction == nil },
		func(c *Context) *types.MessageReactionUpdate { return c.MessageReaction },
		func(c *Context, u *Update) { c.MessageReaction = u.MessageReaction },
		filters,
	)
}

// NewMessageReactionCountHandler creates a handler for anonymous reaction count updates.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.MessageReactionCountUpdate)
//   - func(*Context, *types.MessageReactionCountUpdate)
//   - func(*Context, *Client, *types.MessageReactionCountUpdate)
//
// Optional filters further restrict which updates trigger the handler.
func NewMessageReactionCountHandler(callback any, filters ...Filter) *MessageReactionCountHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.MessageReactionCount == nil },
		func(c *Context) *types.MessageReactionCountUpdate { return c.MessageReactionCount },
		func(c *Context, u *Update) { c.MessageReactionCount = u.MessageReactionCount },
		filters,
	)
}

// NewPollHandler creates a handler for poll state changes (votes, closures).
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.PollUpdated)
//   - func(*Context, *types.PollUpdated)
//   - func(*Context, *Client, *types.PollUpdated)
//
// Optional filters further restrict which updates trigger the handler.
func NewPollHandler(callback any, filters ...Filter) *PollHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.Poll == nil },
		func(c *Context) *types.PollUpdated { return c.Poll },
		func(c *Context, u *Update) { c.Poll = u.Poll },
		filters,
	)
}

// NewBusinessConnectionHandler creates a handler for business connection changes.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.BusinessConnection)
//   - func(*Context, *types.BusinessConnection)
//   - func(*Context, *Client, *types.BusinessConnection)
//
// Optional filters further restrict which updates trigger the handler.
func NewBusinessConnectionHandler(callback any, filters ...Filter) *BusinessConnectionHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.BusinessConnection == nil },
		func(c *Context) *types.BusinessConnection { return c.BusinessConnection },
		func(c *Context, u *Update) { c.BusinessConnection = u.BusinessConnection },
		filters,
	)
}

// NewStoryHandler creates a handler for story events (new, deleted, updated).
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Story)
//   - func(*Context, *types.Story)
//   - func(*Context, *Client, *types.Story)
//
// Optional filters further restrict which updates trigger the handler.
func NewStoryHandler(callback any, filters ...Filter) *StoryHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.Story == nil },
		func(c *Context) *types.Story { return c.Story },
		func(c *Context, u *Update) { c.Story = u.Story },
		filters,
	)
}

// NewChatBoostHandler creates a handler for chat boost changes.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ChatBoostUpdated)
//   - func(*Context, *types.ChatBoostUpdated)
//   - func(*Context, *Client, *types.ChatBoostUpdated)
//
// Optional filters further restrict which updates trigger the handler.
func NewChatBoostHandler(callback any, filters ...Filter) *ChatBoostHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.ChatBoost == nil },
		func(c *Context) *types.ChatBoostUpdated { return c.ChatBoost },
		func(c *Context, u *Update) { c.ChatBoost = u.ChatBoost },
		filters,
	)
}

// NewChatJoinRequestHandler creates a handler for chat join requests.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ChatJoinRequest)
//   - func(*Context, *types.ChatJoinRequest)
//   - func(*Context, *Client, *types.ChatJoinRequest)
//
// Optional filters further restrict which updates trigger the handler.
func NewChatJoinRequestHandler(callback any, filters ...Filter) *ChatJoinRequestHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.ChatJoinRequest == nil },
		func(c *Context) *types.ChatJoinRequest { return c.ChatJoinRequest },
		func(c *Context, u *Update) { c.ChatJoinRequest = u.ChatJoinRequest },
		filters,
	)
}

// NewPreCheckoutQueryHandler creates a handler for pre-checkout payment queries.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.PreCheckoutQuery)
//   - func(*Context, *types.PreCheckoutQuery)
//   - func(*Context, *Client, *types.PreCheckoutQuery)
//
// Optional filters further restrict which updates trigger the handler.
func NewPreCheckoutQueryHandler(callback any, filters ...Filter) *PreCheckoutQueryHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.PreCheckoutQuery == nil },
		func(c *Context) *types.PreCheckoutQuery { return c.PreCheckoutQuery },
		func(c *Context, u *Update) { c.PreCheckoutQuery = u.PreCheckoutQuery },
		filters,
	)
}

// NewShippingQueryHandler creates a handler for shipping query payment updates.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ShippingQuery)
//   - func(*Context, *types.ShippingQuery)
//   - func(*Context, *Client, *types.ShippingQuery)
//
// Optional filters further restrict which updates trigger the handler.
func NewShippingQueryHandler(callback any, filters ...Filter) *ShippingQueryHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.ShippingQuery == nil },
		func(c *Context) *types.ShippingQuery { return c.ShippingQuery },
		func(c *Context, u *Update) { c.ShippingQuery = u.ShippingQuery },
		filters,
	)
}

// NewPurchasedPaidMediaHandler creates a handler for purchased paid media updates.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.PurchasedPaidMedia)
//   - func(*Context, *types.PurchasedPaidMedia)
//   - func(*Context, *Client, *types.PurchasedPaidMedia)
//
// Optional filters further restrict which updates trigger the handler.
func NewPurchasedPaidMediaHandler(callback any, filters ...Filter) *PurchasedPaidMediaHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.PurchasedPaidMedia == nil },
		func(c *Context) *types.PurchasedPaidMedia { return c.PurchasedPaidMedia },
		func(c *Context, u *Update) { c.PurchasedPaidMedia = u.PurchasedPaidMedia },
		filters,
	)
}

// NewManagedBotHandler creates a handler for managed bot connection updates.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ManagedBotUpdated)
//   - func(*Context, *types.ManagedBotUpdated)
//   - func(*Context, *Client, *types.ManagedBotUpdated)
//
// Optional filters further restrict which updates trigger the handler.
func NewManagedBotHandler(callback any, filters ...Filter) *ManagedBotHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.ManagedBot == nil },
		func(c *Context) *types.ManagedBotUpdated { return c.ManagedBot },
		func(c *Context, u *Update) { c.ManagedBot = u.ManagedBot },
		filters,
	)
}

// NewDeletedMessagesHandler creates a handler for deleted message updates.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.DeletedMessages)
//   - func(*Context, *types.DeletedMessages)
//   - func(*Context, *Client, *types.DeletedMessages)
//
// Optional filters further restrict which updates trigger the handler.
func NewDeletedMessagesHandler(callback any, filters ...Filter) *DeletedMessagesHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.DeletedMessages == nil },
		func(c *Context) *types.DeletedMessages { return c.DeletedMessages },
		func(c *Context, u *Update) { c.DeletedMessages = u.DeletedMessages },
		filters,
	)
}

// NewDeletedBusinessMessagesHandler creates a handler for deleted business messages.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.DeletedMessages)
//   - func(*Context, *types.DeletedMessages)
//   - func(*Context, *Client, *types.DeletedMessages)
//
// Optional filters further restrict which updates trigger the handler.
func NewDeletedBusinessMessagesHandler(callback any, filters ...Filter) *DeletedBusinessMessagesHandler {
	return newTypedHandler(callback,
		func(u *Update) bool { return u.DeletedBusinessMessages == nil },
		func(c *Context) *types.DeletedMessages { return c.DeletedBusinessMessages },
		func(c *Context, u *Update) { c.DeletedBusinessMessages = u.DeletedBusinessMessages },
		filters,
	)
}
