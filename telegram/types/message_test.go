package types

import (
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestParseServiceActionRequestedPeer(t *testing.T) {
	action := &tg.MessageActionRequestedPeer{
		ButtonID: 1,
		Peers: []tg.PeerClass{
			&tg.PeerChannel{ChannelID: 100},
			&tg.PeerUser{UserID: 42},
		},
	}

	svc := parseServiceAction(action)
	if svc.Type != ServiceActionRequestedPeer {
		t.Fatalf("type = %d, want ServiceActionRequestedPeer", svc.Type)
	}
	if svc.RequestedPeers == nil {
		t.Fatal("expected RequestedPeers")
	}
	if svc.RequestedPeers.ButtonID != 1 {
		t.Errorf("buttonID = %d, want 1", svc.RequestedPeers.ButtonID)
	}
	if len(svc.RequestedPeers.ChatIDs) != 1 || svc.RequestedPeers.ChatIDs[0] != -1000000000100 {
		t.Errorf("chatIDs = %v, want [-1000000000100]", svc.RequestedPeers.ChatIDs)
	}
	if len(svc.RequestedPeers.UserIDs) != 1 || svc.RequestedPeers.UserIDs[0] != 42 {
		t.Errorf("userIDs = %v, want [42]", svc.RequestedPeers.UserIDs)
	}
}

func TestParseServiceActionRequestedPeerSentMe(t *testing.T) {
	action := &tg.MessageActionRequestedPeerSentMe{
		ButtonID: 3,
		Peers: []tg.RequestedPeerClass{
			&tg.RequestedPeerUser{UserID: 99},
			&tg.RequestedPeerChat{ChatID: 200},
			&tg.RequestedPeerChannel{ChannelID: 1798673537},
		},
	}

	svc := parseServiceAction(action)
	if svc.Type != ServiceActionRequestedPeer {
		t.Fatalf("type = %d, want ServiceActionRequestedPeer", svc.Type)
	}
	if len(svc.RequestedPeers.UserIDs) != 1 || svc.RequestedPeers.UserIDs[0] != 99 {
		t.Errorf("userIDs = %v, want [99]", svc.RequestedPeers.UserIDs)
	}
	wantChatIDs := []int64{-200, -1001798673537}
	if len(svc.RequestedPeers.ChatIDs) != len(wantChatIDs) {
		t.Fatalf("chatIDs = %v, want %v", svc.RequestedPeers.ChatIDs, wantChatIDs)
	}
	for i, want := range wantChatIDs {
		if svc.RequestedPeers.ChatIDs[i] != want {
			t.Errorf("chatIDs[%d] = %d, want %d", i, svc.RequestedPeers.ChatIDs[i], want)
		}
	}
}

func TestParseServiceActionUnknown(t *testing.T) {
	svc := parseServiceAction(&tg.MessageActionEmpty{})
	if svc.Type != ServiceActionUnknown {
		t.Errorf("type = %d, want ServiceActionUnknown", svc.Type)
	}
}

func TestParseServiceActionGroupCreate(t *testing.T) {
	svc := parseServiceAction(&tg.MessageActionChatCreate{})
	if svc.Type != ServiceActionGroupCreate {
		t.Errorf("type = %d, want ServiceActionGroupCreate", svc.Type)
	}
}

func TestParseServiceActionHistoryClear(t *testing.T) {
	svc := parseServiceAction(&tg.MessageActionHistoryClear{})
	if svc.Type != ServiceActionHistoryClear {
		t.Errorf("type = %d, want ServiceActionHistoryClear", svc.Type)
	}
}

func TestParseMessageMetadata(t *testing.T) {
	raw := &tg.Message{
		Out:                    true,
		Mentioned:              true,
		EditHide:               true,
		Noforwards:             true,
		Offline:                true,
		VideoProcessingPending: true,
		PaidSuggestedPostStars: true,
		ID:                     100,
		FromID:                 &tg.PeerUser{UserID: 1},
		FromBoostsApplied:      2,
		FromRank:               "admin",
		PeerID:                 &tg.PeerChat{ChatID: 10},
		ViaBotID:               3,
		ViaBusinessBotID:       4,
		ReplyTo: &tg.MessageReplyHeader{
			ReplyToMsgID: 12,
			ReplyToTopID: 11,
			TodoItemID:   13,
			PollOption:   []byte("option"),
		},
		Date:                 1700000000,
		EditDate:             1700000100,
		PostAuthor:           "signature",
		GroupedID:            99,
		Effect:               1234,
		PaidMessageStars:     5,
		ScheduleRepeatPeriod: 60,
		SummaryFromLanguage:  "en",
	}
	pm := NewPeerMapFromClasses(
		[]tg.UserClass{
			&tg.User{ID: 1, FirstName: "Sender"},
			&tg.User{ID: 3, FirstName: "InlineBot", Bot: true},
			&tg.User{ID: 4, FirstName: "BusinessBot", Bot: true},
		},
		[]tg.ChatClass{
			&tg.Chat{ID: 10, Title: "Group"},
		},
	)

	message := ParseMessage(raw, pm)
	if message == nil {
		t.Fatal("ParseMessage returned nil")
	}

	if message.ID != 100 || !message.Date.Equal(time.Unix(1700000000, 0)) {
		t.Fatalf("basic message fields not mapped: %+v", message)
	}
	if message.Sender == nil || message.FromUser == nil || message.Sender.ID != 1 || message.FromUser.ID != 1 {
		t.Fatalf("sender aliases not mapped: sender=%+v from_user=%+v", message.Sender, message.FromUser)
	}
	if message.Chat == nil || message.Chat.ID != -10 {
		t.Fatalf("Chat = %+v, want group -10", message.Chat)
	}
	if !message.Out || !message.Outgoing {
		t.Fatalf("outgoing aliases not mapped: out=%v outgoing=%v", message.Out, message.Outgoing)
	}
	if !message.Mentioned || !message.EditHidden || !message.HasProtectedContent || !message.FromOffline || !message.VideoProcessingPending || !message.IsPaidPost {
		t.Fatalf("boolean metadata not mapped: %+v", message)
	}
	if message.SenderBoostCount != 2 || message.SenderTag != "admin" {
		t.Fatalf("sender metadata not mapped: boosts=%d tag=%q", message.SenderBoostCount, message.SenderTag)
	}
	if message.ViaBot == nil || message.ViaBot.ID != 3 {
		t.Fatalf("ViaBot = %+v, want user 3", message.ViaBot)
	}
	if message.SenderBusinessBot == nil || message.SenderBusinessBot.ID != 4 {
		t.Fatalf("SenderBusinessBot = %+v, want user 4", message.SenderBusinessBot)
	}
	if message.ReplyToID != 12 || message.ReplyToMessageID != 12 || message.ReplyToTopMessageID != 11 || message.ReplyToChecklistTaskID != 13 || message.ReplyToPollOptionID != "option" {
		t.Fatalf("reply metadata not mapped: %+v", message)
	}
	if !message.EditDate.Equal(time.Unix(1700000100, 0)) || message.PostAuthor != "signature" || message.AuthorSignature != "signature" {
		t.Fatalf("edit/signature metadata not mapped: %+v", message)
	}
	if message.GroupedID != 99 || message.MediaGroupID != 99 {
		t.Fatalf("media group aliases not mapped: grouped=%d media_group=%d", message.GroupedID, message.MediaGroupID)
	}
	if message.EffectID != 1234 || message.SendPaidMessagesStars != 5 || message.RepeatPeriod != 60 || message.SummaryLanguageCode != "en" {
		t.Fatalf("extra metadata not mapped: %+v", message)
	}
}

func TestParseMessageAnimatedDocument(t *testing.T) {
	raw := &tg.Message{
		ID:     100,
		PeerID: &tg.PeerChat{ChatID: 10},
		Date:   1700000000,
		Media: &tg.MessageMediaDocument{
			Video: true,
			Document: &tg.Document{
				ID:         1,
				AccessHash: 2,
				DCID:       4,
				MimeType:   "video/mp4",
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeAnimated{},
					&tg.DocumentAttributeVideo{
						Duration: 3,
						W:        320,
						H:        240,
					},
				},
			},
		},
	}

	message := ParseMessage(raw, nil)
	if message == nil {
		t.Fatal("ParseMessage returned nil")
	}
	if message.Animation == nil {
		t.Fatalf("Animation = nil, media=%T document=%+v", message.Media, message.Document)
	}
	if message.Video != nil {
		t.Fatalf("Video = %+v, want nil for animated document", message.Video)
	}
	if message.Document == nil || message.Document.MediaType() != MessageMediaTypeAnimation {
		t.Fatalf("Document media type = %v, want animation", message.Document.MediaType())
	}
	if message.Document.FileID == "" || message.Document.FileID == "1_2" {
		t.Fatalf("FileID = %q, want encoded file_id", message.Document.FileID)
	}
}

