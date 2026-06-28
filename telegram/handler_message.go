package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// MessageHandler processes incoming new messages in chats where the bot is present.
// This is the most commonly used handler for responding to user messages, commands,
// and other text-based interactions.
type MessageHandler struct {
	baseHandler
	callbackCtx    func(*Context)
	callbackClient func(*Client, *types.Message)
	callbackFull   func(*Context, *types.Message)
	callbackAll    func(*Context, *Client, *types.Message)
}

// NewMessageHandler creates a new MessageHandler with the given callback function
// and optional filters. The callback must be one of four supported signatures.
//
// Example:
//
//	client.AddHandler(telegram.NewMessageHandler(func(ctx *telegram.Context, msg *types.Message) {
//		fmt.Printf("new message in chat %d: %s\n", msg.Chat.ID, msg.Text)
//	}))
//
// With a command filter:
//
//	client.AddHandler(telegram.NewMessageHandler(
//		func(ctx *telegram.Context, client *telegram.Client, msg *types.Message) {
//			client.SendMessage(ctx, msg.Chat.ID, "Pong!")
//		},
//		telegram.CommandFilter("ping"),
//	))
func NewMessageHandler(callback any, filters ...Filter) *MessageHandler {
	h := &MessageHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
	switch fn := callback.(type) {
	case func(*Context):
		h.callbackCtx = fn
	case func(*Client, *types.Message):
		h.callbackClient = fn
	case func(*Context, *types.Message):
		h.callbackFull = fn
	case func(*Context, *Client, *types.Message):
		h.callbackAll = fn
	}
	return h
}

// Check reports whether the incoming update contains a Message field and passes
// the configured filters. Returns false if the update does not represent a new
// message.
func (h *MessageHandler) Check(update *Update) bool {
	if update.Message == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, Message: update.Message}
	return h.filters(ctx)
}

// Handle dispatches the incoming message to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *MessageHandler) Handle(ctx *Context) {
	switch {
	case h.callbackAll != nil:
		h.callbackAll(ctx, ctx.Client, ctx.Message)
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.Message)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.Message)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
