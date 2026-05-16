package types

import "github.com/mtgo-labs/mtgo/tg"

// ChatPermissions represents the set of actions a user is allowed to perform
// in a chat. A true value means the permission is granted.
//
// Example:
//
//	perms := types.ParseChatPermissions(rawBannedRights)
//	if !perms.CanSendMessages {
//	    fmt.Println("User is muted in this chat")
//	}
type ChatPermissions struct {
	CanSendMessages       bool
	CanSendAudios         bool
	CanSendDocuments      bool
	CanSendPhotos         bool
	CanSendVideos         bool
	CanSendVideoNotes     bool
	CanSendVoiceNotes     bool
	CanSendPolls          bool
	CanSendOtherMessages  bool
	CanAddWebPagePreviews bool
	CanReactToMessages    bool
	CanEditTag            bool
	CanChangeInfo         bool
	CanInviteUsers        bool
	CanPinMessages        bool
	CanManageTopics       bool
}

// ParseChatPermissions converts a TL ChatBannedRights into a ChatPermissions.
// When raw is nil, all permissions are granted (fully unrestricted). The banned
// rights are inverted so that true means "allowed".
//
// Example:
//
//	perms := types.ParseChatPermissions(rawBannedRights)
//	fmt.Printf("Can send photos: %v, Can pin: %v\n", perms.CanSendPhotos, perms.CanPinMessages)
func ParseChatPermissions(raw *tg.ChatBannedRights) *ChatPermissions {
	if raw == nil {
		return &ChatPermissions{
			CanSendMessages:       true,
			CanSendAudios:         true,
			CanSendDocuments:      true,
			CanSendPhotos:         true,
			CanSendVideos:         true,
			CanSendVideoNotes:     true,
			CanSendVoiceNotes:     true,
			CanSendPolls:          true,
			CanSendOtherMessages:  true,
			CanAddWebPagePreviews: true,
			CanReactToMessages:    true,
			CanChangeInfo:         true,
			CanInviteUsers:        true,
			CanPinMessages:        true,
			CanManageTopics:       true,
		}
	}
	return &ChatPermissions{
		CanSendMessages:       !raw.SendMessages,
		CanSendAudios:         !raw.SendAudios,
		CanSendDocuments:      !raw.SendDocs,
		CanSendPhotos:         !raw.SendPhotos,
		CanSendVideos:         !raw.SendVideos,
		CanSendVideoNotes:     !raw.SendRoundvideos,
		CanSendVoiceNotes:     !raw.SendVoices,
		CanSendPolls:          !raw.SendPolls,
		CanSendOtherMessages:  !raw.SendInline,
		CanAddWebPagePreviews: !raw.EmbedLinks,
		CanReactToMessages:    !raw.SendReactions,
		CanEditTag:            !raw.EditRank,
		CanChangeInfo:         !raw.ChangeInfo,
		CanInviteUsers:        !raw.InviteUsers,
		CanPinMessages:        !raw.PinMessages,
		CanManageTopics:       !raw.ManageTopics,
	}
}