func TestParseMessageForwardOriginChannel(t *testing.T) {
	raw := &tg.Message{
		ID:     100,
		PeerID: &tg.PeerChat{ChatID: 10},
		Date:   1700000000,
		FwdFrom: &tg.MessageFwdHeader{
			FromID:      &tg.PeerChannel{ChannelID: 20},
			Date:        1690000000,
			ChannelPost: 123,
			PostAuthor:  "signature",
		},
	}
	pm := NewPeerMapFromClasses(nil, []tg.ChatClass{
		&tg.Chat{ID: 10, Title: "Group"},
		&tg.Channel{ID: 20, Title: "Channel", Broadcast: true},
	})

	message := ParseMessage(raw, pm)
	if message == nil {
		t.Fatal("ParseMessage returned nil")
	}
	if message.FwdFrom == nil || message.FwdFrom.ChannelID != 20 || message.FwdFrom.PostID != 123 {
		t.Fatalf("FwdFrom = %+v, want channel 20 post 123", message.FwdFrom)
	}
	origin := message.ForwardOrigin
	if origin == nil {
		t.Fatal("ForwardOrigin is nil")
	}
	if origin.Type != MessageOriginTypeChannel {
		t.Fatalf("Type = %q, want %q", origin.Type, MessageOriginTypeChannel)
	}
	if !origin.Date.Equal(time.Unix(1690000000, 0)) {
		t.Fatalf("Date = %v, want %v", origin.Date, time.Unix(1690000000, 0))
	}
	if origin.Chat == nil || origin.Chat.ID != -20 || origin.Chat.Title != "Channel" {
		t.Fatalf("Chat = %+v, want channel -20", origin.Chat)
	}
	if origin.ChatID != 20 {
		t.Fatalf("ChatID = %d, want 20", origin.ChatID)
	}
	if origin.MessageID != 123 {
		t.Fatalf("MessageID = %d, want 123", origin.MessageID)
	}
	if origin.AuthorSignature != "signature" {
		t.Fatalf("AuthorSignature = %q, want signature", origin.AuthorSignature)
	}
}

