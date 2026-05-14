package types

import (
	"testing"
	"time"

	tl "github.com/mtgo-labs/mtgo/tg"
)

func TestParseStory_Nil(t *testing.T) {
	result := ParseStory(nil, nil)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestParseStory_DeletedItem(t *testing.T) {
	result := ParseStory(&tl.StoryItemDeleted{ID: 5}, nil)
	if result != nil {
		t.Fatalf("expected nil for deleted story, got %v", result)
	}
}

func TestParseStory_SkippedItem(t *testing.T) {
	result := ParseStory(&tl.StoryItemSkipped{ID: 7}, nil)
	if result != nil {
		t.Fatalf("expected nil for skipped story, got %v", result)
	}
}

func TestParseStory_BasicFields(t *testing.T) {
	caption := "Hello world"
	raw := &tl.StoryItem{
		ID:         10,
		Date:       1700000000,
		ExpireDate: 1700003600,
		Caption:    caption,
		Out:        true,
		Pinned:     true,
	}
	s := ParseStory(raw, nil)
	if s == nil {
		t.Fatal("expected non-nil story")
	}
	if s.ID != 10 {
		t.Errorf("ID = %d, want 10", s.ID)
	}
	if !s.Date.Equal(time.Unix(1700000000, 0)) {
		t.Errorf("Date = %v, want %v", s.Date, time.Unix(1700000000, 0))
	}
	if !s.ExpireDate.Equal(time.Unix(1700003600, 0)) {
		t.Errorf("ExpireDate = %v, want %v", s.ExpireDate, time.Unix(1700003600, 0))
	}
	if s.Caption != "Hello world" {
		t.Errorf("Caption = %q, want %q", s.Caption, "Hello world")
	}
	if s.Out != true {
		t.Error("Out = false, want true")
	}
	if s.Pinned != true {
		t.Error("Pinned = false, want true")
	}
}

func TestParseStory_FromID(t *testing.T) {
	raw := &tl.StoryItem{
		ID:     1,
		Date:   1700000000,
		FromID: &tl.PeerUser{UserID: 42},
	}
	s := ParseStory(raw, nil)
	if s == nil {
		t.Fatal("expected non-nil story")
	}
	if s.FromID != 42 {
		t.Errorf("FromID = %d, want 42", s.FromID)
	}
}

func TestParseStory_NilFromID(t *testing.T) {
	raw := &tl.StoryItem{
		ID:     1,
		Date:   1700000000,
		FromID: nil,
	}
	s := ParseStory(raw, nil)
	if s == nil {
		t.Fatal("expected non-nil story")
	}
	if s.FromID != 0 {
		t.Errorf("FromID = %d, want 0", s.FromID)
	}
}

func TestParseStory_NilCaption(t *testing.T) {
	raw := &tl.StoryItem{
		ID:      1,
		Date:    1700000000,
		Caption: "",
	}
	s := ParseStory(raw, nil)
	if s == nil {
		t.Fatal("expected non-nil story")
	}
	if s.Caption != "" {
		t.Errorf("Caption = %q, want empty string", s.Caption)
	}
}

func TestParseStory_ZeroExpireDate(t *testing.T) {
	raw := &tl.StoryItem{
		ID:         1,
		Date:       1700000000,
		ExpireDate: 0,
	}
	s := ParseStory(raw, nil)
	if s == nil {
		t.Fatal("expected non-nil story")
	}
	if !s.ExpireDate.IsZero() {
		t.Errorf("ExpireDate = %v, want zero", s.ExpireDate)
	}
}

func TestParseStory_BoolFlags(t *testing.T) {
	raw := &tl.StoryItem{
		ID:         1,
		Date:       1700000000,
		Public:     true,
		Edited:     true,
		Noforwards: true,
	}
	s := ParseStory(raw, nil)
	if s == nil {
		t.Fatal("expected non-nil story")
	}
	if s.Public != true {
		t.Error("Public = false, want true")
	}
	if s.Edited != true {
		t.Error("Edited = false, want true")
	}
	if s.Noforwards != true {
		t.Error("Noforwards = false, want true")
	}
}

func TestParseStory_Media(t *testing.T) {
	raw := &tl.StoryItem{
		ID:   1,
		Date: 1700000000,
		Media: &tl.MessageMediaPhoto{
			Photo: &tl.Photo{ID: 99},
		},
	}
	s := ParseStory(raw, nil)
	if s == nil {
		t.Fatal("expected non-nil story")
	}
	if s.Media == nil {
		t.Fatal("expected non-nil media")
	}
	pm, ok := s.Media.(*PhotoMedia)
	if !ok {
		t.Fatalf("expected *PhotoMedia, got %T", s.Media)
	}
	if pm.Photo == nil {
		t.Fatal("expected non-nil Photo")
	}
	if pm.Photo.ID != 99 {
		t.Errorf("Photo.ID = %d, want 99", pm.Photo.ID)
	}
}
