package types

import "github.com/mtgo-labs/mtgo/tg"

// ChatAdminRights represents the set of administrative privileges granted to a
// user in a chat. Used when promoting a member to admin.
//
// Example:
//
//	rights := &types.ChatAdminRights{
//	    CanDeleteMessages: true,
//	    CanBanUsers:       true,
//	    CanPinMessages:    true,
//	}
//	_ = rights
type ChatAdminRights struct {
	// CanChangeInfo allows changing the chat title, photo, and description.
	CanChangeInfo bool
	// CanPostMessages allows posting messages in channels.
	CanPostMessages bool
	// CanEditMessages allows editing messages sent by other members.
	CanEditMessages bool
	// CanDeleteMessages allows deleting messages sent by any member.
	CanDeleteMessages bool
	// CanBanUsers allows banning and kicking members.
	CanBanUsers bool
	// CanInviteUsers allows inviting new members via invite link.
	CanInviteUsers bool
	// CanPinMessages allows pinning messages.
	CanPinMessages bool
	// CanAddAdmins allows promoting other members to admin.
	CanAddAdmins bool
	// IsAnonymous hides the admin's identity when sending messages (messages
	// appear as sent by the chat itself).
	IsAnonymous bool
	// CanManageVideoChats allows managing group voice/video calls.
	CanManageVideoChats bool
	// CanManageChat grants general chat management permissions (the "Other"
	// flag in MTProto).
	CanManageChat bool
	// CanManageTopics allows creating, editing, and deleting forum topics.
	CanManageTopics bool
	// CanPostStories allows posting stories on behalf of the channel.
	CanPostStories bool
	// CanEditStories allows editing stories on behalf of the channel.
	CanEditStories bool
	// CanDeleteStories allows deleting stories on behalf of the channel.
	CanDeleteStories bool
}

// ParseChatAdminRights converts an MTProto ChatAdminRights to the domain type.
// Returns nil if raw is nil.
func ParseChatAdminRights(raw *tg.ChatAdminRights) *ChatAdminRights {
	if raw == nil {
		return nil
	}
	return &ChatAdminRights{
		CanChangeInfo:       raw.ChangeInfo,
		CanPostMessages:     raw.PostMessages,
		CanEditMessages:     raw.EditMessages,
		CanDeleteMessages:   raw.DeleteMessages,
		CanBanUsers:         raw.BanUsers,
		CanInviteUsers:      raw.InviteUsers,
		CanPinMessages:      raw.PinMessages,
		CanAddAdmins:        raw.AddAdmins,
		IsAnonymous:         raw.Anonymous,
		CanManageVideoChats: raw.ManageCall,
		CanManageChat:       raw.Other,
		CanManageTopics:     raw.ManageTopics,
		CanPostStories:      raw.PostStories,
		CanEditStories:      raw.EditStories,
		CanDeleteStories:    raw.DeleteStories,
	}
}

// ChatBannedRights represents the set of restrictions applied to a user in a
// chat. Unlike ChatPermissions (which inverts the semantics), this type
// preserves the "true = restricted" convention from the MTProto layer so it can
// be passed directly back to the API.
type ChatBannedRights struct {
	// CanSendMessages is true when sending text messages is restricted.
	CanSendMessages bool
	// CanSendMedia is true when sending media (photos, videos, etc.) is
	// restricted.
	CanSendMedia bool
	// CanSendPolls is true when creating polls is restricted.
	CanSendPolls bool
	// CanSendOtherMessages is true when sending inline/sticker messages is
	// restricted.
	CanSendOtherMessages bool
	// CanAddWebPagePreviews is true when link previews are restricted.
	CanAddWebPagePreviews bool
	// CanChangeInfo is true when changing chat info is restricted.
	CanChangeInfo bool
	// CanInviteUsers is true when inviting users is restricted.
	CanInviteUsers bool
	// CanPinMessages is true when pinning messages is restricted.
	CanPinMessages bool
	// UntilDate is the Unix timestamp when the restriction expires, or 0 for
	// permanent restrictions and forever bans.
	UntilDate int32
}

// ParseChatBannedRights converts an MTProto ChatBannedRights to the domain type.
// Returns nil if raw is nil.
func ParseChatBannedRights(raw *tg.ChatBannedRights) *ChatBannedRights {
	if raw == nil {
		return nil
	}
	return &ChatBannedRights{
		CanSendMessages:       raw.SendMessages,
		CanSendMedia:          raw.SendMedia,
		CanSendPolls:          raw.SendPolls,
		CanSendOtherMessages:  raw.SendInline,
		CanAddWebPagePreviews: raw.EmbedLinks,
		CanChangeInfo:         raw.ChangeInfo,
		CanInviteUsers:        raw.InviteUsers,
		CanPinMessages:        raw.PinMessages,
		UntilDate:             raw.UntilDate,
	}
}
