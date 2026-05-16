package types

import (
	"io"

	"github.com/mtgo-labs/mtgo/telegram/fileid"
)

// InputFile represents a file reference that can be resolved from a file ID string,
// URL, local path, byte slice, or reader. Used when uploading or referencing media.
//
// Example:
//
//	f := types.FileID("AAQBAQMDBAABq64Q4A")
//	f2 := types.Path("/tmp/photo.jpg")
//	f3 := types.FromBytes([]byte{...}, "data.bin")
type InputFile struct {
	fileID     string
	url        string
	path       string
	reader     io.ReadSeeker
	fileName   string
	fileSize   int64
	ID         int64
	AccessHash int64
	FileRef    []byte
	fileType   fileid.FileType
}

// FileID creates an InputFile from a previously generated file ID string.
//
// Example:
//
//	f := types.FileID("AAQBAQMDBAABq64Q4A")
func FileID(s string) *InputFile {
	return &InputFile{fileID: s}
}

// FromIDs creates an InputFile from numeric document ID, access hash, and file reference.
//
// Example:
//
//	f := types.FromIDs(docID, accessHash, fileRef)
func FromIDs(ID, accessHash int64, fileRef []byte) *InputFile {
	return &InputFile{ID: ID, AccessHash: accessHash, FileRef: fileRef}
}

// URL creates an InputFile that will be downloaded from the given URL.
//
// Example:
//
//	f := types.URL("https://example.com/image.jpg")
func URL(u string) *InputFile {
	return &InputFile{url: u}
}

// Path creates an InputFile that will be uploaded from the given local file path.
//
// Example:
//
//	f := types.Path("/tmp/photo.jpg")
func Path(p string) *InputFile {
	return &InputFile{path: p}
}

// Reader creates an InputFile from an io.ReadSeeker with a filename and size hint.
//
// Example:
//
//	f := types.Reader(bytes.NewReader(data), "file.txt", int64(len(data)))
func Reader(r io.ReadSeeker, fileName string, size int64) *InputFile {
	return &InputFile{reader: r, fileName: fileName, fileSize: size}
}

// FromBytes creates an InputFile from a raw byte slice and filename.
//
// Example:
//
//	f := types.FromBytes([]byte{0x89, 0x50, 0x4e, 0x47}, "image.png")
func FromBytes(data []byte, fileName string) *InputFile {
	return &InputFile{
		reader:   io.NewSectionReader(&bytesAt{data}, 0, int64(len(data))),
		fileName: fileName,
		fileSize: int64(len(data)),
	}
}

type bytesAt struct{ data []byte }

func (b *bytesAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n := copy(p, b.data[off:])
	return n, nil
}

func (f *InputFile) GetFileID() string            { return f.fileID }
func (f *InputFile) GetURL() string               { return f.url }
func (f *InputFile) GetPath() string              { return f.path }
func (f *InputFile) GetReader() io.ReadSeeker     { return f.reader }
func (f *InputFile) GetFileName() string          { return f.fileName }
func (f *InputFile) GetFileSize() int64           { return f.fileSize }
func (f *InputFile) GetFileType() fileid.FileType { return f.fileType }
func (f *InputFile) GetID() int64                 { return f.ID }

func (f *InputFile) SetID(v int64)                 { f.ID = v }
func (f *InputFile) SetAccessHash(v int64)         { f.AccessHash = v }
func (f *InputFile) SetFileRef(v []byte)           { f.FileRef = v }
func (f *InputFile) SetFileType(v fileid.FileType) { f.fileType = v }
func (f *InputFile) SetFileName(v string)          { f.fileName = v }
func (f *InputFile) SetReader(v io.ReadSeeker)     { f.reader = v }
func (f *InputFile) SetFileSize(v int64)           { f.fileSize = v }

// MediaKind enumerates the kinds of media that can be sent or uploaded.
type MediaKind int

const (
	MediaKindAuto MediaKind = iota
	MediaKindPhoto
	MediaKindDocument
	MediaKindAudio
	MediaKindVideo
	MediaKindAnimation
	MediaKindVoice
	MediaKindVideoNote
	MediaKindSticker
)
