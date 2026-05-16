package types

import "github.com/mtgo-labs/mtgo/tg"

// InlineQueryResult represents a generic inline query result returned to the user.
// Displayed in the inline mode suggestion list when the user types a query to a bot.
type InlineQueryResult struct {
	// Type is the result type identifier (e.g. "photo", "article", "gif").
	Type string
	// ID is the unique identifier for this inline result within the query session.
	ID string
	// Title is the display title of the result shown to the user.
	Title string
	// Description is a short text description shown below the title.
	Description string
	// URL is the URL of the result, opened when the result is selected.
	URL string
	// ThumbURL is the URL of the thumbnail image for the result.
	ThumbURL string
	// ThumbWidth is the width of the thumbnail in pixels.
	ThumbWidth int32
	// ThumbHeight is the height of the thumbnail in pixels.
	ThumbHeight int32
}

// ParseInlineQueryResult converts a TL InputBotInlineResultClass into an InlineQueryResult.
// Returns nil if raw is nil.
func ParseInlineQueryResult(raw tg.InputBotInlineResultClass) *InlineQueryResult {
	if raw == nil {
		return nil
	}
	r := &InlineQueryResult{}
	switch v := raw.(type) {
	case *tg.InputBotInlineResult:
		r.ID = v.ID
		r.Type = v.Type
		if v.Title != "" {
			r.Title = v.Title
		}
		if v.Description != "" {
			r.Description = v.Description
		}
		if v.URL != "" {
			r.URL = v.URL
		}
		if v.Thumb != nil {
			r.ThumbURL = v.Thumb.URL
		}
		if v.Content != nil {
			r.URL = v.Content.URL
		}
	case *tg.InputBotInlineResultPhoto:
		r.ID = v.ID
		r.Type = v.Type
	case *tg.InputBotInlineResultDocument:
		r.ID = v.ID
		r.Type = v.Type
		if v.Title != "" {
			r.Title = v.Title
		}
		if v.Description != "" {
			r.Description = v.Description
		}
	case *tg.InputBotInlineResultGame:
		r.ID = v.ID
		r.Type = "game"
	}
	return r
}

// InlineQueryResultArticle is an inline result representing an article.
type InlineQueryResultArticle struct {
	ID          string
	Title       string
	Description string
	URL         string
	ThumbURL    string
	HideURL     bool
}

// InlineQueryResultPhoto is an inline result representing a photo.
type InlineQueryResultPhoto struct {
	ID          string
	Title       string
	Description string
	ThumbURL    string
	PhotoURL    string
	Width       int32
	Height      int32
}

// InlineQueryResultAnimation is an inline result representing an animation/GIF.
type InlineQueryResultAnimation struct {
	ID       string
	Title    string
	ThumbURL string
	GifURL   string
	Width    int32
	Height   int32
	Duration int32
}

// InlineQueryResultVideo is an inline result representing a video.
type InlineQueryResultVideo struct {
	ID          string
	Title       string
	Description string
	ThumbURL    string
	VideoURL    string
	MimeType    string
	Width       int32
	Height      int32
	Duration    int32
}

// InlineQueryResultAudio is an inline result representing an audio file.
type InlineQueryResultAudio struct {
	ID        string
	Title     string
	Performer string
	AudioURL  string
	Duration  int32
}

// InlineQueryResultVoice is an inline result representing a voice message.
type InlineQueryResultVoice struct {
	ID       string
	Title    string
	VoiceURL string
	Duration int32
}

// InlineQueryResultDocument is an inline result representing a document.
type InlineQueryResultDocument struct {
	ID          string
	Title       string
	Description string
	ThumbURL    string
	DocumentURL string
	MimeType    string
	Size        int32
}

// InlineQueryResultLocation is an inline result representing a location.
type InlineQueryResultLocation struct {
	ID        string
	Title     string
	Latitude  float64
	Longitude float64
	ThumbURL  string
}

// InlineQueryResultVenue is an inline result representing a venue.
type InlineQueryResultVenue struct {
	ID        string
	Title     string
	Address   string
	Latitude  float64
	Longitude float64
	ThumbURL  string
}

// InlineQueryResultContact is an inline result representing a contact.
type InlineQueryResultContact struct {
	ID          string
	FirstName   string
	LastName    string
	PhoneNumber string
	ThumbURL    string
}

// InlineQueryResultCachedPhoto is a cached photo inline result.
type InlineQueryResultCachedPhoto struct {
	ID          string
	Title       string
	Description string
	FileID      string
}

// InlineQueryResultCachedAnimation is a cached animation inline result.
type InlineQueryResultCachedAnimation struct {
	ID     string
	Title  string
	FileID string
}

// InlineQueryResultCachedVideo is a cached video inline result.
type InlineQueryResultCachedVideo struct {
	ID          string
	Title       string
	Description string
	FileID      string
}

// InlineQueryResultCachedAudio is a cached audio inline result.
type InlineQueryResultCachedAudio struct {
	ID        string
	Title     string
	Performer string
	FileID    string
}

// InlineQueryResultCachedVoice is a cached voice inline result.
type InlineQueryResultCachedVoice struct {
	ID     string
	Title  string
	FileID string
}

// InlineQueryResultCachedDocument is a cached document inline result.
type InlineQueryResultCachedDocument struct {
	ID          string
	Title       string
	Description string
	FileID      string
	MimeType    string
}

// InlineQueryResultCachedSticker is a cached sticker inline result.
type InlineQueryResultCachedSticker struct {
	ID     string
	Title  string
	FileID string
}

// InlineQueryResultType identifies the type of an inline query result.
type InlineQueryResultType string

const (
	InlineResultTypeArticle  InlineQueryResultType = "article"
	InlineResultTypePhoto    InlineQueryResultType = "photo"
	InlineResultTypeGif      InlineQueryResultType = "gif"
	InlineResultTypeVideo    InlineQueryResultType = "video"
	InlineResultTypeAudio    InlineQueryResultType = "audio"
	InlineResultTypeVoice    InlineQueryResultType = "voice"
	InlineResultTypeDocument InlineQueryResultType = "document"
	InlineResultTypeLocation InlineQueryResultType = "location"
	InlineResultTypeVenue    InlineQueryResultType = "venue"
	InlineResultTypeContact  InlineQueryResultType = "contact"
	InlineResultTypeSticker  InlineQueryResultType = "sticker"
)
