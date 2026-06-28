package telegram

import (
	"github.com/mtgo-labs/mtgo/telegram/types"
)

// EditedMessageHandler processes updates about messages that were edited in chats
// where the bot is present. Use this to react to edits on previously sent messages,
// such as updating locally cached message content.
type EditedMessageHandler struct {
	baseHandler
	callbackCtx    func(*Context)
	callbackClient func(*Client, *types.Message)
	callbackFull   func(*Context, *types.Message)
	callbackAll    func(*Context, *Client, *types.Message)
}

// NewEditedMessageHandler creates a new EditedMessageHandler with the given callback
// function and optional filters. The callback must be one of four supported signatures.
//
// Example:
//
//	client.AddHandler(telegram.NewEditedMessageHandler(func(ctx *telegram.Context, msg *types.Message) {
//		fmt.Printf("message %d edited: %s\n", msg.ID, msg.Text)
//	}))
//
// With a filter for a specific chat:
//
//	client.AddHandler(telegram.NewEditedMessageHandler(
//		func(ctx *telegram.Context) {
//			fmt.Println("edited in target chat")
//		},
//		telegram.ChatFilter(chatID),
//	))
func NewEditedMessageHandler(callback any, filters ...Filter) *EditedMessageHandler {
	h := &EditedMessageHandler{baseHandler: baseHandler{filters: mergeFilters(filters)}}
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

// Check reports whether the incoming update contains an EditedMessage field and
// passes the configured filters. Returns false if the update does not represent
// an edited message.
func (h *EditedMessageHandler) Check(update *Update) bool {
	if update.EditedMessage == nil {
		return false
	}
	if h.filters == nil {
		return true
	}
	ctx := &Context{Update: update, EditedMessage: update.EditedMessage}
	return h.filters(ctx)
}

// Handle dispatches the edited message to whichever callback variant was provided
// at construction time. The full callback is preferred, followed by the client
// callback, then the context-only callback.
func (h *EditedMessageHandler) Handle(ctx *Context) {
	switch {
	case h.callbackAll != nil:
		h.callbackAll(ctx, ctx.Client, ctx.EditedMessage)
	case h.callbackFull != nil:
		h.callbackFull(ctx, ctx.EditedMessage)
	case h.callbackClient != nil:
		h.callbackClient(ctx.Client, ctx.EditedMessage)
	case h.callbackCtx != nil:
		h.callbackCtx(ctx)
	}
}
