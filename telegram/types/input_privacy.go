package types

// InputPrivacyRuleType identifies the kind of privacy rule for controlling
// who can perform a particular action (e.g., see status, send invites).
type InputPrivacyRuleType string

const (
	// InputPrivacyAllowContacts allows all contacts to perform the action.
	InputPrivacyAllowContacts InputPrivacyRuleType = "allow_contacts"
	// InputPrivacyAllowAll allows all users to perform the action.
	InputPrivacyAllowAll InputPrivacyRuleType = "allow_all"
	// InputPrivacyAllowUsers allows specific users listed in UserIDs to perform the action.
	InputPrivacyAllowUsers InputPrivacyRuleType = "allow_users"
	// InputPrivacyAllowCloseFriends allows only close friends to perform the action.
	InputPrivacyAllowCloseFriends InputPrivacyRuleType = "allow_close_friends"
	// InputPrivacyDisallowContacts disallows contacts from performing the action.
	InputPrivacyDisallowContacts InputPrivacyRuleType = "disallow_contacts"
	// InputPrivacyDisallowAll disallows all users from performing the action.
	InputPrivacyDisallowAll InputPrivacyRuleType = "disallow_all"
	// InputPrivacyDisallowUsers disallows specific users listed in UserIDs from performing the action.
	InputPrivacyDisallowUsers InputPrivacyRuleType = "disallow_users"
	// InputPrivacyAllowBots allows all bots to perform the action.
	InputPrivacyAllowBots InputPrivacyRuleType = "allow_bots"
	// InputPrivacyAllowPremium allows only Premium subscribers to perform the action.
	InputPrivacyAllowPremium InputPrivacyRuleType = "allow_premium"
)

// InputPrivacyRule represents a single rule for modifying privacy settings.
// Combine multiple rules to build a complete privacy policy.
type InputPrivacyRule struct {
	// Type identifies the rule kind (allow contacts, disallow users, etc.).
	Type InputPrivacyRuleType
	// UserIDs is the list of specific user IDs for allow_users and disallow_users rules.
	UserIDs []int64
}

// InputPrivacyRuleAllowAll allows all users for the privacy action.
type InputPrivacyRuleAllowAll struct{}

// InputPrivacyRuleAllowBots allows all bots for the privacy action.
type InputPrivacyRuleAllowBots struct{}

// InputPrivacyRuleAllowChats allows specific chats for the privacy action.
type InputPrivacyRuleAllowChats struct {
	ChatIDs []int64
}

// InputPrivacyRuleAllowCloseFriends allows only close friends for the privacy action.
type InputPrivacyRuleAllowCloseFriends struct{}

// InputPrivacyRuleAllowContacts allows all contacts for the privacy action.
type InputPrivacyRuleAllowContacts struct{}

// InputPrivacyRuleAllowPremium allows only Premium users for the privacy action.
type InputPrivacyRuleAllowPremium struct{}

// InputPrivacyRuleAllowUsers allows specific users for the privacy action.
type InputPrivacyRuleAllowUsers struct {
	UserIDs []int64
}

// InputPrivacyRuleDisallowAll disallows all users from the privacy action.
type InputPrivacyRuleDisallowAll struct{}

// InputPrivacyRuleDisallowBots disallows all bots from the privacy action.
type InputPrivacyRuleDisallowBots struct{}

// InputPrivacyRuleDisallowChats disallows specific chats from the privacy action.
type InputPrivacyRuleDisallowChats struct {
	ChatIDs []int64
}

// InputPrivacyRuleDisallowContacts disallows all contacts from the privacy action.
type InputPrivacyRuleDisallowContacts struct{}

// InputPrivacyRuleDisallowUsers disallows specific users from the privacy action.
type InputPrivacyRuleDisallowUsers struct {
	UserIDs []int64
}