func TestParseMessageForwardOriginOtherTypes(t *testing.T) {
	pm := NewPeerMapFromClasses(
		[]tg.UserClass{&tg.User{ID: 7, FirstName: "Ada"}},
		[]tg.ChatClass{&tg.Chat{ID: 10, Title: "Group"}},
	)

	tests := []struct {
		name string
		raw  *tg.MessageFwdHeader
		want MessageOriginType
	}{
		{name: "user", raw: &tg.MessageFwdHeader{FromID: &tg.PeerUser{UserID: 7}, Date: 1690000000}, want: MessageOriginTypeUser},
		{name: "chat", raw: &tg.MessageFwdHeader{FromID: &tg.PeerChat{ChatID: 10}, Date: 1690000000}, want: MessageOriginTypeChat},
		{name: "hidden", raw: &tg.MessageFwdHeader{FromName: "Hidden", Date: 1690000000}, want: MessageOriginTypeHiddenUser},
		{name: "import", raw: &tg.MessageFwdHeader{Imported: true, Date: 1690000000}, want: MessageOriginTypeImport},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origin := parseMessageOrigin(tt.raw, pm)
			if origin == nil {
				t.Fatal("origin is nil")
			}
			if origin.Type != tt.want {
				t.Fatalf("Type = %q, want %q", origin.Type, tt.want)
			}
		})
	}
}

func TestParseMessageForwardOriginChat(t *testing.T) {
	pm := NewPeerMapFromClasses(nil, []tg.ChatClass{
		&tg.Chat{ID: 10, Title: "Group"},
	})
	origin := parseMessageOrigin(&tg.MessageFwdHeader{
		FromID:     &tg.PeerChat{ChatID: 10},
		Date:       1690000000,
		PostAuthor: "anonymous admin",
	}, pm)

	if origin == nil {
		t.Fatal("origin is nil")
	}
	if origin.Type != MessageOriginTypeChat {
		t.Fatalf("Type = %q, want %q", origin.Type, MessageOriginTypeChat)
	}
	if !origin.Date.Equal(time.Unix(1690000000, 0)) {
		t.Fatalf("Date = %v, want %v", origin.Date, time.Unix(1690000000, 0))
	}
	if origin.SenderChat == nil || origin.SenderChat.ID != -10 || origin.SenderChat.Title != "Group" {
		t.Fatalf("SenderChat = %+v, want group -10", origin.SenderChat)
	}
	if origin.AuthorSignature != "anonymous admin" {
		t.Fatalf("AuthorSignature = %q, want anonymous admin", origin.AuthorSignature)
	}
}

func TestParseMessageEmpty(t *testing.T) {
	message := ParseMessage(&tg.MessageEmpty{
		ID:     5,
		PeerID: &tg.PeerUser{UserID: 1},
	}, NewPeerMapFromClasses([]tg.UserClass{&tg.User{ID: 1, FirstName: "Ada"}}, nil))

	if message == nil {
		t.Fatal("ParseMessage returned nil")
	}
	if !message.Empty {
		t.Fatal("Empty = false, want true")
	}
	if message.Chat == nil || message.Chat.ID != 1 {
		t.Fatalf("Chat = %+v, want private chat 1", message.Chat)
	}
}
