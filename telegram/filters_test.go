package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/types"
)

// fakeMedia implements types.Media so the media-based filters can be exercised
// without constructing full media objects.
type fakeMedia struct{ t types.MessageMediaType }

func (m fakeMedia) MediaType() types.MessageMediaType { return m.t }

// mkCtx returns a Context whose Message is set.
func mkCtx(m *types.Message) *Context { return &Context{Message: m} }

func TestFilters_MessageBoolean(t *testing.T) {
	cases := []struct {
		name   string
		filter Filter
		ctx    *Context
		want   bool
	}{
		// HasText
		{"HasText present", HasText, mkCtx(&types.Message{Text: "hi"}), true},
		{"HasText empty", HasText, mkCtx(&types.Message{Text: ""}), false},
		{"HasText nil message", HasText, &Context{}, false},

		// Text (exact match)
		{"Text match", Text("ping"), mkCtx(&types.Message{Text: "ping"}), true},
		{"Text mismatch", Text("ping"), mkCtx(&types.Message{Text: "pong"}), false},
		{"Text nil message", Text("ping"), &Context{}, false},

		// All
		{"All true even with nil", All, &Context{}, true},

		// Me / Outgoing (both check Out)
		{"Me outgoing", Me, mkCtx(&types.Message{Out: true}), true},
		{"Me incoming", Me, mkCtx(&types.Message{Out: false}), false},
		{"Me nil", Me, &Context{}, false},
		{"Outgoing true", Outgoing, mkCtx(&types.Message{Out: true}), true},
		{"Incoming true", Incoming, mkCtx(&types.Message{Out: false}), true},

		// Private / Group / Direct
		{"Private positive id", Private, mkCtx(&types.Message{ChatID: 100}), true},
		{"Private negative id", Private, mkCtx(&types.Message{ChatID: -100}), false},
		{"Group negative id", Group, mkCtx(&types.Message{ChatID: -100}), true},
		{"Group positive id", Group, mkCtx(&types.Message{ChatID: 100}), false},
		{"Direct private incoming", Direct, mkCtx(&types.Message{ChatID: 100, Out: false}), true},
		{"Direct private outgoing", Direct, mkCtx(&types.Message{ChatID: 100, Out: true}), false},
		{"Direct group", Direct, mkCtx(&types.Message{ChatID: -100, Out: false}), false},

		// Mentioned / ViaBot / Pinned / Reply / Forwarded / LinkedChannel
		{"Mentioned true", Mentioned, mkCtx(&types.Message{Mentioned: true}), true},
		{"ViaBot set", ViaBot, mkCtx(&types.Message{ViaBotID: 42}), true},
		{"Pinned true", Pinned, mkCtx(&types.Message{Pinned: true}), true},
		{"Reply set", Reply, mkCtx(&types.Message{ReplyToID: 5}), true},
		{"Forwarded set", Forwarded, mkCtx(&types.Message{FwdFrom: &types.ForwardHeader{}}), true},
		{"LinkedChannel author", LinkedChannel, mkCtx(&types.Message{PostAuthor: "Channel"}), true},
		{"MediaGroup grouped", MediaGroup, mkCtx(&types.Message{GroupedID: 9}), true},
		{"GuestMessage pending", GuestMessage, mkCtx(&types.Message{IsFromPending: true}), true},

		// nil-message safety for several filters
		{"Mentioned nil", Mentioned, &Context{}, false},
		{"Reply nil", Reply, &Context{}, false},
		{"Forwarded nil", Forwarded, &Context{}, false},
		{"MediaGroup nil", MediaGroup, &Context{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.filter(c.ctx); got != c.want {
				t.Errorf("filter returned %v, want %v", got, c.want)
			}
		})
	}
}

func TestFilters_AlwaysFalse(t *testing.T) {
	ctx := mkCtx(&types.Message{Text: "anything"})
	for _, f := range []struct {
		name   string
		filter Filter
	}{
		{"Scheduled", Scheduled},
		{"FromScheduled", FromScheduled},
		{"Quote", Quote},
		{"Admin", Admin},
	} {
		t.Run(f.name, func(t *testing.T) {
			if got := f.filter(ctx); got != false {
				t.Errorf("%s = %v, want false (placeholder filter)", f.name, got)
			}
		})
	}
}

