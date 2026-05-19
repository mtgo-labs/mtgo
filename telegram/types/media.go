package types

import (
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/telegram/fileid"
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
	FileID              string
	FileUniqueID        string
	FileName            string
	MimeType            string
	FileSize            int64
	Date                time.Time
	DCID                int32
	Thumbs              []*Thumbnail
	Duration            int
	Width               int
	Height              int
	Codec               string
	Length              int
	Waveform            []byte
	Performer           string
	Title               string
	SupportsStreaming   bool
	VideoCover          *Photo
	VideoStartTimestamp int
	TTLSeconds          int
	IsSpoiler           bool
	IsVoice             bool
	IsRound             bool
	IsAnimation         bool
	AlternativeVideos   []*DocumentMedia
	RawDocument         *tg.Document
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
	Latitude             float64
	Longitude            float64
	AccuracyRadius       int
	Address              string
	LivePeriod           int
	Heading              int
	ProximityAlertRadius int
	Live                 bool
}

// MediaType returns MessageMediaTypeLocation.
func (m *LocationMedia) MediaType() MessageMediaType { return MessageMediaTypeLocation }

// VenueMedia represents a venue (named location) attached to a message.
type VenueMedia struct {
	Location       *LocationMedia
	Title          string
	Address        string
	FoursquareID   string
	FoursquareType string
	Provider       string
	VenueID        string
	VenueType      string
}

// MediaType returns MessageMediaTypeVenue.
func (m *VenueMedia) MediaType() MessageMediaType { return MessageMediaTypeVenue }

// WebPageMedia represents a link preview attached to a message.
type WebPageMedia struct {
	ID               string
	URL              string
	DisplayURL       string
	Type             string
	SiteName         string
	Title            string
	Description      string
	Photo            *Photo
	Document         *DocumentMedia
	Audio            *DocumentMedia
	Animation        *DocumentMedia
	Video            *DocumentMedia
	Duration         int
	Author           string
	EmbedURL         string
	EmbedType        string
	EmbedWidth       int
	EmbedHeight      int
	HasLargeMedia    bool
	PreferLargeMedia bool
	PreferSmallMedia bool
	Manual           bool
	Safe             bool
	Raw              *tg.MessageMediaWebPage
}

// MediaType returns MessageMediaTypeWebPage.
func (m *WebPageMedia) MediaType() MessageMediaType { return MessageMediaTypeWebPage }

// PollMedia represents a poll attached to a message.
type PollMedia struct {
	ID                    int64
	Question              *FormattedText
	Options               []PollOption
	TotalVoterCount       int32
	IsClosed              bool
	IsAnonymous           bool
	Type                  PollType
	AllowsMultipleAnswers bool
	AllowsRevoting        bool
	MembersOnly           bool
	CountryCodes          []string
	ChosenOptionIDs       []int
	CorrectOptionIDs      []int
	Explanation           *FormattedText
	ExplanationMedia      Media
	Description           *FormattedText
	DescriptionMedia      Media
	Voter                 *User
	OpenPeriod            int32
	CloseDate             time.Time
}

// PollOption represents a single option in a poll, including its text, voter
// statistics, and metadata about who added it.
//
// Example:
//
//	for _, opt := range poll.Options {
//		fmt.Printf("Option %s: %d%% voted\n", opt.Text, opt.VotePercentage)
//	}
type PollOption struct {
	PersistentID   string
	Text           *FormattedText
	Media          Media
	VoterCount     int32
	VotePercentage int32
	RecentVoters   []*Chat
	AddedByUser    *User
	AddedByChat    *Chat
	AdditionDate   time.Time
	Data           []byte
	IsChosen       bool
	IsCorrect      bool
}

// MediaType returns MessageMediaTypePoll.
func (m *PollMedia) MediaType() MessageMediaType { return MessageMediaTypePoll }

// DiceMedia represents a dice roll attached to a message.
type DiceMedia struct {
	Emoji string
	Value int
}

// MediaType returns MessageMediaTypeDice.
func (m *DiceMedia) MediaType() MessageMediaType { return MessageMediaTypeDice }

// GameMedia represents a Telegram game attached to a message.
type GameMedia struct {
	ID          int64
	Title       string
	ShortName   string
	Description string
	Photo       *Photo
	Animation   *DocumentMedia
}

