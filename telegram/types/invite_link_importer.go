package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// InviteLinkImporter represents a user who joined a chat via an invite link,
// with the join date.
//
// Example:
//
//	importers := types.ParseInviteLinkImporters(rawResult)
//	for _, imp := range importers {
//	    fmt.Printf("Joined: %s at %s\n", imp.User.FirstName, imp.Date)
//	}
type InviteLinkImporter struct {
	Date time.Time
	User *User
}

// ParseInviteLinkImporters converts a TL MessagesChatInviteImporters into a
// slice of InviteLinkImporter. Returns nil if raw is nil or has no importers.
//
// Example:
//
//	importers := types.ParseInviteLinkImporters(raw)
//	for _, imp := range importers {
//	    fmt.Println(imp.User.FirstName, imp.Date)
//	}
func ParseInviteLinkImporters(raw *tg.MessagesChatInviteImporters) []*InviteLinkImporter {
	if raw == nil || len(raw.Importers) == 0 {
		return nil
	}
	users := make(map[int64]tg.UserClass, len(raw.Users))
	for _, u := range raw.Users {
		if v, ok := u.(*tg.User); ok && v != nil {
			users[v.ID] = u
		}
	}
	result := make([]*InviteLinkImporter, 0, len(raw.Importers))
	for _, imp := range raw.Importers {
		if imp == nil {
			continue
		}
		importer := &InviteLinkImporter{
			Date: time.Unix(int64(imp.Date), 0),
		}
		if u, ok := users[imp.UserID]; ok {
			importer.User = ParseUser(u)
		}
		result = append(result, importer)
	}
	return result
}
