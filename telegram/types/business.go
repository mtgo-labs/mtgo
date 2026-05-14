package types

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// BusinessIntro represents the introductory text and sticker shown on a
// Telegram business account's profile page.
type BusinessIntro struct {
	// Title is the heading text of the business introduction.
	Title string
	// Description is the body text of the business introduction.
	Description string
	// Sticker is the optional sticker displayed alongside the introduction, if set.
	Sticker *DocumentMedia
}

// BusinessRecipients defines which chats a business feature (such as intro or
// away message) applies to, based on chat type and explicit user inclusion/exclusion.
type BusinessRecipients struct {
	// ExistingChats applies the feature to all existing chats when true.
	ExistingChats bool
	// NewChats applies the feature to all new chats when true.
	NewChats bool
	// Contacts applies the feature to all chats with contacts when true.
	Contacts bool
	// NonContacts applies the feature to all chats with non-contacts when true.
	NonContacts bool
	// ExcludeSelected inverts the Users list to exclude rather than include when true.
	ExcludeSelected bool
	// Users is the list of specific user IDs the feature applies to or is excluded from.
	Users []int64
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
	// Reply allows the bot to send replies in connected chats.
	Reply bool
	// ReadMessages allows the bot to read messages in connected chats.
	ReadMessages bool
	// DeleteSentMessages allows the bot to delete messages it has sent.
	DeleteSentMessages bool
	// DeleteReceivedMessages allows the bot to delete messages sent by the user.
	DeleteReceivedMessages bool
	// DeleteMessages allows the bot to delete any message in connected chats.
	DeleteMessages bool
	// EditName allows the bot to change the business account's display name.
	EditName bool
	// EditBio allows the bot to change the business account's bio.
	EditBio bool
	// EditProfilePhoto allows the bot to change the business account's profile photo.
	EditProfilePhoto bool
	// EditUsername allows the bot to change the business account's username.
	EditUsername bool
	// ViewGifts allows the bot to view gifts received by the business account.
	ViewGifts bool
	// SellGifts allows the bot to sell gifts owned by the business account.
	SellGifts bool
	// ChangeGiftSettings allows the bot to modify gift-related settings.
	ChangeGiftSettings bool
	// TransferAndUpgradeGifts allows the bot to transfer and upgrade owned gifts.
	TransferAndUpgradeGifts bool
	// TransferStars allows the bot to transfer Telegram Stars from the account.
	TransferStars bool
	// ManageStories allows the bot to post and manage stories on behalf of the account.
	ManageStories bool
}

// ParseBusinessIntro converts an MTProto BusinessIntro into a BusinessIntro.
// Returns nil if raw is nil.
func ParseBusinessIntro(raw *tg.BusinessIntro) *BusinessIntro {
	if raw == nil {
		return nil
	}
	ni := &BusinessIntro{
		Title:       raw.Title,
		Description: raw.Description,
	}
	if doc, ok := raw.Sticker.(*tg.Document); ok {
		ni.Sticker = parseDocumentFromTL(doc)
	}
	return ni
}

func parseDocumentFromTL(doc *tg.Document) *DocumentMedia {
	if doc == nil {
		return nil
	}
	m := &DocumentMedia{
		FileSize:     doc.Size,
		MimeType:     doc.MimeType,
		RawDocument:  doc,
	}
	m.FileID = fmt.Sprintf("%d_%d", doc.ID, doc.AccessHash)
	for _, attr := range doc.Attributes {
		if a, ok := attr.(*tg.DocumentAttributeFilename); ok {
			m.FileName = a.FileName
		}
	}
	return m
}

// ParseBusinessRecipients converts an MTProto BusinessRecipients into a BusinessRecipients.
// Returns nil if raw is nil.
func ParseBusinessRecipients(raw *tg.BusinessRecipients) *BusinessRecipients {
	if raw == nil {
		return nil
	}
	return &BusinessRecipients{
		ExistingChats:   raw.ExistingChats,
		NewChats:        raw.NewChats,
		Contacts:        raw.Contacts,
		NonContacts:     raw.NonContacts,
		ExcludeSelected: raw.ExcludeSelected,
		Users:           raw.Users,
	}
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
		Reply:                   raw.Reply,
		ReadMessages:            raw.ReadMessages,
		DeleteSentMessages:      raw.DeleteSentMessages,
		DeleteReceivedMessages:  raw.DeleteReceivedMessages,
		EditName:                raw.EditName,
		EditBio:                 raw.EditBio,
		EditProfilePhoto:        raw.EditProfilePhoto,
		EditUsername:            raw.EditUsername,
		ViewGifts:               raw.ViewGifts,
		SellGifts:               raw.SellGifts,
		ChangeGiftSettings:      raw.ChangeGiftSettings,
		TransferAndUpgradeGifts: raw.TransferAndUpgradeGifts,
		TransferStars:           raw.TransferStars,
		ManageStories:           raw.ManageStories,
	}
}
