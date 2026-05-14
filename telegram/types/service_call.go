package types

import "github.com/mtgo-labs/mtgo/tg"

// VideoChatScheduled represents a scheduled video chat (group call).
type VideoChatScheduled struct {
	// GroupCallID is the unique identifier of the group call.
	GroupCallID int64
	// StartDate is the Unix timestamp when the group call is scheduled to start.
	StartDate int32
}

// VideoChatStarted represents a video chat (group call) that has started.
type VideoChatStarted struct {
	// GroupCallID is the unique identifier of the group call.
	GroupCallID int64
}

// VideoChatEnded represents a video chat (group call) that has ended.
type VideoChatEnded struct {
	// Duration is the duration of the group call in seconds.
	Duration int32
}

// VideoChatMembersInvited represents users invited to a video chat.
type VideoChatMembersInvited struct {
	// UserIDs is the list of user IDs invited to the group call.
	UserIDs []int64
}

// PhoneCallStarted represents a phone call that has been initiated.
type PhoneCallStarted struct {
	// CallID is the unique identifier of the phone call.
	CallID int64
	// Video indicates whether this is a video call.
	Video bool
}

// PhoneCallEnded represents a phone call that has ended.
type PhoneCallEnded struct {
	// CallID is the unique identifier of the phone call.
	CallID int64
	// Duration is the duration of the call in seconds.
	Duration int32
	// Reason is the reason the call was ended.
	Reason PhoneCallDiscardReason
	// Video indicates whether this was a video call.
	Video bool
}

// ProximityAlertTriggered represents a proximity alert between two users.
type ProximityAlertTriggered struct {
	// TravelerID is the user ID of the traveler who triggered the alert.
	TravelerID int64
	// WatcherID is the user ID of the watcher who set the proximity alert.
	WatcherID int64
	// Distance is the distance between the two users in meters.
	Distance int32
}

// WriteAccessAllowed indicates that the user was allowed to send messages
// to the bot via a web app.
type WriteAccessAllowed struct {
	// FromRequest indicates whether access was granted from an explicit request.
	FromRequest bool
	// WebAppName is the name of the web app that requested write access.
	WebAppName string
}

// BoostInfo represents a boost applied to a chat.
type BoostInfo struct {
	// ID is the unique identifier of the boost.
	ID string
	// UserID is the ID of the user who applied the boost.
	UserID int64
	// GiveawayMessageID is the message ID of the associated giveaway, if any.
	GiveawayMessageID int32
	// Date is the Unix timestamp when the boost was applied.
	Date int32
	// Expires is the Unix timestamp when the boost expires.
	Expires int32
	// Multiplier is the boost multiplier.
	Multiplier int32
	// Stars is the number of Telegram Stars used for the boost.
	Stars int64
}

// ParseVideoChatScheduled converts a TL MessageActionGroupCallScheduled into a VideoChatScheduled.
func ParseVideoChatScheduled(raw *tg.MessageActionGroupCallScheduled) *VideoChatScheduled {
	if raw == nil {
		return nil
	}
	out := &VideoChatScheduled{
		StartDate: raw.ScheduleDate,
	}
	if call, ok := raw.Call.(*tg.InputGroupCall); ok {
		out.GroupCallID = call.ID
	}
	return out
}

// ParseVideoChatStarted converts a TL MessageActionGroupCall into a VideoChatStarted.
// The Duration field is only set when the call has ended (see ParseVideoChatEnded).
func ParseVideoChatStarted(raw *tg.MessageActionGroupCall) *VideoChatStarted {
	if raw == nil {
		return nil
	}
	out := &VideoChatStarted{}
	if call, ok := raw.Call.(*tg.InputGroupCall); ok {
		out.GroupCallID = call.ID
	}
	return out
}

// ParseVideoChatEnded converts a TL MessageActionGroupCall with a Duration into a VideoChatEnded.
func ParseVideoChatEnded(raw *tg.MessageActionGroupCall) *VideoChatEnded {
	if raw == nil {
		return nil
	}
	out := &VideoChatEnded{}
	if raw.Duration != 0 {
		out.Duration = raw.Duration
	}
	return out
}

