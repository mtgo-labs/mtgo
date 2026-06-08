package types

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestParseChannelParticipantUnwrapsChannelParticipantResult(t *testing.T) {
	const userID = int64(2002)

	member := ParseChannelParticipant(&tg.ChannelsChannelParticipant{
		Participant: &tg.ChannelParticipantAdmin{
			UserID: userID,
			AdminRights: &tg.ChatAdminRights{
				PostMessages: true,
			},
		},
		Users: []tg.UserClass{
			&tg.User{ID: userID, FirstName: "Bot", Bot: true},
		},
	}, nil)

	if member == nil {
		t.Fatal("member is nil")
	}
	if member.Status != ChatMemberStatusAdministrator {
		t.Fatalf("Status = %q", member.Status)
	}
	if member.User == nil || member.User.ID != userID || member.User.FirstName != "Bot" {
		t.Fatalf("User = %+v", member.User)
	}
	if member.Privileges == nil || !member.Privileges.CanPostMessages {
		t.Fatalf("Privileges = %+v", member.Privileges)
	}
}
