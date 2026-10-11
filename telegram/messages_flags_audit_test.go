package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
)

// TestPinMessageFlagBits guards messages.updatePinnedMessage#d2aaf7ec:
// silent=flags.0, unpin=flags.1, pm_oneside=flags.2. Bit 5 does not exist.
func TestPinMessageFlagBits(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	if _, err := c.PinMessage(context.Background(), 10, 7); err != nil {
		t.Fatalf("PinMessage() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesUpdatePinnedMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesUpdatePinnedMessageRequest, got %T", inv.lastCall())
	}
	if uint32(req.Flags) != 0 {
		t.Errorf("plain pin should set no flags, got %032b", uint32(req.Flags))
	}

	if _, err := c.PinMessage(context.Background(), 10, 7, &params.PinMessage{Silent: true}); err != nil {
		t.Fatalf("PinMessage(silent) error: %v", err)
	}
	req, ok = inv.lastCall().(*tg.MessagesUpdatePinnedMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesUpdatePinnedMessageRequest, got %T", inv.lastCall())
	}
	if !req.Flags.Has(0) {
		t.Errorf("silent flag 0 not set, flags=%032b", uint32(req.Flags))
	}
	if req.Flags.Has(5) {
		t.Errorf("nonexistent bit 5 leaked, flags=%032b", uint32(req.Flags))
	}
}

// TestUnpinMessageFlagBits asserts unpin sets only flags.1 (bit 0 is silent).
func TestUnpinMessageFlagBits(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	if _, err := c.UnpinMessage(context.Background(), 10, 7); err != nil {
		t.Fatalf("UnpinMessage() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesUpdatePinnedMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesUpdatePinnedMessageRequest, got %T", inv.lastCall())
	}
	if !req.Unpin || !req.Flags.Has(1) {
		t.Errorf("unpin flag 1 not set, flags=%032b unpin=%v", uint32(req.Flags), req.Unpin)
	}
	if req.Flags.Has(0) {
		t.Errorf("silent bit 0 leaked on unpin, flags=%032b", uint32(req.Flags))
	}
}

// TestSetGameScoreFlagBits guards messages.setGameScore#8ef8ecc0:
// edit_message=flags.0, force=flags.1. Bit 2 does not exist.
func TestSetGameScoreFlagBits(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	c.testResolver = &mockPeerResolver{peers: map[int64]tg.InputPeerClass{
		10:  &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20},
		100: &tg.InputPeerUser{UserID: 100, AccessHash: 1},
	}}

	if _, err := c.SetGameScore(context.Background(), 10, 5, 100, 42, false); err != nil {
		t.Fatalf("SetGameScore() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSetGameScoreRequest)
	if !ok {
		t.Fatalf("expected MessagesSetGameScoreRequest, got %T", inv.lastCall())
	}
	if !req.Flags.Has(0) {
		t.Errorf("edit_message flag 0 not set, flags=%032b", uint32(req.Flags))
	}
	if req.Flags.Has(2) {
		t.Errorf("nonexistent bit 2 leaked, flags=%032b", uint32(req.Flags))
	}
}

// TestSendStoryPeriodFlagBits guards stories.sendStory#8f9e6898:
// pinned=flags.2, noforwards=flags.4, period=flags.3, albums=flags.8.
func TestSendStoryPeriodFlagBits(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.StoriesSendStoryTypeID, &tg.UpdateShort{
		Update: &tg.UpdateStory{Story: &tg.StoryItemSkipped{ID: 1}},
	})

	period := int32(86400)
	_, err := c.SendStory(context.Background(), 10, &tg.InputMediaPhotoExternal{},
		&SendStoryOption{Pinned: true, NoForwards: true, Period: &period})
	if err != nil {
		t.Fatalf("SendStory() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.StoriesSendStoryRequest)
	if !ok {
		t.Fatalf("expected StoriesSendStoryRequest, got %T", inv.lastCall())
	}
	if !req.Flags.Has(2) {
		t.Errorf("pinned flag 2 not set, flags=%032b", uint32(req.Flags))
	}
	if !req.Flags.Has(4) {
		t.Errorf("noforwards flag 4 not set, flags=%032b", uint32(req.Flags))
	}
	if !req.Flags.Has(3) {
		t.Errorf("period flag 3 not set, flags=%032b", uint32(req.Flags))
	}
	if req.Flags.Has(8) {
		t.Errorf("albums bit 8 leaked, flags=%032b", uint32(req.Flags))
	}
}
