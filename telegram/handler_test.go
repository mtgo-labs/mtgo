package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/types"
)

func TestFilter_And(t *testing.T) {
	f1 := Filter(func(ctx *Context) bool { return true })
	f2 := Filter(func(ctx *Context) bool { return true })
	f3 := Filter(func(ctx *Context) bool { return false })

	combined := f1.And(f2)
	if !combined(&Context{}) {
		t.Error("expected true when both filters pass")
	}

	combined2 := f1.And(f3)
	if combined2(&Context{}) {
		t.Error("expected false when second filter fails")
	}
}

func TestFilter_Or(t *testing.T) {
	f1 := Filter(func(ctx *Context) bool { return false })
	f2 := Filter(func(ctx *Context) bool { return true })

	combined := f1.Or(f2)
	if !combined(&Context{}) {
		t.Error("expected true when second filter passes")
	}

	f3 := Filter(func(ctx *Context) bool { return false })
	combined2 := f1.Or(f3)
	if combined2(&Context{}) {
		t.Error("expected false when both filters fail")
	}
}

func TestFilter_Not(t *testing.T) {
	f := Filter(func(ctx *Context) bool { return true })
	negated := f.Not()
	if negated(&Context{}) {
		t.Error("expected false when negating true filter")
	}

	f2 := Filter(func(ctx *Context) bool { return false })
	negated2 := f2.Not()
	if !negated2(&Context{}) {
		t.Error("expected true when negating false filter")
	}
}

func TestFilter_Composition(t *testing.T) {
	isEven := Filter(func(ctx *Context) bool { return ctx.Message != nil && ctx.Message.ID%2 == 0 })
	hasText := Filter(func(ctx *Context) bool { return ctx.Message != nil && ctx.Message.Text != "" })

	both := isEven.And(hasText)
	if !both(&Context{Message: &types.Message{ID: 2, Text: "hi"}}) {
		t.Error("expected true for even ID with text")
	}
	if both(&Context{Message: &types.Message{ID: 2}}) {
		t.Error("expected false for even ID without text")
	}
	if both(&Context{Message: &types.Message{ID: 1, Text: "hi"}}) {
		t.Error("expected false for odd ID with text")
	}

	either := isEven.Or(hasText)
	if !either(&Context{Message: &types.Message{ID: 1, Text: "hi"}}) {
		t.Error("expected true for odd ID with text (hasText passes)")
	}
	if !either(&Context{Message: &types.Message{ID: 2}}) {
		t.Error("expected true for even ID without text (isEven passes)")
	}
	if either(&Context{Message: &types.Message{ID: 1}}) {
		t.Error("expected false for odd ID without text")
	}

	oddOnly := isEven.Not()
	if !oddOnly(&Context{Message: &types.Message{ID: 3}}) {
		t.Error("expected true for odd ID")
	}
	if oddOnly(&Context{Message: &types.Message{ID: 4}}) {
		t.Error("expected false for even ID")
	}
}

func TestMergeFilters_Single(t *testing.T) {
	called := false
	f := mergeFilters([]Filter{func(ctx *Context) bool { called = true; return true }})
	if !f(&Context{}) {
		t.Error("expected single filter to pass")
	}
	if !called {
		t.Error("expected filter to be called")
	}
}

func TestMergeFilters_Multiple(t *testing.T) {
	f := mergeFilters([]Filter{
		func(ctx *Context) bool { return true },
		func(ctx *Context) bool { return false },
	})
	if f(&Context{}) {
		t.Error("expected AND of true+false to be false")
	}
}

func TestMergeFilters_Empty(t *testing.T) {
	f := mergeFilters(nil)
	if f != nil {
		t.Error("expected nil for empty filters")
	}
}

func TestMessageHandler_Check_NilMessage(t *testing.T) {
	h := NewMessageHandler(nil)
	if h.Check(&Update{}) {
		t.Error("expected false for nil message")
	}
}

func TestMessageHandler_Check_NoFilter(t *testing.T) {
	h := NewMessageHandler(nil)
	if !h.Check(&Update{Message: &types.Message{ID: 1}}) {
		t.Error("expected true for non-nil message with no filter")
	}
}

func TestMessageHandler_Check_FilterPasses(t *testing.T) {
	h := NewMessageHandler(nil, func(ctx *Context) bool {
		return ctx.Message != nil && ctx.Message.Text == "hello"
	})
	if !h.Check(&Update{Message: &types.Message{ID: 1, Text: "hello"}}) {
		t.Error("expected true when filter passes")
	}
}

