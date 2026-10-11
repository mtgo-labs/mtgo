package telegram

import (
	"bytes"
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

// TestForwardMessagesFlagBits guards against hand-set flag bits leaking
// unrelated fields onto the wire. Per the TL schema
// (messages.forwardMessages#13704a7c): drop_author is flags.11,
// drop_media_captions is flags.12, while bit 8 is with_my_score and
// bit 9 is top_msg_id.
func TestForwardMessagesFlagBits(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.ForwardMessages(
		context.Background(), 10, 10, []int32{1, 2},
		&params.ForwardMessages{
			DisableNotification: true,
			DropAuthor:          true,
			DropMediaCaptions:   true,
			NoForwards:          true,
		},
	)
	if err != nil {
		t.Fatalf("ForwardMessages() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesForwardMessagesRequest)
	if !ok {
		t.Fatalf("expected MessagesForwardMessagesRequest, got %T", inv.lastCall())
	}
	if !req.Flags.Has(5) {
		t.Errorf("silent flag 5 not set, flags=%032b", uint32(req.Flags))
	}
	if !req.Flags.Has(11) {
		t.Errorf("drop_author flag 11 not set, flags=%032b", uint32(req.Flags))
	}
	if !req.Flags.Has(12) {
		t.Errorf("drop_media_captions flag 12 not set, flags=%032b", uint32(req.Flags))
	}
	if !req.Flags.Has(14) {
		t.Errorf("noforwards flag 14 not set, flags=%032b", uint32(req.Flags))
	}
	if req.Flags.Has(8) {
		t.Errorf("with_my_score bit 8 leaked, flags=%032b", uint32(req.Flags))
	}
	if req.Flags.Has(9) {
		t.Errorf("top_msg_id bit 9 leaked, flags=%032b", uint32(req.Flags))
	}

	// Encoding recomputes flags (OR-only); assert the wire form too.
	before := uint32(req.Flags)
	buf := new(bytes.Buffer)
	if err := req.Encode(buf); err != nil {
		t.Fatalf("Encode() error: %v", err)
	}
	if uint32(req.Flags) != before {
		t.Errorf("Encode changed flags: before=%032b after=%032b", before, uint32(req.Flags))
	}
}

// TestForwardMessagesPlainFlags asserts a plain forward sets no optional bits.
func TestForwardMessagesPlainFlags(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	if _, err := c.ForwardMessages(context.Background(), 10, 10, []int32{7}); err != nil {
		t.Fatalf("ForwardMessages() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesForwardMessagesRequest)
	if !ok {
		t.Fatalf("expected MessagesForwardMessagesRequest, got %T", inv.lastCall())
	}
	if uint32(req.Flags) != 0 {
		t.Errorf("plain forward should set no flags, got %032b", uint32(req.Flags))
	}
}

// TestCopyMessageFlagBits covers the no-caption CopyMessage path: DropAuthor
// must map to flags.11 (not 8) and a reply must map to flags.22 (not bit 0).
// Caption overrides are covered by TestCopyMessageCaptionReSend.
func TestCopyMessageFlagBits(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.CopyMessage(context.Background(), 10, 10, 5,
		&params.CopyMessage{
			DropAuthor:       true,
			ReplyToMessageID: 42,
		},
	)
	if err != nil {
		t.Fatalf("CopyMessage() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesForwardMessagesRequest)
	if !ok {
		t.Fatalf("expected MessagesForwardMessagesRequest, got %T", inv.lastCall())
	}
	if !req.Flags.Has(11) {
		t.Errorf("drop_author flag 11 not set, flags=%032b", uint32(req.Flags))
	}
	if req.Flags.Has(12) {
		t.Errorf("drop_media_captions flag 12 set on plain copy, flags=%032b", uint32(req.Flags))
	}
	if !req.Flags.Has(22) {
		t.Errorf("reply_to flag 22 not set, flags=%032b", uint32(req.Flags))
	}
	if req.Flags.Has(8) {
		t.Errorf("with_my_score bit 8 leaked, flags=%032b", uint32(req.Flags))
	}
	if req.Flags.Has(9) {
		t.Errorf("top_msg_id bit 9 leaked, flags=%032b", uint32(req.Flags))
	}
}
