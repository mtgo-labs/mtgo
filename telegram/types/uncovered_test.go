package types_test

import (
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func TestParseAuctionRound_Nil(t *testing.T) {
	if got := types.ParseAuctionRound(nil); got != nil {
		t.Errorf("ParseAuctionRound(nil) = %v, want nil", got)
	}
}

func TestParseAuctionRound_Basic(t *testing.T) {
	raw := &tg.StarGiftAuctionRound{Num: 1, Duration: 300}
	got := types.ParseAuctionRound(raw)
	if got == nil {
		t.Fatal("ParseAuctionRound returned nil")
	}
	if got.Number != 1 || got.Duration != 300 {
		t.Errorf("ParseAuctionRound basic = {Number:%d, Duration:%d}, want {1, 300}", got.Number, got.Duration)
	}
	if got.ExtendTime != 0 || got.TopWinnerCount != 0 {
		t.Errorf("ParseAuctionRound basic extendable fields should be zero, got ExtendTime=%d TopWinnerCount=%d", got.ExtendTime, got.TopWinnerCount)
	}
}

func TestParseAuctionRound_Extendable(t *testing.T) {
	raw := &tg.StarGiftAuctionRoundExtendable{
		Num:          2,
		Duration:     600,
		ExtendTop:    5,
		ExtendWindow: 120,
	}
	got := types.ParseAuctionRound(raw)
	if got == nil {
		t.Fatal("ParseAuctionRound returned nil")
	}
	if got.Number != 2 || got.Duration != 600 {
		t.Errorf("ParseAuctionRound extendable = {Number:%d, Duration:%d}, want {2, 600}", got.Number, got.Duration)
	}
	if got.ExtendTime != 120 || got.TopWinnerCount != 5 {
		t.Errorf("ParseAuctionRound extendable = {ExtendTime:%d, TopWinnerCount:%d}, want {120, 5}", got.ExtendTime, got.TopWinnerCount)
	}
}

func TestParseAuctionState_Nil(t *testing.T) {
	if got := types.ParseAuctionState(nil); got != nil {
		t.Errorf("ParseAuctionState(nil) = %v, want nil", got)
	}
}

func TestParseAuctionState(t *testing.T) {
	raw := &tg.StarGiftAuctionState{
		MinBidAmount: 500,
		CurrentRound: 3,
		TopBidders:   []int64{42, 99},
	}
	got := types.ParseAuctionState(raw)
	if got == nil {
		t.Fatal("ParseAuctionState returned nil")
	}
	if got.TopBidderID != 42 {
		t.Errorf("TopBidderID = %d, want 42", got.TopBidderID)
	}
	if got.TopBid != 500 {
		t.Errorf("TopBid = %d, want 500", got.TopBid)
	}
	if got.Round != 3 {
		t.Errorf("Round = %d, want 3", got.Round)
	}
}

func TestParseAuctionState_NoBidders(t *testing.T) {
	raw := &tg.StarGiftAuctionState{
		MinBidAmount: 100,
		CurrentRound: 1,
		TopBidders:   nil,
	}
	got := types.ParseAuctionState(raw)
	if got.TopBidderID != 0 {
		t.Errorf("TopBidderID = %d, want 0 when no bidders", got.TopBidderID)
	}
}

func TestParseGiftAuction_Nil(t *testing.T) {
	if got := types.ParseGiftAuction(nil); got != nil {
		t.Errorf("ParseGiftAuction(nil) = %v, want nil", got)
	}
}

func TestParseGiftAuction(t *testing.T) {
	raw := &tg.StarGiftAuctionState{
		StartDate:    1700000000,
		EndDate:      1700086400,
		MinBidAmount: 100,
		TopBidders:   []int64{10, 20, 30},
		Rounds: []tg.StarGiftAuctionRoundClass{
			&tg.StarGiftAuctionRound{Num: 1, Duration: 300},
			&tg.StarGiftAuctionRoundExtendable{Num: 2, Duration: 600, ExtendTop: 3, ExtendWindow: 60},
		},
	}
	got := types.ParseGiftAuction(raw)
	if got == nil {
		t.Fatal("ParseGiftAuction returned nil")
	}
	if got.StartDate != 1700000000 || got.EndDate != 1700086400 {
		t.Errorf("dates = {Start:%d, End:%d}", got.StartDate, got.EndDate)
	}
	if got.MinStars != 100 || got.TopBid != 100 {
		t.Errorf("MinStars=%d TopBid=%d, both want 100", got.MinStars, got.TopBid)
	}
	if got.Bidders != 3 {
		t.Errorf("Bidders = %d, want 3", got.Bidders)
	}
	if len(got.Rounds) != 2 {
		t.Fatalf("Rounds len = %d, want 2", len(got.Rounds))
	}
	if got.Rounds[0].Number != 1 || got.Rounds[0].Duration != 300 {
		t.Errorf("round 0 = %+v", got.Rounds[0])
	}
	if got.Rounds[1].Number != 2 || got.Rounds[1].ExtendTime != 60 {
		t.Errorf("round 1 = %+v", got.Rounds[1])
	}
}

func TestParseSentCodeInfo_Nil(t *testing.T) {
	if got := types.ParseSentCodeInfo(nil); got != nil {
		t.Errorf("ParseSentCodeInfo(nil) = %v, want nil", got)
	}
}

func TestParseSentCodeInfo(t *testing.T) {
	raw := &tg.AuthSentCode{
		Type:          &tg.AuthSentCodeTypeSms{},
		PhoneCodeHash: "abc123",
		NextType:      &tg.AuthCodeTypeCall{},
		Timeout:       120,
	}
	got := types.ParseSentCodeInfo(raw)
	if got == nil {
		t.Fatal("ParseSentCodeInfo returned nil")
	}
	if got.PhoneCodeHash != "abc123" {
		t.Errorf("PhoneCodeHash = %q, want %q", got.PhoneCodeHash, "abc123")
	}
	if got.Type != types.SentCodeTypeSMS {
		t.Errorf("Type = %q, want %q", got.Type, types.SentCodeTypeSMS)
	}
	if got.NextType != types.SentCodeTypeCall {
		t.Errorf("NextType = %q, want %q", got.NextType, types.SentCodeTypeCall)
	}
	if got.Timeout != 120 {
		t.Errorf("Timeout = %d, want 120", got.Timeout)
	}
}

func TestParseSentCodeInfo_NoNextType(t *testing.T) {
	raw := &tg.AuthSentCode{
		Type:          &tg.AuthSentCodeTypeApp{},
		PhoneCodeHash: "hash",
	}
	got := types.ParseSentCodeInfo(raw)
	if got.NextType != "" {
		t.Errorf("NextType = %q, want empty when nil", got.NextType)
	}
	if got.Timeout != 0 {
		t.Errorf("Timeout = %d, want 0 when not set", got.Timeout)
	}
}

func TestSentCodeTypeClass_All(t *testing.T) {
	tests := []struct {
		name string
		raw  tg.SentCodeTypeClass
		want types.SentCodeType
	}{
		{"app", &tg.AuthSentCodeTypeApp{}, types.SentCodeTypeApp},
		{"call", &tg.AuthSentCodeTypeCall{}, types.SentCodeTypeCall},
		{"flash_call", &tg.AuthSentCodeTypeFlashCall{}, types.SentCodeTypeFlashCall},
		{"missed_call", &tg.AuthSentCodeTypeMissedCall{}, types.SentCodeTypeMissedCall},
		{"sms", &tg.AuthSentCodeTypeSms{}, types.SentCodeTypeSMS},
		{"fragment_sms", &tg.AuthSentCodeTypeFragmentSms{}, types.SentCodeTypeFragmentSMS},
		{"email_code", &tg.AuthSentCodeTypeEmailCode{}, types.SentCodeTypeEmailCode},
		{"firebase_sms", &tg.AuthSentCodeTypeFirebaseSms{}, types.SentCodeTypeFirebaseSMS},
		{"setup_email", &tg.AuthSentCodeTypeSetUpEmailRequired{}, types.SentCodeTypeSetupEmailRequired},
		{"sms_phrase", &tg.AuthSentCodeTypeSmsPhrase{}, types.SentCodeTypeSmsPhrase},
		{"sms_word", &tg.AuthSentCodeTypeSmsWord{}, types.SentCodeTypeSmsWord},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := &tg.AuthSentCode{Type: tt.raw, PhoneCodeHash: "h"}
			got := types.ParseSentCodeInfo(raw)
			if got.Type != tt.want {
				t.Errorf("Type = %q, want %q", got.Type, tt.want)
			}
		})
	}
}

