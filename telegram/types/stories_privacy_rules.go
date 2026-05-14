package types

// StoriesPrivacyRules represents the visibility audience for a Telegram
// story. Controls who can view a story when it is posted.
type StoriesPrivacyRules string

const (
	// StoriesPrivacyRulesCloseFriends restricts visibility to close friends only.
	StoriesPrivacyRulesCloseFriends StoriesPrivacyRules = "close_friends"
	// StoriesPrivacyRulesContacts allows all contacts to view the story.
	StoriesPrivacyRulesContacts StoriesPrivacyRules = "contacts"
	// StoriesPrivacyRulesSelected restricts visibility to a manually selected list of users.
	StoriesPrivacyRulesSelected StoriesPrivacyRules = "selected"
	// StoriesPrivacyRulesNobody hides the story from everyone (used for draft/archived stories).
	StoriesPrivacyRulesNobody StoriesPrivacyRules = "nobody"
)

// String returns the string representation of the StoriesPrivacyRules.
func (s StoriesPrivacyRules) String() string { return string(s) }
