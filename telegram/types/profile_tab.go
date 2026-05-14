package types

// ProfileTab represents a content tab displayed on a user or chat profile page.
// Use this to navigate between different media types when browsing profile
// content.
type ProfileTab string

const (
	// ProfileTabMedia displays photos and videos shared in the chat.
	ProfileTabMedia ProfileTab = "media"
	// ProfileTabMedia displays shared links and URLs.
	ProfileTabLinks ProfileTab = "links"
	// ProfileTabAudio displays audio files and voice messages shared in the chat.
	ProfileTabAudio ProfileTab = "audio"
	// ProfileTabVoice displays voice messages specifically.
	ProfileTabVoice ProfileTab = "voice"
)

// String returns the string representation of the ProfileTab.
func (p ProfileTab) String() string { return string(p) }
