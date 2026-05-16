package types

// PaymentFormType distinguishes between different kinds of Telegram payment forms.
// Use this to determine how a payment form should be rendered and processed.
type PaymentFormType string

const (
	// PaymentFormTypeInvoice indicates a payment form created via the invoice
	// flow, typically used for direct payments between user and bot.
	PaymentFormTypeInvoice PaymentFormType = "invoice"
	// PaymentFormTypeRegular indicates a regular payment form used for standard
	// checkout flows.
	PaymentFormTypeRegular PaymentFormType = "regular"
	// PaymentFormTypeStars indicates a payment form for Telegram Stars payments.
	PaymentFormTypeStars PaymentFormType = "stars"
	// PaymentFormTypeStarSubscription indicates a payment form for Telegram Stars
	// subscription payments.
	PaymentFormTypeStarSubscription PaymentFormType = "star_subscription"
)

// String returns the string representation of the PaymentFormType.
func (p PaymentFormType) String() string { return string(p) }