func TestMessageHandler_Check_FilterFails(t *testing.T) {
	h := NewMessageHandler(nil, func(ctx *Context) bool {
		return ctx.Message != nil && ctx.Message.Text == "hello"
	})
	if h.Check(&Update{Message: &types.Message{ID: 1, Text: "bye"}}) {
		t.Error("expected false when filter fails")
	}
}

func TestMessageHandler_Handle(t *testing.T) {
	var received *Context
	h := NewMessageHandler(func(ctx *Context) {
		received = ctx
	})
	ctx := &Context{Message: &types.Message{ID: 42}}
	h.Handle(ctx)
	if received == nil || received.Message.ID != 42 {
		t.Error("expected handler to receive context with message ID 42")
	}
}

func TestMessageHandler_Handle_NilCallback(t *testing.T) {
	h := NewMessageHandler(nil)
	h.Handle(&Context{})
}

func TestEditedMessageHandler_Check(t *testing.T) {
	h := NewEditedMessageHandler(nil)
	if h.Check(&Update{}) {
		t.Error("expected false for nil edited message")
	}
	if !h.Check(&Update{EditedMessage: &types.Message{ID: 1}}) {
		t.Error("expected true for non-nil edited message")
	}
}

func TestEditedMessageHandler_Check_WithFilter(t *testing.T) {
	h := NewEditedMessageHandler(nil, func(ctx *Context) bool {
		return ctx.EditedMessage != nil && ctx.EditedMessage.Text == "edited"
	})
	if !h.Check(&Update{EditedMessage: &types.Message{ID: 1, Text: "edited"}}) {
		t.Error("expected true for matching edited message")
	}
	if h.Check(&Update{EditedMessage: &types.Message{ID: 1, Text: "original"}}) {
		t.Error("expected false for non-matching edited message")
	}
}

func TestDeletedMessagesHandler_Check(t *testing.T) {
	h := NewDeletedMessagesHandler(nil)
	if h.Check(&Update{}) {
		t.Error("expected false for nil deleted messages")
	}
	if !h.Check(&Update{DeletedMessages: &types.DeletedMessages{Messages: []int32{1, 2}}}) {
		t.Error("expected true for non-nil deleted messages")
	}
}

func TestCallbackQueryHandler_Check(t *testing.T) {
	h := NewCallbackQueryHandler(nil)
	if h.Check(&Update{}) {
		t.Error("expected false for nil callback query")
	}
	if !h.Check(&Update{CallbackQuery: &types.CallbackQuery{ID: 1}}) {
		t.Error("expected true for non-nil callback query")
	}
}

func TestCallbackQueryHandler_Check_WithFilter(t *testing.T) {
	h := NewCallbackQueryHandler(nil, func(ctx *Context) bool {
		return ctx.CallbackQuery != nil && string(ctx.CallbackQuery.Data) == "confirm"
	})
	if !h.Check(&Update{CallbackQuery: &types.CallbackQuery{ID: 1, Data: []byte("confirm")}}) {
		t.Error("expected true for matching callback data")
	}
	if h.Check(&Update{CallbackQuery: &types.CallbackQuery{ID: 1, Data: []byte("cancel")}}) {
		t.Error("expected false for non-matching callback data")
	}
}

func TestInlineQueryHandler_Check(t *testing.T) {
	h := NewInlineQueryHandler(nil)
	if h.Check(&Update{}) {
		t.Error("expected false for nil inline query")
	}
	if !h.Check(&Update{InlineQuery: &types.InlineQuery{ID: 1}}) {
		t.Error("expected true for non-nil inline query")
	}
}

func TestUserStatusHandler_Check(t *testing.T) {
	h := NewUserStatusHandler(nil)
	if h.Check(&Update{}) {
		t.Error("expected false for nil user status")
	}
	if !h.Check(&Update{UserStatus: &types.UserStatusUpdated{UserID: 1}}) {
		t.Error("expected true for non-nil user status")
	}
}

func TestChatMemberHandler_Check(t *testing.T) {
	h := NewChatMemberHandler(nil)
	if h.Check(&Update{}) {
		t.Error("expected false for nil chat member")
	}
	if !h.Check(&Update{ChatMember: &types.ChatMemberUpdated{}}) {
		t.Error("expected true for non-nil chat member")
	}
}

func TestPollHandler_Check(t *testing.T) {
	h := NewPollHandler(nil)
	if h.Check(&Update{}) {
		t.Error("expected false for nil poll")
	}
	if !h.Check(&Update{Poll: &types.PollUpdate{PollID: 1}}) {
		t.Error("expected true for non-nil poll")
	}
}