// MediaType returns MessageMediaTypeGame.
func (m *GameMedia) MediaType() MessageMediaType { return MessageMediaTypeGame }

// InvoiceMedia represents a payment invoice attached to a message.
type InvoiceMedia struct {
	Title                    string
	Description              string
	Currency                 string
	TotalAmount              int64
	StartParam               string
	IsTest                   bool
	ShippingAddressRequested bool
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
			Emoji: r.Emoticon,
			Value: int(r.Value),
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
	if raw.Photo != nil {
		if p, ok := raw.Photo.(*tg.Photo); ok {
			m.Photo = parsePhoto(p)
		}
	}
	if raw.TTLSeconds != 0 && m.Photo != nil {
		m.Photo.TTLSeconds = int(raw.TTLSeconds)
		m.TTLSeconds = int(raw.TTLSeconds)
	}
	return m
}

func parsePhoto(raw *tg.Photo) *Photo {
	p := &Photo{
		ID:            raw.ID,
		AccessHash:    raw.AccessHash,
		FileReference: raw.FileReference,
		Date:          time.Unix(int64(raw.Date), 0),
		DCID:          raw.DCID,
		SizeCount:     len(raw.Sizes),
	}
	p.FileUniqueID = fmt.Sprintf("%d", raw.ID)
	if encoded, err := fileid.Encode(fileid.FileID{
		Type:          fileid.FileTypePhoto,
		DCID:          raw.DCID,
		ID:            raw.ID,
		AccessHash:    raw.AccessHash,
		FileReference: raw.FileReference,
		Source: fileid.PhotoSizeSource{
			Type:              fileid.ThumbnailSourceThumbnail,
			ThumbnailFileType: fileid.FileTypePhoto,
			ThumbnailSize:     int32('x'),
		},
	}); err == nil {
		p.FileID = encoded
	}
	for _, s := range raw.Sizes {
		if th := ParseThumbnail(s); th != nil {
			p.Thumbs = append(p.Thumbs, th)
		}
		if ps := parsePhotoSize(s); ps != nil {
			p.Sizes = append(p.Sizes, *ps)
			if ps.Width > p.Width {
				p.Width = ps.Width
			}
			if ps.Height > p.Height {
				p.Height = ps.Height
			}
			if ps.Size > int(p.FileSize) {
				p.FileSize = int64(ps.Size)
			}
		}
	}
	return p
}

func parseDocumentMedia(raw *tg.MessageMediaDocument) *DocumentMedia {
	m := &DocumentMedia{
		IsSpoiler: raw.Spoiler,
		IsVoice:   raw.Voice,
		IsRound:   raw.Round,
	}
	if raw.TTLSeconds != 0 {
		m.TTLSeconds = int(raw.TTLSeconds)
	}
	if raw.Document != nil {
		if doc, ok := raw.Document.(*tg.Document); ok {
			parseDocumentAttrs(doc, m)
		}
	}
	return m
}

func parseDocumentAttrs(doc *tg.Document, m *DocumentMedia) {
	m.FileUniqueID = fmt.Sprintf("%d", doc.ID)
	m.FileSize = doc.Size
	m.MimeType = doc.MimeType
	m.DCID = doc.DCID
	m.Date = time.Unix(int64(doc.Date), 0)
	m.RawDocument = doc
	for _, s := range doc.Thumbs {
		if th := ParseThumbnail(s); th != nil {
			m.Thumbs = append(m.Thumbs, th)
		}
	}
	for _, attr := range doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeFilename:
			m.FileName = a.FileName
		case *tg.DocumentAttributeAudio:
			m.Duration = int(a.Duration)
			if a.Performer != "" {
				m.Performer = a.Performer
			}
			if a.Title != "" {
				m.Title = a.Title
			}
			if len(a.Waveform) > 0 {
				m.Waveform = a.Waveform
			}
		case *tg.DocumentAttributeVideo:
			m.Duration = int(a.Duration)
			m.Width = int(a.W)
			m.Height = int(a.H)
			m.SupportsStreaming = a.SupportsStreaming
			if a.RoundMessage {
				m.Length = int(a.W)
			}
			if a.VideoCodec != "" {
				m.Codec = a.VideoCodec
			}
			if a.VideoStartTs != 0 {
				m.VideoStartTimestamp = int(a.VideoStartTs)
			}
		case *tg.DocumentAttributeAnimated:
			m.IsAnimation = true
		}
	}
	if encoded, err := fileid.Encode(fileid.FileID{
		Type:          documentFileType(m),
		DCID:          doc.DCID,
		ID:            doc.ID,
		AccessHash:    doc.AccessHash,
		FileReference: doc.FileReference,
	}); err == nil {
		m.FileID = encoded
	}
}

