package fileid

import (
	"testing"
)

func TestRLE_RoundTrip(t *testing.T) {
	original := []byte{1, 0, 0, 0, 2, 0, 3}
	encoded := rleEncode(original)
	decoded := rleDecode(encoded)
	if len(decoded) != len(original) {
		t.Fatalf("length mismatch: got %d, want %d", len(decoded), len(original))
	}
	for i := range original {
		if decoded[i] != original[i] {
			t.Errorf("byte %d: got %d, want %d", i, decoded[i], original[i])
		}
	}
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	original := FileID{
		Type:       FileTypePhoto,
		DCID:       2,
		ID:         123456789,
		AccessHash: 987654321,
		VolumeID:   111222333,
		Source: PhotoSizeSource{
			Type:   ThumbnailSourceLegacy,
			Secret: 555666777,
		},
	}
	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if encoded == "" {
		t.Fatal("encoded string is empty")
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type: got %d, want %d", decoded.Type, original.Type)
	}
	if decoded.DCID != original.DCID {
		t.Errorf("DCID: got %d, want %d", decoded.DCID, original.DCID)
	}
	if decoded.ID != original.ID {
		t.Errorf("ID: got %d, want %d", decoded.ID, original.ID)
	}
	if decoded.AccessHash != original.AccessHash {
		t.Errorf("AccessHash: got %d, want %d", decoded.AccessHash, original.AccessHash)
	}
	if decoded.Source.Type != ThumbnailSourceLegacy {
		t.Errorf("SourceType: got %d, want %d", decoded.Source.Type, ThumbnailSourceLegacy)
	}
	if decoded.Source.Secret != original.Source.Secret {
		t.Errorf("Source.Secret: got %d, want %d", decoded.Source.Secret, original.Source.Secret)
	}
}

func TestDecode_InvalidBase64(t *testing.T) {
	_, err := Decode("!!!invalid!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestEncodeDecodeDocument(t *testing.T) {
	f := FileID{
		Type:       FileTypeDocument,
		DCID:       4,
		ID:         100200300,
		AccessHash: 400500600,
	}
	encoded, err := Encode(f)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Type != f.Type {
		t.Fatalf("type mismatch")
	}
	if decoded.DCID != f.DCID {
		t.Fatalf("dcid mismatch")
	}
	if decoded.ID != f.ID {
		t.Fatalf("id mismatch")
	}
	if decoded.AccessHash != f.AccessHash {
		t.Fatalf("access_hash mismatch")
	}
}

func TestEncodeDecodeThumbnailSource(t *testing.T) {
	f := FileID{
		Type:       FileTypePhoto,
		DCID:       1,
		ID:         111,
		AccessHash: 222,
		VolumeID:   333,
		Source: PhotoSizeSource{
			Type:              ThumbnailSourceThumbnail,
			ThumbnailFileType: FileTypePhoto,
			ThumbnailSize:     0x73,
			LocalID:           20,
		},
	}
	encoded, err := Encode(f)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Source.Type != ThumbnailSourceThumbnail {
		t.Fatalf("source type mismatch: got %d", decoded.Source.Type)
	}
	if decoded.Source.ThumbnailFileType != FileTypePhoto {
		t.Fatalf("thumbnail file type mismatch")
	}
	if decoded.Source.ThumbnailSize != 0x73 {
		t.Fatalf("thumbnail size mismatch: got %d", decoded.Source.ThumbnailSize)
	}
}

func TestEncodeDecodeDialogPhoto(t *testing.T) {
	f := FileID{
		Type:       FileTypePhoto,
		DCID:       2,
		ID:         999,
		AccessHash: 888,
		VolumeID:   777,
		Source: PhotoSizeSource{
			Type:           ThumbnailSourceDialogPhotoBig,
			ChatID:         12345,
			ChatAccessHash: 67890,
			LocalID:        5,
		},
	}
	encoded, err := Encode(f)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Source.Type != ThumbnailSourceDialogPhotoBig {
		t.Fatalf("source type mismatch")
	}
	if decoded.Source.ChatID != 12345 {
		t.Fatalf("chat_id mismatch: got %d", decoded.Source.ChatID)
	}
	if decoded.Source.ChatAccessHash != 67890 {
		t.Fatalf("chat_access_hash mismatch: got %d", decoded.Source.ChatAccessHash)
	}
}

func TestEncodeDecodeStickerSetThumb(t *testing.T) {
	f := FileID{
		Type:       FileTypePhoto,
		DCID:       3,
		ID:         555,
		AccessHash: 444,
		VolumeID:   333,
		Source: PhotoSizeSource{
			Type:                 ThumbnailSourceStickerSetThumb,
			StickerSetID:         111222,
			StickerSetAccessHash: 333444,
			LocalID:              7,
		},
	}
	encoded, err := Encode(f)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Source.Type != ThumbnailSourceStickerSetThumb {
		t.Fatalf("source type mismatch")
	}
	if decoded.Source.StickerSetID != 111222 {
		t.Fatalf("sticker_set_id mismatch: got %d", decoded.Source.StickerSetID)
	}
	if decoded.Source.StickerSetAccessHash != 333444 {
		t.Fatalf("sticker_set_access_hash mismatch: got %d", decoded.Source.StickerSetAccessHash)
	}
}

func TestIsPhoto(t *testing.T) {
	tests := []struct {
		ft   FileType
		want bool
	}{
		{FileTypePhoto, true},
		{FileTypeThumbnail, true},
		{FileTypeProfilePhoto, true},
		{FileTypeDocumentPhoto, false},
		{FileTypeDocument, false},
		{FileTypeVideo, false},
		{FileTypeSticker, false},
	}
	for _, tt := range tests {
		if got := tt.ft.IsPhoto(); got != tt.want {
			t.Errorf("FileType(%d).IsPhoto() = %v, want %v", tt.ft, got, tt.want)
		}
	}
}
