package types

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestParseGlobalPrivacySettings(t *testing.T) {
	raw := &tg.GlobalPrivacySettings{
		ArchiveAndMuteNewNoncontactPeers: true,
		KeepArchivedUnmuted:              true,
		KeepArchivedFolders:              true,
		HideReadMarks:                    false,
		NewNoncontactPeersRequirePremium: false,
		DisplayGiftsButton:               true,
		NoncontactPeersPaidStars:         25,
		DisallowedGifts: &tg.DisallowedGiftsSettings{
			DisallowLimitedStargifts:      true,
			DisallowStargiftsFromChannels: true,
		},
	}

	settings := ParseGlobalPrivacySettings(raw)
	if settings == nil {
		t.Fatal("ParseGlobalPrivacySettings returned nil")
	}

	if !settings.ArchiveAndMuteNewChats ||
		!settings.KeepUnmutedChatsArchived ||
		!settings.KeepChatsFromFoldersArchived ||
		!settings.ShowReadDate ||
		!settings.AllowNewChatsFromUnknownUsers ||
		!settings.ShowGiftButton {
		t.Fatalf("boolean settings not fully mapped: %+v", settings)
	}
	if settings.IncomingPaidMessageStarCount != 25 {
		t.Fatalf("IncomingPaidMessageStarCount = %d, want 25", settings.IncomingPaidMessageStarCount)
	}
	if settings.AcceptedGiftTypes == nil {
		t.Fatal("AcceptedGiftTypes is nil")
	}
	if !settings.AcceptedGiftTypes.UnlimitedGifts ||
		settings.AcceptedGiftTypes.LimitedGifts ||
		!settings.AcceptedGiftTypes.UpgradedGifts ||
		settings.AcceptedGiftTypes.GiftsFromChannels ||
		!settings.AcceptedGiftTypes.PremiumSubscription {
		t.Fatalf("AcceptedGiftTypes not mapped from disallowed settings: %+v", settings.AcceptedGiftTypes)
	}
}

func TestParseGlobalPrivacySettings_InvertedBooleans(t *testing.T) {
	settings := ParseGlobalPrivacySettings(&tg.GlobalPrivacySettings{
		HideReadMarks:                    true,
		NewNoncontactPeersRequirePremium: true,
	})

	if settings.ShowReadDate {
		t.Fatal("ShowReadDate = true, want false")
	}
	if settings.AllowNewChatsFromUnknownUsers {
		t.Fatal("AllowNewChatsFromUnknownUsers = true, want false")
	}
}

func TestParseGlobalPrivacySettings_Nil(t *testing.T) {
	if settings := ParseGlobalPrivacySettings(nil); settings != nil {
		t.Fatalf("ParseGlobalPrivacySettings(nil) = %+v, want nil", settings)
	}
}
