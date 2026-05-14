package types

import (
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// Media is the interface implemented by all message media types. It provides a
// unified way to inspect the kind of media attached to a message without
// type-asserting the concrete struct.
//
// Example:
//
//	if msg.Media != nil {
//	    switch msg.Media.MediaType() {
//	    case types.MessageMediaTypePhoto:
//	        fmt.Println("This is a photo")
//	    case types.MessageMediaTypeDocument:
//	        fmt.Println("This is a document")
//	    }
//	}
type Media interface {
	// MediaType returns the MessageMediaType constant for this media.
	MediaType() MessageMediaType
}

// PhotoMedia represents a photo attached to a message.
//
// Example:
//
//	if photo, ok := msg.Media.(*types.PhotoMedia); ok {
//	    fmt.Printf("Photo with %d sizes\n", len(photo.Photo.Sizes))
//	}
type PhotoMedia struct {
	// Photo contains the full photo data with size variants.
	Photo *Photo
	// TTLSeconds is the self-destruct timer in seconds; 0 for persistent photos.
	TTLSeconds int
	// IsSpoiler is true when the photo is hidden behind a spoiler animation.
	IsSpoiler bool
}

// MediaType returns MessageMediaTypePhoto.
func (m *PhotoMedia) MediaType() MessageMediaType { return MessageMediaTypePhoto }

// DocumentMedia represents a generic file document attached to a message. The
// MediaType method disambiguates the document into voice, video note, animation,
// audio, video, sticker, or plain document based on its flags and MIME type.
//
// Example:
//
//	if doc, ok := msg.Media.(*types.DocumentMedia); ok {
//	    fmt.Printf("File: %s (%s, %d bytes)\n", doc.FileName, doc.MimeType, doc.FileSize)
//	}
type DocumentMedia struct {
	// FileID is a composite "id_accessHash" string identifying the file.
	FileID string
	// FileName is the original file name extracted from document attributes.
	FileName string
	// MimeType is the MIME type of the file (e.g. "video/mp4", "audio/ogg").
	MimeType string
	// FileSize is the file size in bytes.
	FileSize int64
	// IsSpoiler is true when the document is hidden behind a spoiler animation.
	IsSpoiler bool
	// IsVoice is true when the document is a voice message.
	IsVoice bool
	// IsRound is true when the document is a round video note.
	IsRound bool
	// IsAnimation is true when the document is an animated GIF (not a video).
	IsAnimation bool
	// TTLSeconds is the self-destruct timer in seconds; 0 for persistent files.
	TTLSeconds int
	// RawDocument holds the underlying TL document for advanced use cases like
	// re-uploading or accessing file locations.
	RawDocument *tg.Document
}

// MediaType returns the specific MessageMediaType derived from the document's
// flags and MIME type.
func (m *DocumentMedia) MediaType() MessageMediaType {
	if m.IsVoice {
		return MessageMediaTypeVoice
	}
	if m.IsRound {
		return MessageMediaTypeVideoNote
	}
	if m.IsAnimation {
		return MessageMediaTypeAnimation
	}
	switch m.MimeType {
	case "audio/mpeg", "audio/ogg", "audio/mp4", "audio/x-m4a":
		return MessageMediaTypeAudio
	case "video/mp4", "video/webm":
		return MessageMediaTypeVideo
	case "image/webp":
		return MessageMediaTypeSticker
	}
	return MessageMediaTypeDocument
}

// ContactMedia represents a contact shared in a message.
type ContactMedia struct {
	// PhoneNumber is the contact's phone number.
	PhoneNumber string
	// FirstName is the contact's first name.
	FirstName string
	// LastName is the contact's last name.
	LastName string
	// VCard contains the vCard data for the contact, if provided.
	VCard string
	// UserID is the Telegram user ID of the contact, or 0 if not on Telegram.
	UserID int64
}

// MediaType returns MessageMediaTypeContact.
func (m *ContactMedia) MediaType() MessageMediaType { return MessageMediaTypeContact }

// LocationMedia represents a geographic location attached to a message.
type LocationMedia struct {
	// Latitude is the geographic latitude.
	Latitude float64
	// Longitude is the geographic longitude.
	Longitude float64
	// Heading is the direction in degrees (0-360) for live locations; 0 when
	// not applicable.
	Heading int
	// Period is the update interval in seconds for live locations.
	Period int
	// Live is true when the location is a live location that updates in real
	// time.
	Live bool
}

// MediaType returns MessageMediaTypeLocation.
func (m *LocationMedia) MediaType() MessageMediaType { return MessageMediaTypeLocation }

// VenueMedia represents a venue (named location) attached to a message.
type VenueMedia struct {
	// Location is the geographic coordinates of the venue.
	Location LocationMedia
	// Title is the name of the venue.
	Title string
	// Address is the street address of the venue.
	Address string
	// Provider is the venue data provider (e.g. "foursquare", "gplaces").
	Provider string
	// VenueID is the provider-specific venue identifier.
	VenueID string
	// VenueType is the provider-specific category type of the venue.
	VenueType string
}

// MediaType returns MessageMediaTypeVenue.
func (m *VenueMedia) MediaType() MessageMediaType { return MessageMediaTypeVenue }

// WebPageMedia represents a link preview attached to a message.
type WebPageMedia struct {
	// URL is the original link URL.
	URL string
	// DisplayURL is the URL shown to the user, which may differ from URL.
	DisplayURL string
	// Type is the content type of the page (e.g. "article", "photo", "video").
	Type string
	// SiteName is the name of the website (e.g. "GitHub", "YouTube").
	SiteName string
	// Title is the page title.
	Title string
	// Description is the page description or snippet.
	Description string
}

// MediaType returns MessageMediaTypeWebPage.
func (m *WebPageMedia) MediaType() MessageMediaType { return MessageMediaTypeWebPage }

// PollMedia represents a poll attached to a message.
type PollMedia struct {
	// ID is the unique poll identifier.
	ID int64
	// Question is the poll question text.
	Question string
	// Answers contains the available answer options.
	Answers []PollAnswer
	// Closed is true when the poll no longer accepts votes.
	Closed bool
}

// PollAnswer represents a single answer option in a poll.
type PollAnswer struct {
	// Text is the display text of the answer option.
	Text string
	// Data is the opaque byte sequence used to identify this option when voting.
	Data []byte
}

// MediaType returns MessageMediaTypePoll.
func (m *PollMedia) MediaType() MessageMediaType { return MessageMediaTypePoll }

// DiceMedia represents a dice roll attached to a message.
type DiceMedia struct {
	// Value is the outcome of the dice roll.
	Value int
	// Emoticon is the emoji used for the dice (e.g. "🎲", "🎯", "🏀").
	Emoticon string
}

// MediaType returns MessageMediaTypeDice.
func (m *DiceMedia) MediaType() MessageMediaType { return MessageMediaTypeDice }

// GameMedia represents a Telegram game attached to a message.
type GameMedia struct {
	// ID is the unique game identifier.
	ID int64
	// Title is the display title of the game.
	Title string
	// Description is the game's short description.
	Description string
	// ShortName is the bot-registered short name used to launch the game.
	ShortName string
}

// MediaType returns MessageMediaTypeGame.
func (m *GameMedia) MediaType() MessageMediaType { return MessageMediaTypeGame }

// InvoiceMedia represents a payment invoice attached to a message.
type InvoiceMedia struct {
	// Title is the product name shown on the invoice.
	Title string
	// Description is the product description.
	Description string
	// Currency is the three-letter ISO 4217 currency code.
	Currency string
	// TotalAmount is the total price in the smallest currency unit (e.g. cents).
	TotalAmount int64
	// StartParam is the parameter passed to the payment provider to start the
	// checkout flow.
	StartParam string
	// IsTest is true when the invoice is in test mode (no real charge).
	IsTest bool
}

// MediaType returns MessageMediaTypeInvoice.
func (m *InvoiceMedia) MediaType() MessageMediaType { return MessageMediaTypeInvoice }

// StoryMedia represents a forwarded Telegram story attached to a message.
type StoryMedia struct {
	// PeerID is the user ID of the story's author.
	PeerID int64
	// ID is the story's unique identifier.
	ID int32
}

// MediaType returns MessageMediaTypeStory.
func (m *StoryMedia) MediaType() MessageMediaType { return MessageMediaTypeStory }

// GiveawayMedia represents a giveaway message.
type GiveawayMedia struct{}

// MediaType returns MessageMediaTypeGiveaway.
func (m *GiveawayMedia) MediaType() MessageMediaType { return MessageMediaTypeGiveaway }

// GiveawayResultsMedia represents the results of a completed giveaway.
type GiveawayResultsMedia struct{}

// MediaType returns MessageMediaTypeGiveawayWinners.
func (m *GiveawayResultsMedia) MediaType() MessageMediaType { return MessageMediaTypeGiveawayWinners }

// PaidMedia represents paid media content that requires Telegram Stars to
// access.
type PaidMedia struct {
	// StarsAmount is the price in Telegram Stars required to unlock the media.
	StarsAmount int64
}

// MediaType returns MessageMediaTypePaidMedia.
func (m *PaidMedia) MediaType() MessageMediaType { return MessageMediaTypePaidMedia }

// ParseMedia converts an MTProto MessageMediaClass into the appropriate Media
// implementation. Returns nil if raw is nil or the media type is not recognized.
//
// Example:
//
//	media := types.ParseMedia(rawMedia)
//	if media != nil {
//	    fmt.Println("Media type:", media.MediaType())
//	}
func ParseMedia(raw tg.MessageMediaClass) Media {
	if raw == nil {
		return nil
	}
	switch r := raw.(type) {
	case *tg.MessageMediaPhoto:
		return parsePhotoMedia(r)
	case *tg.MessageMediaDocument:
		return parseDocumentMedia(r)
	case *tg.MessageMediaContact:
		return &ContactMedia{
			PhoneNumber: r.PhoneNumber,
			FirstName:   r.FirstName,
			LastName:    r.LastName,
			VCard:       r.Vcard,
			UserID:      r.UserID,
		}
	case *tg.MessageMediaGeo:
		return parseGeoMedia(r.Geo)
	case *tg.MessageMediaGeoLive:
		return parseGeoLiveMedia(r)
	case *tg.MessageMediaVenue:
		return parseVenueMedia(r)
	case *tg.MessageMediaWebPage:
		return parseWebPageMedia(r)
	case *tg.MessageMediaPoll:
		return parsePollMedia(r)
	case *tg.MessageMediaDice:
		return &DiceMedia{
			Value:    int(r.Value),
			Emoticon: r.Emoticon,
		}
	case *tg.MessageMediaGame:
		return parseGameMedia(r)
	case *tg.MessageMediaInvoice:
		return parseInvoiceMedia(r)
	case *tg.MessageMediaStory:
		return parseStoryMedia(r)
	case *tg.MessageMediaGiveaway:
		return &GiveawayMedia{}
	case *tg.MessageMediaGiveawayResults:
		return &GiveawayResultsMedia{}
	case *tg.MessageMediaPaidMedia:
		return &PaidMedia{StarsAmount: r.StarsAmount}
	}
	return nil
}

func parsePhotoMedia(raw *tg.MessageMediaPhoto) *PhotoMedia {
	m := &PhotoMedia{
		IsSpoiler: raw.Spoiler,
	}
	if raw.TTLSeconds != 0 {
		m.TTLSeconds = int(raw.TTLSeconds)
	}
	if raw.Photo != nil {
		if p, ok := raw.Photo.(*tg.Photo); ok {
			m.Photo = parsePhoto(p)
		}
	}
	return m
}

func parsePhoto(raw *tg.Photo) *Photo {
	p := &Photo{
		ID:         raw.ID,
		AccessHash: raw.AccessHash,
		Date:       int(raw.Date),
		SizeCount:  len(raw.Sizes),
	}
	for _, s := range raw.Sizes {
		if ps := parsePhotoSize(s); ps != nil {
			p.Sizes = append(p.Sizes, *ps)
		}
	}
	return p
}

func parseDocumentMedia(raw *tg.MessageMediaDocument) *DocumentMedia {
	m := &DocumentMedia{
		IsSpoiler:   raw.Spoiler,
		IsVoice:     raw.Voice,
		IsRound:     raw.Round,
		IsAnimation: raw.Video && !raw.Voice && !raw.Round,
	}
	if raw.TTLSeconds != 0 {
		m.TTLSeconds = int(raw.TTLSeconds)
	}
	if raw.Document != nil {
		if doc, ok := raw.Document.(*tg.Document); ok {
			m.FileID = fmt.Sprintf("%d_%d", doc.ID, doc.AccessHash)
			m.FileSize = doc.Size
			m.MimeType = doc.MimeType
			m.RawDocument = doc
			for _, attr := range doc.Attributes {
				if a, ok := attr.(*tg.DocumentAttributeFilename); ok {
					m.FileName = a.FileName
				}
			}
		}
	}
	return m
}

func parseGeoMedia(geo tg.GeoPointClass) *LocationMedia {
	if geo == nil {
		return nil
	}
	if g, ok := geo.(*tg.GeoPoint); ok {
		return &LocationMedia{
			Latitude:  g.Lat,
			Longitude: g.Long,
		}
	}
	return &LocationMedia{}
}

func parseGeoLiveMedia(raw *tg.MessageMediaGeoLive) *LocationMedia {
	m := parseGeoMedia(raw.Geo)
	if m == nil {
		m = &LocationMedia{}
	}
	m.Live = true
	m.Period = int(raw.Period)
	if raw.Heading != 0 {
		m.Heading = int(raw.Heading)
	}
	return m
}

func parseVenueMedia(raw *tg.MessageMediaVenue) *VenueMedia {
	m := &VenueMedia{
		Title:     raw.Title,
		Address:   raw.Address,
		Provider:  raw.Provider,
		VenueID:   raw.VenueID,
		VenueType: raw.VenueType,
	}
	if raw.Geo != nil {
		loc := parseGeoMedia(raw.Geo)
		if loc != nil {
			m.Location = *loc
		}
	}
	return m
}

func parseWebPageMedia(raw *tg.MessageMediaWebPage) *WebPageMedia {
	m := &WebPageMedia{}
	if raw.Webpage != nil {
		if wp, ok := raw.Webpage.(*tg.WebPage); ok {
			m.URL = wp.URL
			m.DisplayURL = wp.DisplayURL
			if wp.Type != "" {
				m.Type = wp.Type
			}
			if wp.SiteName != "" {
				m.SiteName = wp.SiteName
			}
			if wp.Title != "" {
				m.Title = wp.Title
			}
			if wp.Description != "" {
				m.Description = wp.Description
			}
		}
	}
	return m
}

func parsePollMedia(raw *tg.MessageMediaPoll) *PollMedia {
	if raw.Poll == nil {
		return nil
	}
	m := &PollMedia{
		ID:     raw.Poll.ID,
		Closed: raw.Poll.Closed,
	}
	if raw.Poll.Question != nil {
		m.Question = raw.Poll.Question.Text
	}
	for _, a := range raw.Poll.Answers {
		if ans, ok := a.(*tg.PollAnswer); ok {
			text := ""
			if ans.Text != nil {
				text = ans.Text.Text
			}
			m.Answers = append(m.Answers, PollAnswer{
				Text: text,
				Data: ans.Option,
			})
		}
	}
	return m
}

func parseGameMedia(raw *tg.MessageMediaGame) *GameMedia {
	if raw.Game == nil {
		return nil
	}
	return &GameMedia{
		ID:          raw.Game.ID,
		Title:       raw.Game.Title,
		Description: raw.Game.Description,
		ShortName:   raw.Game.ShortName,
	}
}

func parseInvoiceMedia(raw *tg.MessageMediaInvoice) *InvoiceMedia {
	return &InvoiceMedia{
		Title:       raw.Title,
		Description: raw.Description,
		Currency:    raw.Currency,
		TotalAmount: raw.TotalAmount,
		StartParam:  raw.StartParam,
		IsTest:      raw.Test,
	}
}

func parseStoryMedia(raw *tg.MessageMediaStory) *StoryMedia {
	m := &StoryMedia{ID: raw.ID}
	if raw.Peer != nil {
		if p, ok := raw.Peer.(*tg.PeerUser); ok {
			m.PeerID = p.UserID
		}
	}
	return m
}

// Photo represents a Telegram photo with its size variants and metadata.
type Photo struct {
	// ID is the unique photo identifier.
	ID int64
	// AccessHash is required to access the photo file.
	AccessHash int64
	// Date is the Unix timestamp when the photo was uploaded.
	Date int
	// Sizes contains all available size variants of the photo.
	Sizes []PhotoSize
	// SizeCount is the number of available sizes, cached for quick access.
	SizeCount int
}

// PhotoSize describes a single size variant of a Photo.
type PhotoSize struct {
	// Type is the size label (e.g. "s" for small, "m" for medium, "x" for large,
	// "w" for 2560px).
	Type string
	// Width is the image width in pixels.
	Width int
	// Height is the image height in pixels.
	Height int
	// Size is the file size in bytes. May be 0 for cached sizes where the byte
	// count is not available.
	Size int
}

func parsePhotoSize(raw tg.PhotoSizeClass) *PhotoSize {
	if raw == nil {
		return nil
	}
	switch s := raw.(type) {
	case *tg.PhotoSize:
		return &PhotoSize{
			Type:   s.Type,
			Width:  int(s.W),
			Height: int(s.H),
			Size:   int(s.Size),
		}
	case *tg.PhotoCachedSize:
		return &PhotoSize{
			Type:   s.Type,
			Width:  int(s.W),
			Height: int(s.H),
		}
	case *tg.PhotoSizeProgressive:
		return &PhotoSize{
			Type:   s.Type,
			Width:  int(s.W),
			Height: int(s.H),
			Size: func() int {
				if len(s.Sizes) > 0 {
					return int(s.Sizes[len(s.Sizes)-1])
				}
				return 0
			}(),
		}
	}
	return nil
}
