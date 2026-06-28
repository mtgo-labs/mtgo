package telegram

import "github.com/mtgo-labs/mtgo/telegram/params"

const (
	// uploadPartSize is the size in bytes of each individual part sent during file upload.
	// Telegram requires all parts except the last to be exactly this size (512 KB).
	uploadPartSize = 524288

	// bigFileThreshold is the file size in bytes above which the big-file upload path is used.
	// Files at or above this size are uploaded via UploadSaveBigFilePart and do not require
	// an MD5 hash. Set to 10 MB.
	bigFileThreshold = 10 << 20

	// downloadChunkSize is the default size in bytes of each chunk requested during file download.
	// Set to 1 MB; can be overridden via params.Download.ChunkSize.
	downloadChunkSize = 1 << 20

	// maxFileSize is the maximum allowed file size in bytes for uploads.
	// Set to 2 GB, which is the Telegram Bot API / MTProto limit for file uploads.
	maxFileSize = 2 << 30

	defaultTransferWorkers = 4
	maxTransferWorkers     = 8
)

// UploadOptions configures optional parameters for file upload operations.
// Pass a nil pointer to accept all defaults (automatic workers, no progress reporting).
type UploadOptions struct {
	// Workers controls the number of concurrent goroutines used to upload file parts.
	// Values greater than 8 are clamped to 8. A value of 0 or less lets the client choose
	// a sensible default for known-size files. Set Workers to 1 to force serial upload.
	Workers int

	// Progress is an optional callback invoked after each part is successfully uploaded.
	// It receives a ProgressInfo with the cumulative uploaded byte count.
	// Set to nil to disable progress reporting.
	Progress params.ProgressFunc

	// FileName overrides the file name reported during upload.
	// If empty, the fileName argument passed to the upload function is used instead.
	FileName string
}
