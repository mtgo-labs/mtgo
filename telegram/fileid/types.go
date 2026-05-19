package fileid

// FileType enumerates the kinds of files Telegram stores, matching the numeric
// type codes used in the binary file_id encoding.
type FileType uint32

const (
	// FileTypeThumbnail represents a thumbnail or preview image.
	FileTypeThumbnail FileType = iota
	// FileTypeProfilePhoto represents a profile photo.
	FileTypeProfilePhoto
	// FileTypePhoto represents a full-size photo.
	FileTypePhoto
	// FileTypeVoice represents a voice message.
	FileTypeVoice
	// FileTypeVideo represents a video file.
	FileTypeVideo
	// FileTypeDocument represents a generic document file.
	FileTypeDocument
	// FileTypeEncrypted represents an end-to-end encrypted file.
	FileTypeEncrypted
	// FileTypeTemp represents a temporary file.
	FileTypeTemp
	// FileTypeSticker represents a sticker image.
	FileTypeSticker
	// FileTypeAudio represents an audio file.
	FileTypeAudio
	// FileTypeAnimation represents a GIF or other animation.
	FileTypeAnimation
	// FileTypeEncryptedThumbnail represents an encrypted thumbnail.
	FileTypeEncryptedThumbnail
	// FileTypeWallpaper represents a wallpaper.
	FileTypeWallpaper
	// FileTypeVideoNote represents a round video note.
	FileTypeVideoNote
	// FileTypeSecureRaw represents a raw secure file upload.
	FileTypeSecureRaw
	// FileTypeSecureDocument represents a processed secure document.
	FileTypeSecureDocument
	// FileTypeBackground represents a chat background image.
	FileTypeBackground
	// FileTypeDocumentPhoto represents a photo sent as a document.
	FileTypeDocumentPhoto
)

// IsPhoto reports whether the FileType represents a photo-like file that
// includes volume and source metadata in the file_id encoding.
func (ft FileType) IsPhoto() bool {
	return ft == FileTypePhoto || ft == FileTypeThumbnail || ft == FileTypeProfilePhoto
}

// ThumbnailSource enumerates the possible origins of a photo-size thumbnail,
// used in the Source field of PhotoSizeSource to determine which fields are
// populated.
type ThumbnailSource byte

const (
	// ThumbnailSourceLegacy represents a legacy thumbnail sourced by secret and local_id.
	ThumbnailSourceLegacy ThumbnailSource = 0
	// ThumbnailSourceThumbnail represents a standard thumbnail with explicit file type and size.
	ThumbnailSourceThumbnail ThumbnailSource = 1
	// ThumbnailSourceDialogPhotoSmall represents a small dialog (chat/user) photo.
	ThumbnailSourceDialogPhotoSmall ThumbnailSource = 2
	// ThumbnailSourceDialogPhotoBig represents a big dialog (chat/user) photo.
	ThumbnailSourceDialogPhotoBig ThumbnailSource = 3
	// ThumbnailSourceStickerSetThumb represents a thumbnail for a sticker set.
	ThumbnailSourceStickerSetThumb ThumbnailSource = 4
)

// FileUniqueType enumerates the kinds of unique file identifiers used by
// Telegram to distinguish files independently of file_id rotation.
type FileUniqueType byte

const (
	// FileUniqueTypeWeb represents a file unique to a web location.
	FileUniqueTypeWeb FileUniqueType = 0
	// FileUniqueTypePhoto represents a file unique to a photo.
	FileUniqueTypePhoto FileUniqueType = 1
	// FileUniqueTypeDocument represents a file unique to a document.
	FileUniqueTypeDocument FileUniqueType = 2
	// FileUniqueTypeSecure represents a file unique to a secure upload.
	FileUniqueTypeSecure FileUniqueType = 3
	// FileUniqueTypeEncrypted represents a file unique to an encrypted upload.
	FileUniqueTypeEncrypted FileUniqueType = 4
	// FileUniqueTypeTemp represents a file unique to a temporary upload.
	FileUniqueTypeTemp FileUniqueType = 5
)
