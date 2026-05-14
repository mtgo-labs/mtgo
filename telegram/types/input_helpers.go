package types

// CredentialsType identifies the kind of payment credentials provided by the
// user during a Telegram Stars or payment transaction.
type CredentialsType string

const (
	// CredentialsSaved uses previously saved payment credentials.
	CredentialsSaved CredentialsType = "saved"
	// CredentialsNew provides new payment card credentials.
	CredentialsNew CredentialsType = "new"
	// CredentialsTemp uses temporary payment credentials for a one-time payment.
	CredentialsTemp CredentialsType = "temp"
	// CredentialsApplePay uses Apple Pay as the payment method.
	CredentialsApplePay CredentialsType = "apple_pay"
	// CredentialsGooglePay uses Google Pay as the payment method.
	CredentialsGooglePay CredentialsType = "google_pay"
)

// InvoiceType identifies the source of a payment invoice, determining how
// the invoice is referenced (by slug, message, invite, or Stars).
type InvoiceType string

const (
	// InvoiceSlug references an invoice by its public slug URL.
	InvoiceSlug InvoiceType = "slug"
	// InvoiceMessage references an invoice attached to a specific message.
	InvoiceMessage InvoiceType = "message"
	// InvoiceChatInvite references a chat invite that requires payment.
	InvoiceChatInvite InvoiceType = "chat_invite"
	// InvoiceStars references a Telegram Stars payment invoice.
	InvoiceStars InvoiceType = "stars"
)

// InputPhoneContact represents a phone contact to be imported into the user's
// Telegram contact list.
type InputPhoneContact struct {
	// Phone is the contact's phone number in international format.
	Phone string
	// FirstName is the contact's given name.
	FirstName string
	// LastName is the contact's family name.
	LastName string
}

// InputPollOption represents a single option in a poll to be created.
type InputPollOption struct {
	// Text is the option text displayed to voters.
	Text string
	// ParseMode is the text formatting mode for the option text (Markdown, HTML, etc.).
	ParseMode ParseMode
}

// InputCredentials represents payment credentials supplied by the user for
// completing a payment transaction.
type InputCredentials struct {
	// Type identifies the kind of credentials (saved, new, temp, Apple Pay, Google Pay).
	Type CredentialsType
	// ID is the saved credential identifier for saved credential types.
	ID string
	// Password is the temporary payment password for temp credential types.
	Password string
	// Data is the opaque payment provider data for new and temp credential types.
	Data []byte
}

// InputInvoice identifies an invoice to be paid or viewed. The Type field
// determines which other fields are populated.
type InputInvoice struct {
	// Type identifies the invoice source (slug, message, chat invite, or Stars).
	Type InvoiceType
	// Slug is the public invoice slug for slug-type invoices.
	Slug string
	// ChatID is the chat containing the invoice message, for message-type invoices.
	ChatID int64
	// MsgID is the message ID of the invoice, for message-type invoices.
	MsgID int32
	// GiftCode is the gift code to redeem, for gift code invoices.
	GiftCode string
}