func TestFilters_Bot(t *testing.T) {
	bot := &types.User{IsBot: true}
	human := &types.User{IsBot: false}
	ctx := func(from int64, users map[int64]*types.User) *Context {
		return &Context{Update: &Update{Users: users}, Message: &types.Message{FromID: from}}
	}
	cases := []struct {
		name string
		ctx  *Context
		want bool
	}{
		{"bot author", ctx(1, map[int64]*types.User{1: bot}), true},
		{"human author", ctx(1, map[int64]*types.User{1: human}), false},
		{"author not in map", ctx(2, map[int64]*types.User{1: bot}), false},
		{"nil update", &Context{Message: &types.Message{FromID: 1}}, false},
		{"nil message", &Context{Update: &Update{Users: map[int64]*types.User{1: bot}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Bot(c.ctx); got != c.want {
				t.Errorf("Bot = %v, want %v", got, c.want)
			}
		})
	}
}

func TestFilters_ForumChannel(t *testing.T) {
	chats := map[int64]*types.Chat{
		100: {Type: types.ChatTypeChannel},
		200: {Type: types.ChatTypeSupergroup, IsForum: true},
		300: {Type: types.ChatTypeGroup},
	}
	channelCtx := &Context{Update: &Update{Chats: chats}, Message: &types.Message{ChatID: 100}}
	forumCtx := &Context{Update: &Update{Chats: chats}, Message: &types.Message{ChatID: 200}}
	plainCtx := &Context{Update: &Update{Chats: chats}, Message: &types.Message{ChatID: 300}}

	if !Channel(channelCtx) {
		t.Error("Channel should match a channel-type chat")
	}
	if Channel(forumCtx) {
		t.Error("Channel should not match a supergroup")
	}
	if !Forum(forumCtx) {
		t.Error("Forum should match an IsForum chat")
	}
	if Forum(plainCtx) {
		t.Error("Forum should not match a non-forum chat")
	}
	if Channel(&Context{Message: &types.Message{ChatID: 100}}) {
		t.Error("Channel should be false with nil chats map")
	}
}

func TestFilters_Media(t *testing.T) {
	photo := fakeMedia{types.MessageMediaTypePhoto}
	audio := fakeMedia{types.MessageMediaTypeAudio}
	doc := fakeMedia{types.MessageMediaTypeDocument}

	cases := []struct {
		name   string
		filter Filter
		ctx    *Context
		want   bool
	}{
		{"Photo matches", Photo, mkCtx(&types.Message{Media: photo}), true},
		{"Photo no media", Photo, mkCtx(&types.Message{}), false},
		{"Audio matches", Audio, mkCtx(&types.Message{Media: audio}), true},
		{"Document matches", Document, mkCtx(&types.Message{Media: doc}), true},
		{"Audio rejects photo", Audio, mkCtx(&types.Message{Media: photo}), false},
		{"Media any", Media, mkCtx(&types.Message{Media: photo}), true},
		{"HasMedia alias", HasMedia, mkCtx(&types.Message{Media: photo}), true},
		{"Media nil", Media, mkCtx(&types.Message{}), false},
		{"Caption media no text", Caption, mkCtx(&types.Message{Media: photo, Text: ""}), true},
		{"Caption media with text", Caption, mkCtx(&types.Message{Media: photo, Text: "cap"}), false},
		{"Caption no media", Caption, mkCtx(&types.Message{Text: ""}), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.filter(c.ctx); got != c.want {
				t.Errorf("filter returned %v, want %v", got, c.want)
			}
		})
	}
}

func TestCommand(t *testing.T) {
	f := Command("start", "help")
	cases := []struct {
		text string
		want bool
	}{
		{"/start", true},
		{"/help", true},
		{"/start@mybot", true},
		{"/start arg1 arg2", true},
		{"/START", false}, // NOTE: Command matching is case-sensitive despite the doc comment
		{"/unknown", false},
		{"start", false},  // missing leading slash
		{"", false},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			ctx := mkCtx(&types.Message{Text: c.text})
			if got := f(ctx); got != c.want {
				t.Errorf("Command(%q) = %v, want %v", c.text, got, c.want)
			}
		})
	}
}

