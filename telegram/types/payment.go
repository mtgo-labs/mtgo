package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// LabeledPrice represents a portion of an invoice price with a label.
//
// Example:
//
//	prices := []*types.LabeledPrice{
//	    {Label: "Product", Amount: 999},
//	    {Label: "Shipping", Amount: 200},
//	}
//	_ = prices
type LabeledPrice struct {
	// Label is a human-readable description of the price portion.
	Label string
	// Amount is the price in the smallest units of the currency.
	Amount int64
}

// Invoice represents a payment invoice attached to a message.
type Invoice struct {
	Currency                   string
	IsTest                     bool
	Title                      string
	Description                string
	TotalAmount                int64
	StartParameter             string
	Prices                     []*LabeledPrice
	IsNameRequested            bool
	IsPhoneRequested           bool
	IsEmailRequested           bool
	IsShippingAddressRequested bool
	IsFlexible                 bool
	IsPhoneToProvider          bool
	IsEmailToProvider          bool
	IsRecurring                bool
	MaxTipAmount               int64
	SuggestedTipAmounts        []int64
	TermsURL                   string
	SubscriptionPeriod         int32
}

// SuccessfulPayment represents a confirmation of a successful payment.
type SuccessfulPayment struct {
	// Currency is the three-letter ISO 4217 currency code.
	Currency string
	// TotalAmount is the total amount charged in smallest currency units.
	TotalAmount int64
	// Payload is the bot-specified payment payload.
	Payload []byte
	// ChargeID is the payment charge ID from the payment provider.
	ChargeID string
	// ShippingOptionID is the selected shipping option ID, if applicable.
	ShippingOptionID string
	// ProviderPaymentChargeID is the charge ID from the payment provider.
	ProviderPaymentChargeID string
	// OrderInfo is the payment information provided by the user.
	OrderInfo *OrderInfo
	// InvoiceSlug is the name of the invoice.
	InvoiceSlug string
	// SubscriptionExpirationDate is the expiration date of the subscription for recurring payments.
	SubscriptionExpirationDate time.Time
	// IsRecurring is true if the payment is a recurring payment for a subscription.
	IsRecurring bool
	// IsFirstRecurring is true if the payment is the first payment for a subscription.
	IsFirstRecurring bool
}

// LinkPreviewOptions controls how link previews are generated for a message.
type LinkPreviewOptions struct {
	IsDisabled       bool
	URL              string
	PreferSmallMedia bool
	PreferLargeMedia bool
	ShowAboveText    bool
}

// ReplyParameters describes how a message replies to another message.
type ReplyParameters struct {
	MessageID                int32
	ChatID                   int64
	StoryID                  int32
	AllowSendingWithoutReply bool
	Quote                    bool
	QuoteText                string
	QuoteParseMode           string
	QuoteEntities            []*MessageEntity
	QuotePosition            int32
	ChecklistTaskID          int32
	PollOptionID             string
}

// ParseInvoice converts a TL Invoice into an Invoice.
func ParseInvoice(raw *tg.Invoice) *Invoice {
	if raw == nil {
		return nil
	}
	out := &Invoice{
		Currency:                   raw.Currency,
		IsTest:                     raw.Test,
		IsNameRequested:            raw.NameRequested,
		IsPhoneRequested:           raw.PhoneRequested,
		IsEmailRequested:           raw.EmailRequested,
		IsShippingAddressRequested: raw.ShippingAddressRequested,
		IsFlexible:                 raw.Flexible,
		IsPhoneToProvider:          raw.PhoneToProvider,
		IsEmailToProvider:          raw.EmailToProvider,
		IsRecurring:                raw.Recurring,
	}
	for _, p := range raw.Prices {
		if p != nil {
			out.Prices = append(out.Prices, &LabeledPrice{
				Label:  p.Label,
				Amount: p.Amount,
			})
		}
	}
	if raw.MaxTipAmount != 0 {
		out.MaxTipAmount = raw.MaxTipAmount
	}
	if len(raw.SuggestedTipAmounts) > 0 {
		out.SuggestedTipAmounts = raw.SuggestedTipAmounts
	}
	if raw.TermsURL != "" {
		out.TermsURL = raw.TermsURL
	}
	if raw.SubscriptionPeriod != 0 {
		out.SubscriptionPeriod = raw.SubscriptionPeriod
	}
	return out
}

// ParseSuccessfulPayment converts a TL MessageActionPaymentSentMe into a SuccessfulPayment.
func ParseSuccessfulPayment(raw *tg.MessageActionPaymentSentMe) *SuccessfulPayment {
	if raw == nil {
		return nil
	}
	out := &SuccessfulPayment{
		Currency:    raw.Currency,
		TotalAmount: raw.TotalAmount,
		Payload:     raw.Payload,
	}
	if raw.ShippingOptionID != "" {
		out.ShippingOptionID = raw.ShippingOptionID
	}
	if raw.Charge != nil {
		out.ChargeID = raw.Charge.ID
		out.ProviderPaymentChargeID = raw.Charge.ProviderChargeID
	}
	return out
}

// ParseShippingAddress converts a TL PostAddress into a ShippingAddress.
func ParseShippingAddress(raw *tg.PostAddress) *ShippingAddress {
	if raw == nil {
		return nil
	}
	return &ShippingAddress{
		StreetLine1: raw.StreetLine1,
		StreetLine2: raw.StreetLine2,
		City:        raw.City,
		State:       raw.State,
		CountryCode: raw.CountryIso2,
		PostCode:    raw.PostCode,
	}
}

// ParseOrderInfo converts a TL PaymentRequestedInfo into an OrderInfo.
func ParseOrderInfo(raw *tg.PaymentRequestedInfo) *OrderInfo {
	if raw == nil {
		return nil
	}
	out := &OrderInfo{}
	if raw.Name != "" {
		out.Name = raw.Name
	}
	if raw.Phone != "" {
		out.Phone = raw.Phone
	}
	if raw.Email != "" {
		out.Email = raw.Email
	}
	if raw.ShippingAddress != nil {
		out.ShippingAddress = ParseShippingAddress(raw.ShippingAddress)
	}
	return out
}
