package types

// ChatEventAction enumerates the kinds of changes recorded in a chat's admin log
// (e.g., title changed, message pinned, member promoted).
//
// Example:
//
//	switch event.Action {
//	case types.ChatEventActionTitleChanged:
//	    fmt.Printf("Title: %s -> %s\n", event.OldTitle, event.NewTitle)
//	case types.ChatEventActionMessageDeleted:
//	    fmt.Printf("Deleted message: %s\n", event.DeletedMessage.Text)
//	}
type ChatEventAction string

const (
	ChatEventActionTitleChanged                   ChatEventAction = "title_changed"
	ChatEventActionPhotoChanged                   ChatEventAction = "photo_changed"
	ChatEventActionDescriptionChanged             ChatEventAction = "description_changed"
	ChatEventActionUsernameChanged                ChatEventAction = "username_changed"
	ChatEventActionHistoryTTLChanged              ChatEventAction = "history_ttl_changed"
	ChatEventActionLinkedChatChanged              ChatEventAction = "linked_chat_changed"
	ChatEventActionChatPermissionsChanged         ChatEventAction = "chat_permissions_changed"
	ChatEventActionSignaturesEnabled              ChatEventAction = "signatures_enabled"
	ChatEventActionInvitesEnabled                 ChatEventAction = "invites_enabled"
	ChatEventActionHistoryHidden                  ChatEventAction = "history_hidden"
	ChatEventActionSlowModeChanged                ChatEventAction = "slow_mode_changed"
	ChatEventActionStickersetChanged              ChatEventAction = "stickerset_changed"
	ChatEventActionMessageDeleted                 ChatEventAction = "message_deleted"
	ChatEventActionMessageEdited                  ChatEventAction = "message_edited"
	ChatEventActionMessagePinned                  ChatEventAction = "message_pinned"
	ChatEventActionMessageUnpinned                ChatEventAction = "message_unpinned"
	ChatEventActionMemberInvited                  ChatEventAction = "member_invited"
	ChatEventActionMemberJoined                   ChatEventAction = "member_joined"
	ChatEventActionMemberLeft                     ChatEventAction = "member_left"
	ChatEventActionAdministratorPrivilegesChanged ChatEventAction = "administrator_privileges_changed"
	ChatEventActionMemberPermissionsChanged       ChatEventAction = "member_permissions_changed"
	ChatEventActionPollStopped                    ChatEventAction = "poll_stopped"
	ChatEventActionInviteLinkEdited               ChatEventAction = "invite_link_edited"
	ChatEventActionInviteLinkRevoked              ChatEventAction = "invite_link_revoked"
	ChatEventActionInviteLinkDeleted              ChatEventAction = "invite_link_deleted"
	ChatEventActionCreatedForumTopic              ChatEventAction = "created_forum_topic"
	ChatEventActionEditedForumTopic               ChatEventAction = "edited_forum_topic"
	ChatEventActionDeletedForumTopic              ChatEventAction = "deleted_forum_topic"
)

func (c ChatEventAction) String() string { return string(c) }