func documentFileType(m *DocumentMedia) fileid.FileType {
	switch m.MediaType() {
	case MessageMediaTypeVoice:
		return fileid.FileTypeVoice
	case MessageMediaTypeVideoNote:
		return fileid.FileTypeVideoNote
	case MessageMediaTypeAnimation:
		return fileid.FileTypeAnimation
	case MessageMediaTypeAudio:
		return fileid.FileTypeAudio
	case MessageMediaTypeVideo:
		return fileid.FileTypeVideo
	case MessageMediaTypeSticker:
		return fileid.FileTypeSticker
	default:
		return fileid.FileTypeDocument
	}
}

func parseGeoMedia(geo tg.GeoPointClass) *LocationMedia {
	if geo == nil {
		return nil
	}
	if g, ok := geo.(*tg.GeoPoint); ok {
		return &LocationMedia{
			Latitude:       g.Lat,
			Longitude:      g.Long,
			AccuracyRadius: int(g.AccuracyRadius),
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
	m.LivePeriod = int(raw.Period)
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
			m.Location = loc
		}
	}
	return m
}

func parseWebPageMedia(raw *tg.MessageMediaWebPage) *WebPageMedia {
	m := &WebPageMedia{Raw: raw}
	if raw.Webpage == nil {
		return m
	}
	wp, ok := raw.Webpage.(*tg.WebPage)
	if !ok {
		return m
	}
	m.ID = fmt.Sprintf("%d", wp.ID)
	m.URL = wp.URL
	m.DisplayURL = wp.DisplayURL
	m.Type = wp.Type
	m.SiteName = wp.SiteName
	m.Title = wp.Title
	m.Description = wp.Description
	m.Duration = int(wp.Duration)
	m.Author = wp.Author
	if wp.Photo != nil {
		if p, ok := wp.Photo.(*tg.Photo); ok {
			m.Photo = parsePhoto(p)
		}
	}
	if wp.Document != nil {
		if doc, ok := wp.Document.(*tg.Document); ok {
			dm := &DocumentMedia{}
			parseDocumentAttrs(doc, dm)
			m.Document = dm
		}
	}
	if wp.EmbedURL != "" {
		m.EmbedURL = wp.EmbedURL
		m.EmbedType = wp.EmbedType
		m.EmbedWidth = int(wp.EmbedWidth)
		m.EmbedHeight = int(wp.EmbedHeight)
	}
	return m
}

func parsePollMedia(raw *tg.MessageMediaPoll) *PollMedia {
	if raw.Poll == nil {
		return nil
	}
	p := raw.Poll
	m := &PollMedia{
		ID:                    p.ID,
		IsClosed:              p.Closed,
		IsAnonymous:           !p.PublicVoters,
		AllowsMultipleAnswers: p.MultipleChoice,
		AllowsRevoting:        !p.RevotingDisabled,
		MembersOnly:           p.SubscribersOnly,
	}
	if raw.Results != nil {
		m.TotalVoterCount = raw.Results.TotalVoters
	}
	if p.PublicVoters {
		m.Type = PollTypeRegular
	} else {
		m.Type = PollTypeQuiz
	}
	if p.Question != nil {
		m.Question = &FormattedText{Text: p.Question.Text}
		for _, e := range p.Question.Entities {
			if me := ParseMessageEntity(e); me != nil {
				m.Question.Entities = append(m.Question.Entities, me)
			}
		}
	}
	for _, a := range p.Answers {
		if ans, ok := a.(*tg.PollAnswer); ok {
			opt := PollOption{Data: ans.Option}
			if ans.Text != nil {
				opt.Text = &FormattedText{Text: ans.Text.Text}
				for _, e := range ans.Text.Entities {
					if me := ParseMessageEntity(e); me != nil {
						opt.Text.Entities = append(opt.Text.Entities, me)
					}
				}
			}
			m.Options = append(m.Options, opt)
		}
	}
	if raw.Results != nil {
		for _, r := range raw.Results.Results {
			for i := range m.Options {
				if string(m.Options[i].Data) == string(r.Option) {
					m.Options[i].VoterCount = r.Voters
					m.Options[i].IsChosen = r.Chosen
					m.Options[i].IsCorrect = r.Correct
					break
				}
			}
		}
	}
	if p.ClosePeriod > 0 {
		m.OpenPeriod = p.ClosePeriod
	}
	if p.CloseDate != 0 {
		m.CloseDate = time.Unix(int64(p.CloseDate), 0)
	}
	if len(p.CountriesIso2) > 0 {
		m.CountryCodes = p.CountriesIso2
	}
	if raw.Results != nil {
		if raw.Results.Solution != "" {
			m.Explanation = &FormattedText{Text: raw.Results.Solution}
			for _, e := range raw.Results.SolutionEntities {
				if me := ParseMessageEntity(e); me != nil {
					m.Explanation.Entities = append(m.Explanation.Entities, me)
				}
			}
		}
	}
	return m
}

