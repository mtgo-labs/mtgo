package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/fileid"
	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSearchMessagesOptions(t *testing.T) {
	cfg := &SearchMessagesOption{
		Limit:    50,
		OffsetID: 100,
		MinDate:  1640000000,
		MaxDate:  1672444800,
		FromID:   &tg.InputPeerUser{UserID: 1, AccessHash: 2},
		Filter:   &tg.InputMessagesFilterEmpty{},
	}

	if cfg.Limit != 50 {
		t.Errorf("Limit = %d, want 50", cfg.Limit)
	}
	if cfg.OffsetID != 100 {
		t.Errorf("OffsetID = %d, want 100", cfg.OffsetID)
	}
	if cfg.MinDate != 1640000000 {
		t.Errorf("MinDate = %d, want 1640000000", cfg.MinDate)
	}
	if cfg.MaxDate != 1672444800 {
		t.Errorf("MaxDate = %d, want 1672444800", cfg.MaxDate)
	}
	if cfg.FromID == nil {
		t.Error("FromID should not be nil")
	}
	if cfg.Filter == nil {
		t.Error("Filter should not be nil")
	}
}

func TestSearchGlobalOptions(t *testing.T) {
	cfg := &SearchGlobalOption{
		Limit:          20,
		OffsetRate:     5,
		OffsetID:       999,
		OffsetPeer:     &tg.InputPeerEmpty{},
		MinDate:        1000000,
		MaxDate:        2000000,
		BroadcastsOnly: true,
		GroupsOnly:     true,
	}

	if cfg.Limit != 20 {
		t.Errorf("Limit = %d, want 20", cfg.Limit)
	}
	if cfg.OffsetRate != 5 {
		t.Errorf("OffsetRate = %d, want 5", cfg.OffsetRate)
	}
	if cfg.OffsetID != 999 {
		t.Errorf("OffsetID = %d, want 999", cfg.OffsetID)
	}
	if cfg.OffsetPeer == nil {
		t.Error("OffsetPeer should not be nil")
	}
	if cfg.MinDate != 1000000 {
		t.Errorf("MinDate = %d, want 1000000", cfg.MinDate)
	}
	if cfg.MaxDate != 2000000 {
		t.Errorf("MaxDate = %d, want 2000000", cfg.MaxDate)
	}
	if !cfg.BroadcastsOnly {
		t.Error("BroadcastsOnly should be true")
	}
	if !cfg.GroupsOnly {
		t.Error("GroupsOnly should be true")
	}
}

func TestCopyMessages(t *testing.T) {
	scheduleDate := int32(12345)
	cfg := &params.CopyMessage{
		Caption:             "test caption",
		DisableNotification: true,
		ReplyToMessageID:    42,
		ReplyMarkup:         &tg.ReplyInlineMarkup{},
		ScheduleDate:        &scheduleDate,
		DropAuthor:          true,
	}

	if cfg.Caption != "test caption" {
		t.Errorf("Caption = %q, want %q", cfg.Caption, "test caption")
	}
	if !cfg.DisableNotification {
		t.Error("DisableNotification should be true")
	}
	if cfg.ReplyToMessageID != 42 {
		t.Errorf("ReplyToMessageID = %d, want 42", cfg.ReplyToMessageID)
	}
	if cfg.ReplyMarkup == nil {
		t.Error("ReplyMarkup should not be nil")
	}
	if cfg.ScheduleDate == nil || *cfg.ScheduleDate != 12345 {
		t.Error("ScheduleDate should be 12345")
	}
	if !cfg.DropAuthor {
		t.Error("DropAuthor should be true")
	}
}

func TestEditMessageCaption(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.EditMessageCaption(context.Background(), 10, 55, "new caption", nil)
	if err != nil {
		t.Fatalf("EditMessageCaption() error: %v", err)
	}
	if inv.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", inv.callCount())
	}
	req, ok := inv.lastCall().(*tg.MessagesEditMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesEditMessageRequest, got %T", inv.lastCall())
	}
	if req.ID != 55 {
		t.Errorf("ID = %d, want 55", req.ID)
	}
	if req.Media != nil {
		t.Error("Media should be nil for caption-only edit")
	}
}

