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
