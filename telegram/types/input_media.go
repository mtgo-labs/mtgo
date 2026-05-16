package types

// InputMediaType identifies the kind of media attached to an input message
// being sent or forwarded through the Telegram API.
type InputMediaType string

const (
	// InputMediaPhoto is a photo to be sent.
	InputMediaPhoto InputMediaType = "photo"
	// InputMediaVideo is a video to be sent.
	InputMediaVideo InputMediaType = "video"
	// InputMediaAudio is an audio file to be sent.
	InputMediaAudio InputMediaType = "audio"
	// InputMediaDocument is a generic document to be sent.
	InputMediaDocument InputMediaType = "document"
	// InputMediaSticker is a sticker to be sent.
	InputMediaSticker InputMediaType = "sticker"
	// InputMediaAnimation is an animated GIF or MPEG4 to be sent.
	InputMediaAnimation InputMediaType = "animation"
	// InputMediaLocation is a geographic location to be sent.
	InputMediaLocation InputMediaType = "location"
	// InputMediaVenue is a named venue to be sent.
	InputMediaVenue InputMediaType = "venue"
	// InputMediaContact is a contact to be shared.
	InputMediaContact InputMediaType = "contact"
)

// InputMedia is a union type representing media content that can be sent or
// forwarded through the Telegram API. The Type field determines which fields
// are populated.
type InputMedia struct {
	// Type identifies the media kind (photo, video, document, etc.).
	Type InputMediaType
	// FileID is the Telegram file ID for previously uploaded files.
	FileID int64
	// URL is the HTTP URL for external media resources.
	URL string
	// Caption is the text caption to attach to the media.
	Caption string
	// ParseMode is the text formatting mode for the caption.
	ParseMode ParseMode
	// FileName is the original file name for document uploads.
	FileName string
	// MimeType is the MIME type of the file being uploaded.
	MimeType string
	// Latitude is the geographic latitude for location media.
	Latitude float64
	// Longitude is the geographic longitude for location media.
	Longitude float64
	// Title is the venue name for venue media.
	Title string
	// Address is the venue address for venue media.
	Address string
	// Provider is the venue data provider name (e.g. "foursquare").
	Provider string
	// VenueID is the unique venue identifier from the provider.
	VenueID string
}

// InputMediaLivePhoto references a live photo file for uploading.
type InputMediaLivePhoto struct {
	FileID     int64
	AccessHash int64
}

// InputPollMedia represents a poll as a media attachment.
type InputPollMedia struct {
	Type string
}

// InputPollOptionMedia represents a single poll option as a media attachment.
type InputPollOptionMedia struct {
	Type string
}