func TestRawUpdateHandler_Check_NoFilter(t *testing.T) {
	h := NewRawUpdateHandler(nil)
	if !h.Check(&Update{}) {
		t.Error("expected true for any update (raw handler)")
	}
}

func TestRawUpdateHandler_Check_WithFilter(t *testing.T) {
	h := NewRawUpdateHandler(nil, func(ctx *Context) bool {
		return ctx.Update != nil && ctx.Update.Message != nil
	})
	if h.Check(&Update{}) {
		t.Error("expected false when update is empty but filter checks Message")
	}
	if !h.Check(&Update{Message: &types.Message{ID: 1}}) {
		t.Error("expected true when update has message (via Update.Message)")
	}
}

func TestRawUpdateHandler_Check_FilterUsesUpdateDirectly(t *testing.T) {
	h := NewRawUpdateHandler(nil, func(ctx *Context) bool {
		return ctx.Update != nil && ctx.Update.CallbackQuery != nil
	})
	if h.Check(&Update{}) {
		t.Error("expected false for update without callback")
	}
	if !h.Check(&Update{CallbackQuery: &types.CallbackQuery{ID: 1}}) {
		t.Error("expected true for update with callback")
	}
}

func TestTextFilter(t *testing.T) {
	f := Text("hello")
	if !f(&Context{Message: &types.Message{Text: "hello"}}) {
		t.Error("expected true for matching text")
	}
	if f(&Context{Message: &types.Message{Text: "bye"}}) {
		t.Error("expected false for non-matching text")
	}
	if f(&Context{}) {
		t.Error("expected false for nil message")
	}
}

func TestTextFilter_EmptyString(t *testing.T) {
	f := Text("")
	if !f(&Context{Message: &types.Message{Text: ""}}) {
		t.Error("expected true for empty text match")
	}
	if f(&Context{Message: &types.Message{Text: "hello"}}) {
		t.Error("expected false for non-empty text")
	}
}

func TestCommandFilter_SingleCommand(t *testing.T) {
	f := Command("start")
	if !f(&Context{Message: &types.Message{Text: "/start"}}) {
		t.Error("expected true for /start")
	}
	if !f(&Context{Message: &types.Message{Text: "/start@bot"}}) {
		t.Error("expected true for /start@bot")
	}
	if !f(&Context{Message: &types.Message{Text: "/start args here"}}) {
		t.Error("expected true for /start with args")
	}
	if f(&Context{Message: &types.Message{Text: "/help"}}) {
		t.Error("expected false for /help")
	}
	if f(&Context{Message: &types.Message{Text: "start"}}) {
		t.Error("expected false without /")
	}
	if f(&Context{Message: &types.Message{Text: ""}}) {
		t.Error("expected false for empty text")
	}
	if f(&Context{}) {
		t.Error("expected false for nil message")
	}
}

func TestCommandFilter_MultipleCommands(t *testing.T) {
	f := Command("start", "help", "settings")
	cases := []struct {
		text string
		want bool
	}{
		{"/start", true},
		{"/help", true},
		{"/settings", true},
		{"/unknown", false},
		{"/start@mybot", true},
	}
	for _, tc := range cases {
		got := f(&Context{Message: &types.Message{Text: tc.text}})
		if got != tc.want {
			t.Errorf("Command(%q) for %q: got %v, want %v", tc.text, tc.text, got, tc.want)
		}
	}
}

func TestRegexFilter(t *testing.T) {
	f := Regex(`^hello \w+$`)
	if !f(&Context{Message: &types.Message{Text: "hello world"}}) {
		t.Error("expected true for matching regex")
	}
	if f(&Context{Message: &types.Message{Text: "hello"}}) {
		t.Error("expected false for non-matching regex")
	}
	if f(&Context{Message: &types.Message{Text: ""}}) {
		t.Error("expected false for empty text")
	}
	if f(&Context{}) {
		t.Error("expected false for nil message")
	}
}

func TestPrivateFilter(t *testing.T) {
	f := Private
	if !f(&Context{Message: &types.Message{ChatID: 42}}) {
		t.Error("expected true for positive chat ID")
	}
	if f(&Context{Message: &types.Message{ChatID: -100}}) {
		t.Error("expected false for negative chat ID")
	}
	if f(&Context{Message: &types.Message{ChatID: 0}}) {
		t.Error("expected false for zero chat ID")
	}
	if f(&Context{}) {
		t.Error("expected false for nil message")
	}
}

