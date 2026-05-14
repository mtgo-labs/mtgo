package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestGetFileLocation_PhotoMedia(t *testing.T) {
	media := &types.PhotoMedia{
		Photo: &types.Photo{
			ID:         123,
			AccessHash: 456,
			Sizes: []types.PhotoSize{
				{Type: "s", Width: 90, Height: 90, Size: 1000},
				{Type: "m", Width: 320, Height: 320, Size: 5000},
				{Type: "x", Width: 800, Height: 800, Size: 20000},
			},
		},
	}

	loc, _, err := GetFileLocation(media, "x")
	if err != nil {
		t.Fatalf("GetFileLocation() error: %v", err)
	}
	photoLoc, ok := loc.(*tg.InputPhotoFileLocation)
	if !ok {
		t.Fatalf("expected *tg.InputPhotoFileLocation, got %T", loc)
	}
	if photoLoc.ID != 123 {
		t.Errorf("ID = %d, want 123", photoLoc.ID)
	}
	if photoLoc.AccessHash != 456 {
		t.Errorf("AccessHash = %d, want 456", photoLoc.AccessHash)
	}
	if photoLoc.ThumbSize != "x" {
		t.Errorf("ThumbSize = %q, want %q", photoLoc.ThumbSize, "x")
	}
}

func TestGetFileLocation_PhotoMediaDefaultSize(t *testing.T) {
	media := &types.PhotoMedia{
		Photo: &types.Photo{
			ID:         123,
			AccessHash: 456,
			Sizes: []types.PhotoSize{
				{Type: "s", Width: 90, Height: 90, Size: 1000},
				{Type: "x", Width: 800, Height: 800, Size: 20000},
			},
		},
	}

	loc, _, err := GetFileLocation(media, "")
	if err != nil {
		t.Fatalf("GetFileLocation() error: %v", err)
	}
	photoLoc, ok := loc.(*tg.InputPhotoFileLocation)
	if !ok {
		t.Fatalf("expected *tg.InputPhotoFileLocation, got %T", loc)
	}
	if photoLoc.ThumbSize != "x" {
		t.Errorf("ThumbSize = %q, want %q (largest)", photoLoc.ThumbSize, "x")
	}
}

func TestGetFileLocation_PhotoMediaNoSizes(t *testing.T) {
	media := &types.PhotoMedia{
		Photo: &types.Photo{
			ID:         123,
			AccessHash: 456,
		},
	}

	_, _, err := GetFileLocation(media, "x")
	if err == nil {
		t.Fatal("expected error for photo with no sizes")
	}
}

func TestGetFileLocation_DocumentMedia(t *testing.T) {
	media := &types.DocumentMedia{
		FileID:   "123_456",
		FileName: "doc.pdf",
		MimeType: "application/pdf",
		FileSize: 1024,
	}

	loc, _, err := GetFileLocation(media, "")
	if err != nil {
		t.Fatalf("GetFileLocation() error: %v", err)
	}
	docLoc, ok := loc.(*tg.InputDocumentFileLocation)
	if !ok {
		t.Fatalf("expected *tg.InputDocumentFileLocation, got %T", loc)
	}
	if docLoc.ID != 123 {
		t.Errorf("ID = %d, want 123", docLoc.ID)
	}
	if docLoc.AccessHash != 456 {
		t.Errorf("AccessHash = %d, want 456", docLoc.AccessHash)
	}
}

func TestGetFileLocation_DocumentMediaThumb(t *testing.T) {
	media := &types.DocumentMedia{
		FileID:   "123_456",
		FileName: "video.mp4",
	}

	loc, _, err := GetFileLocation(media, "x")
	if err != nil {
		t.Fatalf("GetFileLocation() error: %v", err)
	}
	docLoc, ok := loc.(*tg.InputDocumentFileLocation)
	if !ok {
		t.Fatalf("expected *tg.InputDocumentFileLocation, got %T", loc)
	}
	if docLoc.ThumbSize != "x" {
		t.Errorf("ThumbSize = %q, want %q", docLoc.ThumbSize, "x")
	}
}

func TestGetFileLocation_UnsupportedMedia(t *testing.T) {
	media := &types.ContactMedia{
		PhoneNumber: "+1234567890",
		FirstName:   "Test",
	}

	_, _, err := GetFileLocation(media, "")
	if err == nil {
		t.Fatal("expected error for contact media")
	}
}

