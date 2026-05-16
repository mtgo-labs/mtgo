package types

import "github.com/mtgo-labs/mtgo/tg"

// ChatAdministratorRights represents the granular permissions granted to an
// administrator in a chat or channel.
//
// Example:
//
//	rights := types.ParseChatAdministratorRights(rawAdminRights)
//	fmt.Printf("Can post: %v, Can delete: %v\n", rights.CanPostMessages, rights.CanDeleteMessages)
type ChatAdministratorRights struct {
	IsAnonymous             bool
	CanManageChat           bool
	CanDeleteMessages       bool
	CanManageVideoChats     bool
	CanRestrictMembers      bool
	CanPromoteMembers       bool
	CanChangeInfo           bool
	CanInviteUsers          bool
	CanPostStories          bool
	CanEditStories          bool
	CanDeleteStories        bool
	CanPostMessages         bool
	CanEditMessages         bool
	CanPinMessages          bool
	CanManageTopics         bool
	CanManageDirectMessages bool
	CanManageTags           bool
}

// ParseChatAdministratorRights converts a TL ChatAdminRights into a ChatAdministratorRights.
// Returns nil if raw is nil.
//
// Example:
//
//	rights := types.ParseChatAdministratorRights(rawRights)
//	if rights != nil && rights.CanManageChat {
//	    fmt.Println("User can manage this chat")
//	}
func ParseChatAdministratorRights(raw *tg.ChatAdminRights) *ChatAdministratorRights {
	if raw == nil {
		return nil
	}
	return &ChatAdministratorRights{
		IsAnonymous:             raw.Anonymous,
		CanManageChat:           raw.Other,
		CanDeleteMessages:       raw.DeleteMessages,
		CanManageVideoChats:     raw.ManageCall,
		CanRestrictMembers:      raw.BanUsers,
		CanPromoteMembers:       raw.AddAdmins,
		CanChangeInfo:           raw.ChangeInfo,
		CanInviteUsers:          raw.InviteUsers,
		CanPostStories:          raw.PostStories,
		CanEditStories:          raw.EditStories,
		CanDeleteStories:        raw.DeleteStories,
		CanPostMessages:         raw.PostMessages,
		CanEditMessages:         raw.EditMessages,
		CanPinMessages:          raw.PinMessages,
		CanManageTopics:         raw.ManageTopics,
		CanManageDirectMessages: raw.ManageDirectMessages,
		CanManageTags:           raw.ManageRanks,
	}
}

// ChatAdminRights is an alias for ChatAdministratorRights.
type ChatAdminRights = ChatAdministratorRights

// ParseChatAdminRights is an alias for ParseChatAdministratorRights.
var ParseChatAdminRights = ParseChatAdministratorRights

// ChatBannedRights represents the set of actions a restricted user is allowed or
// denied, along with an expiration date.
//
// Example:
//
//	banned := types.ParseChatBannedRights(rawBannedRights)
//	fmt.Printf("Can send messages: %v, until: %d\n", banned.CanSendMessages, banned.UntilDate)
type ChatBannedRights struct {
	CanSendMessages       bool
	CanSendMedia          bool
	CanSendPolls          bool
	CanSendOtherMessages  bool
	CanAddWebPagePreviews bool
	CanChangeInfo         bool
	CanInviteUsers        bool
	CanPinMessages        bool
	UntilDate             int32
}

// ParseChatBannedRights converts a TL ChatBannedRights into a ChatBannedRights.
// Returns nil if raw is nil.
//
// Example:
//
//	rights := types.ParseChatBannedRights(rawRights)
//	if rights != nil && !rights.CanSendMessages {
//	    fmt.Println("User is muted")
//	}
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
