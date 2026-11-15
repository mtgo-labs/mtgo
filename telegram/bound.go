package telegram

import (
	"strings"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

func parseCommonChats(result tg.ChatsClass) []*types.Chat {
	var raw []tg.ChatClass
	switch v := result.(type) {
	case *tg.MessagesChats:
		raw = v.Chats
	case *tg.MessagesChatsSlice:
		raw = v.Chats
	}
	chats := make([]*types.Chat, 0, len(raw))
	for _, ch := range raw {
		if parsed := types.ParseChatFromChat(ch); parsed != nil {
			chats = append(chats, parsed)
		}
	}
	return chats
}

// storyPrivacyRules converts a high-level EditPrivacy option into the
// []InputPrivacyRuleClass expected by stories.editStory. Without this the edit
// is a silent no-op: SetFlags only emits the privacy_rules flag when the field
// is non-nil, so omitting it makes the server ignore the privacy change.
func storyPrivacyRules(o *params.EditPrivacy) []tg.InputPrivacyRuleClass {
	if o == nil {
		return []tg.InputPrivacyRuleClass{&tg.InputPrivacyValueAllowAll{}}
	}

	var rules []tg.InputPrivacyRuleClass
	switch strings.ToLower(strings.TrimSpace(o.Privacy)) {
	case "", "everybody", "all", "public":
		rules = append(rules, &tg.InputPrivacyValueAllowAll{})
	case "contacts":
		rules = append(rules, &tg.InputPrivacyValueAllowContacts{})
	case "close_friends", "closefriends":
		rules = append(rules, &tg.InputPrivacyValueAllowCloseFriends{})
	case "nobody", "none", "private":
		rules = append(rules, &tg.InputPrivacyValueDisallowAll{})
	default:
		rules = append(rules, &tg.InputPrivacyValueAllowAll{})
	}

	if len(o.AllowedUsers) > 0 {
		users := make([]tg.InputUserClass, 0, len(o.AllowedUsers))
		for _, id := range o.AllowedUsers {
			users = append(users, &tg.InputUser{UserID: id})
		}
		rules = append(rules, &tg.InputPrivacyValueAllowUsers{Users: users})
	}
	if len(o.DisallowedUsers) > 0 {
		users := make([]tg.InputUserClass, 0, len(o.DisallowedUsers))
		for _, id := range o.DisallowedUsers {
			users = append(users, &tg.InputUser{UserID: id})
		}
		rules = append(rules, &tg.InputPrivacyValueDisallowUsers{Users: users})
	}
	return rules
}