func parseGameMedia(raw *tg.MessageMediaGame) *GameMedia {
	if raw.Game == nil {
		return nil
	}
	g := &GameMedia{
		ID:          raw.Game.ID,
		Title:       raw.Game.Title,
		Description: raw.Game.Description,
		ShortName:   raw.Game.ShortName,
	}
	if raw.Game.Photo != nil {
		if p, ok := raw.Game.Photo.(*tg.Photo); ok {
			g.Photo = parsePhoto(p)
		}
	}
	if raw.Game.Document != nil {
		if doc, ok := raw.Game.Document.(*tg.Document); ok {
			dm := &DocumentMedia{}
			parseDocumentAttrs(doc, dm)
			g.Animation = dm
		}
	}
	return g
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
	FileID        string
	FileUniqueID  string
	Width         int
	Height        int
	FileSize      int64
	Date          time.Time
	TTLSeconds    int
	Thumbs        []*Thumbnail
	ID            int64
	AccessHash    int64
	FileReference []byte
	DCID          int32
	Sizes         []PhotoSize
	SizeCount     int
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

// LivePhoto represents a live photo (short video attached to a photo).
type LivePhoto struct {
	FileID       string
	FileUniqueID string
	Width        int
	Height       int
	Duration     int
	MimeType     string
	FileSize     int64
	ID           int64
	AccessHash   int64
	Date         time.Time
	Sizes        []PhotoSize
	VideoSizes   []VideoSize
}

// VideoSize represents a video size variant.
type VideoSize struct {
	Type   string
	Width  int32
	Height int32
	Size   int32
}

// Thumbnail represents a photo or file thumbnail (wraps PhotoSize).
type Thumbnail struct {
	FileID       string
	FileUniqueID string
	Width        int32
	Height       int32
	FileSize     int32
}

// ParseThumbnail converts a raw Telegram PhotoSizeClass into a structured
// Thumbnail with dimensions, size, and an optional data URI.
//
// Example:
//
//	thumb := types.ParseThumbnail(photoSize)
//	fmt.Printf("Thumbnail: %dx%d (%d bytes)\n", thumb.Width, thumb.Height, thumb.FileSize)
func ParseThumbnail(raw tg.PhotoSizeClass) *Thumbnail {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.PhotoSize:
		return &Thumbnail{Width: v.W, Height: v.H, FileSize: v.Size}
	case *tg.PhotoSizeProgressive:
		sz := int32(0)
		if len(v.Sizes) > 0 {
			sz = v.Sizes[len(v.Sizes)-1]
		}
		return &Thumbnail{Width: v.W, Height: v.H, FileSize: sz}
	case *tg.PhotoCachedSize:
		return &Thumbnail{Width: v.W, Height: v.H}
	}
	return nil
}

// StrippedThumbnail represents a stripped inline thumbnail.
type StrippedThumbnail struct {
	Data []byte
}

// ParseStrippedThumbnail converts a raw Telegram PhotoStrippedSize into a
// StrippedThumbnail containing the inline thumbnail bytes.
//
// Example:
//
//	st := types.ParseStrippedThumbnail(strippedSize)
//	if st != nil {
//		fmt.Printf("Stripped thumbnail: %d bytes\n", len(st.Data))
//	}
func ParseStrippedThumbnail(raw *tg.PhotoStrippedSize) *StrippedThumbnail {
	if raw == nil {
		return nil
	}
	return &StrippedThumbnail{Data: raw.Bytes}
}

