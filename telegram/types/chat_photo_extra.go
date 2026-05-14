package types

// InputChatPhotoPrevious references a previously used chat photo.
type InputChatPhotoPrevious struct {
	PhotoID int64
}

// InputChatPhotoStatic references a static image file for a chat photo.
type InputChatPhotoStatic struct {
	FileID     int64
	AccessHash int64
}

// InputChatPhotoAnimation references an animated file for a chat photo.
type InputChatPhotoAnimation struct {
	FileID     int64
	AccessHash int64
}