func TestGetMessagesUsesChannelsGetMessagesForChannels(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	inv.setResult(tg.ChannelsGetMessagesTypeID, &tg.MessagesMessages{
		Messages: []tg.MessageClass{&tg.Message{ID: 42}},
	})

	msgs, err := c.GetMessages(context.Background(), 10, []int32{42})
	if err != nil {
		t.Fatalf("GetMessages() error: %v", err)
	}
	if len(msgs) != 1 || msgs[0].ID != 42 {
		t.Fatalf("GetMessages() = %#v, want message 42", msgs)
	}

	req, ok := inv.lastCall().(*tg.ChannelsGetMessagesRequest)
	if !ok {
		t.Fatalf("expected ChannelsGetMessagesRequest, got %T", inv.lastCall())
	}
	ch, ok := req.Channel.(*tg.InputChannel)
	if !ok {
		t.Fatalf("expected InputChannel, got %T", req.Channel)
	}
	if ch.ChannelID != 10 || ch.AccessHash != 20 {
		t.Fatalf("channel = (%d, %d), want (10, 20)", ch.ChannelID, ch.AccessHash)
	}
}

func TestEditMessageCaptionWithOptions(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.EditMessageCaption(context.Background(), 10, 55, "cap",
		&params.EditMessage{ReplyMarkup: &tg.ReplyInlineMarkup{}},
	)
	if err != nil {
		t.Fatalf("EditMessageCaption() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesEditMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesEditMessageRequest, got %T", inv.lastCall())
	}
	if req.ReplyMarkup == nil {
		t.Error("expected ReplyMarkup to be set")
	}
}

func TestEditMessageMedia(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	media := &tg.InputMediaPhoto{
		ID: &tg.InputPhoto{ID: 1, AccessHash: 2, FileReference: []byte{}},
	}
	_, err := c.EditMessageMedia(context.Background(), 10, 55, media)
	if err != nil {
		t.Fatalf("EditMessageMedia() error: %v", err)
	}
	if inv.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", inv.callCount())
	}
	req, ok := inv.lastCall().(*tg.MessagesEditMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesEditMessageRequest, got %T", inv.lastCall())
	}
	if req.Media == nil {
		t.Error("Media should not be nil")
	}
}

func TestEditMessageReplyMarkup(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	markup := &tg.ReplyInlineMarkup{}
	_, err := c.EditMessageReplyMarkup(context.Background(), 10, 55, markup)
	if err != nil {
		t.Fatalf("EditMessageReplyMarkup() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesEditMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesEditMessageRequest, got %T", inv.lastCall())
	}
	if req.ReplyMarkup == nil {
		t.Error("ReplyMarkup should not be nil")
	}
}

func TestEditMessageCaptionPeerNotFound(t *testing.T) {
	c, _ := newClientWithMock(t)
	_, err := c.EditMessageCaption(context.Background(), 999, 1, "x")
	if err == nil {
		t.Fatal("expected error for unresolved peer")
	}
}

func TestSearchMessages(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	inv.setResult(tg.MessagesSearchTypeID, &tg.MessagesMessages{
		Messages: []tg.MessageClass{},
		Users:    []tg.UserClass{},
		Chats:    []tg.ChatClass{},
	})

	_, err := c.SearchMessages(context.Background(), 10, "hello",
		&SearchMessagesOption{Limit: 25},
	)
	if err != nil {
		t.Fatalf("SearchMessages() error: %v", err)
	}
	if inv.callCount() != 1 {
		t.Fatalf("expected 1 RPC call, got %d", inv.callCount())
	}
	req, ok := inv.lastCall().(*tg.MessagesSearchRequest)
	if !ok {
		t.Fatalf("expected MessagesSearchRequest, got %T", inv.lastCall())
	}
	if req.Q != "hello" {
		t.Errorf("Q = %q, want %q", req.Q, "hello")
	}
	if req.Limit != 25 {
		t.Errorf("Limit = %d, want 25", req.Limit)
	}
}

