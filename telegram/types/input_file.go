package types

import (
	"io"

	"github.com/mtgo-labs/mtgo/telegram/fileid"
)

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

func FileID(s string) *InputFile {
	return &InputFile{fileID: s}
}

func FromIDs(ID, accessHash int64, fileRef []byte) *InputFile {
	return &InputFile{ID: ID, AccessHash: accessHash, FileRef: fileRef}
}

func URL(u string) *InputFile {
	return &InputFile{url: u}
}

func Path(p string) *InputFile {
	return &InputFile{path: p}
}

func Reader(r io.ReadSeeker, fileName string, size int64) *InputFile {
	return &InputFile{reader: r, fileName: fileName, fileSize: size}
}

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

func (f *InputFile) GetFileID() string     { return f.fileID }
func (f *InputFile) GetURL() string        { return f.url }
func (f *InputFile) GetPath() string       { return f.path }
func (f *InputFile) GetReader() io.ReadSeeker { return f.reader }
func (f *InputFile) GetFileName() string   { return f.fileName }
func (f *InputFile) GetFileSize() int64    { return f.fileSize }
func (f *InputFile) GetFileType() fileid.FileType { return f.fileType }
func (f *InputFile) GetID() int64          { return f.ID }

func (f *InputFile) SetID(v int64)                      { f.ID = v }
func (f *InputFile) SetAccessHash(v int64)               { f.AccessHash = v }
func (f *InputFile) SetFileRef(v []byte)                 { f.FileRef = v }
func (f *InputFile) SetFileType(v fileid.FileType)       { f.fileType = v }
func (f *InputFile) SetFileName(v string)                { f.fileName = v }
func (f *InputFile) SetReader(v io.ReadSeeker)           { f.reader = v }
func (f *InputFile) SetFileSize(v int64)                 { f.fileSize = v }

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
