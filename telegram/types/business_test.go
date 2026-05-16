package types

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestParseBusinessBotRights(t *testing.T) {
	raw := &tg.BusinessBotRights{
		Reply:                   true,
		ReadMessages:            true,
		DeleteSentMessages:      true,
		DeleteReceivedMessages:  true,
		EditName:                true,
		EditBio:                 true,
		EditProfilePhoto:        true,
		EditUsername:            true,
		ViewGifts:               true,
		SellGifts:               true,
		ChangeGiftSettings:      true,
		TransferAndUpgradeGifts: true,
		TransferStars:           true,
		ManageStories:           true,
	}

	permissions := ParseBusinessBotRights(raw)
	if permissions == nil {
		t.Fatal("ParseBusinessBotRights returned nil")
	}

	if !permissions.CanReply ||
		!permissions.CanReadMessages ||
		!permissions.CanDeleteSentMessages ||
		!permissions.CanDeleteAllMessages ||
		!permissions.CanEditName ||
		!permissions.CanEditBio ||
		!permissions.CanEditProfilePhoto ||
		!permissions.CanEditUsername ||
		!permissions.CanViewGiftsAndStars ||
		!permissions.CanSellGifts ||
		!permissions.CanChangeGiftSettings ||
		!permissions.CanTransferAndUpgradeGifts ||
		!permissions.CanTransferStars ||
		!permissions.CanManageStories {
		t.Fatalf("permissions not fully mapped: %+v", permissions)
	}
}

func TestParseBusinessBotRights_Nil(t *testing.T) {
	if permissions := ParseBusinessBotRights(nil); permissions != nil {
		t.Fatalf("ParseBusinessBotRights(nil) = %+v, want nil", permissions)
	}
}
