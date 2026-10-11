package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

// TestCopyMessageCaptionReSend asserts the caption-override path: the source
// message is fetched via messages.getMessages and re-sent through
// messages.sendMedia with the replacement caption, because
// messages.forwardMessages has no caption field.
func TestCopyMessageCaptionReSend(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.MessagesGetMessagesTypeID, &tg.MessagesMessages{
		Messages: []tg.MessageClass{&tg.Message{
			ID:     55,
			PeerID: &tg.PeerUser{UserID: 10},
			Media: &tg.MessageMediaPhoto{Photo: &tg.Photo{
				ID:            777,
				AccessHash:    888,
				FileReference: []byte{1, 2, 3},
			}},
			Message: "original caption",
		}},
		Users: []tg.UserClass{},
		Chats: []tg.ChatClass{},
	})

	newID, err := c.CopyMessage(context.Background(), 10, 10, 55,
		&params.CopyMessage{Caption: "replaced caption"})
	if err != nil {
		t.Fatalf("CopyMessage(caption) error: %v", err)
	}
	if int32(newID) != 1 {
		t.Errorf("new message id = %d, want 1 (mock)", newID)
	}

	// first call: getMessages; second: sendMedia
	req, ok := inv.lastCall().(*tg.MessagesSendMediaRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMediaRequest, got %T", inv.lastCall())
	}
	if req.Message != "replaced caption" {
		t.Errorf("caption = %q, want %q", req.Message, "replaced caption")
	}
	photo, ok := req.Media.(*tg.InputMediaPhoto)
	if !ok {
		t.Fatalf("expected InputMediaPhoto, got %T", req.Media)
	}
	ref, ok := photo.ID.(*tg.InputPhoto)
	if !ok {
		t.Fatalf("expected InputPhoto, got %T", photo.ID)
	}
	if ref.ID != 777 || ref.AccessHash != 888 || len(ref.FileReference) != 3 {
		t.Errorf("photo ref mismatch: %+v", ref)
	}
}

// TestCopyMessageNoCaptionStillForwards asserts plain copies keep using the
// forwardMessages drop_author path.
func TestCopyMessageNoCaptionStillForwards(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})

	if _, err := c.CopyMessage(context.Background(), 10, 10, 55); err != nil {
		t.Fatalf("CopyMessage() error: %v", err)
	}
	if inv.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", inv.callCount())
	}
	req, ok := inv.lastCall().(*tg.MessagesForwardMessagesRequest)
	if !ok {
		t.Fatalf("expected MessagesForwardMessagesRequest, got %T", inv.lastCall())
	}
	if req.DropMediaCaptions {
		t.Error("plain copy must not strip captions")
	}
}