// AvailableEffect describes an available message effect.
type AvailableEffect struct {
	ID                int64
	Emoji             string
	EffectStickerID   int64
	Sticker           *Sticker
	IsPremium         bool
	StaticIconID      int64
	EffectAnimationID int64
}

// ParseAvailableEffect converts a raw Telegram AvailableEffect into a
// structured AvailableEffect with emoji, sticker, and premium details.
//
// Example:
//
//	effect := types.ParseAvailableEffect(rawEffect)
//	fmt.Printf("Effect: %s (premium=%v)\n", effect.Emoji, effect.IsPremium)
func ParseAvailableEffect(raw *tg.AvailableEffect) *AvailableEffect {
	if raw == nil {
		return nil
	}
	e := &AvailableEffect{
		ID:              raw.ID,
		Emoji:           raw.Emoticon,
		EffectStickerID: raw.EffectStickerID,
		IsPremium:       raw.PremiumRequired,
	}
	if raw.StaticIconID != 0 {
		e.StaticIconID = raw.StaticIconID
	}
	if raw.EffectAnimationID != 0 {
		e.EffectAnimationID = raw.EffectAnimationID
	}
	return e
}

// FormattedText contains text with formatting entities.
type FormattedText struct {
	Text      string
	ParseMode string
	Entities  []*MessageEntity
}

// MessageOrigin describes the origin of a forwarded/replied-to message.
type MessageOrigin struct {
	Type            MessageOriginType
	Date            time.Time
	SenderUser      *User
	SenderUserID    int64
	SenderChat      *Chat
	SenderChatID    int64
	Chat            *Chat
	ChatID          int64
	MessageID       int32
	AuthorSignature string
	SenderUserName  string
	Imported        bool
}

// ExternalReplyInfo contains info about a message being replied to from another chat.
type ExternalReplyInfo struct {
	Origin             *MessageOrigin
	Chat               *Chat
	MessageID          int32
	HasMediaSpoiler    bool
	Media              MessageMediaType
	LinkPreviewOptions *LinkPreviewOptions
	Photo              *Photo
	Animation          *Animation
	Audio              *DocumentMedia
	Document           *DocumentMedia
	PaidMedia          *PaidMedia
	Sticker            *Sticker
	Video              *DocumentMedia
	VideoNote          *DocumentMedia
	Voice              *DocumentMedia
	Checklist          *Checklist
	Contact            *ContactMedia
	Location           *LocationMedia
	Venue              *VenueMedia
	Game               *GameMedia
	Poll               *PollMedia
	Dice               *DiceMedia
	Invoice            *InvoiceMedia
	Giveaway           *GiveawayMedia
	GiveawayWinners    *GiveawayWinners
	GiveawayResults    *GiveawayResultsMedia
	Story              *StoryMedia
}

func parseLinkPreviewOptions(raw tg.MessageMediaClass) *LinkPreviewOptions {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.MessageMediaWebPage:
		return nil
	default:
		_ = v
	}
	return nil
}

func parseAnimationFromDoc(doc *DocumentMedia) *Animation {
	if doc == nil || doc.RawDocument == nil {
		return nil
	}
	return ParseAnimation(doc.RawDocument)
}

func parseStickerFromDoc(doc *DocumentMedia) *Sticker {
	if doc == nil || doc.RawDocument == nil {
		return nil
	}
	return ParseSticker(doc.RawDocument)
}

// ParseGiveawayWinnersFromReply extracts GiveawayWinners from a
// MessageMediaClass when the reply contains giveaway results.
//
// Example:
//
//	winners := types.ParseGiveawayWinnersFromReply(media, pm)
//	if winners != nil {
//		fmt.Printf("Giveaway winners: %d\n", len(winners.Winners))
//	}
func ParseGiveawayWinnersFromReply(raw tg.MessageMediaClass, pm *PeerMap) *GiveawayWinners {
	if raw == nil {
		return nil
	}
	if v, ok := raw.(*tg.MessageMediaGiveawayResults); ok {
		return ParseGiveawayWinners(v, pm)
	}
	return nil
}

