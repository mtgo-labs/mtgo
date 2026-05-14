package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSendPhoto_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.SendPhoto(context.Background(), 0, FromBytes([]byte("x"), "photo.jpg"), "caption")
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestSendDocument_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.SendDocument(context.Background(), 0, FromBytes([]byte("x"), "doc.pdf"), "caption")
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestBuildInputMediaUploadedPhoto(t *testing.T) {
	inputFile := &tg.InputFile{
		ID:    123,
		Parts: 1,
		Name:  "photo.jpg",
	}

	media := buildInputMediaUploadedPhoto(inputFile)
	if media == nil {
		t.Fatal("expected non-nil media")
	}
	uploaded, ok := media.(*tg.InputMediaUploadedPhoto)
	if !ok {
		t.Fatalf("expected *tg.InputMediaUploadedPhoto, got %T", media)
	}
	if uploaded.File != inputFile {
		t.Error("File mismatch")
	}
}

func TestBuildInputMediaUploadedDocument(t *testing.T) {
	inputFile := &tg.InputFile{
		ID:    456,
		Parts: 1,
		Name:  "doc.pdf",
	}

	attrs := []tg.DocumentAttributeClass{
		&tg.DocumentAttributeFilename{FileName: "doc.pdf"},
	}

	media := buildInputMediaUploadedDocument(inputFile, nil, "application/pdf", attrs)
	if media == nil {
		t.Fatal("expected non-nil media")
	}
	uploaded, ok := media.(*tg.InputMediaUploadedDocument)
	if !ok {
		t.Fatalf("expected *tg.InputMediaUploadedDocument, got %T", media)
	}
	if uploaded.File != inputFile {
		t.Error("File mismatch")
	}
	if uploaded.MimeType != "application/pdf" {
		t.Errorf("MimeType = %q, want %q", uploaded.MimeType, "application/pdf")
	}
}

func TestBuildDocumentAttributes(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		mime     string
		want     int
	}{
		{"video gets video attr", "clip.mp4", "video/mp4", 2},
		{"audio gets audio attr", "song.mp3", "audio/mpeg", 2},
		{"plain doc gets filename", "readme.txt", "text/plain", 1},
		{"sticker webp", "sticker.webp", "image/webp", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := buildDocumentAttributes(tt.fileName, tt.mime, 0, 0)
			if len(attrs) < tt.want {
				t.Errorf("got %d attrs, want at least %d", len(attrs), tt.want)
			}

			hasFilename := false
			for _, a := range attrs {
				if fn, ok := a.(*tg.DocumentAttributeFilename); ok {
					hasFilename = true
					if fn.FileName != tt.fileName {
						t.Errorf("filename = %q, want %q", fn.FileName, tt.fileName)
					}
				}
			}
			if !hasFilename {
				t.Error("expected DocumentAttributeFilename")
			}
		})
	}
}

func TestSendMediaMethodsGuardNotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	ctx := context.Background()
	data := FromBytes(make([]byte, 100), "test")

	methods := []struct {
		name string
		fn   func() error
	}{
		{"SendPhoto", func() error { _, err := c.SendPhoto(ctx, 0, data, ""); return err }},
		{"SendDocument", func() error { _, err := c.SendDocument(ctx, 0, data, ""); return err }},
		{"SendVideo", func() error { _, err := c.SendVideo(ctx, 0, data, ""); return err }},
		{"SendAudio", func() error { _, err := c.SendAudio(ctx, 0, data, ""); return err }},
		{"SendAnimation", func() error { _, err := c.SendAnimation(ctx, 0, data, ""); return err }},
		{"SendVoice", func() error { _, err := c.SendVoice(ctx, 0, data, ""); return err }},
		{"SendVideoNote", func() error { _, err := c.SendVideoNote(ctx, 0, data); return err }},
		{"SendSticker", func() error { _, err := c.SendSticker(ctx, 0, data); return err }},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			if err := m.fn(); err == nil {
				t.Errorf("%s: expected error when not connected", m.name)
			}
		})
	}
}