func TestGroupFilter(t *testing.T) {
	f := Group
	if !f(&Context{Message: &types.Message{ChatID: -100}}) {
		t.Error("expected true for negative chat ID")
	}
	if f(&Context{Message: &types.Message{ChatID: 42}}) {
		t.Error("expected false for positive chat ID")
	}
}

func TestChannelFilter(t *testing.T) {
	f := Channel
	chatID := int64(-100)
	if !f(&Context{
		Message: &types.Message{ChatID: chatID},
		Update:  &Update{Chats: map[int64]*types.Chat{chatID: {ID: chatID, Type: types.ChatTypeChannel}}},
	}) {
		t.Error("expected true for channel chat")
	}
	if f(&Context{
		Message: &types.Message{ChatID: chatID},
		Update:  &Update{Chats: map[int64]*types.Chat{chatID: {ID: chatID, Type: types.ChatTypeSupergroup}}},
	}) {
		t.Error("expected false for supergroup chat")
	}
	if f(&Context{Message: &types.Message{ChatID: 42}}) {
		t.Error("expected false for positive chat ID")
	}
}

func TestUserFilter_SingleUser(t *testing.T) {
	f := User(42)
	if !f(&Context{Message: &types.Message{FromID: 42}}) {
		t.Error("expected true for matching user")
	}
	if f(&Context{Message: &types.Message{FromID: 99}}) {
		t.Error("expected false for non-matching user")
	}
	if f(&Context{}) {
		t.Error("expected false for nil message")
	}
}

func TestUserFilter_MultipleUsers(t *testing.T) {
	f := User(42, 99, 100)
	if !f(&Context{Message: &types.Message{FromID: 42}}) {
		t.Error("expected true for user 42")
	}
	if !f(&Context{Message: &types.Message{FromID: 100}}) {
		t.Error("expected true for user 100")
	}
	if f(&Context{Message: &types.Message{FromID: 1}}) {
		t.Error("expected false for user 1")
	}
}

func TestForwardedFilter(t *testing.T) {
	f := Forwarded
	if !f(&Context{Message: &types.Message{FwdFrom: &types.ForwardHeader{}}}) {
		t.Error("expected true for forwarded message")
	}
	if f(&Context{Message: &types.Message{}}) {
		t.Error("expected false for non-forwarded")
	}
	if f(&Context{}) {
		t.Error("expected false for nil message")
	}
}

func TestReplyFilter(t *testing.T) {
	f := Reply
	if !f(&Context{Message: &types.Message{ReplyToID: 5}}) {
		t.Error("expected true for reply")
	}
	if f(&Context{Message: &types.Message{}}) {
		t.Error("expected false for non-reply (zero ReplyToID)")
	}
	if f(&Context{}) {
		t.Error("expected false for nil message")
	}
}

func TestMentionedFilter(t *testing.T) {
	f := Mentioned
	if !f(&Context{Message: &types.Message{Mentioned: true}}) {
		t.Error("expected true for mentioned")
	}
	if f(&Context{Message: &types.Message{Mentioned: false}}) {
		t.Error("expected false for non-mentioned")
	}
}

func TestServiceFilter(t *testing.T) {
	f := Service
	if !f(&Context{Message: &types.Message{Service: &types.ServiceMessage{}}}) {
		t.Error("expected true for service message")
	}
	if f(&Context{Message: &types.Message{}}) {
		t.Error("expected false for non-service")
	}
}

func TestHasMediaFilter(t *testing.T) {
	f := HasMedia
	if f(&Context{Message: &types.Message{}}) {
		t.Error("expected false for nil media")
	}
	if f(&Context{}) {
		t.Error("expected false for nil message")
	}
	if !f(&Context{Message: &types.Message{Media: &types.PhotoMedia{}}}) {
		t.Error("expected true for message with media")
	}
}

func TestPhotoFilter(t *testing.T) {
	f := Photo
	if !f(&Context{Message: &types.Message{Media: &types.PhotoMedia{}}}) {
		t.Error("expected true for photo media")
	}
	if f(&Context{Message: &types.Message{Media: &types.DocumentMedia{}}}) {
		t.Error("expected false for document media")
	}
	if f(&Context{Message: &types.Message{}}) {
		t.Error("expected false for no media")
	}
}

func TestDocumentFilter(t *testing.T) {
	f := Document
	if !f(&Context{Message: &types.Message{Media: &types.DocumentMedia{}}}) {
		t.Error("expected true for document media")
	}
	if f(&Context{Message: &types.Message{Media: &types.PhotoMedia{}}}) {
		t.Error("expected false for photo media")
	}
}