func TestCodeTypeClass(t *testing.T) {
	tests := []struct {
		name string
		raw  tg.CodeTypeClass
		want types.SentCodeType
	}{
		{"sms", &tg.AuthCodeTypeSms{}, types.SentCodeTypeSMS},
		{"call", &tg.AuthCodeTypeCall{}, types.SentCodeTypeCall},
		{"flash_call", &tg.AuthCodeTypeFlashCall{}, types.SentCodeTypeFlashCall},
		{"missed_call", &tg.AuthCodeTypeMissedCall{}, types.SentCodeTypeMissedCall},
		{"fragment_sms", &tg.AuthCodeTypeFragmentSms{}, types.SentCodeTypeFragmentSMS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := &tg.AuthSentCode{
				Type:     &tg.AuthSentCodeTypeSms{},
				NextType: tt.raw,
			}
			got := types.ParseSentCodeInfo(raw)
			if got.NextType != tt.want {
				t.Errorf("NextType = %q, want %q", got.NextType, tt.want)
			}
		})
	}
}

func TestParseTermsOfService_Nil(t *testing.T) {
	if got := types.ParseTermsOfService(nil); got != nil {
		t.Errorf("ParseTermsOfService(nil) = %v, want nil", got)
	}
}

func TestParseTermsOfService(t *testing.T) {
	raw := &tg.HelpTermsOfService{
		ID:            &tg.DataJSON{Data: "tos-id-123"},
		Text:          "Welcome to Telegram",
		MinAgeConfirm: 16,
	}
	got := types.ParseTermsOfService(raw)
	if got == nil {
		t.Fatal("ParseTermsOfService returned nil")
	}
	if got.ID != "tos-id-123" {
		t.Errorf("ID = %q, want %q", got.ID, "tos-id-123")
	}
	if got.Text != "Welcome to Telegram" {
		t.Errorf("Text = %q, want %q", got.Text, "Welcome to Telegram")
	}
	if got.MinAgeConfirm != 16 {
		t.Errorf("MinAgeConfirm = %d, want 16", got.MinAgeConfirm)
	}
}

