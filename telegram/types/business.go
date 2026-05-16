package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// BusinessIntro represents the introductory text and sticker shown on a
// Telegram business account's profile page.
type BusinessIntro struct {
	Title   string
	Text    string
	Sticker *Sticker
}

// BusinessRecipients defines which chats a business feature (such as intro or
// away message) applies to, based on chat type and explicit user inclusion/exclusion.
type BusinessRecipients struct {
	ExistingChats   bool
	NewChats        bool
	Contacts        bool
	NonContacts     bool
	ExcludeSelected bool
	Users           []*User
}

// BusinessWorkingHours represents the weekly operating schedule for a Telegram
// business account, including timezone and open/close intervals.
type BusinessWorkingHours struct {
	// OpenNow indicates whether the business is currently open.
	OpenNow bool
	// TimeZone is the IANA timezone identifier for the schedule (e.g. "Europe/Berlin").
	TimeZone string
	// WeeklyOpen is the list of open intervals for each day of the week.
	WeeklyOpen []BusinessWeeklyOpen
}

// BusinessWeeklyOpen represents a single open interval within a business's weekly schedule.
// Times are expressed as minutes from the start of the week (Monday 00:00 = 0).
type BusinessWeeklyOpen struct {
	// StartMinute is the minute offset from Monday 00:00 when the interval starts.
	StartMinute int32
	// EndMinute is the minute offset from Monday 00:00 when the interval ends.
	EndMinute int32
}

// BusinessBotRights enumerates the granular permissions a business account has
// granted to a connected bot. Each field corresponds to an independent capability.
type BusinessBotRights struct {
	CanReply                   bool
	CanReadMessages            bool
	CanDeleteSentMessages      bool
	CanDeleteAllMessages       bool
	CanEditName                bool
	CanEditBio                 bool
	CanEditProfilePhoto        bool
	CanEditUsername            bool
	CanViewGiftsAndStars       bool
	CanSellGifts               bool
	CanChangeGiftSettings      bool
	CanTransferAndUpgradeGifts bool
	CanTransferStars           bool
	CanManageStories           bool
}

// ParseBusinessIntro converts an MTProto BusinessIntro into a BusinessIntro.
// Returns nil if raw is nil.
func ParseBusinessIntro(raw *tg.BusinessIntro) *BusinessIntro {
	if raw == nil {
		return nil
	}
	ni := &BusinessIntro{
		Title: raw.Title,
		Text:  raw.Description,
	}
	if doc, ok := raw.Sticker.(*tg.Document); ok {
		ni.Sticker = ParseSticker(doc)
	}
	return ni
}

// ParseBusinessRecipients converts an MTProto BusinessRecipients into a BusinessRecipients.
// Returns nil if raw is nil.
func ParseBusinessRecipients(raw *tg.BusinessRecipients, users map[int64]tg.UserClass) *BusinessRecipients {
	if raw == nil {
		return nil
	}
	r := &BusinessRecipients{
		ExistingChats:   raw.ExistingChats,
		NewChats:        raw.NewChats,
		Contacts:        raw.Contacts,
		NonContacts:     raw.NonContacts,
		ExcludeSelected: raw.ExcludeSelected,
	}
	for _, id := range raw.Users {
		if u := getUser(users, id); u != nil {
			r.Users = append(r.Users, u)
		}
	}
	return r
}

// ParseBusinessWorkingHours converts an MTProto BusinessWorkHours into a BusinessWorkingHours.
// Returns nil if raw is nil.
func ParseBusinessWorkingHours(raw *tg.BusinessWorkHours) *BusinessWorkingHours {
	if raw == nil {
		return nil
	}
	h := &BusinessWorkingHours{
		OpenNow:  raw.OpenNow,
		TimeZone: raw.TimezoneID,
	}
	for _, wo := range raw.WeeklyOpen {
		if wo != nil {
			h.WeeklyOpen = append(h.WeeklyOpen, BusinessWeeklyOpen{
				StartMinute: wo.StartMinute,
				EndMinute:   wo.EndMinute,
			})
		}
	}
	return h
}

// ParseBusinessBotRights converts an MTProto BusinessBotRights into a BusinessBotRights.
// Returns nil if raw is nil.
func ParseBusinessBotRights(raw *tg.BusinessBotRights) *BusinessBotRights {
	if raw == nil {
		return nil
	}
	return &BusinessBotRights{
		CanReply:                   raw.Reply,
		CanReadMessages:            raw.ReadMessages,
		CanDeleteSentMessages:      raw.DeleteSentMessages,
		CanDeleteAllMessages:       raw.DeleteReceivedMessages,
		CanEditName:                raw.EditName,
		CanEditBio:                 raw.EditBio,
		CanEditProfilePhoto:        raw.EditProfilePhoto,
		CanEditUsername:            raw.EditUsername,
		CanViewGiftsAndStars:       raw.ViewGifts,
		CanSellGifts:               raw.SellGifts,
		CanChangeGiftSettings:      raw.ChangeGiftSettings,
		CanTransferAndUpgradeGifts: raw.TransferAndUpgradeGifts,
		CanTransferStars:           raw.TransferStars,
		CanManageStories:           raw.ManageStories,
	}
}

// BusinessMessage represents a message sent through a business connection.
type BusinessMessage struct {
	ShortcutID     int32
	IsGreeting     bool
	IsAway         bool
	NoActivityDays int32
	OfflineOnly    bool
	Recipients     []*User
	Schedule       BusinessSchedule
	StartDate      time.Time
	EndDate        time.Time
}

// ParseBusinessMessage converts a TL BusinessGreetingMessage or BusinessAwayMessage
// into a BusinessMessage, resolving recipients from the user map. Returns nil if raw is nil.
//
// Example:
//
//	msg := types.ParseBusinessMessage(rawGreeting, users)
//	if msg != nil && msg.IsGreeting {
//	    fmt.Printf("Greeting shortcut: %d, no-activity days: %d\n", msg.ShortcutID, msg.NoActivityDays)
//	}
func ParseBusinessMessage(raw any, users map[int64]tg.UserClass) *BusinessMessage {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.BusinessGreetingMessage:
		recipients := ParseBusinessRecipients(v.Recipients, users)
		message := &BusinessMessage{
			ShortcutID:     v.ShortcutID,
			IsGreeting:     true,
			NoActivityDays: v.NoActivityDays,
		}
		if recipients != nil {
			message.Recipients = recipients.Users
		}
		return message
	case *tg.BusinessAwayMessage:
		recipients := ParseBusinessRecipients(v.Recipients, users)
		message := &BusinessMessage{
			ShortcutID:  v.ShortcutID,
			IsAway:      true,
			OfflineOnly: v.OfflineOnly,
		}
		if recipients != nil {
			message.Recipients = recipients.Users
		}
		applyBusinessSchedule(message, v.Schedule)
		return message
	default:
		return nil
	}
}

func applyBusinessSchedule(message *BusinessMessage, raw tg.BusinessAwayMessageScheduleClass) {
	switch v := raw.(type) {
	case *tg.BusinessAwayMessageScheduleAlways:
		message.Schedule = BusinessScheduleAlwaysOpen
	case *tg.BusinessAwayMessageScheduleOutsideWorkHours:
		message.Schedule = BusinessScheduleOutside
	case *tg.BusinessAwayMessageScheduleCustom:
		message.Schedule = BusinessScheduleCustom
		message.StartDate = time.Unix(int64(v.StartDate), 0)
		message.EndDate = time.Unix(int64(v.EndDate), 0)
	}
}

// MessageContent is the interface for typed message content payloads.
type MessageContent interface {
	ContentType() string
}
