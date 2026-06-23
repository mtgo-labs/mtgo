package fileid

import (
	"testing"
)

func BenchmarkEncodeFileID(b *testing.B) {
	fid := FileID{
		Type:        FileTypePhoto,
		ID:          123456789,
		AccessHash:  987654321,
		FileReference: []byte("abc123ref"),
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = Encode(fid)
	}
}

func BenchmarkDecodeFileID(b *testing.B) {
	fid := FileID{
		Type:        FileTypeDocument,
		ID:          123456789,
		AccessHash:  987654321,
		FileReference: []byte("abc123ref"),
	}
	encoded, _ := Encode(fid)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = Decode(encoded)
	}
}

func BenchmarkEncodeDecodeRoundTrip(b *testing.B) {
	fid := FileID{
		Type:        FileTypeVideo,
		ID:          555555,
		AccessHash:  777777,
		FileReference: []byte("ref"),
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		enc, _ := Encode(fid)
		_, _ = Decode(enc)
	}
}
