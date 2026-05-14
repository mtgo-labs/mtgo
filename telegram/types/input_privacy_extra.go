package types

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
