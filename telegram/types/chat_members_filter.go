package types

// ChatMembersFilter represents a filter criterion when querying chat members.
// Passed to member listing methods to scope the results.
type ChatMembersFilter string

const (
	// ChatMembersFilterSearch searches members by name/query string.
	ChatMembersFilterSearch ChatMembersFilter = "search"
	// ChatMembersFilterBanned returns only banned members.
	ChatMembersFilterBanned ChatMembersFilter = "banned"
	// ChatMembersFilterRestricted returns only restricted members.
	ChatMembersFilterRestricted ChatMembersFilter = "restricted"
	// ChatMembersFilterBots returns only bot members.
	ChatMembersFilterBots ChatMembersFilter = "bots"
	// ChatMembersFilterRecent returns recently active members.
	ChatMembersFilterRecent ChatMembersFilter = "recent"
	// ChatMembersFilterAdministrators returns only administrators and the owner.
	ChatMembersFilterAdministrators ChatMembersFilter = "administrators"
)

// String returns the string representation of the ChatMembersFilter.
func (c ChatMembersFilter) String() string { return string(c) }
