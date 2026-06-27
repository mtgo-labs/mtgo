package types

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

// --- User parsing ---

func BenchmarkParseUser(b *testing.B) {
	raw := &tg.User{
		ID:         123456789,
		FirstName:  "Test",
		LastName:   "User",
		Username:   "testuser",
		Phone:      "1234567890",
		AccessHash: 987654321,
		Bot:        true,
	}
	b.ReportAllocs()
	for b.Loop() {
		ParseUser(raw)
	}
}

// --- Chat parsing ---

func BenchmarkParseChatFromChat(b *testing.B) {
	raw := &tg.Chat{
		ID:                123456,
		Title:             "Test Group",
		ParticipantsCount: 100,
		Date:              1700000000,
		Version:           1,
	}
	b.ReportAllocs()
	for b.Loop() {
		ParseChatFromChat(raw)
	}
}

// --- Message entity parsing ---

func BenchmarkParseMessageEntityBold(b *testing.B) {
	raw := &tg.MessageEntityBold{Offset: 0, Length: 10}
	b.ReportAllocs()
	for b.Loop() {
		ParseMessageEntity(raw)
	}
}

func BenchmarkParseMessageEntityTextLink(b *testing.B) {
	raw := &tg.MessageEntityTextURL{
		Offset: 0,
		Length: 20,
		URL:    "https://example.com/very/long/url/path",
	}
	b.ReportAllocs()
	for b.Loop() {
		ParseMessageEntity(raw)
	}
}

func BenchmarkParseMessageEntities(b *testing.B) {
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityBold{Offset: 0, Length: 5},
		&tg.MessageEntityItalic{Offset: 6, Length: 3},
		&tg.MessageEntityCode{Offset: 10, Length: 8},
		&tg.MessageEntityURL{Offset: 19, Length: 15},
		&tg.MessageEntityMention{Offset: 35, Length: 8},
	}
	b.ReportAllocs()
	for b.Loop() {
		ParseMessageEntities(entities)
	}
}

// --- Message parsing ---

func BenchmarkParseMessage(b *testing.B) {
	raw := &tg.Message{
		ID:      12345,
		PeerID:  &tg.PeerUser{UserID: 123456789},
		FromID:  &tg.PeerUser{UserID: 987654321},
		Message: "Hello, this is a test message with some length",
		Date:    1700000000,
		Entities: []tg.MessageEntityClass{
			&tg.MessageEntityBold{Offset: 0, Length: 5},
			&tg.MessageEntityCode{Offset: 7, Length: 4},
		},
	}
	pm := &PeerMap{
		Users: map[int64]*tg.User{
			123456789: {ID: 123456789, FirstName: "Test", AccessHash: 1},
			987654321: {ID: 987654321, FirstName: "Other", AccessHash: 2},
		},
	}
	b.ReportAllocs()
	for b.Loop() {
		ParseMessage(raw, pm)
	}
}

// --- Peer map ---

func BenchmarkPeerMapResolve(b *testing.B) {
	pm := &PeerMap{
		Users: map[int64]*tg.User{
			123456789: {ID: 123456789, FirstName: "Test", AccessHash: 1},
		},
		Chats: map[int64]*tg.Chat{
			100: {ID: 100, Title: "Group"},
		},
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = pm.Users[123456789]
	}
}