// ParseExternalReplyInfo converts a raw Telegram MessageReplyHeader into
// structured ExternalReplyInfo with origin, chat, and optional media.
//
// Example:
//
//	info := types.ParseExternalReplyInfo(replyHeader, pm)
//	fmt.Printf("Reply to message %d in %s\n", info.MessageID, info.Chat.Title)
func ParseExternalReplyInfo(raw *tg.MessageReplyHeader, pm *PeerMap) *ExternalReplyInfo {
	if raw == nil {
		return nil
	}
	info := &ExternalReplyInfo{}
	if raw.ReplyToMsgID != 0 {
		info.MessageID = raw.ReplyToMsgID
	}
	if raw.ReplyFrom != nil {
		info.Origin = parseMessageOrigin(raw.ReplyFrom, pm)
	}
	if raw.ReplyToPeerID != nil {
		info.Chat = ParseChatFromPeer(raw.ReplyToPeerID, pm)
	}
	if raw.ReplyMedia != nil {
		info.HasMediaSpoiler = raw.Quote
		info.LinkPreviewOptions = parseLinkPreviewOptions(raw.ReplyMedia)
		if m := ParseMedia(raw.ReplyMedia); m != nil {
			switch v := m.(type) {
			case *PhotoMedia:
				info.Media = MessageMediaTypePhoto
				info.Photo = v.Photo
			case *DocumentMedia:
				info.Media = v.MediaType()
				info.Document = v
				switch info.Media {
				case MessageMediaTypeAnimation:
					info.Animation = parseAnimationFromDoc(v)
				case MessageMediaTypeAudio:
					info.Audio = v
				case MessageMediaTypeSticker:
					info.Sticker = parseStickerFromDoc(v)
				case MessageMediaTypeVideo:
					info.Video = v
				case MessageMediaTypeVideoNote:
					info.VideoNote = v
				case MessageMediaTypeVoice:
					info.Voice = v
				}
			case *PaidMedia:
				info.Media = MessageMediaTypePaidMedia
				info.PaidMedia = v
			case *ContactMedia:
				info.Media = MessageMediaTypeContact
				info.Contact = v
			case *LocationMedia:
				info.Media = MessageMediaTypeLocation
				info.Location = v
			case *VenueMedia:
				info.Media = MessageMediaTypeVenue
				info.Venue = v
			case *GameMedia:
				info.Media = MessageMediaTypeGame
				info.Game = v
			case *PollMedia:
				info.Media = MessageMediaTypePoll
				info.Poll = v
			case *DiceMedia:
				info.Media = MessageMediaTypeDice
				info.Dice = v
			case *InvoiceMedia:
				info.Media = MessageMediaTypeInvoice
				info.Invoice = v
			case *GiveawayMedia:
				info.Media = MessageMediaTypeGiveaway
				info.Giveaway = v
			case *GiveawayResultsMedia:
				info.Media = MessageMediaTypeGiveawayWinners
				info.GiveawayResults = v
				info.GiveawayWinners = ParseGiveawayWinnersFromReply(raw.ReplyMedia, pm)
			case *StoryMedia:
				info.Media = MessageMediaTypeStory
				info.Story = v
			}
		}
		switch r := raw.ReplyMedia.(type) {
		case *tg.MessageMediaToDo:
			info.Checklist = ParseChecklist(r)
			info.Media = MessageMediaTypeChecklist
		}
	}
	return info
}

// StarAmount describes a possibly non-integer amount of Telegram Stars.
type StarAmount struct {
	Amount int64
	Nanos  int32
}

// ParseStarAmount converts a raw Telegram StarsAmountClass into a StarAmount
// with whole and fractional (nanos) components.
//
// Example:
//
//	sa := types.ParseStarAmount(rawAmount)
//	fmt.Printf("Stars: %d.%09d\n", sa.Amount, sa.Nanos)
func ParseStarAmount(raw tg.StarsAmountClass) *StarAmount {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.StarsAmount:
		return &StarAmount{Amount: v.Amount, Nanos: v.Nanos}
	}
	return nil
}

// StoryView contains information about a story view.
type StoryView struct {
	User                   *User
	Date                   time.Time
	IsBlocked              bool
	IsBlockedMyStoriesFrom bool
	Reaction               *Reaction
}

