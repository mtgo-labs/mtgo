package telegram

// OnMessage registers a handler for new incoming messages. The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Message)
//   - func(*Context, *types.Message)
//
// Optional filters restrict which messages trigger the handler. Returns the registered
// Handler for later removal.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    if ctx.Message != nil {
//	        ctx.Reply("Got your message!")
//	    }
//	})
//
// Example (with filters):
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Reply("You sent a photo!")
//	}, telegram.Photo())
func (c *Client) OnMessage(callback interface{}, filters ...Filter) Handler {
	h := NewMessageHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnEditedMessage registers a handler for edited message updates. The callback must
// be one of:
//   - func(*Context)
//   - func(*Client, *types.Message)
//   - func(*Context, *types.Message)
//
// Optional filters restrict which edited messages trigger the handler. Returns the
// registered Handler for later removal.
//
// Example:
//
//	client.OnEditedMessage(func(ctx *telegram.Context) {
//	    ctx.Reply("I saw you edit that!")
//	})
func (c *Client) OnEditedMessage(callback interface{}, filters ...Filter) Handler {
	h := NewEditedMessageHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnBusinessMessage registers a handler for new messages sent through a business
// connection. The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Message)
//   - func(*Context, *types.Message)
//
// Optional filters restrict which business messages trigger the handler. Returns the
// registered Handler for later removal.
func (c *Client) OnBusinessMessage(callback interface{}, filters ...Filter) Handler {
	h := NewBusinessMessageHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnEditedBusinessMessage registers a handler for edited messages sent through a
// business connection. The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Message)
//   - func(*Context, *types.Message)
//
// Optional filters restrict which edited business messages trigger the handler.
// Returns the registered Handler for later removal.
func (c *Client) OnEditedBusinessMessage(callback interface{}, filters ...Filter) Handler {
	h := NewEditedBusinessMessageHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnDeletedMessages registers a handler for message deletion updates. The callback
// must be one of:
//   - func(*Context)
//   - func(*Client, *types.DeletedMessages)
//   - func(*Context, *types.DeletedMessages)
//
// Optional filters restrict which deletions trigger the handler. Returns the registered
// Handler for later removal.
func (c *Client) OnDeletedMessages(callback interface{}, filters ...Filter) Handler {
	h := NewDeletedMessagesHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnDeletedBusinessMessages registers a handler for deleted business message updates.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.DeletedMessages)
//   - func(*Context, *types.DeletedMessages)
//
// Optional filters restrict which deletions trigger the handler. Returns the registered
// Handler for later removal.
func (c *Client) OnDeletedBusinessMessages(callback interface{}, filters ...Filter) Handler {
	h := NewDeletedBusinessMessagesHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnGuestMessage registers a handler for messages sent by guest users pending
// approval in a Telegram Business chat. The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Message)
//   - func(*Context, *types.Message)
//
// The GuestMessage filter is applied automatically. Additional optional filters
// can further restrict which guest messages trigger the handler. Returns the
// registered Handler for later removal.
func (c *Client) OnGuestMessage(callback interface{}, filters ...Filter) Handler {
	h := NewGuestMessageHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnCallbackQuery registers a handler for inline keyboard callback queries. The
// callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.CallbackQuery)
//   - func(*Context, *types.CallbackQuery)
//
// Optional filters restrict which callback queries trigger the handler. Returns the
// registered Handler for later removal.
//
// Example:
//
//	client.OnCallbackQuery(func(ctx *telegram.Context) {
//	    ctx.Answer("Button pressed!", false)
//	    ctx.CallbackEditText("Processing...")
//	})
func (c *Client) OnCallbackQuery(callback interface{}, filters ...Filter) Handler {
	h := NewCallbackQueryHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnInlineQuery registers a handler for inline query updates sent when a user types
// the bot's username. The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.InlineQuery)
//   - func(*Context, *types.InlineQuery)
//
// Optional filters restrict which inline queries trigger the handler. Returns the
// registered Handler for later removal.
//
// Example:
//
//	client.OnInlineQuery(func(ctx *telegram.Context) {
//	    results := []tg.InputBotInlineResultClass{...}
//	    ctx.AnswerInlineQuery(results)
//	})
func (c *Client) OnInlineQuery(callback interface{}, filters ...Filter) Handler {
	h := NewInlineQueryHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnChosenInlineResult registers a handler for chosen inline result updates sent
// when a user selects an inline result. The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ChosenInlineResult)
//   - func(*Context, *types.ChosenInlineResult)
//
// Optional filters restrict which results trigger the handler. Returns the registered
// Handler for later removal.
func (c *Client) OnChosenInlineResult(callback interface{}, filters ...Filter) Handler {
	h := NewChosenInlineResultHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnUserStatus registers a handler for user online status changes. The callback
// must be one of:
//   - func(*Context)
//   - func(*Client, *types.UserStatusUpdated)
//   - func(*Context, *types.UserStatusUpdated)
//
// Optional filters restrict which status changes trigger the handler. Returns the
// registered Handler for later removal.
func (c *Client) OnUserStatus(callback interface{}, filters ...Filter) Handler {
	h := NewUserStatusHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnChatMember registers a handler for chat member status changes (join, leave,
// kick, ban, promote, demote). The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ChatMemberUpdated)
//   - func(*Context, *types.ChatMemberUpdated)
//
// Optional filters restrict which member changes trigger the handler. Returns the
// registered Handler for later removal.
//
// Example:
//
//	client.OnChatMember(func(ctx *telegram.Context) {
//	    log.Println("Chat member status changed")
//	})
func (c *Client) OnChatMember(callback interface{}, filters ...Filter) Handler {
	h := NewChatMemberHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnMessageReaction registers a handler for message reaction changes. The callback
// must be one of:
//   - func(*Context)
//   - func(*Client, *types.MessageReactions)
//   - func(*Context, *types.MessageReactions)
//
// Optional filters restrict which reactions trigger the handler. Returns the
// registered Handler for later removal.
func (c *Client) OnMessageReaction(callback interface{}, filters ...Filter) Handler {
	h := NewMessageReactionHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnMessageReactionCount registers a handler for anonymous reaction count changes.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.MessageReactions)
//   - func(*Context, *types.MessageReactions)
//
// Optional filters restrict which updates trigger the handler. Returns the registered
// Handler for later removal.
func (c *Client) OnMessageReactionCount(callback interface{}, filters ...Filter) Handler {
	h := NewMessageReactionCountHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnPoll registers a handler for poll state updates (votes, closure). The callback
// must be one of:
//   - func(*Context)
//   - func(*Client, *types.PollUpdate)
//   - func(*Context, *types.PollUpdate)
//
// Optional filters restrict which poll updates trigger the handler. Returns the
// registered Handler for later removal.
func (c *Client) OnPoll(callback interface{}, filters ...Filter) Handler {
	h := NewPollHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnBusinessConnection registers a handler for business connection status changes.
// The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.BusinessConnection)
//   - func(*Context, *types.BusinessConnection)
//
// Optional filters restrict which updates trigger the handler. Returns the registered
// Handler for later removal.
func (c *Client) OnBusinessConnection(callback interface{}, filters ...Filter) Handler {
	h := NewBusinessConnectionHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnStory registers a handler for Telegram story updates. The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.Story)
//   - func(*Context, *types.Story)
//
// Optional filters restrict which story updates trigger the handler. Returns the
// registered Handler for later removal.
func (c *Client) OnStory(callback interface{}, filters ...Filter) Handler {
	h := NewStoryHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnChatBoost registers a handler for chat boost updates. The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ChatBoostUpdated)
//   - func(*Context, *types.ChatBoostUpdated)
//
// Optional filters restrict which boost updates trigger the handler. Returns the
// registered Handler for later removal.
func (c *Client) OnChatBoost(callback interface{}, filters ...Filter) Handler {
	h := NewChatBoostHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnChatJoinRequest registers a handler for chat join request updates. The callback
// must be one of:
//   - func(*Context)
//   - func(*Client, *types.ChatJoinRequest)
//   - func(*Context, *types.ChatJoinRequest)
//
// Optional filters restrict which join requests trigger the handler. Returns the
// registered Handler for later removal.
func (c *Client) OnChatJoinRequest(callback interface{}, filters ...Filter) Handler {
	h := NewChatJoinRequestHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnPreCheckoutQuery registers a handler for pre-checkout payment queries. The
// callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.PreCheckoutQuery)
//   - func(*Context, *types.PreCheckoutQuery)
//
// Optional filters restrict which queries trigger the handler. Returns the registered
// Handler for later removal.
func (c *Client) OnPreCheckoutQuery(callback interface{}, filters ...Filter) Handler {
	h := NewPreCheckoutQueryHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnShippingQuery registers a handler for shipping queries during the Telegram
// payment flow. The callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ShippingQuery)
//   - func(*Context, *types.ShippingQuery)
//
// Optional filters restrict which queries trigger the handler. Returns the registered
// Handler for later removal.
func (c *Client) OnShippingQuery(callback interface{}, filters ...Filter) Handler {
	h := NewShippingQueryHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnPurchasedPaidMedia registers a handler for paid media purchase updates. The
// callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.PurchasedPaidMedia)
//   - func(*Context, *types.PurchasedPaidMedia)
//
// Optional filters restrict which purchases trigger the handler. Returns the
// registered Handler for later removal.
func (c *Client) OnPurchasedPaidMedia(callback interface{}, filters ...Filter) Handler {
	h := NewPurchasedPaidMediaHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnManagedBot registers a handler for managed bot connection state changes. The
// callback must be one of:
//   - func(*Context)
//   - func(*Client, *types.ManagedBotUpdated)
//   - func(*Context, *types.ManagedBotUpdated)
//
// Optional filters restrict which updates trigger the handler. Returns the registered
// Handler for later removal.
func (c *Client) OnManagedBot(callback interface{}, filters ...Filter) Handler {
	h := NewManagedBotHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnRawUpdate registers a catch-all handler that fires for every incoming update,
// regardless of type. The callback must be of type func(*Context). Optional filters
// can restrict which updates trigger the handler. Returns the registered Handler
// for later removal.
//
// Example:
//
//	client.OnRawUpdate(func(ctx *telegram.Context) {
//	    log.Printf("raw update received: %+v", ctx.Update)
//	})
func (c *Client) OnRawUpdate(callback interface{}, filters ...Filter) Handler {
	h := NewRawUpdateHandler(callback, filters...)
	c.registerHandler(h)
	return h
}

// OnStart registers a handler that fires once when the client starts up, before any
// connection is established. Returns the registered Handler for later removal.
func (c *Client) OnStart(callback func(*Context)) Handler {
	h := NewStartHandler(callback)
	c.registerHandler(h)
	return h
}

// OnStop registers a handler that fires when the client is shutting down. Use this
// for final cleanup. Returns the registered Handler for later removal.
func (c *Client) OnStop(callback func(*Context)) Handler {
	h := NewStopHandler(callback)
	c.registerHandler(h)
	return h
}

// OnConnect registers a handler that fires each time the client successfully connects
// to the Telegram servers. Use this for post-connection initialization. Returns the
// registered Handler for later removal.
func (c *Client) OnConnect(callback func(*Context)) Handler {
	h := NewConnectHandler(callback)
	c.registerHandler(h)
	return h
}

// OnDisconnect registers a handler that fires each time the client disconnects from
// the Telegram servers. Use this for cleanup on connection loss. Returns the registered
// Handler for later removal.
func (c *Client) OnDisconnect(callback func(*Context)) Handler {
	h := NewDisconnectHandler(callback)
	c.registerHandler(h)
	return h
}

// OnError registers a handler for runtime errors encountered by the client. Optional
// exception types can be provided to restrict the handler to specific error categories.
// Returns the registered Handler for later removal.
//
// Example:
//
//	client.OnError(func(ctx *telegram.Context) {
//	    log.Printf("bot error: %v", ctx.Error)
//	})
func (c *Client) OnError(callback func(*Context), exceptions ...error) Handler {
	h := NewErrorHandler(callback, exceptions...)
	c.registerHandler(h)
	return h
}

// AddHandler registers a pre-constructed Handler with the client's dispatcher. An
// optional group number controls handler execution order; handlers in lower-numbered
// groups are executed first.
//
// Example:
//
//	h := telegram.NewMessageHandler(func(ctx *telegram.Context) {
//	    ctx.Reply("Handled via AddHandler")
//	}, telegram.Command("start"))
//	client.AddHandler(h, 0)
func (c *Client) AddHandler(handler Handler, group ...int) {
	if c.handlerDispatcher != nil {
		c.handlerDispatcher.AddHandler(handler, group...)
	}
}

// RemoveHandler unregisters a previously added handler so it no longer receives
// updates. This is a no-op if the handler was never registered or the dispatcher
// has not been initialized.
func (c *Client) RemoveHandler(handler Handler) {
	if c.handlerDispatcher != nil {
		c.handlerDispatcher.RemoveHandler(handler)
	}
}

// registerHandler is an internal helper that adds a handler to the dispatcher and
// logs the registration for debugging purposes.
func (c *Client) registerHandler(h Handler) {
	if c.handlerDispatcher != nil {
		c.Log.Debugf("registerHandler %T", h)
		c.handlerDispatcher.AddHandler(h)
	}
}
