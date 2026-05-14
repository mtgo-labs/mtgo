package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// ChatSettings holds the peer-specific settings and suggested actions for a chat,
// such as whether to report spam, invite members, or manage a connected business bot.
type ChatSettings struct {
	// ReportSpam suggests the user report the chat for spam.
	ReportSpam bool
	// AddContact suggests the user add the peer as a contact.
	AddContact bool
	// BlockContact suggests the user block the peer.
	BlockContact bool
	// ShareContact suggests the user share their phone number with the peer.
	ShareContact bool
	// NeedContactsException indicates the peer needs a contacts exception to message the user.
	NeedContactsException bool
	// ReportGeo suggests the user report a geolocation-related issue.
	ReportGeo bool
	// AutoArchived indicates the chat was automatically archived by the system.
	AutoArchived bool
	// InviteMembers suggests the user invite members to the chat.
	InviteMembers bool
	// RequestChatBroadcast indicates a request to broadcast in the chat.
	RequestChatBroadcast bool
	// BusinessBotPaused indicates the connected business bot is paused.
	BusinessBotPaused bool
	// BusinessBotCanReply indicates the connected business bot can reply to messages.
	BusinessBotCanReply bool
	// GeoDistance is the distance in meters from the user to the peer's shared location.
	GeoDistance int32
	// RequestChatTitle is the title of the chat being requested.
	RequestChatTitle string
	// RequestChatDate is the Unix timestamp of the chat request.
	RequestChatDate int32
	// BusinessBotID is the user ID of the business bot connected to this chat.
	BusinessBotID int64
	// BusinessBotManageURL is the URL for managing the connected business bot's settings.
	BusinessBotManageURL string
}

// ChatEventFilter represents a filter for selecting which admin log events to retrieve
// from a channel or supergroup. Each boolean enables or disables a category of events.
type ChatEventFilter struct {
	// Join includes member join events.
	Join bool
	// Leave includes member leave events.
	Leave bool
	// Invite includes invite link events.
	Invite bool
	// Ban includes member ban events.
	Ban bool
	// Unban includes member unban events.
	Unban bool
	// Kick includes member kick events.
	Kick bool
	// Unkick includes member unkick events.
	Unkick bool
	// Promote includes admin promotion events.
	Promote bool
	// Demote includes admin demotion events.
	Demote bool
	// Join includes chat info change events (title, description, etc.).
	Info bool
	// Settings includes chat settings change events.
	Settings bool
	// Pinned includes message pin/unpin events.
	Pinned bool
	// Edit includes message edit events.
	Edit bool
	// Delete includes message delete events.
	Delete bool
	// GroupCall includes group call events.
	GroupCall bool
	// Invites includes invite link creation and revocation events.
	Invites bool
	// Send includes new message events.
	Send bool
	// Forums includes forum topic management events.
	Forums bool
	// SubExtend includes subscription extension events.
	SubExtend bool
	// EditRank includes admin rank change events.
	EditRank bool
}

// ParseChatSettings converts an MTProto PeerSettingsTL into a ChatSettings.
// Returns nil if raw is nil.
func ParseChatSettings(raw *tg.PeerSettings) *ChatSettings {
	if raw == nil {
		return nil
	}
	s := &ChatSettings{
		ReportSpam:            raw.ReportSpam,
		AddContact:            raw.AddContact,
		BlockContact:          raw.BlockContact,
		ShareContact:          raw.ShareContact,
		NeedContactsException: raw.NeedContactsException,
		ReportGeo:             raw.ReportGeo,
		AutoArchived:          raw.Autoarchived,
		InviteMembers:         raw.InviteMembers,
		RequestChatBroadcast:  raw.RequestChatBroadcast,
		BusinessBotPaused:     raw.BusinessBotPaused,
		BusinessBotCanReply:   raw.BusinessBotCanReply,
	}
	if raw.GeoDistance != 0 {
		s.GeoDistance = raw.GeoDistance
	}
	if raw.RequestChatTitle != "" {
		s.RequestChatTitle = raw.RequestChatTitle
	}
	if raw.RequestChatDate != 0 {
		s.RequestChatDate = raw.RequestChatDate
	}
	if raw.BusinessBotID != 0 {
		s.BusinessBotID = raw.BusinessBotID
	}
	if raw.BusinessBotManageURL != "" {
		s.BusinessBotManageURL = raw.BusinessBotManageURL
	}
	return s
}

// ParseChatEventFilter converts an MTProto ChannelAdminLogEventsFilter into a ChatEventFilter.
// Returns nil if raw is nil.
func ParseChatEventFilter(raw *tg.ChannelAdminLogEventsFilter) *ChatEventFilter {
	if raw == nil {
		return nil
	}
	return &ChatEventFilter{
		Join:      raw.Join,
		Leave:     raw.Leave,
		Invite:    raw.Invite,
		Ban:       raw.Ban,
		Unban:     raw.Unban,
		Kick:      raw.Kick,
		Unkick:    raw.Unkick,
		Promote:   raw.Promote,
		Demote:    raw.Demote,
		Info:      raw.Info,
		Settings:  raw.Settings,
		Pinned:    raw.Pinned,
		Edit:      raw.Edit,
		Delete:    raw.Delete,
		GroupCall: raw.GroupCall,
		Invites:   raw.Invites,
		Send:      raw.Send,
		Forums:    raw.Forums,
		SubExtend: raw.SubExtend,
		EditRank:  raw.EditRank,
	}
}
