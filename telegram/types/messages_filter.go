package types

// MessagesFilter specifies a category filter when searching or listing messages,
// such as only photos, documents, or mentions. Used to narrow search results to
// a specific media or message type.
type MessagesFilter string

const (
	// MessagesFilterEmpty returns all message types without filtering.
	MessagesFilterEmpty MessagesFilter = "empty"
	// MessagesFilterPhoto returns only photo messages.
	MessagesFilterPhoto MessagesFilter = "photo"
	// MessagesFilterVideo returns only video messages.
	MessagesFilterVideo MessagesFilter = "video"
	// MessagesFilterPhotoVideo returns photos and videos combined.
	MessagesFilterPhotoVideo MessagesFilter = "photo_video"
	// MessagesFilterDocument returns only document (file) messages.
	MessagesFilterDocument MessagesFilter = "document"
	// MessagesFilterURL returns messages containing a URL.
	MessagesFilterURL MessagesFilter = "url"
	// MessagesFilterAnimation returns only animation (GIF) messages.
	MessagesFilterAnimation MessagesFilter = "animation"
	// MessagesFilterVoiceNote returns only voice message messages.
	MessagesFilterVoiceNote MessagesFilter = "voice_note"
	// MessagesFilterVideoNote returns only round video message messages.
	MessagesFilterVideoNote MessagesFilter = "video_note"
	// MessagesFilterAudioVideoNote returns voice and video notes combined.
	MessagesFilterAudioVideoNote MessagesFilter = "audio_video_note"
	// MessagesFilterAudio returns only audio file messages.
	MessagesFilterAudio MessagesFilter = "audio"
	// MessagesFilterChatPhoto returns chat photo change service messages.
	MessagesFilterChatPhoto MessagesFilter = "chat_photo"
	// MessagesFilterPhoneCall returns phone call service messages.
	MessagesFilterPhoneCall MessagesFilter = "phone_call"
	// MessagesFilterMention returns messages that mention the current user.
	MessagesFilterMention MessagesFilter = "mention"
	// MessagesFilterLocation returns location and venue messages.
	MessagesFilterLocation MessagesFilter = "location"
	// MessagesFilterContact returns contact sharing messages.
	MessagesFilterContact MessagesFilter = "contact"
	// MessagesFilterPinned returns pinned messages only.
	MessagesFilterPinned MessagesFilter = "pinned"
	// MessagesFilterPoll returns poll messages only.
	MessagesFilterPoll MessagesFilter = "poll"
)

// String returns the string representation of the MessagesFilter.
func (m MessagesFilter) String() string { return string(m) }
