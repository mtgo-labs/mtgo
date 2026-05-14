package types

// PrivacyRuleType represents a category of subjects that a privacy rule
// applies to, either as an allowlist or a denylist entry. Values prefixed with
// "allow" grant access; values prefixed with "disallow" restrict access.
type PrivacyRuleType string

const (
	// PrivacyRuleTypeAllowAll grants access to everyone.
	PrivacyRuleTypeAllowAll PrivacyRuleType = "allow_all"
	// PrivacyRuleTypeAllowBots grants access to all bots.
	PrivacyRuleTypeAllowBots PrivacyRuleType = "allow_bots"
	// PrivacyRuleTypeAllowChatParticipants grants access to members of specific
	// chats listed in the associated PrivacyRule.ChatIDs.
	PrivacyRuleTypeAllowChatParticipants PrivacyRuleType = "allow_chat_participants"
	// PrivacyRuleTypeAllowCloseFriends grants access only to the user's close
	// friends list.
	PrivacyRuleTypeAllowCloseFriends PrivacyRuleType = "allow_close_friends"
	// PrivacyRuleTypeAllowContacts grants access to the user's contacts.
	PrivacyRuleTypeAllowContacts PrivacyRuleType = "allow_contacts"
	// PrivacyRuleTypeAllowPremium grants access to all Telegram Premium
	// subscribers.
	PrivacyRuleTypeAllowPremium PrivacyRuleType = "allow_premium"
	// PrivacyRuleTypeAllowUsers grants access to specific users listed in the
	// associated PrivacyRule.UserIDs.
	PrivacyRuleTypeAllowUsers PrivacyRuleType = "allow_users"
	// PrivacyRuleTypeDisallowAll denies access to everyone.
	PrivacyRuleTypeDisallowAll PrivacyRuleType = "disallow_all"
	// PrivacyRuleTypeDisallowBots denies access to all bots.
	PrivacyRuleTypeDisallowBots PrivacyRuleType = "disallow_bots"
	// PrivacyRuleTypeDisallowChatParticipants denies access to members of
	// specific chats listed in the associated PrivacyRule.ChatIDs.
	PrivacyRuleTypeDisallowChatParticipants PrivacyRuleType = "disallow_chat_participants"
	// PrivacyRuleTypeDisallowContacts denies access to the user's contacts.
	PrivacyRuleTypeDisallowContacts PrivacyRuleType = "disallow_contacts"
	// PrivacyRuleTypeDisallowUsers denies access to specific users listed in
	// the associated PrivacyRule.UserIDs.
	PrivacyRuleTypeDisallowUsers PrivacyRuleType = "disallow_users"
)

// String returns the string representation of the PrivacyRuleType.
func (p PrivacyRuleType) String() string { return string(p) }
