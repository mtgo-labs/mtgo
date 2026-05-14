package types

// PaidReactionPrivacy controls who can see that the current user sent a paid
// reaction. Paid reactions cost Telegram Stars, and the user may want to
// control their visibility.
type PaidReactionPrivacy string

const (
	// PaidReactionPrivacyEveryone makes the paid reaction visible to all users.
	PaidReactionPrivacyEveryone PaidReactionPrivacy = "everyone"
	// PaidReactionPrivacyNobody hides the paid reaction sender identity from
	// everyone except the message author.
	PaidReactionPrivacyNobody PaidReactionPrivacy = "nobody"
	// PaidReactionPrivacyCloseFriends makes the paid reaction visible only to
	// the sender's close friends.
	PaidReactionPrivacyCloseFriends PaidReactionPrivacy = "close_friends"
	// PaidReactionPrivacyContacts makes the paid reaction visible only to
	// the sender's contacts.
	PaidReactionPrivacyContacts PaidReactionPrivacy = "contacts"
)

// String returns the string representation of the PaidReactionPrivacy.
func (p PaidReactionPrivacy) String() string { return string(p) }
