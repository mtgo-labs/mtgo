package types

// InputMediaLivePhoto references a live photo file for uploading.
type InputMediaLivePhoto struct {
	FileID     int64
	AccessHash int64
}

// InputPollMedia represents a poll as a media attachment.
type InputPollMedia struct {
	Type string
}

// InputPollOptionMedia represents a single poll option as a media attachment.
type InputPollOptionMedia struct {
	Type string
}
