package generic

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
)

func assertPeerInput[T PeerInput](t *testing.T, v T) {
	_ = v
	_ = t
}

func TestInputPeerToID(t *testing.T) {
	tests := []struct {
		name string
		peer tg.InputPeerClass
		want int64
	}{
		{"user", &tg.InputPeerUser{UserID: 42}, 42},
		{"chat", &tg.InputPeerChat{ChatID: 100}, -100},
		{"channel", &tg.InputPeerChannel{ChannelID: 555}, -1000000000555},
		{"self", &tg.InputPeerSelf{}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := inputPeerToID(tc.peer)
			if got != tc.want {
				t.Errorf("inputPeerToID(%T) = %d, want %d", tc.peer, got, tc.want)
			}
		})
	}
}

func TestResolveIDNumeric(t *testing.T) {
	ctx := t.Context()

	id, err := resolveID[int](ctx, nil, 42)
	if err != nil || id != 42 {
		t.Fatalf("resolveID[int](42) = (%d, %v), want (42, nil)", id, err)
	}

	id, err = resolveID[int64](ctx, nil, -100)
	if err != nil || id != -100 {
		t.Fatalf("resolveID[int64](-100) = (%d, %v), want (-100, nil)", id, err)
	}
}

func TestResolveIDNumericString(t *testing.T) {
	ctx := t.Context()

	// Numeric strings should parse without network calls — nil client is fine.
	id, err := resolveID[string](ctx, nil, "12345")
	if err != nil || id != 12345 {
		t.Fatalf(`resolveID[string]("12345") = (%d, %v), want (12345, nil)`, id, err)
	}

	id, err = resolveID[string](ctx, nil, "-1001234567890")
	if err != nil || id != -1001234567890 {
		t.Fatalf(`resolveID[string]("-1001234567890") = (%d, %v), want (-1001234567890, nil)`, id, err)
	}
}

func TestAsMessageUpdates(t *testing.T) {
	rawMsg := &tg.Message{
		ID:      42,
		PeerID:  &tg.PeerUser{UserID: 1},
		FromID:  &tg.PeerUser{UserID: 2},
		Message: "hello",
	}
	result := &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewMessage{Message: rawMsg},
		},
	}

	msg, err := AsMessage(result)
	if err != nil {
		t.Fatalf("AsMessage: %v", err)
	}
	if msg == nil || msg.ID != 42 {
		t.Fatalf("AsMessage: got %v, want message ID 42", msg)
	}
}

func TestAsMessageEmptyUpdates(t *testing.T) {
	result := &tg.Updates{Updates: nil}
	_, err := AsMessage(result)
	if err != ErrNoMessageUpdates {
		t.Fatalf("AsMessage empty updates: got %v, want %v", err, ErrNoMessageUpdates)
	}
}

func TestAsMessageUpdateShortSentMessage(t *testing.T) {
	result := &tg.UpdateShortSentMessage{ID: 99}
	msg, err := AsMessage(result)
	if err != nil {
		t.Fatalf("AsMessage: %v", err)
	}
	if msg == nil || msg.ID != 99 {
		t.Fatalf("AsMessage: got %v, want message ID 99", msg)
	}
}

func TestAsMessages(t *testing.T) {
	result := &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewMessage{Message: &tg.Message{ID: 1, Message: "a"}},
			&tg.UpdateNewChannelMessage{Message: &tg.Message{ID: 2, Message: "b"}},
		},
	}

	msgs, err := AsMessages(result)
	if err != nil {
		t.Fatalf("AsMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("AsMessages: got %d messages, want 2", len(msgs))
	}
}

func TestExtractUsers(t *testing.T) {
	result := &tg.Updates{
		Users: []tg.UserClass{
			&tg.User{ID: 10, FirstName: "Alice"},
			&tg.User{ID: 20, FirstName: "Bob"},
			// Duplicate ID should be deduplicated.
			&tg.User{ID: 10, FirstName: "Alice2"},
		},
	}

	users, err := ExtractUsers(result)
	if err != nil {
		t.Fatalf("ExtractUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("ExtractUsers: got %d users, want 2 (deduplicated)", len(users))
	}
}

func TestExtractChats(t *testing.T) {
	result := &tg.Updates{
		Chats: []tg.ChatClass{
			&tg.Chat{ID: 100, Title: "Group"},
			&tg.Channel{ID: 200, Title: "Channel"},
		},
	}

	chats, err := ExtractChats(result)
	if err != nil {
		t.Fatalf("ExtractChats: %v", err)
	}
	if len(chats) != 2 {
		t.Fatalf("ExtractChats: got %d chats, want 2", len(chats))
	}
}

func TestExtractChatsDeduplication(t *testing.T) {
	result := &tg.Updates{
		Chats: []tg.ChatClass{
			&tg.Chat{ID: 100, Title: "Group"},
			&tg.Chat{ID: 100, Title: "Group2"},
		},
	}

	chats, err := ExtractChats(result)
	if err != nil {
		t.Fatalf("ExtractChats: %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("ExtractChats: got %d chats, want 1 (deduplicated)", len(chats))
	}
}

func TestExtractClient(t *testing.T) {
	ctx := t.Context()
	client := &telegram.Client{}

	gotClient, gotCtx := extractClient(client)
	if gotClient != client {
		t.Errorf("extractClient(*Client).client = %v, want %v", gotClient, client)
	}
	if gotCtx != context.Background() {
		t.Errorf("extractClient(*Client).ctx = %v, want context.Background()", gotCtx)
	}

	tc := &telegram.Context{Ctx: ctx, Client: client}
	gotClient, gotCtx = extractClient(tc)
	if gotClient != client {
		t.Errorf("extractClient(*Context).client = %v, want %v", gotClient, client)
	}
	if gotCtx != ctx {
		t.Errorf("extractClient(*Context).ctx = %v, want %v", gotCtx, ctx)
	}
}

func TestGeneratedWrappersCompile(t *testing.T) {
	// Compile-time check: the generated generic wrappers must accept int,
	// int64, and string as the chat/user identifier type, and both
	// *telegram.Client and *telegram.Context as the Caller type.
	_ = SendMessage[int, *telegram.Client]
	_ = SendMessage[int64, *telegram.Client]
	_ = SendMessage[string, *telegram.Client]
	_ = SendMessage[int, *telegram.Context]
	_ = SendMessage[string, *telegram.Context]
	_ = GetUser[int, *telegram.Client]
	_ = GetUser[int64, *telegram.Client]
	_ = GetUser[string, *telegram.Client]
	_ = GetUser[int, *telegram.Context]
	_ = BanChatMember[int, *telegram.Client]
}

// Compile-time verification that both *telegram.Client and *telegram.Context
// satisfy the Caller constraint and can be used as the first argument to
// generated wrappers.
func assertCallerSatisfies[C Caller]() {}

func TestCallerConstraint(t *testing.T) {
	assertCallerSatisfies[*telegram.Client]()
	assertCallerSatisfies[*telegram.Context]()
}