func TestParseTermsOfService_NoID(t *testing.T) {
	raw := &tg.HelpTermsOfService{
		Text: "some text",
	}
	got := types.ParseTermsOfService(raw)
	if got.ID != "" {
		t.Errorf("ID = %q, want empty when ID is nil", got.ID)
	}
	if got.MinAgeConfirm != 0 {
		t.Errorf("MinAgeConfirm = %d, want 0 when not set", got.MinAgeConfirm)
	}
}

func TestParseActiveSession_Nil(t *testing.T) {
	if got := types.ParseActiveSession(nil); got != nil {
		t.Errorf("ParseActiveSession(nil) = %v, want nil", got)
	}
}

func TestParseActiveSession(t *testing.T) {
	raw := &tg.Authorization{
		Hash:          12345,
		DeviceModel:   "Pixel 8",
		Platform:      "Android",
		SystemVersion: "14",
		AppName:       "Telegram",
		AppVersion:    "10.0",
		DateCreated:   1700000000,
		DateActive:    1700086400,
		Ip:            "1.2.3.4",
		Country:       "US",
		Region:        "CA",
	}
	got := types.ParseActiveSession(raw)
	if got == nil {
		t.Fatal("ParseActiveSession returned nil")
	}
	if got.Hash != 12345 {
		t.Errorf("Hash = %d, want 12345", got.Hash)
	}
	if got.DeviceModel != "Pixel 8" {
		t.Errorf("DeviceModel = %q, want %q", got.DeviceModel, "Pixel 8")
	}
	if got.Platform != "Android" {
		t.Errorf("Platform = %q, want %q", got.Platform, "Android")
	}
	if got.ApplicationName != "Telegram" {
		t.Errorf("ApplicationName = %q, want %q", got.ApplicationName, "Telegram")
	}
	if got.IP != "1.2.3.4" {
		t.Errorf("IP = %q, want %q", got.IP, "1.2.3.4")
	}
	if got.Country != "US" {
		t.Errorf("Country = %q, want %q", got.Country, "US")
	}
	wantCreated := time.Unix(1700000000, 0)
	if !got.DateCreated.Equal(wantCreated) {
		t.Errorf("DateCreated = %v, want %v", got.DateCreated, wantCreated)
	}
}

