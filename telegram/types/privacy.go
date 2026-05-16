package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// PrivacyRule describes a single allow or deny rule that governs who can access
// a specific privacy-controlled feature (e.g., who can see the user's phone
// number). Rules are ordered; the first matching rule determines the outcome.
type PrivacyRule struct {
	Type  PrivacyRuleType
	Users []*User
	Chats []*Chat
}

// ParsePrivacyRules converts a slice of TL PrivacyRuleClass into typed PrivacyRule
// entries, resolving user and chat references from the provided maps.
// Returns nil if raw is empty.
//
// Example:
//
//	rules := types.ParsePrivacyRules(rawRules, users, peerMap)
//	for _, r := range rules {
//	    fmt.Printf("Rule: %s (users: %d, chats: %d)\n", r.Type, len(r.Users), len(r.Chats))
//	}
func ParsePrivacyRules(raw []tg.PrivacyRuleClass, users map[int64]tg.UserClass, chats *PeerMap) []PrivacyRule {
	if len(raw) == 0 {
		return nil
	}
	var rules []PrivacyRule
	for _, r := range raw {
		if rule := parsePrivacyRule(r, users, chats); rule != nil {
			rules = append(rules, *rule)
		}
	}
	return rules
}

func parsePrivacyRule(raw tg.PrivacyRuleClass, users map[int64]tg.UserClass, chats *PeerMap) *PrivacyRule {
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
		return &PrivacyRule{Type: PrivacyRuleTypeAllowUsers, Users: resolveUsers(r.Users, users)}
	case *tg.PrivacyValueAllowChatParticipants:
		return &PrivacyRule{Type: PrivacyRuleTypeAllowChatParticipants, Chats: resolveChats(r.Chats, chats)}
	case *tg.PrivacyValueDisallowAll:
		return &PrivacyRule{Type: PrivacyRuleTypeDisallowAll}
	case *tg.PrivacyValueDisallowContacts:
		return &PrivacyRule{Type: PrivacyRuleTypeDisallowContacts}
	case *tg.PrivacyValueDisallowUsers:
		return &PrivacyRule{Type: PrivacyRuleTypeDisallowUsers, Users: resolveUsers(r.Users, users)}
	case *tg.PrivacyValueDisallowChatParticipants:
		return &PrivacyRule{Type: PrivacyRuleTypeDisallowChatParticipants, Chats: resolveChats(r.Chats, chats)}
	}
	return nil
}

func resolveUsers(ids []int64, users map[int64]tg.UserClass) []*User {
	if len(ids) == 0 {
		return nil
	}
	out := make([]*User, 0, len(ids))
	for _, id := range ids {
		if u := getUser(users, id); u != nil {
			out = append(out, u)
		}
	}
	return out
}

func resolveChats(ids []int64, pm *PeerMap) []*Chat {
	if len(ids) == 0 || pm == nil {
		return nil
	}
	out := make([]*Chat, 0, len(ids))
	for _, id := range ids {
		if c, ok := pm.Chats[id]; ok {
			out = append(out, ParseChatFromChat(c))
		} else if c, ok := pm.Channels[id]; ok {
			out = append(out, ParseChatFromChat(c))
		}
	}
	return out
}

// GlobalPrivacySettings holds account-wide privacy preferences that are not tied
// to a single PrivacyKey. These settings control behavior for archiving, read
// receipts, and non-contact interactions.
type GlobalPrivacySettings struct {
	ArchiveAndMuteNewChats        bool
	KeepUnmutedChatsArchived      bool
	KeepChatsFromFoldersArchived  bool
	ShowReadDate                  bool
	AllowNewChatsFromUnknownUsers bool
	IncomingPaidMessageStarCount  int64
	ShowGiftButton                bool
	AcceptedGiftTypes             *AcceptedGiftTypes
}

// ParseGlobalPrivacySettings converts a TL GlobalPrivacySettings into a
// GlobalPrivacySettings. Returns nil if raw is nil.
func ParseGlobalPrivacySettings(raw *tg.GlobalPrivacySettings) *GlobalPrivacySettings {
	if raw == nil {
		return nil
	}
	return &GlobalPrivacySettings{
		ArchiveAndMuteNewChats:        raw.ArchiveAndMuteNewNoncontactPeers,
		KeepUnmutedChatsArchived:      raw.KeepArchivedUnmuted,
		KeepChatsFromFoldersArchived:  raw.KeepArchivedFolders,
		ShowReadDate:                  !raw.HideReadMarks,
		AllowNewChatsFromUnknownUsers: !raw.NewNoncontactPeersRequirePremium,
		IncomingPaidMessageStarCount:  raw.NoncontactPeersPaidStars,
		ShowGiftButton:                raw.DisplayGiftsButton,
		AcceptedGiftTypes:             ParseAcceptedGiftTypes(raw.DisallowedGifts),
	}
}
