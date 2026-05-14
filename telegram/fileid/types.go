package fileid

// FileType enumerates the kinds of files Telegram stores, matching the numeric
// type codes used in the binary file_id encoding.
type FileType byte

const (
	// FileTypeThumbnail represents a thumbnail or preview image.
	FileTypeThumbnail      FileType = 0
	// FileTypePhoto represents a full-size photo.
	FileTypePhoto          FileType = 1
	// FileTypeVoice represents a voice message.
	FileTypeVoice          FileType = 2
	// FileTypeVideo represents a video file.
	FileTypeVideo          FileType = 3
	// FileTypeDocument represents a generic document file.
	FileTypeDocument       FileType = 4
	// FileTypeEncrypted represents an end-to-end encrypted file.
	FileTypeEncrypted      FileType = 5
	// FileTypeTemp represents a temporary file.
	FileTypeTemp           FileType = 6
	// FileTypeSticker represents a sticker image.
	FileTypeSticker        FileType = 7
	// FileTypeAudio represents an audio file.
	FileTypeAudio          FileType = 8
	// FileTypeAnimation represents a GIF or other animation.
	FileTypeAnimation      FileType = 9
	// FileTypeVideoNote represents a round video note.
	FileTypeVideoNote      FileType = 10
	// FileTypeSecureRaw represents a raw secure file upload.
	FileTypeSecureRaw      FileType = 11
	// FileTypeSecureDocument represents a processed secure document.
	FileTypeSecureDocument FileType = 12
	// FileTypeBackground represents a chat background image.
	FileTypeBackground     FileType = 13
	// FileTypeDocumentPhoto represents a photo sent as a document.
	FileTypeDocumentPhoto  FileType = 14
)

// IsPhoto reports whether the FileType represents a photo-like file that
// includes volume and source metadata in the file_id encoding.
func (ft FileType) IsPhoto() bool {
	return ft == FileTypePhoto || ft == FileTypeThumbnail || ft == FileTypeDocumentPhoto
}

// ThumbnailSource enumerates the possible origins of a photo-size thumbnail,
// used in the Source field of PhotoSizeSource to determine which fields are
// populated.
type ThumbnailSource byte

const (
	// ThumbnailSourceLegacy represents a legacy thumbnail sourced by secret and local_id.
	ThumbnailSourceLegacy           ThumbnailSource = 0
	// ThumbnailSourceThumbnail represents a standard thumbnail with explicit file type and size.
	ThumbnailSourceThumbnail        ThumbnailSource = 1
	// ThumbnailSourceDialogPhotoSmall represents a small dialog (chat/user) photo.
	ThumbnailSourceDialogPhotoSmall ThumbnailSource = 2
	// ThumbnailSourceDialogPhotoBig represents a big dialog (chat/user) photo.
	ThumbnailSourceDialogPhotoBig   ThumbnailSource = 3
	// ThumbnailSourceStickerSetThumb represents a thumbnail for a sticker set.
	ThumbnailSourceStickerSetThumb  ThumbnailSource = 4
)

// FileUniqueType enumerates the kinds of unique file identifiers used by
// Telegram to distinguish files independently of file_id rotation.
type FileUniqueType byte

const (
	// FileUniqueTypeWeb represents a file unique to a web location.
	FileUniqueTypeWeb       FileUniqueType = 0
	// FileUniqueTypePhoto represents a file unique to a photo.
	FileUniqueTypePhoto     FileUniqueType = 1
	// FileUniqueTypeDocument represents a file unique to a document.
	FileUniqueTypeDocument  FileUniqueType = 2
	// FileUniqueTypeSecure represents a file unique to a secure upload.
	FileUniqueTypeSecure    FileUniqueType = 3
	// FileUniqueTypeEncrypted represents a file unique to an encrypted upload.
	FileUniqueTypeEncrypted FileUniqueType = 4
	// FileUniqueTypeTemp represents a file unique to a temporary upload.
	FileUniqueTypeTemp      FileUniqueType = 5
)
