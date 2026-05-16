package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// ChatSettings holds the peer-specific settings and suggested actions for a chat,
// such as whether to report spam, invite members, or manage a connected business bot.
type ChatSettings struct {
	CanReportSpam          bool
	CanAddContact          bool
	CanBlockContact        bool
	CanShareContact        bool
	CanReportGeo           bool
	CanInviteMembers       bool
	IsAutoArchived         bool
	IsBusinessBotPaused    bool
	IsBusinessBotCanReply  bool
	NeedContactsException  bool
	RequestChatBroadcast   bool
	GeoDistance            int32
	RequestChatTitle       string
	RequestChatDate        time.Time
	BusinessBot            *User
	BusinessBotManageURL   string
	ChargePaidMessageStars int64
	RegistrationDate       string
	PhoneNumberCountryCode string
	LastNameChangeDate     time.Time
	LastPhotoChangeDate    time.Time
}

// ChatEventFilter represents a filter for selecting which admin log events to retrieve
// from a channel or supergroup. Each boolean enables or disables a category of events.
type ChatEventFilter struct {
	NewRestrictions bool
	NewPrivileges   bool
	NewMembers      bool
	ChatInfo        bool
	ChatSettings    bool
	InviteLinks     bool
	DeletedMessages bool
	EditedMessages  bool
	PinnedMessages  bool
	LeavingMembers  bool
	VideoChats      bool
}

// ParseChatEventFilter converts a TL ChannelAdminLogEventsFilter into a ChatEventFilter.
// Returns nil if raw is nil.
//
// Example:
//
//	filter := types.ParseChatEventFilter(rawFilter)
//	if filter != nil && filter.DeletedMessages {
//	    fmt.Println("Filter includes deleted messages")
//	}
func ParseChatEventFilter(raw *tg.ChannelAdminLogEventsFilter) *ChatEventFilter {
	if raw == nil {
		return nil
	}
	return &ChatEventFilter{
		NewRestrictions: raw.Ban || raw.Kick || raw.Unban || raw.Unkick,
		NewPrivileges:   raw.Promote || raw.Demote,
		NewMembers:      raw.Join,
		ChatInfo:        raw.Info,
		ChatSettings:    raw.Settings,
		InviteLinks:     raw.Invites,
		DeletedMessages: raw.Delete,
		EditedMessages:  raw.Edit,
		PinnedMessages:  raw.Pinned,
		LeavingMembers:  raw.Leave,
		VideoChats:      raw.GroupCall,
	}
}

// ParseChatSettings converts a TL PeerSettings into a ChatSettings, optionally
// resolving the connected business bot from the users map. Returns nil if raw is nil.
//
// Example:
//
//	settings := types.ParseChatSettings(rawSettings, users)
//	if settings.CanReportSpam {
//	    fmt.Println("User can report spam")
//	}
func ParseChatSettings(raw *tg.PeerSettings, users ...map[int64]tg.UserClass) *ChatSettings {
	if raw == nil {
		return nil
	}
	s := &ChatSettings{
		CanReportSpam:          raw.ReportSpam,
		CanAddContact:          raw.AddContact,
		CanBlockContact:        raw.BlockContact,
		CanShareContact:        raw.ShareContact,
		NeedContactsException:  raw.NeedContactsException,
		CanReportGeo:           raw.ReportGeo,
		IsAutoArchived:         raw.Autoarchived,
		CanInviteMembers:       raw.InviteMembers,
		RequestChatBroadcast:   raw.RequestChatBroadcast,
		IsBusinessBotPaused:    raw.BusinessBotPaused,
		IsBusinessBotCanReply:  raw.BusinessBotCanReply,
		BusinessBotManageURL:   raw.BusinessBotManageURL,
		ChargePaidMessageStars: raw.ChargePaidMessageStars,
		RegistrationDate:       raw.RegistrationMonth,
		PhoneNumberCountryCode: raw.PhoneCountry,
	}
	if raw.GeoDistance != 0 {
		s.GeoDistance = raw.GeoDistance
	}
	if raw.RequestChatTitle != "" {
		s.RequestChatTitle = raw.RequestChatTitle
	}
	if raw.RequestChatDate != 0 {
		s.RequestChatDate = time.Unix(int64(raw.RequestChatDate), 0)
	}
	if raw.BusinessBotID != 0 && len(users) > 0 {
		s.BusinessBot = getUser(users[0], raw.BusinessBotID)
	}
	if raw.NameChangeDate != 0 {
		s.LastNameChangeDate = time.Unix(int64(raw.NameChangeDate), 0)
	}
	if raw.PhotoChangeDate != 0 {
		s.LastPhotoChangeDate = time.Unix(int64(raw.PhotoChangeDate), 0)
	}
	return s
}
