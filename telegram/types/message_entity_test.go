package types

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestParseMessageEntity_TextMentionWithUser(t *testing.T) {
	entity := ParseMessageEntityWithUsers(&tg.MessageEntityMentionName{
		Offset: 1,
		Length: 4,
		UserID: 42,
	}, map[int64]*tg.User{
		42: {ID: 42, FirstName: "Ada"},
	})

	if entity == nil {
		t.Fatal("expected entity, got nil")
	}
	if entity.Type != MessageEntityTypeTextMention {
		t.Errorf("Type = %q, want %q", entity.Type, MessageEntityTypeTextMention)
	}
	if entity.Offset != 1 || entity.Length != 4 {
		t.Errorf("Offset/Length = %d/%d, want 1/4", entity.Offset, entity.Length)
	}
	if entity.UserID != 42 {
		t.Errorf("UserID = %d, want 42", entity.UserID)
	}
	if entity.User == nil || entity.User.ID != 42 || entity.User.FirstName != "Ada" {
		t.Fatalf("User = %#v, want ID 42 FirstName Ada", entity.User)
	}
}

func TestParseMessageEntity_CustomEmoji(t *testing.T) {
	entity := ParseMessageEntity(&tg.MessageEntityCustomEmoji{
		Offset:     0,
		Length:     2,
		DocumentID: 987654321,
	})

	if entity == nil {
		t.Fatal("expected entity, got nil")
	}
	if entity.Type != MessageEntityTypeCustomEmoji {
		t.Errorf("Type = %q, want %q", entity.Type, MessageEntityTypeCustomEmoji)
	}
	if entity.CustomEmojiID != "987654321" {
		t.Errorf("CustomEmojiID = %q, want %q", entity.CustomEmojiID, "987654321")
	}
}

func TestParseMessageEntity_BlockquoteExpandable(t *testing.T) {
	entity := ParseMessageEntity(&tg.MessageEntityBlockquote{
		Collapsed: true,
		Offset:    3,
		Length:    8,
	})

	if entity == nil {
		t.Fatal("expected entity, got nil")
	}
	if entity.Type != MessageEntityTypeBlockquote {
		t.Errorf("Type = %q, want %q", entity.Type, MessageEntityTypeBlockquote)
	}
	if !entity.Expandable {
		t.Fatal("Expandable = false, want true")
	}
}

func TestParseMessageEntity_FormattedDate(t *testing.T) {
	entity := ParseMessageEntity(&tg.MessageEntityFormattedDate{
		DayOfWeek: true,
		LongDate:  true,
		LongTime:  true,
		Offset:    5,
		Length:    6,
		Date:      1710000000,
	})

	if entity == nil {
		t.Fatal("expected entity, got nil")
	}
	if entity.Type != MessageEntityTypeDateTime {
		t.Errorf("Type = %q, want %q", entity.Type, MessageEntityTypeDateTime)
	}
	if entity.UnixTime != 1710000000 {
		t.Errorf("UnixTime = %d, want 1710000000", entity.UnixTime)
	}
	if entity.DateTimeFormat != "wDT" {
		t.Errorf("DateTimeFormat = %q, want %q", entity.DateTimeFormat, "wDT")
	}
}

func TestParseMessageEntity_RelativeFormattedDate(t *testing.T) {
	entity := ParseMessageEntity(&tg.MessageEntityFormattedDate{
		Relative: true,
		Offset:   5,
		Length:   6,
		Date:     1710000000,
	})

	if entity == nil {
		t.Fatal("expected entity, got nil")
	}
	if entity.DateTimeFormat != "r" {
		t.Errorf("DateTimeFormat = %q, want %q", entity.DateTimeFormat, "r")
	}
}