func TestMediaFilterFunc(t *testing.T) {
	f := Media
	if f(&Context{}) {
		t.Error("expected false for nil message")
	}
	if f(&Context{Message: &types.Message{}}) {
		t.Error("expected false for nil media")
	}
	if !f(&Context{Message: &types.Message{Media: &types.PhotoMedia{}}}) {
		t.Error("expected true for message with media")
	}
}

func TestCallbackDataFilter(t *testing.T) {
	f := CallbackData("click")
	if !f(&Context{CallbackQuery: &types.CallbackQuery{Data: []byte("click")}}) {
		t.Error("expected true for matching callback data")
	}
	if f(&Context{CallbackQuery: &types.CallbackQuery{Data: []byte("other")}}) {
		t.Error("expected false for non-matching data")
	}
	if f(&Context{}) {
		t.Error("expected false for nil callback query")
	}
}

func TestCallbackDataFilter_Empty(t *testing.T) {
	f := CallbackData("")
	if !f(&Context{CallbackQuery: &types.CallbackQuery{Data: []byte{}}}) {
		t.Error("expected true for empty callback data")
	}
	if f(&Context{CallbackQuery: &types.CallbackQuery{Data: []byte("x")}}) {
		t.Error("expected false for non-empty data")
	}
}

func TestInlineQueryTextFilter(t *testing.T) {
	f := InlineQueryText("search term")
	if !f(&Context{InlineQuery: &types.InlineQuery{Query: "search term"}}) {
		t.Error("expected true for matching inline query")
	}
	if f(&Context{InlineQuery: &types.InlineQuery{Query: "other"}}) {
		t.Error("expected false for non-matching query")
	}
	if f(&Context{}) {
		t.Error("expected false for nil inline query")
	}
}

func TestFilterChaining_Complex(t *testing.T) {
	f := Private.And(Text("hello")).Or(Command("start"))
	if !f(&Context{Message: &types.Message{ChatID: 1, Text: "hello"}}) {
		t.Error("expected true: private + text match")
	}
	if !f(&Context{Message: &types.Message{ChatID: -1, Text: "/start"}}) {
		t.Error("expected true: not private but command match via Or")
	}
	if f(&Context{Message: &types.Message{ChatID: -1, Text: "hello"}}) {
		t.Error("expected false: group chat with text but not command")
	}
}

func TestDispatcher_New(t *testing.T) {
	d := NewHandlerDispatcher()
	if d == nil {
		t.Fatal("expected non-nil dispatcher")
	}
	if len(d.handlers) != 0 {
		t.Error("expected empty handlers")
	}
}

func TestDispatcher_AddAndDispatch(t *testing.T) {
	d := NewHandlerDispatcher()
	called := false
	h := NewMessageHandler(func(ctx *Context) {
		called = true
	})
	d.AddHandler(h)
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1}})
	if !called {
		t.Error("expected handler to be called")
	}
}

func TestDispatcher_DispatchPopulatesContext(t *testing.T) {
	d := NewHandlerDispatcher()
	var captured *Context
	h := NewMessageHandler(func(ctx *Context) {
		captured = ctx
	})
	d.AddHandler(h)
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 42, Text: "test"}})
	if captured == nil {
		t.Fatal("expected context to be captured")
	}
	if captured.Message == nil || captured.Message.ID != 42 {
		t.Error("expected message ID 42 in context")
	}
	if captured.Message.Text != "test" {
		t.Error("expected message text 'test' in context")
	}
}

func TestDispatcher_StopPropagation(t *testing.T) {
	d := NewHandlerDispatcher()
	called1 := false
	called2 := false
	h1 := NewMessageHandler(func(ctx *Context) {
		called1 = true
		ctx.StopPropagation()
	})
	h2 := NewMessageHandler(func(ctx *Context) {
		called2 = true
	})
	d.AddHandler(h1)
	d.AddHandler(h2)
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1}})
	if !called1 {
		t.Error("expected first handler to be called")
	}
	if called2 {
		t.Error("expected second handler NOT to be called (stopped)")
	}
}

func TestDispatcher_NoStopPropagation(t *testing.T) {
	d := NewHandlerDispatcher()
	called1 := false
	called2 := false
	h1 := NewMessageHandler(func(ctx *Context) {
		called1 = true
	})
	h2 := NewMessageHandler(func(ctx *Context) {
		called2 = true
	})
	d.AddHandler(h1)
	d.AddHandler(h2)
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1}})
	if !called1 {
		t.Error("expected first handler to be called")
	}
	if !called2 {
		t.Error("expected second handler to be called (no stop)")
	}
}

