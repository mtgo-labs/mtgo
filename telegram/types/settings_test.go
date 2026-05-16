package types

import (
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestParseChatSettings(t *testing.T) {
	raw := &tg.PeerSettings{
		ReportSpam:             true,
		AddContact:             true,
		BlockContact:           true,
		ShareContact:           true,
		NeedContactsException:  true,
		ReportGeo:              true,
		Autoarchived:           true,
		InviteMembers:          true,
		RequestChatBroadcast:   true,
		BusinessBotPaused:      true,
		BusinessBotCanReply:    true,
		GeoDistance:            120,
		RequestChatTitle:       "join target",
		RequestChatDate:        1700000000,
		BusinessBotID:          42,
		BusinessBotManageURL:   "https://t.me/example_bot/manage",
		ChargePaidMessageStars: 15,
		RegistrationMonth:      "05.2026",
		PhoneCountry:           "IQ",
		NameChangeDate:         1700000100,
		PhotoChangeDate:        1700000200,
	}
	users := map[int64]tg.UserClass{
		42: &tg.User{
			ID:        42,
			FirstName: "Business",
			Bot:       true,
		},
	}

	settings := ParseChatSettings(raw, users)
	if settings == nil {
		t.Fatal("ParseChatSettings returned nil")
	}

	if !settings.CanReportSpam ||
		!settings.CanAddContact ||
		!settings.CanBlockContact ||
		!settings.CanShareContact ||
		!settings.CanReportGeo ||
		!settings.CanInviteMembers ||
		!settings.IsAutoArchived ||
		!settings.IsBusinessBotPaused ||
		!settings.IsBusinessBotCanReply ||
		!settings.NeedContactsException ||
		!settings.RequestChatBroadcast {
		t.Fatalf("boolean settings not fully mapped: %+v", settings)
	}
	if settings.GeoDistance != 120 {
		t.Fatalf("GeoDistance = %d, want 120", settings.GeoDistance)
	}
	if settings.RequestChatTitle != "join target" {
		t.Fatalf("RequestChatTitle = %q, want %q", settings.RequestChatTitle, "join target")
	}
	if !settings.RequestChatDate.Equal(time.Unix(1700000000, 0)) {
		t.Fatalf("RequestChatDate = %v, want %v", settings.RequestChatDate, time.Unix(1700000000, 0))
	}
	if settings.BusinessBot == nil || settings.BusinessBot.ID != 42 {
		t.Fatalf("BusinessBot = %+v, want user 42", settings.BusinessBot)
	}
	if settings.BusinessBotManageURL != "https://t.me/example_bot/manage" {
		t.Fatalf("BusinessBotManageURL = %q", settings.BusinessBotManageURL)
	}
	if settings.ChargePaidMessageStars != 15 {
		t.Fatalf("ChargePaidMessageStars = %d, want 15", settings.ChargePaidMessageStars)
	}
	if settings.RegistrationDate != "05.2026" {
		t.Fatalf("RegistrationDate = %q, want %q", settings.RegistrationDate, "05.2026")
	}
	if settings.PhoneNumberCountryCode != "IQ" {
		t.Fatalf("PhoneNumberCountryCode = %q, want IQ", settings.PhoneNumberCountryCode)
	}
	if !settings.LastNameChangeDate.Equal(time.Unix(1700000100, 0)) {
		t.Fatalf("LastNameChangeDate = %v, want %v", settings.LastNameChangeDate, time.Unix(1700000100, 0))
	}
	if !settings.LastPhotoChangeDate.Equal(time.Unix(1700000200, 0)) {
		t.Fatalf("LastPhotoChangeDate = %v, want %v", settings.LastPhotoChangeDate, time.Unix(1700000200, 0))
	}
}

func TestParseChatSettings_Nil(t *testing.T) {
	if settings := ParseChatSettings(nil); settings != nil {
		t.Fatalf("ParseChatSettings(nil) = %+v, want nil", settings)
	}
}
