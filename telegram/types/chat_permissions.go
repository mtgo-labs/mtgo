package types

import "github.com/mtgo-labs/mtgo/tg"

// ChatPermissions represents the set of actions a non-admin user is allowed to
// take in a chat. Each field corresponds to a specific permission; a false value
// means the action is restricted. This type inverts the MTProto ChatBannedRights
// (where true = banned) into a positive permission model.
//
// Example:
//
//	perms := &types.ChatPermissions{
//	    CanSendMessages:  true,
//	    CanSendMedia:     true,
//	    CanPinMessages:   false,
//	    CanInviteUsers:   false,
//	}
//	_ = perms
type ChatPermissions struct {
	// CanSendMessages controls whether the user can send text messages.
	CanSendMessages bool
	// CanSendMedia controls whether the user can send photos, videos, documents,
	// and other media.
	CanSendMedia bool
	// CanSendPolls controls whether the user can create polls.
	CanSendPolls bool
	// CanSendOtherMessages controls whether the user can send inline messages,
	// stickers, and other non-media content.
	CanSendOtherMessages bool
	// CanAddWebPagePreviews controls whether link previews are shown for URLs the
	// user sends.
	CanAddWebPagePreviews bool
	// CanChangeInfo controls whether the user can change the chat title, photo,
	// and description.
	CanChangeInfo bool
	// CanInviteUsers controls whether the user can invite new members via link.
	CanInviteUsers bool
	// CanPinMessages controls whether the user can pin messages.
	CanPinMessages bool
}

// ParseChatPermissions converts an MTProto ChatBannedRights into ChatPermissions.
// If raw is nil, all permissions are returned as enabled (default unrestricted
// state), since the absence of banned rights implies full access.
func ParseChatPermissions(raw *tg.ChatBannedRights) *ChatPermissions {
	if raw == nil {
		return &ChatPermissions{
			CanSendMessages:       true,
			CanSendMedia:          true,
			CanSendPolls:          true,
			CanSendOtherMessages:  true,
			CanAddWebPagePreviews: true,
			CanChangeInfo:         true,
			CanInviteUsers:        true,
			CanPinMessages:        true,
		}
	}
	return &ChatPermissions{
		CanSendMessages:       !raw.SendMessages,
		CanSendMedia:          !raw.SendMedia,
		CanSendPolls:          !raw.SendPolls,
		CanSendOtherMessages:  !raw.SendInline,
		CanAddWebPagePreviews: !raw.EmbedLinks,
		CanChangeInfo:         !raw.ChangeInfo,
		CanInviteUsers:        !raw.InviteUsers,
		CanPinMessages:        !raw.PinMessages,
	}
}
