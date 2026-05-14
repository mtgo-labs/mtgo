package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

type ManagedBotCreated struct {
	BotID int64
}

type ContactRegistered struct{}

type ScreenshotTaken struct{}

type ChatOwnerChanged struct {
	NewOwnerID int64
}

type ChatOwnerLeft struct{}

type ChatHasProtectedContentToggled struct {
	Enabled bool
}

type ChatHasProtectedContentDisableRequested struct{}

type PaidMessagesRefunded struct {
	Amount int64
}

type PaidMessageReactor struct {
	PeerID    int64
	Count     int32
	Top       bool
	My        bool
	Anonymous bool
}

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
			r.PeerID = p.ChatID
		case *tg.PeerChannel:
			r.PeerID = p.ChannelID
		}
	}
	return r
}

type PaidMessagesPriceChanged struct {
	Stars int64
}

type DirectMessagePriceChanged struct {
	Stars int64
}

type DirectMessagesTopic struct {
	PeerID               int64
	TopMessage           int32
	ReadInboxMaxID       int32
	ReadOutboxMaxID      int32
	UnreadCount          int32
	UnreadReactionsCount int32
}

type PollOptionAdded struct {
	Text string
}

type PollOptionDeleted struct {
	Text string
}

type RefundedPayment struct {
	Currency         string
	TotalAmount      int64
	Payload          []byte
	ChargeID         string
	ProviderChargeID string
}

type PaidMediaPreview struct {
	Width    int32
	Height   int32
	Duration int32
}

type SuggestedPostApprovalFailed struct{}

type SuggestedPostApproved struct{}

type SuggestedPostDeclined struct{}

type SuggestedPostRefunded struct{}

type SuggestedPostPaid struct{}

type SuggestedPostInfo struct {
	Price        *SuggestedPostPrice
	ScheduleDate int32
	Accepted     bool
	Rejected     bool
}

type SuggestedPostPriceStar struct {
	Stars int64
}

type SuggestedPostPriceTon struct {
	Amount int64
}

type SavedCredentials struct {
	ID    string
	Title string
}

type PaymentOption struct {
	URL   string
	Title string
}

type PaymentResult struct {
	Success         bool
	VerificationURL string
}

type InputCredentialsSaved struct {
	ID            string
	TemporaryHash []byte
}

type InputCredentialsNew struct {
	Data string
	Save bool
}

type InputCredentialsApplePay struct {
	PaymentData string
}

type InputCredentialsGooglePay struct {
	PaymentData string
}

type InputInvoiceExtended struct {
	Type   InputInvoiceTypeExtended
	Slug   string
	MsgID  int32
	ChatID int64
}

type InputInvoiceTypeExtended string

const (
	InputInvoiceExtSlug      InputInvoiceTypeExtended = "slug"
	InputInvoiceExtMessage   InputInvoiceTypeExtended = "message"
	InputInvoiceExtStars     InputInvoiceTypeExtended = "stars"
	InputInvoiceExtGiftCode  InputInvoiceTypeExtended = "gift_code"
	InputInvoiceExtGift      InputInvoiceTypeExtended = "gift"
)

type InputChecklist struct {
	Title             string
	Tasks             []InputChecklistTask
	OthersCanAppend   bool
	OthersCanComplete bool
}

type InputChecklistTask struct {
	ID    int32
	Title string
}
