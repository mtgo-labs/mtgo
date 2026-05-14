package types

import (
	"testing"

	tl "github.com/mtgo-labs/mtgo/tg"
)

func TestParseChatAdminRights_Nil(t *testing.T) {
	if ParseChatAdminRights(nil) != nil {
		t.Error("ParseChatAdminRights(nil) should return nil")
	}
}

func TestParseChatAdminRights_Full(t *testing.T) {
	raw := &tl.ChatAdminRights{
		ChangeInfo:   true,
		PostMessages: true,
		EditMessages: true,
		DeleteMessages: true,
		BanUsers:     true,
		InviteUsers:  true,
		PinMessages:  true,
		AddAdmins:    true,
		Anonymous:    true,
		ManageCall:   true,
		Other:        true,
		ManageTopics: true,
	}
	r := ParseChatAdminRights(raw)
	if r == nil {
		t.Fatal("ParseChatAdminRights returned nil")
	}
	if !r.CanChangeInfo || !r.CanPostMessages || !r.CanDeleteMessages {
		t.Error("expected admin rights to be true")
	}
	if !r.CanBanUsers || !r.CanInviteUsers || !r.CanPinMessages {
		t.Error("expected admin rights to be true")
	}
	if !r.CanAddAdmins || !r.IsAnonymous || !r.CanManageVideoChats {
		t.Error("expected admin rights to be true")
	}
	if !r.CanManageChat || !r.CanManageTopics {
		t.Error("expected admin rights to be true")
	}
}

func TestParseChatBannedRights_Nil(t *testing.T) {
	if ParseChatBannedRights(nil) != nil {
		t.Error("ParseChatBannedRights(nil) should return nil")
	}
}

func TestParseChatBannedRights(t *testing.T) {
	raw := &tl.ChatBannedRights{
		SendMessages: true,
		UntilDate:    1700000000,
	}
	r := ParseChatBannedRights(raw)
	if r == nil {
		t.Fatal("ParseChatBannedRights returned nil")
	}
	if !r.CanSendMessages {
		t.Error("CanSendMessages should be true (direct mapping)")
	}
	if r.UntilDate != 1700000000 {
		t.Errorf("UntilDate = %d, want 1700000000", r.UntilDate)
	}
}
