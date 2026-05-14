package types

// InputContentType identifies the kind of content in an inline or input
// message, such as text, location, venue, contact, or invoice.
type InputContentType string

const (
	// InputContentText is plain or formatted text content.
	InputContentText InputContentType = "text"
	// InputContentLocation is a geographic location.
	InputContentLocation InputContentType = "location"
	// InputContentVenue is a named venue with address.
	InputContentVenue InputContentType = "venue"
	// InputContentContact is a phone contact.
	InputContentContact InputContentType = "contact"
	// InputContentInvoice is a payment invoice.
	InputContentInvoice InputContentType = "invoice"
)

// InputMessageContent represents content that can be used as the payload of an
// inline result or a bot message. The Type field determines which fields are relevant.
type InputMessageContent struct {
	// Type identifies the content kind (text, location, venue, contact, invoice).
	Type InputContentType
	// Text is the message text for text-type content.
	Text string
	// ParseMode is the text formatting mode for text content.
	ParseMode ParseMode
	// DisablePreview disables link preview for text content when true.
	DisablePreview bool
	// Latitude is the geographic latitude for location and venue content.
	Latitude float64
	// Longitude is the geographic longitude for location and venue content.
	Longitude float64
	// Title is the venue name for venue-type content.
	Title string
	// Address is the venue address for venue-type content.
	Address string
	// Provider is the venue data provider for venue-type content.
	Provider string
	// VenueID is the venue identifier from the provider for venue-type content.
	VenueID string
	// PhoneNumber is the contact's phone number for contact-type content.
	PhoneNumber string
	// FirstName is the contact's first name for contact-type content.
	FirstName string
	// LastName is the contact's last name for contact-type content.
	LastName string
	// VCard is the vCard data for contact-type content.
	VCard string
}