// ParseStoryView converts a raw Telegram StoryViewClass into a StoryView
// with viewer info, date, and optional reaction.
//
// Example:
//
//	view := types.ParseStoryView(rawView, pm)
//	fmt.Printf("Viewed by user %d at %s\n", view.User.ID, view.Date)
func ParseStoryView(raw tg.StoryViewClass, pm *PeerMap) *StoryView {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.StoryView:
		sv := &StoryView{
			Date:                   time.Unix(int64(v.Date), 0),
			IsBlocked:              v.Blocked,
			IsBlockedMyStoriesFrom: v.BlockedMyStoriesFrom,
		}
		if pm != nil {
			sv.User = getUserFromPM(pm, v.UserID)
		} else {
			sv.User = &User{ID: v.UserID}
		}
		if v.Reaction != nil {
			sv.Reaction = ParseReaction(v.Reaction)
		}
		return sv
	}
	return nil
}

// ChatBoost contains info about boosts applied by a user.
type ChatBoost struct {
	ID                string
	User              *User
	Date              time.Time
	ExpireDate        time.Time
	Multiplier        int32
	IsGift            bool
	IsGiveaway        bool
	IsUnclaimed       bool
	GiveawayMessageID int32
	UsedGiftSlug      string
	StarCount         int64
}

// ParseChatBoost converts a raw Telegram Boost into a ChatBoost with user,
// dates, multiplier, and giveaway details.
//
// Example:
//
//	boost := types.ParseChatBoost(rawBoost, pm)
//	fmt.Printf("Boost %s: multiplier=%d gift=%v\n", boost.ID, boost.Multiplier, boost.IsGift)
func ParseChatBoost(raw *tg.Boost, pm *PeerMap) *ChatBoost {
	if raw == nil {
		return nil
	}
	b := &ChatBoost{
		ID:          raw.ID,
		Date:        time.Unix(int64(raw.Date), 0),
		Multiplier:  raw.Multiplier,
		IsGift:      raw.Gift,
		IsGiveaway:  raw.Giveaway,
		IsUnclaimed: raw.Unclaimed,
		StarCount:   raw.Stars,
	}
	if raw.Expires != 0 {
		b.ExpireDate = time.Unix(int64(raw.Expires), 0)
	}
	if raw.UserID != 0 && pm != nil {
		b.User = getUserFromPM(pm, raw.UserID)
	} else if raw.UserID != 0 {
		b.User = &User{ID: raw.UserID}
	}
	if raw.GiveawayMsgID != 0 {
		b.GiveawayMessageID = raw.GiveawayMsgID
	}
	if raw.UsedGiftSlug != "" {
		b.UsedGiftSlug = raw.UsedGiftSlug
	}
	return b
}

// BoostsStatus contains info about boost status of a chat.
type BoostsStatus struct {
	Level              int32
	CurrentLevelBoosts int32
	Boosts             int32
	BoostURL           string
	MyBoost            bool
	GiftBoosts         int32
	NextLevelBoosts    int32
	MyBoostSlots       []int32
}

// ParseBoostsStatus converts a raw Telegram PremiumBoostsStatus into a
// BoostsStatus with level, boost counts, and the boost URL.
//
// Example:
//
//	status := types.ParseBoostsStatus(rawStatus)
//	fmt.Printf("Level %d: %d/%d boosts\n", status.Level, status.Boosts, status.NextLevelBoosts)
func ParseBoostsStatus(raw *tg.PremiumBoostsStatus) *BoostsStatus {
	if raw == nil {
		return nil
	}
	bs := &BoostsStatus{
		Level:              raw.Level,
		CurrentLevelBoosts: raw.CurrentLevelBoosts,
		Boosts:             raw.Boosts,
		MyBoost:            raw.MyBoost,
		BoostURL:           raw.BoostURL,
		MyBoostSlots:       raw.MyBoostSlots,
	}
	if raw.NextLevelBoosts != 0 {
		bs.NextLevelBoosts = raw.NextLevelBoosts
	}
	if raw.GiftBoosts != 0 {
		bs.GiftBoosts = raw.GiftBoosts
	}
	return bs
}

// ChatReactions describes available reactions in a chat.
type ChatReactions struct {
	AllAreEnabled    bool
	AllowCustomEmoji bool
	Reactions        []Reaction
}

