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
	// Type classifies the kind of interactive area.
	Type MediaAreaType
	// X is the horizontal position as a percentage of the media width (0–100).
	X float64
	// Y is the vertical position as a percentage of the media height (0–100).
	Y float64
	// W is the width as a percentage of the media width (0–100).
	W float64
	// H is the height as a percentage of the media height (0–100).
	H float64
	// Rotation is the clockwise rotation angle in degrees.
	Rotation float64
	// Color is the 24-bit RGB fill color of the area, or zero for no fill.
	Color int32
}
