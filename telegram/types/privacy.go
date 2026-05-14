package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// PrivacyRule describes a single allow or deny rule that governs who can access
// a specific privacy-controlled feature (e.g., who can see the user's phone
// number). Rules are ordered; the first matching rule determines the outcome.
type PrivacyRule struct {
	// Type identifies the category of subjects this rule applies to, such as
	// "allow_all", "allow_contacts", "disallow_users", etc.
	Type PrivacyRuleType
	// UserIDs lists the specific user IDs the rule applies to when Type is
	// PrivacyRuleTypeAllowUsers or PrivacyRuleTypeDisallowUsers. Empty for
	// rules that target broad categories.
	UserIDs []int64
	// ChatIDs lists the specific chat IDs the rule applies to when Type is
	// PrivacyRuleTypeAllowChatParticipants or PrivacyRuleTypeDisallowChatParticipants.
	// Empty for rules that target broad categories.
	ChatIDs []int64
}

// GlobalPrivacySettings holds account-wide privacy preferences that are not tied
// to a single PrivacyKey. These settings control behavior for archiving, read
// receipts, and non-contact interactions.
type GlobalPrivacySettings struct {
	// ArchiveAndMuteNonContacts indicates whether new messages from non-contacts
	// are automatically archived and muted.
	ArchiveAndMuteNonContacts bool
	// KeepArchivedUnmuted indicates whether chats that were archived remain in
	// the archive even when the user unmutes them.
	KeepArchivedUnmuted bool
	// KeepArchivedFolders indicates whether chats that were archived remain in
	// the archive even when moved to a different folder.
	KeepArchivedFolders bool
	// HideReadMarks indicates whether the user's read receipts are hidden
	// globally, preventing others from seeing when messages were read.
	HideReadMarks bool
	// NonContactsRequirePremium indicates whether only Premium users can
	// initiate direct messages with non-contacts.
	NonContactsRequirePremium bool
	// DisplayGiftsButton indicates whether the gifts button is visible on the
	// user's profile page.
	DisplayGiftsButton bool
}

// ParsePrivacyRules converts a slice of TL PrivacyRuleClass values into a slice
// of PrivacyRule. Returns nil if the input is empty.
func ParsePrivacyRules(raw []tg.PrivacyRuleClass) []PrivacyRule {
	if len(raw) == 0 {
		return nil
	}
	var rules []PrivacyRule
	for _, r := range raw {
		if rule := parsePrivacyRule(r); rule != nil {
			rules = append(rules, *rule)
		}
	}
	return rules
}

func parsePrivacyRule(raw tg.PrivacyRuleClass) *PrivacyRule {
	if raw == nil {
		return nil
	}
	switch r := raw.(type) {
	case *tg.PrivacyValueAllowAll:
		return &PrivacyRule{Type: PrivacyRuleTypeAllowAll}
	case *tg.PrivacyValueAllowContacts:
		return &PrivacyRule{Type: PrivacyRuleTypeAllowContacts}
	case *tg.PrivacyValueAllowCloseFriends:
		return &PrivacyRule{Type: PrivacyRuleTypeAllowCloseFriends}
	case *tg.PrivacyValueAllowPremium:
		return &PrivacyRule{Type: PrivacyRuleTypeAllowPremium}
	case *tg.PrivacyValueAllowUsers:
		return &PrivacyRule{Type: PrivacyRuleTypeAllowUsers, UserIDs: r.Users}
	case *tg.PrivacyValueAllowChatParticipants:
		return &PrivacyRule{Type: PrivacyRuleTypeAllowChatParticipants, ChatIDs: r.Chats}
	case *tg.PrivacyValueDisallowAll:
		return &PrivacyRule{Type: PrivacyRuleTypeDisallowAll}
	case *tg.PrivacyValueDisallowContacts:
		return &PrivacyRule{Type: PrivacyRuleTypeDisallowContacts}
	case *tg.PrivacyValueDisallowUsers:
		return &PrivacyRule{Type: PrivacyRuleTypeDisallowUsers, UserIDs: r.Users}
	case *tg.PrivacyValueDisallowChatParticipants:
		return &PrivacyRule{Type: PrivacyRuleTypeDisallowChatParticipants, ChatIDs: r.Chats}
	}
	return nil
}

// ParseGlobalPrivacySettings converts a TL GlobalPrivacySettings into a
// GlobalPrivacySettings. Returns nil if raw is nil.
func ParseGlobalPrivacySettings(raw *tg.GlobalPrivacySettings) *GlobalPrivacySettings {
	if raw == nil {
		return nil
	}
	return &GlobalPrivacySettings{
		ArchiveAndMuteNonContacts: raw.ArchiveAndMuteNewNoncontactPeers,
		KeepArchivedUnmuted:       raw.KeepArchivedUnmuted,
		KeepArchivedFolders:       raw.KeepArchivedFolders,
		HideReadMarks:             raw.HideReadMarks,
		NonContactsRequirePremium: raw.NewNoncontactPeersRequirePremium,
		DisplayGiftsButton:        raw.DisplayGiftsButton,
	}
}
