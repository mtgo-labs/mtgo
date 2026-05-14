package types

// ChatAction represents a "typing" or activity indicator shown in a chat.
type ChatAction string

// Chat action types indicating the current activity of a user or bot.
const (
	ChatActionTyping          ChatAction = "typing"
	ChatActionUploadPhoto     ChatAction = "upload_photo"
	ChatActionRecordVideo     ChatAction = "record_video"
	ChatActionUploadVideo     ChatAction = "upload_video"
	ChatActionRecordAudio     ChatAction = "record_audio"
	ChatActionUploadAudio     ChatAction = "upload_audio"
	ChatActionUploadDocument  ChatAction = "upload_document"
	ChatActionFindLocation    ChatAction = "find_location"
	ChatActionRecordVideoNote ChatAction = "record_video_note"
	ChatActionUploadVideoNote ChatAction = "upload_video_note"
	ChatActionPlaying         ChatAction = "playing"
	ChatActionChooseContact   ChatAction = "choose_contact"
	ChatActionSpeaking        ChatAction = "speaking"
	ChatActionImportHistory   ChatAction = "import_history"
	ChatActionChooseSticker   ChatAction = "choose_sticker"
	ChatActionCancel          ChatAction = "cancel"
)

// String returns the string representation of the ChatAction.
func (c ChatAction) String() string { return string(c) }
