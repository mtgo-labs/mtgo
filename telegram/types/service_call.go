package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// VideoChatScheduled represents a service message indicating that a group
// video/voice chat has been scheduled for a specific date.
//
// Example:
//
//	svc := msg.Service()
//	if svc.Scheduled != nil {
//	    fmt.Printf("call scheduled for %v\n", svc.Scheduled.StartDate)
//	}
type VideoChatScheduled struct {
	StartDate time.Time
}

// VideoChatStarted represents a service message indicating that a group
// video/voice chat has started.
//
// Example:
//
//	svc := msg.Service()
//	if svc.Started != nil {
//	    fmt.Println("group call is now active")
//	}
type VideoChatStarted struct{}

// VideoChatEnded represents a service message indicating that a group
// video/voice chat has ended, including its total duration in seconds.
//
// Example:
//
//	svc := msg.Service()
//	if svc.Ended != nil {
//	    fmt.Printf("call lasted %d seconds\n", svc.Ended.Duration)
//	}
type VideoChatEnded struct {
	Duration int32
}

// VideoChatMembersInvited represents a service message indicating that
// specific members were invited to an active group video/voice chat.
//
// Example:
//
//	svc := msg.Service()
//	if svc.MembersInvited != nil {
//	    for _, u := range svc.MembersInvited.Users {
//	        fmt.Printf("invited user %d\n", u.ID)
//	    }
//	}
type VideoChatMembersInvited struct {
	Users []*User
}

// PhoneCallStarted represents a phone call that has been initiated.
type PhoneCallStarted struct {
	ID      int64
	IsVideo bool
}

// PhoneCallEnded represents a service message indicating that a phone call
// has ended, including its duration and the reason for ending.
//
// Example:
//
//	svc := msg.Service()
//	if svc.PhoneCallEnded != nil {
//	    fmt.Printf("call %d ended after %ds: %s\n",
//	        svc.PhoneCallEnded.ID, svc.PhoneCallEnded.Duration, svc.PhoneCallEnded.Reason)
//	}
type PhoneCallEnded struct {
	ID       int64
	IsVideo  bool
	Reason   PhoneCallDiscardReason
	Duration int32
}

// ProximityAlertTriggered represents a proximity alert between two users.
type ProximityAlertTriggered struct {
	Traveler *User
	Watcher  *User
	Distance int32
}

// WriteAccessAllowed indicates that the user was allowed to send messages
// to the bot via a web app.
type WriteAccessAllowed struct {
	FromRequest        bool
	WebAppName         string
	FromAttachmentMenu bool
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
	return &VideoChatScheduled{
		StartDate: time.Unix(int64(raw.ScheduleDate), 0),
	}
}

// ParseVideoChatStarted converts a TL MessageActionGroupCall into a VideoChatStarted.
// Returns nil if the input is nil.
func ParseVideoChatStarted(raw *tg.MessageActionGroupCall) *VideoChatStarted {
	if raw == nil {
		return nil
	}
	return &VideoChatStarted{}
}

// ParseVideoChatEnded converts a TL MessageActionGroupCall into a VideoChatEnded,
// extracting the call duration if available. Returns nil if the input is nil.
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

// ParseVideoChatMembersInvited converts a TL MessageActionInviteToGroupCall
// into a VideoChatMembersInvited, mapping invited user IDs to User objects.
// Returns nil if the input is nil.
func ParseVideoChatMembersInvited(raw *tg.MessageActionInviteToGroupCall) *VideoChatMembersInvited {
	if raw == nil {
		return nil
	}
	out := &VideoChatMembersInvited{}
	for _, id := range raw.Users {
		out.Users = append(out.Users, &User{ID: id})
	}
	return out
}

// ParsePhoneCallStarted converts a TL MessageActionPhoneCall into a PhoneCallStarted.
func ParsePhoneCallStarted(raw *tg.MessageActionPhoneCall) *PhoneCallStarted {
	if raw == nil {
		return nil
	}
	return &PhoneCallStarted{
		ID:      raw.CallID,
		IsVideo: raw.Video,
	}
}

// ParsePhoneCallEnded converts a TL MessageActionPhoneCall into a PhoneCallEnded.
func ParsePhoneCallEnded(raw *tg.MessageActionPhoneCall) *PhoneCallEnded {
	if raw == nil {
		return nil
	}
	out := &PhoneCallEnded{
		ID:      raw.CallID,
		IsVideo: raw.Video,
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
		Traveler: &User{ID: GetPeerID(raw.FromID)},
		Watcher:  &User{ID: GetPeerID(raw.ToID)},
		Distance: raw.Distance,
	}
}

