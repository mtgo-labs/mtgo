package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

type PaidReactionPrivacy string

const (
	PaidReactionPrivacyEveryone     PaidReactionPrivacy = "everyone"
	PaidReactionPrivacyNobody       PaidReactionPrivacy = "nobody"
	PaidReactionPrivacyCloseFriends PaidReactionPrivacy = "close_friends"
	PaidReactionPrivacyContacts     PaidReactionPrivacy = "contacts"
)

func (p PaidReactionPrivacy) String() string { return string(p) }

func (p PaidReactionPrivacy) ToTL() tg.PaidReactionPrivacyClass {
	switch p {
	case PaidReactionPrivacyNobody:
		return &tg.PaidReactionPrivacyAnonymous{}
	default:
		return &tg.PaidReactionPrivacyDefault{}
	}
}
