package types

// ChatEventAction represents the type of action recorded in a chat admin event
// log. It is a string-based enum for easy comparison and logging.
type ChatEventAction string

const (
	// ChatEventActionTitleChanged indicates the chat title was modified.
	ChatEventActionTitleChanged ChatEventAction = "title_changed"
	// ChatEventActionPhotoChanged indicates the chat photo was changed.
	ChatEventActionPhotoChanged ChatEventAction = "photo_changed"
	// ChatEventActionDescriptionChanged indicates the chat description/about
	// text was modified.
	ChatEventActionDescriptionChanged ChatEventAction = "description_changed"
	// ChatEventActionUsernameChanged indicates the chat's public username was
	// changed.
	ChatEventActionUsernameChanged ChatEventAction = "username_changed"
	// ChatEventActionStickersetChanged indicates the group sticker pack was
	// changed.
	ChatEventActionStickersetChanged ChatEventAction = "stickerset_changed"
	// ChatEventActionLinkedChatChanged indicates the linked chat was changed.
	ChatEventActionLinkedChatChanged ChatEventAction = "linked_chat_changed"
	// ChatEventActionDefaultBannedRightsChanged indicates the default
	// permissions for non-admin members were updated.
	ChatEventActionDefaultBannedRightsChanged ChatEventAction = "default_banned_rights_changed"
	// ChatEventActionSignMessagesChanged indicates the message signature
	// setting was toggled.
	ChatEventActionSignMessagesChanged ChatEventAction = "sign_messages_changed"
	// ChatEventActionInvitesToggled indicates the invite link setting was
	// enabled or disabled.
	ChatEventActionInvitesToggled ChatEventAction = "invites_toggled"
)

// String returns the string representation of the ChatEventAction.
func (c ChatEventAction) String() string { return string(c) }