// ParseWriteAccessAllowed converts a TL MessageActionBotAllowed into a WriteAccessAllowed.
func ParseWriteAccessAllowed(raw *tg.MessageActionBotAllowed) *WriteAccessAllowed {
	if raw == nil {
		return nil
	}
	out := &WriteAccessAllowed{
		FromRequest:        raw.FromRequest,
		FromAttachmentMenu: raw.AttachMenu,
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

// ManagedBotCreated represents a service message indicating that a managed
// bot was created in the chat.
//
// Example:
//
//	svc := msg.Service()
//	if svc.ManagedBotCreated != nil {
//	    fmt.Printf("bot created: %d\n", svc.ManagedBotCreated.Bot.ID)
//	}
type ManagedBotCreated struct {
	Bot *User
}

// ContactRegistered represents a service message indicating that a contact
// has registered on Telegram.
type ContactRegistered struct{}

// ScreenshotTaken represents a service message indicating that a user took
// a screenshot of the chat content.
type ScreenshotTaken struct{}

// ChatOwnerChanged represents a service message indicating that the
// ownership of a chat has been transferred to a new owner.
//
// Example:
//
//	svc := msg.Service()
//	if svc.OwnerChanged != nil {
//	    fmt.Printf("new owner: %d\n", svc.OwnerChanged.NewOwnerID)
//	}
type ChatOwnerChanged struct {
	NewOwnerID int64
}

// ChatOwnerLeft represents a service message indicating that the chat owner
// has left the chat.
type ChatOwnerLeft struct{}

// ChatHasProtectedContentToggled represents a service message indicating
// that content protection has been toggled on or off in the chat.
//
// Example:
//
//	svc := msg.Service()
//	if svc.ProtectedContentToggled != nil {
//	    fmt.Printf("protected: %v\n", svc.ProtectedContentToggled.Enabled)
//	}
type ChatHasProtectedContentToggled struct {
	Enabled                  bool
	RequestMessageID         int32
	IsOldHasProtectedContent bool
	IsNewHasProtectedContent bool
}

// ChatHasProtectedContentDisableRequested represents a service message
// indicating that a request was made to disable content protection in the chat.
type ChatHasProtectedContentDisableRequested struct{}

// PaidMessagesRefunded represents a service message indicating that paid
// messages were refunded, including the message count and star count.
//
// Example:
//
//	svc := msg.Service()
//	if svc.PaidMessagesRefunded != nil {
//	    fmt.Printf("refunded %d messages (%d stars)\n",
//	        svc.PaidMessagesRefunded.MessageCount, svc.PaidMessagesRefunded.StarCount)
//	}
type PaidMessagesRefunded struct {
	MessageCount int32
	StarCount    int64
	Amount       int64
}

// PaidMessageReactor represents a user who reacted to a paid message,
// including the reaction count and whether the reactor is anonymous.
//
// Example:
//
//	reactor := &PaidMessageReactor{PeerID: 123, Count: 2, Top: true}
//	fmt.Printf("user %d reacted %d times\n", reactor.PeerID, reactor.Count)
type PaidMessageReactor struct {
	PeerID    int64
	Count     int32
	Top       bool
	My        bool
	Anonymous bool
}

// ParsePaidMessageReactor converts a TL MessageReactor into a PaidMessageReactor,
// extracting the peer ID, count, and flags. Returns nil if the input is nil.
func ParsePaidMessageReactor(raw *tg.MessageReactor) *PaidMessageReactor {
	if raw == nil {
		return nil
	}
	r := &PaidMessageReactor{
		Count:     raw.Count,
		Top:       raw.Top,
		My:        raw.My,
		Anonymous: raw.Anonymous,
	}
	if raw.PeerID != nil {
		switch p := raw.PeerID.(type) {
		case *tg.PeerUser:
			r.PeerID = p.UserID
		case *tg.PeerChat:
			r.PeerID = -p.ChatID
		case *tg.PeerChannel:
			r.PeerID = channelChatID(p.ChannelID)
		}
	}
	return r
}

// PaidMessagesPriceChanged represents a service message indicating that the
// price for paid messages in a chat has changed.
//
// Example:
//
//	svc := msg.Service()
//	if svc.PaidMessagesPriceChanged != nil {
//	    fmt.Printf("new price: %d stars\n", svc.PaidMessagesPriceChanged.Stars)
//	}
type PaidMessagesPriceChanged struct {
	Stars                int64
	PaidMessageStarCount int64
}

// DirectMessagePriceChanged represents a service message indicating that the
// price for direct messages in a chat has changed.
//
// Example:
//
//	svc := msg.Service()
//	if svc.DirectMessagePriceChanged != nil {
//	    fmt.Printf("new DM price: %d stars\n", svc.DirectMessagePriceChanged.Stars)
//	}
type DirectMessagePriceChanged struct {
	Stars                int64
	PaidMessageStarCount int64
}

// DirectMessagesTopic represents a direct messages topic with metadata
// including read state, unread counts, and the top message.
//
// Example:
//
//	svc := msg.Service()
//	if svc.DirectMessagesTopic != nil {
//	    fmt.Printf("topic %d: %d unread\n",
//	        svc.DirectMessagesTopic.ID, svc.DirectMessagesTopic.UnreadCount)
//	}
type DirectMessagesTopic struct {
	ID                      int32
	PeerID                  int64
	User                    *User
	TopMessage              *Message
	LastReadInboxMessageID  int32
	LastReadOutboxMessageID int32
	ReadInboxMaxID          int32
	ReadOutboxMaxID         int32
	UnreadCount             int32
	UnreadReactionsCount    int32
}

// PollOptionAdded represents a service message indicating that a new option
// was added to an existing poll.
//
// Example:
//
//	svc := msg.Service()
//	if svc.PollOptionAdded != nil {
//	    fmt.Printf("new option: %s\n", svc.PollOptionAdded.Text.Text)
//	}
type PollOptionAdded struct {
	PollMessage        *Message
	OptionPersistentID string
	Text               *FormattedText
}

// PollOptionDeleted represents a service message indicating that an option
// was removed from an existing poll.
//
// Example:
//
//	svc := msg.Service()
//	if svc.PollOptionDeleted != nil {
//	    fmt.Printf("removed option: %s\n", svc.PollOptionDeleted.Text.Text)
//	}
type PollOptionDeleted struct {
	PollMessage        *Message
	OptionPersistentID string
	Text               *FormattedText
}

// FactCheck represents a fact-check annotation attached to a message,
// including the check text, country, and entity-formatted content.
//
// Example:
//
//	svc := msg.Service()
//	if svc.FactCheck != nil {
//	    fmt.Printf("fact-check: %s\n", svc.FactCheck.Text)
//	}
type FactCheck struct {
	NeedCheck bool
	Country   string
	Text      string
	Entities  []*MessageEntity
	Hash      int64
}

// ParseFactCheck converts a TL FactCheck into a FactCheck, extracting the
// text, entities, country, and hash. Returns nil if the input is nil.
func ParseFactCheck(raw *tg.FactCheck) *FactCheck {
	if raw == nil {
		return nil
	}
	fc := &FactCheck{
		NeedCheck: raw.NeedCheck,
		Country:   raw.Country,
		Hash:      raw.Hash,
	}
	if raw.Text != nil {
		fc.Text = raw.Text.Text
		for _, e := range raw.Text.Entities {
			if me := ParseMessageEntity(e); me != nil {
				fc.Entities = append(fc.Entities, me)
			}
		}
	}
	return fc
}

// WebAppData represents data received from a Web App launched via a
// keyboard button, including the raw data and the button text.
//
// Example:
//
//	svc := msg.Service()
//	if svc.WebAppData != nil {
//	    fmt.Printf("web app data: %s\n", svc.WebAppData.Data)
//	}
type WebAppData struct {
	Data       string
	ButtonText string
}

// RefundedPayment represents a service message indicating that a payment
// was refunded, including currency, amount, and charge identifiers.
//
// Example:
//
//	svc := msg.Service()
//	if svc.RefundedPayment != nil {
//	    fmt.Printf("refunded %d %s\n", svc.RefundedPayment.TotalAmount, svc.RefundedPayment.Currency)
//	}
type RefundedPayment struct {
	Currency         string
	TotalAmount      int64
	Payload          []byte
	ChargeID         string
	ProviderChargeID string
}

// PaidMediaPreview represents a preview of paid media content, including
// dimensions, duration, and an optional thumbnail.
//
// Example:
//
//	svc := msg.Service()
//	if svc.PaidMediaPreview != nil {
//	    fmt.Printf("preview: %dx%d, %ds\n",
//	        svc.PaidMediaPreview.Width, svc.PaidMediaPreview.Height, svc.PaidMediaPreview.Duration)
//	}
type PaidMediaPreview struct {
	Width     int32
	Height    int32
	Duration  int32
	Thumbnail []byte
}

// SuggestedPostApprovalFailed represents a service message indicating that
// the approval of a suggested post has failed.
type SuggestedPostApprovalFailed struct {
	SuggestedPostMessageID int
	SuggestedPostMessage   *Message
	Price                  *SuggestedPostPrice
}

// SuggestedPostApproved represents a service message indicating that a
// suggested post has been approved with an optional send date.
type SuggestedPostApproved struct {
	SuggestedPostMessageID int
	SuggestedPostMessage   *Message
	Price                  *SuggestedPostPrice
	SendDate               time.Time
}

// SuggestedPostDeclined represents a service message indicating that a
// suggested post has been declined with an optional comment.
type SuggestedPostDeclined struct {
	SuggestedPostMessageID int
	SuggestedPostMessage   *Message
	Comment                string
}

// SuggestedPostRefunded represents a service message indicating that a
// suggested post payment has been refunded with a reason.
type SuggestedPostRefunded struct {
	SuggestedPostMessageID int
	SuggestedPostMessage   *Message
	Reason                 SuggestedPostRefundReason
}

// SuggestedPostPaid represents a service message indicating that a
// suggested post has been paid for, including the payment amount.
type SuggestedPostPaid struct {
	SuggestedPostMessageID int
	SuggestedPostMessage   *Message
	Amount                 int64
	StarAmount             *StarAmount
}

// SuggestedPostInfo represents metadata about a suggested post including
// its price, scheduled send date, and current state.
type SuggestedPostInfo struct {
	Price    *SuggestedPostPrice
	SendDate time.Time
	State    SuggestedPostState
}

// SuggestedPostPriceStar represents a suggested post price denominated in
// Telegram Stars.
type SuggestedPostPriceStar struct {
	Stars int64
}

// SuggestedPostPriceTon represents a suggested post price denominated in
// TON cryptocurrency.
type SuggestedPostPriceTon struct {
	Amount int64
}

// SavedCredentials represents previously saved payment credentials that can
// be reused for future payments.
type SavedCredentials struct {
	ID    string
	Title string
}

// PaymentOption represents an available payment method option with a
// redirect URL and display title.
type PaymentOption struct {
	URL   string
	Title string
}

// PaymentResult represents the outcome of a payment attempt, including
// success status and an optional verification URL.
type PaymentResult struct {
	Success         bool
	VerificationURL string
}

// InputCredentialsSaved represents saved payment credentials referenced by
// their stored ID and a temporary hash for authentication.
type InputCredentialsSaved struct {
	ID            string
	TemporaryHash []byte
}

// InputCredentialsNew represents new payment credentials provided directly
// by the user, with an option to save them for future use.
type InputCredentialsNew struct {
	Data string
	Save bool
}

// InputCredentialsApplePay represents payment credentials provided through
// Apple Pay.
type InputCredentialsApplePay struct {
	PaymentData string
}

// InputCredentialsGooglePay represents payment credentials provided through
// Google Pay.
type InputCredentialsGooglePay struct {
	PaymentData string
}

// InputInvoiceExtended represents an extended invoice input that identifies
// the invoice by type, slug, or message/chat reference.
//
// Example:
//
//	inv := &InputInvoiceExtended{
//	    Type:   InputInvoiceExtSlug,
//	    Slug:   "my-invoice",
//	}
type InputInvoiceExtended struct {
	Type   InputInvoiceTypeExtended
	Slug   string
	MsgID  int32
	ChatID int64
}

// InputInvoiceTypeExtended is the type of invoice source used with
// InputInvoiceExtended.
type InputInvoiceTypeExtended string

const (
	InputInvoiceExtSlug     InputInvoiceTypeExtended = "slug"
	InputInvoiceExtMessage  InputInvoiceTypeExtended = "message"
	InputInvoiceExtStars    InputInvoiceTypeExtended = "stars"
	InputInvoiceExtGiftCode InputInvoiceTypeExtended = "gift_code"
	InputInvoiceExtGift     InputInvoiceTypeExtended = "gift"
)

// InputChecklist represents a checklist input with a title and tasks,
// along with permissions for other users to append or complete tasks.
//
// Example:
//
//	cl := &InputChecklist{
//	    Title: "Shopping list",
//	    Tasks: []InputChecklistTask{
//	        {ID: 1, Text: "Milk"},
//	        {ID: 2, Text: "Eggs"},
//	    },
//	    OthersCanAppend: true,
//	}
type InputChecklist struct {
	Title             string
	Tasks             []InputChecklistTask
	OthersCanAppend   bool
	OthersCanComplete bool
}

// InputChecklistTask represents a single task within an InputChecklist,
// with optional formatting via parse mode and entities.
type InputChecklistTask struct {
	ID        int32
	Text      string
	ParseMode string
	Entities  []*MessageEntity
}