func TestActiveSession_Reset_NoBinder(t *testing.T) {
	s := types.ParseActiveSession(&tg.Authorization{Hash: 1})
	err := s.Reset()
	if err != types.ErrNoBinder {
		t.Errorf("Reset() without binder = %v, want ErrNoBinder", err)
	}
}

func TestParseBotCommand_Nil(t *testing.T) {
	if got := types.ParseBotCommand(nil); got != nil {
		t.Errorf("ParseBotCommand(nil) = %v, want nil", got)
	}
}

func TestParseBotCommand(t *testing.T) {
	raw := &tg.BotCommand{
		Command:     "start",
		Description: "Start the bot",
	}
	got := types.ParseBotCommand(raw)
	if got == nil {
		t.Fatal("ParseBotCommand returned nil")
	}
	if got.Command != "start" || got.Description != "Start the bot" {
		t.Errorf("ParseBotCommand = {Command:%q, Description:%q}", got.Command, got.Description)
	}
}

func TestParseBotCommandScope_Nil(t *testing.T) {
	if got := types.ParseBotCommandScope(nil); got != nil {
		t.Errorf("ParseBotCommandScope(nil) = %v, want nil", got)
	}
}

func TestParseBotCommandScope_Types(t *testing.T) {
	tests := []struct {
		name     string
		raw      tg.BotCommandScopeClass
		wantTyp  types.BotCommandScopeType
		wantChat int64
		wantUser int64
	}{
		{
			"default", &tg.BotCommandScopeDefault{},
			types.BotCommandScopeDefault, 0, 0,
		},
		{
			"users", &tg.BotCommandScopeUsers{},
			types.BotCommandScopeAllPrivate, 0, 0,
		},
		{
			"chats", &tg.BotCommandScopeChats{},
			types.BotCommandScopeAllGroups, 0, 0,
		},
		{
			"chat_admins", &tg.BotCommandScopeChatAdmins{},
			types.BotCommandScopeAllChatAdmins, 0, 0,
		},
		{
			"peer",
			&tg.BotCommandScopePeer{Peer: &tg.InputPeerChat{ChatID: 100}},
			types.BotCommandScopeChat, 100, 0,
		},
		{
			"peer_admins",
			&tg.BotCommandScopePeerAdmins{Peer: &tg.InputPeerChannel{ChannelID: 200}},
			types.BotCommandScopeChatAdmins, 200, 0,
		},
		{
			"peer_user",
			&tg.BotCommandScopePeerUser{
				Peer:   &tg.InputPeerUser{UserID: 300},
				UserID: &tg.InputUser{UserID: 400},
			},
			types.BotCommandScopeChatMember, 300, 400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := types.ParseBotCommandScope(tt.raw)
			if got == nil {
				t.Fatal("ParseBotCommandScope returned nil")
			}
			if got.Type != tt.wantTyp {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantTyp)
			}
			if got.ChatID != tt.wantChat {
				t.Errorf("ChatID = %d, want %d", got.ChatID, tt.wantChat)
			}
			if got.UserID != tt.wantUser {
				t.Errorf("UserID = %d, want %d", got.UserID, tt.wantUser)
			}
		})
	}
}

func TestParseBotCommandScope_PeerFromMessage(t *testing.T) {
	raw := &tg.BotCommandScopePeerUser{
		Peer:   &tg.InputPeerUser{UserID: 500},
		UserID: &tg.InputUserFromMessage{UserID: 600},
	}
	got := types.ParseBotCommandScope(raw)
	if got.UserID != 600 {
		t.Errorf("UserID = %d, want 600 (InputUserFromMessage)", got.UserID)
	}
}

func TestParseChatShared_Nil(t *testing.T) {
	if got := types.ParseChatShared(nil); got != nil {
		t.Errorf("ParseChatShared(nil) = %v, want nil", got)
	}
}

