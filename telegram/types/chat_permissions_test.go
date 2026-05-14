package types

import (
	"testing"

	tl "github.com/mtgo-labs/mtgo/tg"
)

func TestParseChatPermissions_Nil(t *testing.T) {
	p := ParseChatPermissions(nil)
	if p == nil {
		t.Fatal("ParseChatPermissions(nil) should return full permissions")
	}
	if !p.CanSendMessages {
		t.Error("nil rights should mean full permissions")
	}
}

func TestParseChatPermissions_Full(t *testing.T) {
	raw := &tl.ChatBannedRights{
		SendMessages: true,
		SendMedia:    true,
		SendPolls:    true,
		SendInline:   true,
		EmbedLinks:   true,
		ChangeInfo:   true,
		InviteUsers:  true,
		PinMessages:  true,
	}
	p := ParseChatPermissions(raw)
	if p.CanSendMessages {
		t.Error("CanSendMessages should be false when banned")
	}
	if p.CanSendMedia {
		t.Error("CanSendMedia should be false when banned")
	}
}

func TestParseChatPermissions_Allowed(t *testing.T) {
	raw := &tl.ChatBannedRights{}
	p := ParseChatPermissions(raw)
	if !p.CanSendMessages {
		t.Error("CanSendMessages should be true when not banned")
	}
}
