package types

// ChatJoinType represents how a user joined a chat. Used in member join events
// to distinguish between self-joins, invitations, linked-chat joins, and
// migrations.
type ChatJoinType string

const (
	// ChatJoinTypeSelf indicates the user joined on their own (e.g. via search or
	// direct link).
	ChatJoinTypeSelf ChatJoinType = "self"
	// ChatJoinTypeBot indicates the user was added by a bot.
	ChatJoinTypeBot ChatJoinType = "bot"
	// ChatJoinTypeInvited indicates the user was invited by another member.
	ChatJoinTypeInvited ChatJoinType = "invited"
	// ChatJoinTypeLinked indicates the user joined via a linked chat (e.g. a
	// discussion group linked to a channel).
	ChatJoinTypeLinked ChatJoinType = "linked"
	// ChatJoinTypeMigrated indicates the user was migrated from another chat
	// (e.g. when a basic group is upgraded to a supergroup).
	ChatJoinTypeMigrated ChatJoinType = "migrated"
)

// String returns the string representation of the ChatJoinType.
func (c ChatJoinType) String() string { return string(c) }