func TestParseChatShared(t *testing.T) {
	raw := &tg.RequestedPeerChat{
		ChatID: 123,
		Title:  "Test Chat",
	}
	got := types.ParseChatShared(raw)
	if got == nil {
		t.Fatal("ParseChatShared returned nil")
	}
	if got.ChatID != 123 {
		t.Errorf("ChatID = %d, want 123", got.ChatID)
	}
	if got.Title != "Test Chat" {
		t.Errorf("Title = %q, want %q", got.Title, "Test Chat")
	}
}

func TestParseChatShared_NoTitle(t *testing.T) {
	raw := &tg.RequestedPeerChat{ChatID: 456}
	got := types.ParseChatShared(raw)
	if got.Title != "" {
		t.Errorf("Title = %q, want empty when not set", got.Title)
	}
}

func TestParseMessageReactionUpdated_Nil(t *testing.T) {
	if got := types.ParseMessageReactionUpdated(nil); got != nil {
		t.Errorf("ParseMessageReactionUpdated(nil) = %v, want nil", got)
	}
}

func TestParseMessageReactionUpdated(t *testing.T) {
	raw := &tg.UpdateBotMessageReaction{
		MsgID: 42,
		Date:  1700000000,
		OldReactions: []tg.ReactionClass{
			&tg.ReactionEmoji{Emoticon: "\xf0\x9f\x91\x8d"},
		},
		NewReactions: []tg.ReactionClass{
			&tg.ReactionEmoji{Emoticon: "\xe2\x9d\xa4\xef\xb8\x8f"},
			&tg.ReactionCustomEmoji{DocumentID: 999},
			&tg.ReactionPaid{},
		},
	}
	got := types.ParseMessageReactionUpdated(raw)
	if got == nil {
		t.Fatal("ParseMessageReactionUpdated returned nil")
	}
	if got.MessageID != 42 {
		t.Errorf("MessageID = %d, want 42", got.MessageID)
	}
	wantDate := time.Unix(1700000000, 0)
	if !got.Date.Equal(wantDate) {
		t.Errorf("Date = %v, want %v", got.Date, wantDate)
	}
	if len(got.OldReaction) != 1 || got.OldReaction[0].Emoji != "\xf0\x9f\x91\x8d" {
		t.Errorf("OldReaction = %+v", got.OldReaction)
	}
	if len(got.NewReaction) != 3 {
		t.Fatalf("NewReaction len = %d, want 3", len(got.NewReaction))
	}
	if got.NewReaction[0].Emoji != "\xe2\x9d\xa4\xef\xb8\x8f" {
		t.Errorf("NewReaction[0].Emoji = %q", got.NewReaction[0].Emoji)
	}
	if got.NewReaction[1].CustomEmojiID == "" {
		t.Error("NewReaction[1].CustomEmojiID should not be empty")
	}
	if !got.NewReaction[2].IsPaid {
		t.Error("NewReaction[2].IsPaid should be true")
	}
	if len(got.Reactions) != 3 {
		t.Fatalf("Reactions len = %d, want 3", len(got.Reactions))
	}
	if got.Reactions[0].Type != "\xe2\x9d\xa4\xef\xb8\x8f" || got.Reactions[0].Count != 1 {
		t.Errorf("Reactions[0] = %+v", got.Reactions[0])
	}
	if got.Reactions[1].Type != "custom" {
		t.Errorf("Reactions[1].Type = %q, want %q", got.Reactions[1].Type, "custom")
	}
	if got.Reactions[2].Type != "paid" {
		t.Errorf("Reactions[2].Type = %q, want %q", got.Reactions[2].Type, "paid")
	}
}

func TestParseMessageReactionCountUpdated_Nil(t *testing.T) {
	if got := types.ParseMessageReactionCountUpdated(nil); got != nil {
		t.Errorf("ParseMessageReactionCountUpdated(nil) = %v, want nil", got)
	}
}