func TestDispatcher_FilterBlocksDispatch(t *testing.T) {
	d := NewHandlerDispatcher()
	called := false
	h := NewMessageHandler(func(ctx *Context) {
		called = true
	}, func(ctx *Context) bool {
		return ctx.Message != nil && ctx.Message.Text == "hello"
	})
	d.AddHandler(h)
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1, Text: "bye"}})
	if called {
		t.Error("expected handler NOT to be called (filter blocked)")
	}
}

func TestDispatcher_RemoveHandler(t *testing.T) {
	d := NewHandlerDispatcher()
	called := false
	h := NewMessageHandler(func(ctx *Context) {
		called = true
	})
	d.AddHandler(h)
	d.RemoveHandler(h)
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1}})
	if called {
		t.Error("expected handler NOT to be called after removal")
	}
}

func TestDispatcher_RemoveHandler_NotFound(t *testing.T) {
	d := NewHandlerDispatcher()
	h1 := NewMessageHandler(func(ctx *Context) {})
	h2 := NewMessageHandler(func(ctx *Context) {})
	d.AddHandler(h1)
	d.RemoveHandler(h2)
	if len(d.handlers) != 1 {
		t.Error("expected 1 handler after removing non-existent handler")
	}
}

func TestDispatcher_MultipleHandlerTypes(t *testing.T) {
	d := NewHandlerDispatcher()
	msgCalled := false
	cbCalled := false
	d.AddHandler(NewMessageHandler(func(ctx *Context) { msgCalled = true }))
	d.AddHandler(NewCallbackQueryHandler(func(ctx *Context) { cbCalled = true }))

	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1}})
	if !msgCalled {
		t.Error("expected message handler to be called")
	}
	if cbCalled {
		t.Error("expected callback handler NOT to be called for message update")
	}

	msgCalled = false
	cbCalled = false
	d.Dispatch(nil, &Update{CallbackQuery: &types.CallbackQuery{ID: 1}})
	if msgCalled {
		t.Error("expected message handler NOT to be called for callback update")
	}
	if !cbCalled {
		t.Error("expected callback handler to be called")
	}
}

func TestDispatcher_RawUpdateCatchesAll(t *testing.T) {
	d := NewHandlerDispatcher()
	rawCalled := false
	msgCalled := false
	d.AddHandler(NewRawUpdateHandler(func(ctx *Context) { rawCalled = true }))
	d.AddHandler(NewMessageHandler(func(ctx *Context) { msgCalled = true }))

	d.Dispatch(nil, &Update{})
	if !rawCalled {
		t.Error("expected raw handler to be called for empty update")
	}
	if msgCalled {
		t.Error("expected message handler NOT to be called for empty update")
	}
}

func TestDispatcher_PriorityRecorded(t *testing.T) {
	d := NewHandlerDispatcher()
	h1 := NewMessageHandler(func(ctx *Context) {})
	h2 := NewMessageHandler(func(ctx *Context) {})
	h3 := NewMessageHandler(func(ctx *Context) {})
	d.AddHandler(h1, 10)
	d.AddHandler(h2, 1)
	d.AddHandler(h3, 5)
	if len(d.handlers) != 3 {
		t.Fatalf("expected 3 handlers, got %d", len(d.handlers))
	}
	if d.handlers[0].group != 10 {
		t.Errorf("expected group 10, got %d", d.handlers[0].group)
	}
	if d.handlers[1].group != 1 {
		t.Errorf("expected group 1, got %d", d.handlers[1].group)
	}
	if d.handlers[2].group != 5 {
		t.Errorf("expected group 5, got %d", d.handlers[2].group)
	}
}

func TestDispatcher_InsertionOrderDispatch(t *testing.T) {
	d := NewHandlerDispatcher()
	var order []int
	d.AddHandler(NewMessageHandler(func(ctx *Context) { order = append(order, 1) }))
	d.AddHandler(NewMessageHandler(func(ctx *Context) { order = append(order, 2) }))
	d.AddHandler(NewMessageHandler(func(ctx *Context) { order = append(order, 3) }))
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1}})
	if len(order) != 3 {
		t.Fatalf("expected 3 handlers called, got %d", len(order))
	}
	if order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("expected insertion order [1,2,3], got %v", order)
	}
}

