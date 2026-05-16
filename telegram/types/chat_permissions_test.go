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
		SendMessages:    true,
		SendAudios:      true,
		SendDocs:        true,
		SendPhotos:      true,
		SendVideos:      true,
		SendRoundvideos: true,
		SendVoices:      true,
		SendPolls:       true,
		SendInline:      true,
		EmbedLinks:      true,
		ChangeInfo:      true,
		InviteUsers:     true,
		PinMessages:     true,
		ManageTopics:    true,
		SendReactions:   true,
		EditRank:        true,
	}
	p := ParseChatPermissions(raw)
	if p.CanSendMessages {
		t.Error("CanSendMessages should be false when banned")
	}
	if p.CanSendAudios {
		t.Error("CanSendAudios should be false when banned")
	}
	if p.CanSendDocuments {
		t.Error("CanSendDocuments should be false when banned")
	}
	if p.CanSendPhotos {
		t.Error("CanSendPhotos should be false when banned")
	}
	if p.CanSendVideos {
		t.Error("CanSendVideos should be false when banned")
	}
	if p.CanSendVideoNotes {
		t.Error("CanSendVideoNotes should be false when banned")
	}
	if p.CanSendVoiceNotes {
		t.Error("CanSendVoiceNotes should be false when banned")
	}
}

func TestParseChatPermissions_Allowed(t *testing.T) {
	raw := &tl.ChatBannedRights{}
	p := ParseChatPermissions(raw)
	if !p.CanSendMessages {
		t.Error("CanSendMessages should be true when not banned")
	}
}
