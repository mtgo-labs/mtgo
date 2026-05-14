package types

// ChatMemberUpdated represents a change in a user's chat membership, such as
// joining, leaving, being promoted, or being banned. Delivered via update
// handlers when member events occur.
type ChatMemberUpdated struct {
	// Chat is the chat where the membership change occurred.
	Chat *Chat
	// From is the user who performed the action that triggered the change (e.g.
	// the admin who promoted or banned).
	From *User
	// OldChatMember is the member's previous state before the change.
	OldChatMember *ChatMember
	// NewChatMember is the member's current state after the change.
	NewChatMember *ChatMember
	// Date is the Unix timestamp when the change occurred.
	Date int32
	// InviteLink is the invite link associated with the membership change, if
	// the user joined via a link.
	InviteLink *ChatInviteLink
	// ViaChatFolder is true when the membership change was triggered via a chat
	// folder (e.g. adding a chat to a folder).
	ViaChatFolder bool
}
