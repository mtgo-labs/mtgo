package types

// MessageMediaType identifies the kind of media attached to a message,
// such as photo, video, document, sticker, or poll.
type MessageMediaType string

const (
	// MessageMediaTypeUnsupported represents a media type not recognized by this library.
	MessageMediaTypeUnsupported MessageMediaType = "unsupported"
	// MessageMediaTypeAudio is an audio file (MP3, FLAC, etc.).
	MessageMediaTypeAudio MessageMediaType = "audio"
	// MessageMediaTypeDocument is a generic file attachment.
	MessageMediaTypeDocument MessageMediaType = "document"
	// MessageMediaTypePhoto is a photo image.
	MessageMediaTypePhoto MessageMediaType = "photo"
	// MessageMediaTypeSticker is a static, animated, or video sticker.
	MessageMediaTypeSticker MessageMediaType = "sticker"
	// MessageMediaTypeVideo is a video file.
	MessageMediaTypeVideo MessageMediaType = "video"
	// MessageMediaTypeAnimation is an animated GIF or MPEG4 animation.
	MessageMediaTypeAnimation MessageMediaType = "animation"
	// MessageMediaTypeVoice is a voice message.
	MessageMediaTypeVoice MessageMediaType = "voice"
	// MessageMediaTypeVideoNote is a round video message.
	MessageMediaTypeVideoNote MessageMediaType = "video_note"
	// MessageMediaTypeContact is a shared contact.
	MessageMediaTypeContact MessageMediaType = "contact"
	// MessageMediaTypeLocation is a geographic location.
	MessageMediaTypeLocation MessageMediaType = "location"
	// MessageMediaTypeVenue is a named venue with address.
	MessageMediaTypeVenue MessageMediaType = "venue"
	// MessageMediaTypePoll is an interactive poll.
	MessageMediaTypePoll MessageMediaType = "poll"
	// MessageMediaTypeWebPage is a webpage link preview.
	MessageMediaTypeWebPage MessageMediaType = "web_page"
	// MessageMediaTypeDice is a dice roll result.
	MessageMediaTypeDice MessageMediaType = "dice"
	// MessageMediaTypeGame is a Telegram game.
	MessageMediaTypeGame MessageMediaType = "game"
	// MessageMediaTypeGiveaway is a giveaway announcement.
	MessageMediaTypeGiveaway MessageMediaType = "giveaway"
	// MessageMediaTypeGiveawayWinners lists the winners of a completed giveaway.
	MessageMediaTypeGiveawayWinners MessageMediaType = "giveaway_winners"
	// MessageMediaTypeStory is a forwarded story.
	MessageMediaTypeStory MessageMediaType = "story"
	// MessageMediaTypeInvoice is a payment invoice.
	MessageMediaTypeInvoice MessageMediaType = "invoice"
	// MessageMediaTypePaidMedia is media behind a paywall requiring Telegram Stars.
	MessageMediaTypePaidMedia MessageMediaType = "paid_media"
	// MessageMediaTypeChecklist is a task checklist.
	MessageMediaTypeChecklist MessageMediaType = "checklist"
)

// String returns the string representation of the MessageMediaType.
func (m MessageMediaType) String() string { return string(m) }
