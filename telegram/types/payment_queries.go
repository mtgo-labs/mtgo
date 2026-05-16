package types

import (
	"github.com/mtgo-labs/mtgo/telegram/params"
)

// PreCheckoutQuery is sent to the bot when a user confirms a payment but before
// the charge is finalized. The bot must respond within 10 seconds to approve or
// reject the payment.
type PreCheckoutQuery struct {
	ID               int64
	UserID           int64
	FromUser         *User
	Currency         string
	TotalAmount      int64
	InvoicePayload   string
	InvoiceSlug      string
	ShippingOptionID string
	OrderInfo        *OrderInfo
	binder           Binder
}

func (q *PreCheckoutQuery) SetBinder(b Binder) {
	q.binder = b
}

func (q *PreCheckoutQuery) Answer(ok bool, errorMsg string) error {
	if q.binder == nil {
		return ErrNoBinder
	}
	return q.binder.BoundAnswerPreCheckout(q.ID, &params.AnswerPreCheckout{Ok: ok, ErrorMsg: errorMsg})
}

// ShippingQuery is sent to the bot when a user selects a shipping address during
// checkout so the bot can return available shipping options.
type ShippingQuery struct {
	ID             int64
	UserID         int64
	FromUser       *User
	InvoicePayload string
	InvoiceSlug    string
	Address        *ShippingAddress
	binder         Binder
}

func (q *ShippingQuery) SetBinder(b Binder) {
	q.binder = b
}

func (q *ShippingQuery) Answer(ok bool, errorMsg string) error {
	if q.binder == nil {
		return ErrNoBinder
	}
	return q.binder.BoundAnswerShipping(q.ID, &params.AnswerShipping{Ok: ok, ErrorMsg: errorMsg})
}

// OrderInfo contains the contact and shipping details provided by the buyer
// during a payment flow.
type OrderInfo struct {
	// Name is the buyer's full name, empty when not requested by the invoice.
	Name string
	// Phone is the buyer's phone number, empty when not requested by the invoice.
	Phone string
	// Email is the buyer's email address, empty when not requested by the invoice.
	Email string
	// ShippingAddress is the shipping address provided by the buyer, or nil when
	// the invoice does not request a shipping address.
	ShippingAddress *ShippingAddress
}

// ShippingAddress represents a postal address provided during a payment checkout.
type ShippingAddress struct {
	// CountryCode is the two-letter ISO 3166-1 alpha-2 country code.
	CountryCode string
	// State is the state or province name, if applicable.
	State string
	// City is the city name.
	City string
	// StreetLine1 is the first line of the street address.
	StreetLine1 string
	// StreetLine2 is the second line of the street address, if applicable.
	StreetLine2 string
	// PostCode is the postal or ZIP code.
	PostCode string
}

// ShippingOption represents a single shipping option presented to the user
// during checkout. The bot returns a list of these in response to a ShippingQuery.
type ShippingOption struct {
	ID     string
	Title  string
	Prices []*LabeledPrice
}

// PurchasedPaidMedia is sent when a user purchases access to paid media
// attached to a message. Use it to verify and fulfill the media purchase.
type PurchasedPaidMedia struct {
	FromUser    *User
	Payload     string
	UserID      int64
	PaidMediaID string
}