func TestParseMessageReactionCountUpdated(t *testing.T) {
	raw := &tg.UpdateBotMessageReactions{
		MsgID: 55,
		Date:  1700000000,
		Reactions: []*tg.ReactionCount{
			{Reaction: &tg.ReactionEmoji{Emoticon: "\xf0\x9f\x91\x8d"}, Count: 5},
			{Reaction: &tg.ReactionCustomEmoji{DocumentID: 100}, Count: 3},
			{Reaction: &tg.ReactionPaid{}, Count: 2},
		},
	}
	got := types.ParseMessageReactionCountUpdated(raw)
	if got == nil {
		t.Fatal("ParseMessageReactionCountUpdated returned nil")
	}
	if got.MessageID != 55 {
		t.Errorf("MessageID = %d, want 55", got.MessageID)
	}
	if len(got.Reactions) != 3 {
		t.Fatalf("Reactions len = %d, want 3", len(got.Reactions))
	}
	if got.Reactions[0].Type != "\xf0\x9f\x91\x8d" || got.Reactions[0].Count != 5 {
		t.Errorf("Reactions[0] = %+v", got.Reactions[0])
	}
	if got.Reactions[1].Type != "custom" || got.Reactions[1].Count != 3 {
		t.Errorf("Reactions[1] = %+v", got.Reactions[1])
	}
	if got.Reactions[2].Type != "paid" || got.Reactions[2].Count != 2 {
		t.Errorf("Reactions[2] = %+v", got.Reactions[2])
	}
}

func TestParseBusinessConnection_Nil(t *testing.T) {
	if got := types.ParseBusinessConnection(nil, nil); got != nil {
		t.Errorf("ParseBusinessConnection(nil, nil) = %v, want nil", got)
	}
}

func TestParseBusinessConnection(t *testing.T) {
	user := &types.User{ID: 100}
	raw := &tg.BotBusinessConnection{
		ConnectionID: "conn-1",
		DCID:         2,
		Date:         1700000000,
		Disabled:     false,
		Rights: &tg.BusinessBotRights{
			Reply:        true,
			ReadMessages: true,
		},
	}
	got := types.ParseBusinessConnection(raw, user)
	if got == nil {
		t.Fatal("ParseBusinessConnection returned nil")
	}
	if got.ID != "conn-1" {
		t.Errorf("ID = %q, want %q", got.ID, "conn-1")
	}
	if got.DC != 2 {
		t.Errorf("DC = %d, want 2", got.DC)
	}
	if !got.IsEnabled {
		t.Error("IsEnabled = false, want true (Disabled=false)")
	}
	if got.User != user {
		t.Error("User not set correctly")
	}
	if got.Rights == nil || !got.Rights.CanReply || !got.Rights.CanReadMessages {
		t.Errorf("Rights = %+v", got.Rights)
	}
}

func TestParseBusinessConnection_Disabled(t *testing.T) {
	raw := &tg.BotBusinessConnection{
		ConnectionID: "conn-2",
		Disabled:     true,
	}
	got := types.ParseBusinessConnection(raw, nil)
	if got.IsEnabled {
		t.Error("IsEnabled = true, want false (Disabled=true)")
	}
}

func TestParseBusinessIntro_Nil(t *testing.T) {
	if got := types.ParseBusinessIntro(nil); got != nil {
		t.Errorf("ParseBusinessIntro(nil) = %v, want nil", got)
	}
}

func TestParseBusinessIntro(t *testing.T) {
	raw := &tg.BusinessIntro{
		Title:       "Welcome",
		Description: "We are open!",
	}
	got := types.ParseBusinessIntro(raw)
	if got == nil {
		t.Fatal("ParseBusinessIntro returned nil")
	}
	if got.Title != "Welcome" {
		t.Errorf("Title = %q, want %q", got.Title, "Welcome")
	}
	if got.Text != "We are open!" {
		t.Errorf("Text = %q, want %q", got.Text, "We are open!")
	}
	if got.Sticker != nil {
		t.Error("Sticker should be nil when not a *tg.Document")
	}
}

func TestParseBusinessWorkingHours_Nil(t *testing.T) {
	if got := types.ParseBusinessWorkingHours(nil); got != nil {
		t.Errorf("ParseBusinessWorkingHours(nil) = %v, want nil", got)
	}
}

