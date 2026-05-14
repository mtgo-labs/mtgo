package telegram

import (
	"path/filepath"
	"strings"
)

var mimeMap = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".bmp":  "image/bmp",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".ico":  "image/x-icon",
	".mp4":  "video/mp4",
	".mov":  "video/quicktime",
	".avi":  "video/x-msvideo",
	".mkv":  "video/x-matroska",
	".webm": "video/webm",
	".flv":  "video/x-flv",
	".wmv":  "video/x-ms-wmv",
	".3gp":  "video/3gpp",
	".mp3":  "audio/mpeg",
	".flac": "audio/flac",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".aac":  "audio/aac",
	".m4a":  "audio/mp4",
	".wma":  "audio/x-ms-wma",
	".opus": "audio/opus",
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".zip":  "application/zip",
	".gz":   "application/gzip",
	".tar":  "application/x-tar",
	".rar":  "application/vnd.rar",
	".7z":   "application/x-7z-compressed",
	".bz2":  "application/x-bzip2",
	".txt":  "text/plain",
	".html": "text/html",
	".htm":  "text/html",
	".css":  "text/css",
	".js":   "application/javascript",
	".json": "application/json",
	".xml":  "application/xml",
	".apk":  "application/vnd.android.package-archive",
	".stl":  "model/stl",
}

var extMap map[string]string

func init() {
	extMap = make(map[string]string, len(mimeMap))
	for ext, mime := range mimeMap {
		if existing, exists := extMap[mime]; !exists || len(ext) > len(existing) {
			extMap[mime] = ext
		}
	}
}

// GuessMIMEType returns the MIME type for the given filename based on its extension.
// Returns "application/octet-stream" when the extension is unrecognized.
func GuessMIMEType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if m, ok := mimeMap[ext]; ok {
		return m
	}
	return "application/octet-stream"
}

// GuessExtension returns the canonical file extension (e.g. ".jpg") for the given MIME
// type. Returns an empty string when the MIME type is unrecognized.
func GuessExtension(mime string) string {
	mime = strings.ToLower(mime)
	if ext, ok := extMap[mime]; ok {
		return ext
	}
	return ""
}
