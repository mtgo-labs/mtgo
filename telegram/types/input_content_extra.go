package types

// InputContactMessageContent represents contact content for an inline message.
type InputContactMessageContent struct {
	PhoneNumber string
	FirstName   string
	LastName    string
	VCard       string
}

// InputLocationMessageContent represents location content for an inline message.
type InputLocationMessageContent struct {
	Latitude        float64
	Longitude       float64
	Heading         int32
	Period          int32
	ProximityRadius int32
}

// InputVenueMessageContent represents venue content for an inline message.
type InputVenueMessageContent struct {
	Latitude  float64
	Longitude float64
	Title     string
	Address   string
	Provider  string
	VenueID   string
	VenueType string
}

// InputTextMessageContent represents text content for an inline message.
type InputTextMessageContent struct {
	Text           string
	ParseMode      ParseMode
	DisablePreview bool
}

// InputInvoiceMessage references an invoice attached to a message.
type InputInvoiceMessage struct {
	ChatID int64
	MsgID  int32
}

// InputInvoiceMessageContent represents invoice content for an inline message.
type InputInvoiceMessageContent struct {
	Title       string
	Description string
	Payload     []byte
	Provider    string
	StartParam  string
}

// InputInvoiceName references an invoice by its slug name.
type InputInvoiceName struct {
	Slug string
}