func TestDispatcher_DispatchCallbackQueryPopulatesContext(t *testing.T) {
	d := NewHandlerDispatcher()
	var captured *Context
	d.AddHandler(NewCallbackQueryHandler(func(ctx *Context) {
		captured = ctx
	}))
	d.Dispatch(nil, &Update{CallbackQuery: &types.CallbackQuery{ID: 7, Data: []byte("test")}})
	if captured == nil {
		t.Fatal("expected context to be captured")
	}
	if captured.CallbackQuery == nil || captured.CallbackQuery.ID != 7 {
		t.Error("expected callback query ID 7 in context")
	}
}

func TestDispatcher_DispatchEditedMessage(t *testing.T) {
	d := NewHandlerDispatcher()
	called := false
	d.AddHandler(NewEditedMessageHandler(func(ctx *Context) { called = true }))
	d.Dispatch(nil, &Update{EditedMessage: &types.Message{ID: 1}})
	if !called {
		t.Error("expected edited message handler to be called")
	}
}

func TestDispatcher_DispatchDeletedMessages(t *testing.T) {
	d := NewHandlerDispatcher()
	called := false
	d.AddHandler(NewDeletedMessagesHandler(func(ctx *Context) { called = true }))
	d.Dispatch(nil, &Update{DeletedMessages: &types.DeletedMessages{Messages: []int32{1, 2}}})
	if !called {
		t.Error("expected deleted messages handler to be called")
	}
}

func TestDispatcher_DispatchInlineQuery(t *testing.T) {
	d := NewHandlerDispatcher()
	called := false
	d.AddHandler(NewInlineQueryHandler(func(ctx *Context) { called = true }))
	d.Dispatch(nil, &Update{InlineQuery: &types.InlineQuery{ID: 1}})
	if !called {
		t.Error("expected inline query handler to be called")
	}
}

func TestDispatcher_DispatchUserStatus(t *testing.T) {
	d := NewHandlerDispatcher()
	called := false
	d.AddHandler(NewUserStatusHandler(func(ctx *Context) { called = true }))
	d.Dispatch(nil, &Update{UserStatus: &types.UserStatusUpdated{UserID: 1}})
	if !called {
		t.Error("expected user status handler to be called")
	}
}

func TestDispatcher_DispatchChatMember(t *testing.T) {
	d := NewHandlerDispatcher()
	called := false
	d.AddHandler(NewChatMemberHandler(func(ctx *Context) { called = true }))
	d.Dispatch(nil, &Update{ChatMember: &types.ChatMemberUpdated{}})
	if !called {
		t.Error("expected chat member handler to be called")
	}
}

func TestDispatcher_DispatchPoll(t *testing.T) {
	d := NewHandlerDispatcher()
	called := false
	d.AddHandler(NewPollHandler(func(ctx *Context) { called = true }))
	d.Dispatch(nil, &Update{Poll: &types.PollUpdate{PollID: 1}})
	if !called {
		t.Error("expected poll handler to be called")
	}
}

func TestContext_StopPropagation(t *testing.T) {
	ctx := (&Client{}).NewContext(context.Background())
	if ctx.Stopped {
		t.Error("expected Stopped to be false initially")
	}
	ctx.StopPropagation()
	if !ctx.Stopped {
		t.Error("expected Stopped to be true after StopPropagation")
	}
}

func TestContext_NewContext(t *testing.T) {
	var c *Client
	ctx := c.NewContext(context.Background())
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if ctx.Ctx == nil {
		t.Error("expected non-nil Ctx")
	}
	if ctx.Update != nil {
		t.Error("expected nil update")
	}
}

func TestContext_ResolvePeer_User(t *testing.T) {
	ctx := (&Client{}).NewContext(context.Background())
	ctx.Update = &Update{
		Users: map[int64]*types.User{1: {ID: 1, FirstName: "Test"}},
	}
	u := ctx.ResolvePeer(1)
	if u == nil {
		t.Fatal("expected to resolve user")
	}
	user := u.(*types.User)
	if user.FirstName != "Test" {
		t.Error("expected user FirstName 'Test'")
	}
}

func TestContext_ResolvePeer_Chat(t *testing.T) {
	ctx := (&Client{}).NewContext(context.Background())
	ctx.Update = &Update{
		Chats: map[int64]*types.Chat{-100: {ID: -100, Title: "Chat"}},
	}
	c := ctx.ResolvePeer(-100)
	if c == nil {
		t.Fatal("expected to resolve chat")
	}
	chat := c.(*types.Chat)
	if chat.Title != "Chat" {
		t.Error("expected chat Title 'Chat'")
	}
}