func TestParseBusinessWorkingHours(t *testing.T) {
	raw := &tg.BusinessWorkHours{
		OpenNow:    true,
		TimezoneID: "Europe/Berlin",
		WeeklyOpen: []*tg.BusinessWeeklyOpen{
			{StartMinute: 0, EndMinute: 480},
			{StartMinute: 600, EndMinute: 1080},
		},
	}
	got := types.ParseBusinessWorkingHours(raw)
	if got == nil {
		t.Fatal("ParseBusinessWorkingHours returned nil")
	}
	if !got.OpenNow {
		t.Error("OpenNow = false, want true")
	}
	if got.TimeZone != "Europe/Berlin" {
		t.Errorf("TimeZone = %q, want %q", got.TimeZone, "Europe/Berlin")
	}
	if len(got.WeeklyOpen) != 2 {
		t.Fatalf("WeeklyOpen len = %d, want 2", len(got.WeeklyOpen))
	}
	if got.WeeklyOpen[0].StartMinute != 0 || got.WeeklyOpen[0].EndMinute != 480 {
		t.Errorf("WeeklyOpen[0] = %+v", got.WeeklyOpen[0])
	}
	if got.WeeklyOpen[1].StartMinute != 600 || got.WeeklyOpen[1].EndMinute != 1080 {
		t.Errorf("WeeklyOpen[1] = %+v", got.WeeklyOpen[1])
	}
}

func TestParseBusinessWorkingHours_NilWeeklyOpen(t *testing.T) {
	raw := &tg.BusinessWorkHours{
		OpenNow:    false,
		TimezoneID: "UTC",
		WeeklyOpen: []*tg.BusinessWeeklyOpen{nil, {StartMinute: 100, EndMinute: 200}},
	}
	got := types.ParseBusinessWorkingHours(raw)
	if len(got.WeeklyOpen) != 1 {
		t.Errorf("WeeklyOpen len = %d, want 1 (nil entry skipped)", len(got.WeeklyOpen))
	}
}

func TestParseBusinessBotRights_Nil(t *testing.T) {
	if got := types.ParseBusinessBotRights(nil); got != nil {
		t.Errorf("ParseBusinessBotRights(nil) = %v, want nil", got)
	}
}

func TestParseBusinessBotRights(t *testing.T) {
	raw := &tg.BusinessBotRights{
		Reply:              true,
		ReadMessages:       false,
		DeleteSentMessages: true,
		EditName:           true,
		ManageStories:      true,
		TransferStars:      false,
	}
	got := types.ParseBusinessBotRights(raw)
	if got == nil {
		t.Fatal("ParseBusinessBotRights returned nil")
	}
	if !got.CanReply {
		t.Error("CanReply = false, want true")
	}
	if got.CanReadMessages {
		t.Error("CanReadMessages = true, want false")
	}
	if !got.CanDeleteSentMessages {
		t.Error("CanDeleteSentMessages = false, want true")
	}
	if !got.CanEditName {
		t.Error("CanEditName = false, want true")
	}
	if !got.CanManageStories {
		t.Error("CanManageStories = false, want true")
	}
	if got.CanTransferStars {
		t.Error("CanTransferStars = true, want false")
	}
}

func TestBusinessScheduleWeekday_String(t *testing.T) {
	tests := []struct {
		name string
		s    types.BusinessSchedule
		want string
	}{
		{"always_open", types.BusinessScheduleAlwaysOpen, "always_open"},
		{"by_date", types.BusinessScheduleByDate, "by_date"},
		{"custom", types.BusinessScheduleCustom, "custom"},
		{"outside", types.BusinessScheduleOutside, "outside"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReactionString_All(t *testing.T) {
	raw := &tg.UpdateBotMessageReaction{
		MsgID: 1,
		Date:  1,
		NewReactions: []tg.ReactionClass{
			&tg.ReactionEmpty{},
			&tg.ReactionEmoji{Emoticon: "\xf0\x9f\x91\x8d"},
			&tg.ReactionCustomEmoji{DocumentID: 1},
			&tg.ReactionPaid{},
		},
	}
	got := types.ParseMessageReactionUpdated(raw)
	if len(got.Reactions) != 4 {
		t.Fatalf("Reactions len = %d, want 4", len(got.Reactions))
	}
	if got.Reactions[0].Type != "" {
		t.Errorf("empty reaction Type = %q, want empty", got.Reactions[0].Type)
	}
	if got.Reactions[1].Type != "\xf0\x9f\x91\x8d" {
		t.Errorf("emoji reaction Type = %q", got.Reactions[1].Type)
	}
	if got.Reactions[2].Type != "custom" {
		t.Errorf("custom emoji Type = %q, want %q", got.Reactions[2].Type, "custom")
	}
	if got.Reactions[3].Type != "paid" {
		t.Errorf("paid reaction Type = %q, want %q", got.Reactions[3].Type, "paid")
	}
}