func TestSearchMessagesDefaultLimit(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	inv.setResult(tg.MessagesSearchTypeID, &tg.MessagesMessages{
		Messages: []tg.MessageClass{},
		Users:    []tg.UserClass{},
		Chats:    []tg.ChatClass{},
	})

	_, _ = c.SearchMessages(context.Background(), 10, "test")
	req, ok := inv.lastCall().(*tg.MessagesSearchRequest)
	if !ok {
		t.Fatalf("expected MessagesSearchRequest, got %T", inv.lastCall())
	}
	if req.Limit != 100 {
		t.Errorf("default Limit = %d, want 100", req.Limit)
	}
}

func TestSearchGlobal(t *testing.T) {
	c, inv := newClientWithMock(t)

	inv.setResult(tg.MessagesSearchGlobalTypeID, &tg.MessagesMessages{
		Messages: []tg.MessageClass{},
		Users:    []tg.UserClass{},
		Chats:    []tg.ChatClass{},
	})

	_, err := c.SearchGlobal(context.Background(), "world",
		&SearchGlobalOption{Limit: 10},
	)
	if err != nil {
		t.Fatalf("SearchGlobal() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSearchGlobalRequest)
	if !ok {
		t.Fatalf("expected MessagesSearchGlobalRequest, got %T", inv.lastCall())
	}
	if req.Q != "world" {
		t.Errorf("Q = %q, want %q", req.Q, "world")
	}
	if req.Limit != 10 {
		t.Errorf("Limit = %d, want 10", req.Limit)
	}
}

func TestSearchMessagesPeerError(t *testing.T) {
	c, _ := newClientWithMock(t)
	_, err := c.SearchMessages(context.Background(), 999, "test")
	if err == nil {
		t.Fatal("expected error for unresolved peer")
	}
}

func TestSearchMessagesCount(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	inv.setResult(tg.MessagesSearchTypeID, &tg.MessagesMessagesSlice{
		Count:    42,
		Messages: []tg.MessageClass{},
		Users:    []tg.UserClass{},
		Chats:    []tg.ChatClass{},
	})

	count, err := c.SearchMessagesCount(context.Background(), 10, "hello")
	if err != nil {
		t.Fatalf("SearchMessagesCount() error: %v", err)
	}
	if count != 42 {
		t.Errorf("count = %d, want 42", count)
	}
	req, ok := inv.lastCall().(*tg.MessagesSearchRequest)
	if !ok {
		t.Fatalf("expected MessagesSearchRequest, got %T", inv.lastCall())
	}
	if req.Limit != 1 {
		t.Errorf("Limit = %d, want 1 (minimal for count)", req.Limit)
	}
}

func TestSearchGlobalCount(t *testing.T) {
	c, inv := newClientWithMock(t)

	inv.setResult(tg.MessagesSearchGlobalTypeID, &tg.MessagesChannelMessages{
		Count:    99,
		Messages: []tg.MessageClass{},
		Users:    []tg.UserClass{},
		Chats:    []tg.ChatClass{},
	})

	count, err := c.SearchGlobalCount(context.Background(), "test")
	if err != nil {
		t.Fatalf("SearchGlobalCount() error: %v", err)
	}
	if count != 99 {
		t.Errorf("count = %d, want 99", count)
	}
	req, ok := inv.lastCall().(*tg.MessagesSearchGlobalRequest)
	if !ok {
		t.Fatalf("expected MessagesSearchGlobalRequest, got %T", inv.lastCall())
	}
	if req.Limit != 1 {
		t.Errorf("Limit = %d, want 1 (minimal for count)", req.Limit)
	}
}

func TestReadHistory(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	inv.setResult(tg.MessagesReadHistoryTypeID, &tg.MessagesAffectedMessages{
		PTS:      100,
		PTSCount: 5,
	})

	err := c.ReadHistory(context.Background(), 10, 500)
	if err != nil {
		t.Fatalf("ReadHistory() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesReadHistoryRequest)
	if !ok {
		t.Fatalf("expected MessagesReadHistoryRequest, got %T", inv.lastCall())
	}
	if req.MaxID != 500 {
		t.Errorf("MaxID = %d, want 500", req.MaxID)
	}
}

func TestReadMentions(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	inv.setResult(tg.MessagesReadMentionsTypeID, &tg.MessagesAffectedHistory{
		PTS:      100,
		PTSCount: 3,
		Offset:   0,
	})

	err := c.ReadMentions(context.Background(), 10)
	if err != nil {
		t.Fatalf("ReadMentions() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesReadMentionsRequest)
	if !ok {
		t.Fatalf("expected MessagesReadMentionsRequest, got %T", inv.lastCall())
	}
	if req.Peer == nil {
		t.Error("Peer should not be nil")
	}
}

func TestReadReactions(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	inv.setResult(tg.MessagesReadReactionsTypeID, &tg.MessagesAffectedHistory{
		PTS:      100,
		PTSCount: 2,
		Offset:   0,
	})

	err := c.ReadReactions(context.Background(), 10)
	if err != nil {
		t.Fatalf("ReadReactions() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesReadReactionsRequest)
	if !ok {
		t.Fatalf("expected MessagesReadReactionsRequest, got %T", inv.lastCall())
	}
	if req.Peer == nil {
		t.Error("Peer should not be nil")
	}
}

func TestReadHistoryPeerError(t *testing.T) {
	c, _ := newClientWithMock(t)
	err := c.ReadHistory(context.Background(), 999, 0)
	if err == nil {
		t.Fatal("expected error for unresolved peer")
	}
}

func TestSendReaction(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	err := c.SendReaction(context.Background(), 10, 55, []types.Reaction{{Emoji: "\U0001F525"}})
	if err != nil {
		t.Fatalf("SendReaction() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendReactionRequest)
	if !ok {
		t.Fatalf("expected MessagesSendReactionRequest, got %T", inv.lastCall())
	}
	if req.MsgID != 55 {
		t.Errorf("MsgID = %d, want 55", req.MsgID)
	}
	if len(req.Reaction) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(req.Reaction))
	}
}

func TestSendReactionMultiple(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	err := c.SendReaction(context.Background(), 10, 55,
		[]types.Reaction{
			{Emoji: "\U0001F525"},
			{CustomEmojiID: "12345"},
		},
	)
	if err != nil {
		t.Fatalf("SendReaction() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendReactionRequest)
	if !ok {
		t.Fatalf("expected MessagesSendReactionRequest, got %T", inv.lastCall())
	}
	if len(req.Reaction) != 2 {
		t.Errorf("expected 2 reactions, got %d", len(req.Reaction))
	}
}

func TestSendReactionWithOptions(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	err := c.SendReaction(context.Background(), 10, 55,
		[]types.Reaction{{Emoji: "\U0001F525"}},
		&params.SendReactionOption{Big: true, AddToRecent: true},
	)
	if err != nil {
		t.Fatalf("SendReaction() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendReactionRequest)
	if !ok {
		t.Fatalf("expected MessagesSendReactionRequest, got %T", inv.lastCall())
	}
	if !req.Big {
		t.Error("Big = false, want true")
	}
	if !req.AddToRecent {
		t.Error("AddToRecent = false, want true")
	}
}

func TestSendPaidReaction(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	err := c.SendPaidReaction(context.Background(), 10, 55, 100)
	if err != nil {
		t.Fatalf("SendPaidReaction() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendPaidReactionRequest)
	if !ok {
		t.Fatalf("expected MessagesSendPaidReactionRequest, got %T", inv.lastCall())
	}
	if req.MsgID != 55 {
		t.Errorf("MsgID = %d, want 55", req.MsgID)
	}
	if req.Count != 100 {
		t.Errorf("Count = %d, want 100", req.Count)
	}
}

func TestSendReactionPeerError(t *testing.T) {
	c, _ := newClientWithMock(t)
	err := c.SendReaction(context.Background(), 999, 1, []types.Reaction{{Emoji: "\U0001F44D"}})
	if err == nil {
		t.Fatal("expected error for unresolved peer")
	}
}

func TestVotePoll(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	options := [][]byte{[]byte("opt1"), []byte("opt2")}
	err := c.VotePoll(context.Background(), 10, 55, options)
	if err != nil {
		t.Fatalf("VotePoll() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendVoteRequest)
	if !ok {
		t.Fatalf("expected MessagesSendVoteRequest, got %T", inv.lastCall())
	}
	if req.MsgID != 55 {
		t.Errorf("MsgID = %d, want 55", req.MsgID)
	}
	if len(req.Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(req.Options))
	}
}

func TestStopPoll(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	err := c.StopPoll(context.Background(), 10, 55)
	if err != nil {
		t.Fatalf("StopPoll() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesEditMessageRequest)
	if !ok {
		t.Fatalf("expected MessagesEditMessageRequest, got %T", inv.lastCall())
	}
	if req.ID != 55 {
		t.Errorf("ID = %d, want 55", req.ID)
	}
	if req.Media == nil {
		t.Error("Media should not be nil for StopPoll")
	}
}

func TestVotePollPeerError(t *testing.T) {
	c, _ := newClientWithMock(t)
	err := c.VotePoll(context.Background(), 999, 1, [][]byte{[]byte("a")})
	if err == nil {
		t.Fatal("expected error for unresolved peer")
	}
}

func TestSendContact(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.SendContact(context.Background(), 10, "+1234567890", "John", "Doe", nil)
	if err != nil {
		t.Fatalf("SendContact() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMediaRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMediaRequest, got %T", inv.lastCall())
	}
	media, ok := req.Media.(*tg.InputMediaContact)
	if !ok {
		t.Fatalf("expected InputMediaContact, got %T", req.Media)
	}
	if media.PhoneNumber != "+1234567890" {
		t.Errorf("PhoneNumber = %q, want +1234567890", media.PhoneNumber)
	}
	if media.FirstName != "John" {
		t.Errorf("FirstName = %q, want John", media.FirstName)
	}
	if media.LastName != "Doe" {
		t.Errorf("LastName = %q, want Doe", media.LastName)
	}
}

func TestSendLocation(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.SendLocation(context.Background(), 10, 40.7128, -74.0060, nil)
	if err != nil {
		t.Fatalf("SendLocation() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMediaRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMediaRequest, got %T", inv.lastCall())
	}
	media, ok := req.Media.(*tg.InputMediaGeoPoint)
	if !ok {
		t.Fatalf("expected InputMediaGeoPoint, got %T", req.Media)
	}
	geo, ok := media.GeoPoint.(*tg.InputGeoPoint)
	if !ok {
		t.Fatalf("expected InputGeoPointTL, got %T", media.GeoPoint)
	}
	if geo.Lat != 40.7128 {
		t.Errorf("Lat = %f, want 40.7128", geo.Lat)
	}
	if geo.Long != -74.0060 {
		t.Errorf("Long = %f, want -74.0060", geo.Long)
	}
}

func TestSendLocationWithReply(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.SendLocation(context.Background(), 10, 1.0, 2.0,
		&params.SendMessage{ReplyToMessageID: 42},
	)
	if err != nil {
		t.Fatalf("SendLocation() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMediaRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMediaRequest, got %T", inv.lastCall())
	}
	if req.ReplyTo == nil {
		t.Error("expected ReplyTo to be set")
	}
}

func TestSendContactPeerError(t *testing.T) {
	c, _ := newClientWithMock(t)
	_, err := c.SendContact(context.Background(), 999, "+1", "A", "B", nil)
	if err == nil {
		t.Fatal("expected error for unresolved peer")
	}
}

func TestSendVenue(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.SendVenue(context.Background(), 10, 40.7128, -74.0060, "NYC", "123 Main St", nil)
	if err != nil {
		t.Fatalf("SendVenue() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMediaRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMediaRequest, got %T", inv.lastCall())
	}
	media, ok := req.Media.(*tg.InputMediaVenue)
	if !ok {
		t.Fatalf("expected InputMediaVenue, got %T", req.Media)
	}
	if media.Title != "NYC" {
		t.Errorf("Title = %q, want NYC", media.Title)
	}
	if media.Address != "123 Main St" {
		t.Errorf("Address = %q, want 123 Main St", media.Address)
	}
}

func TestSendDice(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.SendDice(context.Background(), 10, nil)
	if err != nil {
		t.Fatalf("SendDice() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMediaRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMediaRequest, got %T", inv.lastCall())
	}
	media, ok := req.Media.(*tg.InputMediaDice)
	if !ok {
		t.Fatalf("expected InputMediaDice, got %T", req.Media)
	}
	if media.Emoticon != "\U0001F3B2" {
		t.Errorf("Emoticon = %q, want \U0001F3B2", media.Emoticon)
	}
}

func TestSendDiceCustomEmoticon(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.SendDice(context.Background(), 10,
		&SendDiceOption{Emoticon: "\U0001F3AF"},
	)
	if err != nil {
		t.Fatalf("SendDice() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMediaRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMediaRequest, got %T", inv.lastCall())
	}
	media, ok := req.Media.(*tg.InputMediaDice)
	if !ok {
		t.Fatalf("expected InputMediaDice, got %T", req.Media)
	}
	if media.Emoticon != "\U0001F3AF" {
		t.Errorf("Emoticon = %q, want \U0001F3AF", media.Emoticon)
	}
}

func TestSendPoll(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})

	_, err := c.SendPoll(context.Background(), 10, "What is your fav?", []string{"A", "B", "C"}, nil)
	if err != nil {
		t.Fatalf("SendPoll() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMediaRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMediaRequest, got %T", inv.lastCall())
	}
	media, ok := req.Media.(*tg.InputMediaPoll)
	if !ok {
		t.Fatalf("expected InputMediaPoll, got %T", req.Media)
	}
	if media.Poll == nil {
		t.Fatal("Poll should not be nil")
	}
	questionText := ""
	if media.Poll.Question != nil {
		questionText = media.Poll.Question.Text
	}
	if questionText != "What is your fav?" {
		t.Errorf("Question = %q, want %q", questionText, "What is your fav?")
	}
	if len(media.Poll.Answers) != 3 {
		t.Errorf("expected 3 answers, got %d", len(media.Poll.Answers))
	}
}

func TestSendCachedMedia(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerChannel{ChannelID: 10, AccessHash: 20})
	cachedFileID, err := fileid.Encode(fileid.FileID{
		Type:          fileid.FileTypeDocument,
		DCID:          4,
		ID:            100,
		AccessHash:    200,
		FileReference: []byte{1, 2, 3},
	})
	if err != nil {
		t.Fatalf("encode file_id: %v", err)
	}

	_, err = c.SendCachedMedia(context.Background(), 10, cachedFileID,
		&params.SendMessage{ReplyToMessageID: 10},
	)
	if err != nil {
		t.Fatalf("SendCachedMedia() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSendMediaRequest)
	if !ok {
		t.Fatalf("expected MessagesSendMediaRequest, got %T", inv.lastCall())
	}
	_, ok = req.Media.(*tg.InputMediaDocument)
	if !ok {
		t.Fatalf("expected InputMediaDocument, got %T", req.Media)
	}
}

func TestSendPollPeerError(t *testing.T) {
	c, _ := newClientWithMock(t)
	_, err := c.SendPoll(context.Background(), 999, "Q?", []string{"A"}, nil)
	if err == nil {
		t.Fatal("expected error for unresolved peer")
	}
}
