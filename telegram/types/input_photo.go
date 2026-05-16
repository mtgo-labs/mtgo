package types

// InputChatPhoto specifies a photo to use as a chat or profile photo.
type InputChatPhoto struct {
	// Type identifies the kind of chat photo input.
	//   0 = Empty (remove photo)
	//   1 = Static (upload a new photo)
	//   2 = Animation (upload an animated photo)
	//   3 = Previous (reuse a previous photo)
	Type int
	// FileID is the uploaded file ID for static or animation types, or zero.
	FileID int64
	// PhotoID is the existing photo ID for the previous type, or zero.
	PhotoID int64
}

// MediaArea represents an interactive area positioned on a media item such as a story.
type MediaArea struct {
	Type        MediaAreaType
	X           float64
	Y           float64
	Width       float64
	Height      float64
	Rotation    float64
	Radius      float64
	Color       int32
	SenderChat  *Chat
	MessageID   int32
	Message     *Message
	Location    *LocationMedia
	Reaction    *Reaction
	IsDark      bool
	IsFlipped   bool
	URL         string
	Emoji       string
	Temperature float64
	Gift        *Gift
}