func TestContext_ResolvePeer_NotFound(t *testing.T) {
	ctx := (&Client{}).NewContext(context.Background())
	ctx.Update = &Update{
		Users: map[int64]*types.User{},
		Chats: map[int64]*types.Chat{},
	}
	n := ctx.ResolvePeer(999)
	if n != nil {
		t.Error("expected nil for unknown peer")
	}
}

func TestContext_ResolvePeer_NilUpdate(t *testing.T) {
	ctx := (&Client{}).NewContext(context.Background())
	n := ctx.ResolvePeer(1)
	if n != nil {
		t.Error("expected nil when update is nil")
	}
}

func TestContext_ResolvePeer_NoMaps(t *testing.T) {
	ctx := (&Client{}).NewContext(context.Background())
	ctx.Update = &Update{}
	n := ctx.ResolvePeer(1)
	if n != nil {
		t.Error("expected nil when maps are nil")
	}
}

func TestIntegration_Dispatch_MessageWithFilter(t *testing.T) {
	d := NewHandlerDispatcher()
	received := ""
	d.AddHandler(NewMessageHandler(func(ctx *Context) {
		received = ctx.Message.Text
		ctx.StopPropagation()
	}, func(ctx *Context) bool {
		return ctx.Message != nil && ctx.Message.Text == "hello"
	}))
	d.AddHandler(NewMessageHandler(func(ctx *Context) {
		received = "fallback"
	}))

	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1, Text: "hello"}})
	if received != "hello" {
		t.Errorf("expected first handler to match, got %q", received)
	}
}

func TestIntegration_Dispatch_StopPropagation(t *testing.T) {
	d := NewHandlerDispatcher()
	order := []int{}
	d.AddHandler(NewMessageHandler(func(ctx *Context) {
		order = append(order, 1)
		ctx.StopPropagation()
	}))
	d.AddHandler(NewMessageHandler(func(ctx *Context) {
		order = append(order, 2)
	}))
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1}})
	if len(order) != 1 || order[0] != 1 {
		t.Errorf("expected [1], got %v", order)
	}
}

func TestIntegration_Dispatch_CallbackQuery(t *testing.T) {
	d := NewHandlerDispatcher()
	var data []byte
	d.AddHandler(NewCallbackQueryHandler(func(ctx *Context) {
		data = ctx.CallbackQuery.Data
	}))
	d.Dispatch(nil, &Update{CallbackQuery: &types.CallbackQuery{ID: 1, Data: []byte("click")}})
	if string(data) != "click" {
		t.Errorf("expected click, got %q", string(data))
	}
}

func TestIntegration_Dispatch_RawUpdateCatchAll(t *testing.T) {
	d := NewHandlerDispatcher()
	called := false
	d.AddHandler(NewRawUpdateHandler(func(ctx *Context) {
		called = true
	}))
	d.Dispatch(nil, &Update{})
	if !called {
		t.Error("expected raw handler to be called for any update")
	}
}

func TestIntegration_FilterChaining(t *testing.T) {
	d := NewHandlerDispatcher()
	received := ""
	d.AddHandler(NewMessageHandler(func(ctx *Context) {
		received = ctx.Message.Text
	}, Private.And(func(ctx *Context) bool {
		return ctx.Message.Text != ""
	})))
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1, ChatID: -100, Text: "hello"}})
	if received != "" {
		t.Error("expected no match for group chat")
	}
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 2, ChatID: 42, Text: "hello"}})
	if received != "hello" {
		t.Errorf("expected match for private chat, got %q", received)
	}
}

func TestIntegration_CommandFilter(t *testing.T) {
	d := NewHandlerDispatcher()
	received := ""
	d.AddHandler(NewMessageHandler(func(ctx *Context) {
		received = ctx.Message.Text
	}, Command("start")))
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 1, Text: "/start"}})
	if received != "/start" {
		t.Errorf("expected /start, got %q", received)
	}
	received = ""
	d.Dispatch(nil, &Update{Message: &types.Message{ID: 2, Text: "hello"}})
	if received != "" {
		t.Error("expected no match for non-command")
	}
}

func TestIntegration_ContextResolvePeer(t *testing.T) {
	d := NewHandlerDispatcher()
	var resolved interface{}
	d.AddHandler(NewMessageHandler(func(ctx *Context) {
		resolved = ctx.ResolvePeer(1)
	}))
	d.Dispatch(nil, &Update{
		Message: &types.Message{ID: 1},
		Users:   map[int64]*types.User{1: {ID: 1, FirstName: "Alice"}},
	})
	if resolved == nil {
		t.Error("expected to resolve user peer")
	}
}