// ParseVideoChatMembersInvited converts a TL MessageActionInviteToGroupCall into a VideoChatMembersInvited.
func ParseVideoChatMembersInvited(raw *tg.MessageActionInviteToGroupCall) *VideoChatMembersInvited {
	if raw == nil {
		return nil
	}
	return &VideoChatMembersInvited{
		UserIDs: raw.Users,
	}
}

// ParsePhoneCallStarted converts a TL MessageActionPhoneCall into a PhoneCallStarted.
func ParsePhoneCallStarted(raw *tg.MessageActionPhoneCall) *PhoneCallStarted {
	if raw == nil {
		return nil
	}
	return &PhoneCallStarted{
		CallID: raw.CallID,
		Video:  raw.Video,
	}
}

// ParsePhoneCallEnded converts a TL MessageActionPhoneCall into a PhoneCallEnded.
func ParsePhoneCallEnded(raw *tg.MessageActionPhoneCall) *PhoneCallEnded {
	if raw == nil {
		return nil
	}
	out := &PhoneCallEnded{
		CallID: raw.CallID,
		Video:  raw.Video,
	}
	if raw.Duration != 0 {
		out.Duration = raw.Duration
	}
	if raw.Reason != nil {
		out.Reason = parsePhoneCallDiscardReason(raw.Reason)
	}
	return out
}

// ParseProximityAlertTriggered converts a TL MessageActionGeoProximityReached into a ProximityAlertTriggered.
func ParseProximityAlertTriggered(raw *tg.MessageActionGeoProximityReached) *ProximityAlertTriggered {
	if raw == nil {
		return nil
	}
	return &ProximityAlertTriggered{
		TravelerID: GetPeerID(raw.FromID),
		WatcherID:  GetPeerID(raw.ToID),
		Distance:   raw.Distance,
	}
}

// ParseWriteAccessAllowed converts a TL MessageActionBotAllowed into a WriteAccessAllowed.
func ParseWriteAccessAllowed(raw *tg.MessageActionBotAllowed) *WriteAccessAllowed {
	if raw == nil {
		return nil
	}
	out := &WriteAccessAllowed{
		FromRequest: raw.FromRequest,
	}
	if raw.Domain != "" {
		out.WebAppName = raw.Domain
	}
	return out
}

// ParseChatBackground converts a TL MessageActionSetChatWallPaper into a ChatBackground.
func ParseChatBackground(raw *tg.MessageActionSetChatWallPaper) *ChatBackground {
	if raw == nil {
		return nil
	}
	out := &ChatBackground{}
	switch wp := raw.Wallpaper.(type) {
	case *tg.WallPaper:
		out.ID = wp.ID
		if doc, ok := wp.Document.(*tg.Document); ok {
			out.WallpaperDocID = doc.ID
		}
	case *tg.WallPaperNoFile:
		out.ID = wp.ID
	}
	return out
}

// ParseBoostInfo converts a TL Boost into a BoostInfo.
func ParseBoostInfo(raw *tg.Boost) *BoostInfo {
	if raw == nil {
		return nil
	}
	out := &BoostInfo{
		ID:      raw.ID,
		Date:    raw.Date,
		Expires: raw.Expires,
	}
	if raw.UserID != 0 {
		out.UserID = raw.UserID
	}
	if raw.GiveawayMsgID != 0 {
		out.GiveawayMessageID = raw.GiveawayMsgID
	}
	if raw.Multiplier != 0 {
		out.Multiplier = raw.Multiplier
	}
	if raw.Stars != 0 {
		out.Stars = raw.Stars
	}
	return out
}

func parsePhoneCallDiscardReason(raw tg.PhoneCallDiscardReasonClass) PhoneCallDiscardReason {
	if raw == nil {
		return ""
	}
	switch raw.(type) {
	case *tg.PhoneCallDiscardReasonMissed:
		return PhoneCallDiscardReasonMissed
	case *tg.PhoneCallDiscardReasonDisconnect:
		return PhoneCallDiscardReasonDisconnected
	case *tg.PhoneCallDiscardReasonHangup:
		return PhoneCallDiscardReasonHungUp
	case *tg.PhoneCallDiscardReasonBusy:
		return PhoneCallDiscardReasonDeclined
	case *tg.PhoneCallDiscardReasonMigrateConferenceCall:
		return PhoneCallDiscardReasonUpgradeToConferenceCall
	}
	return ""
}