func TestGetFileLocation_NilMedia(t *testing.T) {
	_, _, err := GetFileLocation(nil, "")
	if err == nil {
		t.Fatal("expected error for nil media")
	}
}

func TestGetPhotoSize(t *testing.T) {
	sizes := []types.PhotoSize{
		{Type: "s", Size: 500},
		{Type: "m", Size: 2000},
		{Type: "x", Size: 8000},
		{Type: "y", Size: 16000},
	}

	tests := []struct {
		name      string
		thumbSize string
		want      string
	}{
		{"specific size m", "m", "m"},
		{"specific size x", "x", "x"},
		{"empty defaults to largest", "", "y"},
		{"missing size falls back to largest", "z", "y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPhotoSize(sizes, tt.thumbSize)
			if got == nil {
				t.Fatal("got nil size")
			}
			if got.Type != tt.want {
				t.Errorf("got type %q, want %q", got.Type, tt.want)
			}
		})
	}
}

func TestGetPhotoSize_EmptySizes(t *testing.T) {
	got := getPhotoSize(nil, "x")
	if got != nil {
		t.Errorf("expected nil for empty sizes, got %+v", got)
	}
}

func TestFileLocationFromTL(t *testing.T) {
	tests := []struct {
		name    string
		media   tg.MessageMediaClass
		want    string
		wantErr bool
	}{
		{
			name: "photo media",
			media: &tg.MessageMediaPhoto{
				Photo: &tg.Photo{
					ID:            100,
					AccessHash:    200,
					DCID:          4,
					FileReference: []byte{1, 2, 3},
					Sizes: []tg.PhotoSizeClass{
						&tg.PhotoSize{Type: "x", W: 800, H: 800, Size: 20000},
					},
				},
			},
			want:    "photo",
			wantErr: false,
		},
		{
			name: "document media",
			media: &tg.MessageMediaDocument{
				Document: &tg.Document{
					ID:            300,
					AccessHash:    400,
					DCID:          2,
					FileReference: []byte{4, 5, 6},
					Size:          1024,
				},
			},
			want:    "document",
			wantErr: false,
		},
		{
			name:    "empty media",
			media:   &tg.MessageMediaEmpty{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, _, err := getFileLocationFromTL(tt.media, "x")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if loc == nil {
				t.Fatal("got nil location")
			}
		})
	}
}

func TestGetFileLocationFromTL_PhotoLocation(t *testing.T) {
	media := &tg.MessageMediaPhoto{
		Photo: &tg.Photo{
			ID:            100,
			AccessHash:    200,
			DCID:          4,
			FileReference: []byte{1, 2, 3},
			Sizes: []tg.PhotoSizeClass{
				&tg.PhotoSize{Type: "m", W: 320, H: 320, Size: 5000},
				&tg.PhotoSize{Type: "x", W: 800, H: 800, Size: 20000},
			},
		},
	}

	loc, dcID, err := getFileLocationFromTL(media, "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	photoLoc, ok := loc.(*tg.InputPhotoFileLocation)
	if !ok {
		t.Fatalf("expected *tg.InputPhotoFileLocation, got %T", loc)
	}
	if photoLoc.ID != 100 {
		t.Errorf("ID = %d, want 100", photoLoc.ID)
	}
	if photoLoc.AccessHash != 200 {
		t.Errorf("AccessHash = %d, want 200", photoLoc.AccessHash)
	}
	if photoLoc.ThumbSize != "x" {
		t.Errorf("ThumbSize = %q, want %q", photoLoc.ThumbSize, "x")
	}
	if dcID != 4 {
		t.Errorf("dcID = %d, want 4", dcID)
	}
}

func TestGetFileLocationFromTL_DocumentLocation(t *testing.T) {
	media := &tg.MessageMediaDocument{
		Document: &tg.Document{
			ID:            300,
			AccessHash:    400,
			DCID:          2,
			FileReference: []byte{4, 5, 6},
			Size:          1024,
		},
	}

	loc, dcID, err := getFileLocationFromTL(media, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	docLoc, ok := loc.(*tg.InputDocumentFileLocation)
	if !ok {
		t.Fatalf("expected *tg.InputDocumentFileLocation, got %T", loc)
	}
	if docLoc.ID != 300 {
		t.Errorf("ID = %d, want 300", docLoc.ID)
	}
	if docLoc.AccessHash != 400 {
		t.Errorf("AccessHash = %d, want 400", docLoc.AccessHash)
	}
	if dcID != 2 {
		t.Errorf("dcID = %d, want 2", dcID)
	}
}
