package types

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
