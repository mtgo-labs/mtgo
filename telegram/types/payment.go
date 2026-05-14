package types

import "github.com/mtgo-labs/mtgo/tg"

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
	// Currency is the three-letter ISO 4217 currency code.
	Currency string
	// Prices is the list of priced portions of the total amount.
	Prices []*LabeledPrice
	// Test indicates whether this is a test payment.
	Test bool
	// NameRequested indicates whether the buyer's name is requested.
	NameRequested bool
	// PhoneRequested indicates whether the buyer's phone number is requested.
	PhoneRequested bool
	// EmailRequested indicates whether the buyer's email is requested.
	EmailRequested bool
	// ShippingAddressRequested indicates whether a shipping address is requested.
	ShippingAddressRequested bool
	// Flexible indicates whether the total amount is flexible.
	Flexible bool
	// PhoneToProvider indicates whether the phone is sent to the payment provider.
	PhoneToProvider bool
	// EmailToProvider indicates whether the email is sent to the payment provider.
	EmailToProvider bool
	// Recurring indicates whether this is a recurring payment.
	Recurring bool
	// MaxTipAmount is the maximum accepted tip amount.
	MaxTipAmount int64
	// SubscriptionPeriod is the period in seconds between recurring payments.
	SubscriptionPeriod int32
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
}

// LinkPreviewOptions controls how link previews are generated for a message.
type LinkPreviewOptions struct {
	// Disabled indicates whether link previews are disabled.
	Disabled bool
	// URL is the URL to use for the link preview.
	URL string
	// SmallMedia requests a small-sized media preview.
	SmallMedia bool
	// LargeMedia requests a large-sized media preview.
	LargeMedia bool
	// PreferSmall indicates a preference for small media previews.
	PreferSmall bool
	// PreferLarge indicates a preference for large media previews.
	PreferLarge bool
}

// ReplyParameters describes how a message replies to another message.
type ReplyParameters struct {
	// MessageID is the ID of the message being replied to.
	MessageID int32
	// ChatID is the ID of the chat containing the message being replied to.
	ChatID int64
	// AllowSendingWithoutReply indicates the reply can be sent even if the
	// referenced message does not exist.
	AllowSendingWithoutReply bool
	// Quote indicates whether the reply quotes a portion of the original message.
	Quote bool
	// QuoteText is the quoted text from the original message.
	QuoteText string
}

// ParseInvoice converts a TL Invoice into an Invoice.
func ParseInvoice(raw *tg.Invoice) *Invoice {
	if raw == nil {
		return nil
	}
	out := &Invoice{
		Currency:                 raw.Currency,
		Test:                     raw.Test,
		NameRequested:            raw.NameRequested,
		PhoneRequested:           raw.PhoneRequested,
		EmailRequested:           raw.EmailRequested,
		ShippingAddressRequested: raw.ShippingAddressRequested,
		Flexible:                 raw.Flexible,
		PhoneToProvider:          raw.PhoneToProvider,
		EmailToProvider:          raw.EmailToProvider,
		Recurring:                raw.Recurring,
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
