package types

// ChatMemberStatus represents the membership status of a user within a chat.
// Used to classify a ChatMember's role without inspecting the underlying TL
// participant type.
type ChatMemberStatus string

const (
	// ChatMemberStatusOwner indicates the user is the chat owner with full
	// control.
	ChatMemberStatusOwner ChatMemberStatus = "owner"
	// ChatMemberStatusAdministrator indicates the user is an administrator with
	// some or all admin privileges.
	ChatMemberStatusAdministrator ChatMemberStatus = "administrator"
	// ChatMemberStatusMember indicates a regular member with no special rights.
	ChatMemberStatusMember ChatMemberStatus = "member"
	// ChatMemberStatusRestricted indicates a restricted member with limited
	// permissions.
	ChatMemberStatusRestricted ChatMemberStatus = "restricted"
	// ChatMemberStatusLeft indicates the user has left the chat voluntarily.
	ChatMemberStatusLeft ChatMemberStatus = "left"
	// ChatMemberStatusBanned indicates the user has been banned from the chat.
	ChatMemberStatusBanned ChatMemberStatus = "banned"
)

// String returns the string representation of the ChatMemberStatus.
func (c ChatMemberStatus) String() string { return string(c) }