// ParseChatReactions converts a raw Telegram ChatReactionsClass into a
// ChatReactions describing whether all, none, or specific reactions are enabled.
//
// Example:
//
//	cr := types.ParseChatReactions(rawReactions)
//	if cr.AllAreEnabled {
//		fmt.Println("All reactions enabled")
//	}
func ParseChatReactions(raw tg.ChatReactionsClass) *ChatReactions {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.ChatReactionsAll:
		return &ChatReactions{AllAreEnabled: true, AllowCustomEmoji: v.AllowCustom}
	case *tg.ChatReactionsNone:
		return &ChatReactions{}
	case *tg.ChatReactionsSome:
		cr := &ChatReactions{}
		for _, r := range v.Reactions {
			if e, ok := r.(*tg.ReactionEmoji); ok {
				cr.Reactions = append(cr.Reactions, Reaction{Emoji: e.Emoticon})
			}
			if c, ok := r.(*tg.ReactionCustomEmoji); ok {
				cr.Reactions = append(cr.Reactions, Reaction{CustomEmojiID: fmt.Sprintf("%d", c.DocumentID)})
			}
		}
		return cr
	}
	return nil
}

// GroupCallMember contains information about a group call participant.
type GroupCallMember struct {
	Chat                   *Chat
	Date                   time.Time
	ActiveDate             time.Time
	Volume                 int32
	CanSelfUnmute          bool
	IsMuted                bool
	IsLeft                 bool
	IsJustJoined           bool
	IsMutedByYou           bool
	IsVolumeByAdmin        bool
	IsSelf                 bool
	IsVideoJoined          bool
	IsHandRaised           bool
	IsVideoEnabled         bool
	IsScreenSharingEnabled bool
}

// ParseGroupCallMember converts a raw Telegram GroupCallParticipant into a
// GroupCallMember with chat, mute state, and participation details.
//
// Example:
//
//	member := types.ParseGroupCallMember(rawParticipant)
//	fmt.Printf("Member muted=%v joined=%v\n", member.IsMuted, member.IsJustJoined)
func ParseGroupCallMember(raw *tg.GroupCallParticipant) *GroupCallMember {
	if raw == nil {
		return nil
	}
	m := &GroupCallMember{
		Date:            time.Unix(int64(raw.Date), 0),
		IsMuted:         raw.Muted,
		IsLeft:          raw.Left,
		CanSelfUnmute:   raw.CanSelfUnmute,
		IsJustJoined:    raw.JustJoined,
		IsMutedByYou:    raw.MutedByYou,
		IsVolumeByAdmin: raw.VolumeByAdmin,
		IsSelf:          raw.Self,
		IsVideoJoined:   raw.VideoJoined,
	}
	if raw.Peer != nil {
		m.Chat = ParseChatFromPeer(raw.Peer, nil)
	}
	if raw.ActiveDate != 0 {
		m.ActiveDate = time.Unix(int64(raw.ActiveDate), 0)
	}
	if raw.Volume != 0 {
		m.Volume = raw.Volume
	}
	if raw.RaiseHandRating != 0 {
		m.IsHandRaised = true
	}
	if raw.Video != nil {
		m.IsVideoEnabled = true
	}
	if raw.Presentation != nil {
		m.IsScreenSharingEnabled = true
	}
	return m
}

// ChatBackground describes a background set for a specific chat.
type ChatBackground struct {
	ID                    int64
	WallpaperDocID        int64
	Document              *DocumentMedia
	IsOnlyForSelf         bool
	BackgroundColor       int32
	SecondBackgroundColor int32
	ThirdBackgroundColor  int32
	FourthBackgroundColor int32
	Intensity             int32
	RotationAngle         int32
	Emoji                 string
}

// ChatTheme describes a chat theme.
type ChatTheme struct {
	Emoticon string
}

// ParseChatTheme converts a raw Telegram ChatTheme into a ChatTheme
// containing the emoticon identifier for the theme.
//
// Example:
//
//	theme := types.ParseChatTheme(rawTheme)
//	fmt.Printf("Chat theme emoticon: %s\n", theme.Emoticon)
func ParseChatTheme(raw *tg.ChatTheme) *ChatTheme {
	if raw == nil {
		return nil
	}
	return &ChatTheme{
		Emoticon: raw.Emoticon,
	}
}
