package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/fileid"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestGetCustomEmojiStickers(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setResult(tg.MessagesGetCustomEmojiDocumentsTypeID, &tg.GenericVector{Items: []tg.TLObject{
		&tg.Document{
			ID:         12345,
			AccessHash: 67890,
			DCID:       4,
			Date:       1700000000,
			MimeType:   "application/x-tgsticker",
			Size:       2048,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeCustomEmoji{
					Alt:       "🙂",
					TextColor: true,
					Stickerset: &tg.InputStickerSetShortName{
						ShortName: "emoji_set",
					},
				},
				&tg.DocumentAttributeImageSize{W: 64, H: 64},
				&tg.DocumentAttributeFilename{FileName: "emoji.tgs"},
			},
		},
	}})

	stickers, err := c.GetCustomEmojiStickers(context.Background(), []string{"12345"})
	if err != nil {
		t.Fatalf("GetCustomEmojiStickers() error: %v", err)
	}
	if len(stickers) != 1 {
		t.Fatalf("len(stickers) = %d, want 1", len(stickers))
	}
	sticker := stickers[0]
	if sticker.CustomEmojiID != "12345" {
		t.Errorf("CustomEmojiID = %q, want %q", sticker.CustomEmojiID, "12345")
	}
	decoded, err := fileid.Decode(sticker.FileID)
	if err != nil {
		t.Fatalf("decode FileID: %v", err)
	}
	if decoded.Type != fileid.FileTypeSticker {
		t.Errorf("FileID type = %v, want %v", decoded.Type, fileid.FileTypeSticker)
	}
	if decoded.ID != 12345 || decoded.AccessHash != 67890 || decoded.DCID != 4 {
		t.Errorf("decoded FileID = %#v, want id/access_hash/dcid 12345/67890/4", decoded)
	}
	if sticker.Emoji != "🙂" {
		t.Errorf("Emoji = %q, want 🙂", sticker.Emoji)
	}
	if !sticker.NeedsRepainting {
		t.Error("NeedsRepainting = false, want true")
	}
	if !sticker.Date.Equal(time.Unix(1700000000, 0)) {
		t.Errorf("Date = %v, want %v", sticker.Date, time.Unix(1700000000, 0))
	}

	req, ok := mock.lastCall().(*tg.MessagesGetCustomEmojiDocumentsRequest)
	if !ok {
		t.Fatalf("expected MessagesGetCustomEmojiDocumentsRequest, got %T", mock.lastCall())
	}
	if len(req.DocumentID) != 1 || req.DocumentID[0] != 12345 {
		t.Fatalf("DocumentID = %v, want [12345]", req.DocumentID)
	}
}

func TestGetCustomEmojiStickers_Empty(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	stickers, err := c.GetCustomEmojiStickers(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetCustomEmojiStickers() error: %v", err)
	}
	if len(stickers) != 0 {
		t.Fatalf("len(stickers) = %d, want 0", len(stickers))
	}
	if mock.callCount() != 0 {
		t.Fatalf("expected no RPC calls, got %d", mock.callCount())
	}
}

func TestGetCustomEmojiStickers_InvalidID(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	_, err := c.GetCustomEmojiStickers(context.Background(), []string{"bad"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.callCount() != 0 {
		t.Fatalf("expected no RPC calls, got %d", mock.callCount())
	}
}

func TestGetCustomEmojiStickers_TooManyIDs(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	ids := make([]string, 201)
	for i := range ids {
		ids[i] = "1"
	}
	_, err := c.GetCustomEmojiStickers(context.Background(), ids)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.callCount() != 0 {
		t.Fatalf("expected no RPC calls, got %d", mock.callCount())
	}
}

func TestGetCustomEmojiStickers_RPCError(t *testing.T) {
	c, mock := newClientWithBotRPCMock(t)

	mock.setError(tg.MessagesGetCustomEmojiDocumentsTypeID, errors.New("rpc error"))

	_, err := c.GetCustomEmojiStickers(context.Background(), []string{"12345"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