func TestNewCommand(t *testing.T) {
	// Custom prefix "!", case-insensitive (default).
	caseInsensitive := NewCommand([]string{"start"}, []string{"!"}, false)
	caseSensitive := NewCommand([]string{"start"}, []string{"!"}, true)

	cases := []struct {
		name   string
		filter Filter
		text   string
		want   bool
	}{
		{"ci matches lowercase", caseInsensitive, "!start", true},
		{"ci matches uppercase", caseInsensitive, "!START", true},
		{"ci rejects slash prefix", caseInsensitive, "/start", false},
		{"cs matches exact", caseSensitive, "!start", true},
		{"cs rejects uppercase", caseSensitive, "!START", false},
		{"cs rejects slash", caseSensitive, "/start", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := mkCtx(&types.Message{Text: c.text})
			if got := c.filter(ctx); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}

	// Empty prefixes defaults to "/".
	def := NewCommand([]string{"ping"}, nil, false)
	if !def(mkCtx(&types.Message{Text: "/ping"})) {
		t.Error("NewCommand with nil prefixes should default to '/'")
	}
}

func TestRegex_MultiSource(t *testing.T) {
	f := Regex(`\d+`)
	cases := []struct {
		name string
		ctx  *Context
		want bool
	}{
		{"message text", &Context{Message: &types.Message{Text: "order 42"}}, true},
		{"message no digits", &Context{Message: &types.Message{Text: "no number"}}, false},
		{"callback data", &Context{CallbackQuery: &types.CallbackQuery{Data: []byte("page_7")}}, true},
		{"inline query", &Context{InlineQuery: &types.InlineQuery{Query: "item 3"}}, true},
		{"chosen inline result", &Context{ChosenInlineResult: &types.ChosenInlineResult{Query: "9"}}, true},
		{"empty context", &Context{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := f(c.ctx); got != c.want {
				t.Errorf("Regex = %v, want %v", got, c.want)
			}
		})
	}
}

func TestFilters_Constructors(t *testing.T) {
	t.Run("User", func(t *testing.T) {
		f := User(int64(10), int64(20))
		if !f(&Context{Message: &types.Message{FromID: 10}}) {
			t.Error("User should match Message.FromID")
		}
		if !f(&Context{CallbackQuery: &types.CallbackQuery{UserID: 20}}) {
			t.Error("User should match CallbackQuery.UserID")
		}
		if !f(&Context{InlineQuery: &types.InlineQuery{UserID: 10}}) {
			t.Error("User should match InlineQuery.UserID")
		}
		if f(&Context{Message: &types.Message{FromID: 99}}) {
			t.Error("User should not match unlisted id")
		}
	})

	t.Run("Chat", func(t *testing.T) {
		f := Chat(int64(-1))
		if !f(&Context{Message: &types.Message{ChatID: -1}}) {
			t.Error("Chat should match Message.ChatID")
		}
		if !f(&Context{EditedMessage: &types.Message{ChatID: -1}}) {
			t.Error("Chat should match EditedMessage.ChatID")
		}
		if !f(&Context{CallbackQuery: &types.CallbackQuery{ChatID: -1}}) {
			t.Error("Chat should match CallbackQuery.ChatID")
		}
		if f(&Context{Message: &types.Message{ChatID: -2}}) {
			t.Error("Chat should not match unlisted chat")
		}
	})

	t.Run("Topic", func(t *testing.T) {
		f := Topic(int32(42))
		if !f(&Context{Message: &types.Message{TopicID: 42}}) {
			t.Error("Topic should match")
		}
		if f(&Context{Message: &types.Message{TopicID: 7}}) {
			t.Error("Topic should not match other id")
		}
		if f(&Context{}) {
			t.Error("Topic should be false with nil message")
		}
	})

	t.Run("SenderChat", func(t *testing.T) {
		f := SenderChat(int64(-100))
		if !f(&Context{Message: &types.Message{SenderChatID: -100}}) {
			t.Error("SenderChat should match")
		}
		if f(&Context{Message: &types.Message{SenderChatID: -200}}) {
			t.Error("SenderChat should not match other id")
		}
		if f(&Context{Message: &types.Message{}}) {
			t.Error("SenderChat should be false when SenderChatID is 0")
		}
	})

	t.Run("CallbackData", func(t *testing.T) {
		f := CallbackData("approve")
		if !f(&Context{CallbackQuery: &types.CallbackQuery{Data: []byte("approve")}}) {
			t.Error("CallbackData should match exact")
		}
		if f(&Context{CallbackQuery: &types.CallbackQuery{Data: []byte("approved")}}) {
			t.Error("CallbackData should not match partial")
		}
		if f(&Context{}) {
			t.Error("CallbackData should be false with nil callback")
		}
	})

	t.Run("CallbackRegex", func(t *testing.T) {
		f := CallbackRegex(`^page_\d+$`)
		if !f(&Context{CallbackQuery: &types.CallbackQuery{Data: []byte("page_3")}}) {
			t.Error("CallbackRegex should match")
		}
		if f(&Context{CallbackQuery: &types.CallbackQuery{Data: []byte("page_")}}) {
			t.Error("CallbackRegex should not match without digits")
		}
	})

	t.Run("InlineQueryText", func(t *testing.T) {
		f := InlineQueryText("search")
		if !f(&Context{InlineQuery: &types.InlineQuery{Query: "search"}}) {
			t.Error("InlineQueryText should match")
		}
		if f(&Context{InlineQuery: &types.InlineQuery{Query: "search more"}}) {
			t.Error("InlineQueryText should require exact match")
		}
		if f(&Context{}) {
			t.Error("InlineQueryText should be false with nil inline query")
		}
	})
}
