package telegram

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

func TestLocationDocID(t *testing.T) {
	tests := []struct {
		name string
		loc  tg.InputFileLocationClass
		want int64
	}{
		{"document", &tg.InputDocumentFileLocation{ID: 42}, 42},
		{"photo", &tg.InputPhotoFileLocation{ID: 99}, 99},
		{"other", &tg.InputPeerPhotoFileLocation{}, 0},
		{"nil", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := locationDocID(tt.loc)
			if got != tt.want {
				t.Errorf("locationDocID() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestUpdateLocationFileRef(t *testing.T) {
	newRef := []byte("fresh-ref")

	t.Run("document", func(t *testing.T) {
		orig := &tg.InputDocumentFileLocation{
			ID:            1,
			AccessHash:    2,
			FileReference: []byte("old"),
			ThumbSize:     "m",
		}
		updated, err := updateLocationFileRef(orig, newRef)
		if err != nil {
			t.Fatal(err)
		}
		doc, ok := updated.(*tg.InputDocumentFileLocation)
		if !ok {
			t.Fatalf("wrong type: %T", updated)
		}
		if doc.ID != 1 || doc.AccessHash != 2 || doc.ThumbSize != "m" {
			t.Error("fields not preserved")
		}
		if !bytes.Equal(doc.FileReference, newRef) {
			t.Error("file reference not updated")
		}
		// Original should be unchanged (copy, not mutation).
		if !bytes.Equal(orig.FileReference, []byte("old")) {
			t.Error("original location was mutated")
		}
	})

	t.Run("photo", func(t *testing.T) {
		orig := &tg.InputPhotoFileLocation{
			ID:            3,
			AccessHash:    4,
			FileReference: []byte("old"),
			ThumbSize:     "x",
		}
		updated, err := updateLocationFileRef(orig, newRef)
		if err != nil {
			t.Fatal(err)
		}
		photo, ok := updated.(*tg.InputPhotoFileLocation)
		if !ok {
			t.Fatalf("wrong type: %T", updated)
		}
		if !bytes.Equal(photo.FileReference, newRef) {
			t.Error("file reference not updated")
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		_, err := updateLocationFileRef(&tg.InputPeerPhotoFileLocation{}, newRef)
		if err == nil {
			t.Error("expected error for unsupported type")
		}
	})
}

func TestFileRefresher_RecordAndGetOrigin(t *testing.T) {
	fr := NewFileRefresher(nil)

	fr.Record(100, FileRefOrigin{
		Kind:      FileRefOriginMessage,
		ChatID:    555,
		MessageID: 42,
	})

	origin, ok := fr.GetOrigin(100)
	if !ok {
		t.Fatal("origin not found")
	}
	if origin.ChatID != 555 || origin.MessageID != 42 {
		t.Errorf("unexpected origin: %+v", origin)
	}

	_, ok = fr.GetOrigin(999)
	if ok {
		t.Error("should not find unrecorded origin")
	}

	// Recording with id=0 should be ignored.
	fr.Record(0, FileRefOrigin{Kind: FileRefOriginMessage})
	_, ok = fr.GetOrigin(0)
	if ok {
		t.Error("id=0 should not be recorded")
	}
}

func TestFileRefresher_RefreshFileReference_NoOrigin(t *testing.T) {
	fr := NewFileRefresher(nil)
	_, err := fr.RefreshFileReference(context.Background(), 123)
	if err == nil {
		t.Error("expected error when no origin recorded")
	}
}

func TestFileRefresher_RefreshFileReference_UnsupportedKind(t *testing.T) {
	fr := NewFileRefresher(nil)
	fr.Record(123, FileRefOrigin{Kind: FileRefOriginUnknown})
	_, err := fr.RefreshFileReference(context.Background(), 123)
	if err == nil {
		t.Error("expected error for unknown origin kind")
	}
}

func TestExtractFileRefFromMessages_Document(t *testing.T) {
	docID := int64(777)
	newRef := []byte("refreshed-ref")

	msgs := &tg.MessagesMessages{
		Messages: []tg.MessageClass{
			&tg.Message{
				ID: 1,
				Media: &tg.MessageMediaDocument{
					Document: &tg.Document{
						ID:            docID,
						FileReference: newRef,
					},
				},
			},
			&tg.Message{ID: 2},
		},
	}

	ref, err := extractFileRefFromMessages(msgs, docID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ref, newRef) {
		t.Errorf("got %v, want %v", ref, newRef)
	}
}

func TestExtractFileRefFromMessages_Photo(t *testing.T) {
	photoID := int64(888)
	newRef := []byte("photo-ref")

	msgs := &tg.MessagesMessages{
		Messages: []tg.MessageClass{
			&tg.Message{
				ID: 5,
				Media: &tg.MessageMediaPhoto{
					Photo: &tg.Photo{
						ID:            photoID,
						FileReference: newRef,
					},
				},
			},
		},
	}

	ref, err := extractFileRefFromMessages(msgs, photoID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ref, newRef) {
		t.Errorf("got %v, want %v", ref, newRef)
	}
}

func TestExtractFileRefFromMessages_NotFound(t *testing.T) {
	msgs := &tg.MessagesMessages{
		Messages: []tg.MessageClass{
			&tg.Message{ID: 1},
		},
	}
	_, err := extractFileRefFromMessages(msgs, 999)
	if err == nil {
		t.Error("expected error when document not found")
	}
}

func TestExtractFileRefFromMessages_SliceType(t *testing.T) {
	docID := int64(111)
	newRef := []byte("slice-ref")

	msgs := &tg.MessagesMessagesSlice{
		Messages: []tg.MessageClass{
			&tg.Message{
				ID: 1,
				Media: &tg.MessageMediaDocument{
					Document: &tg.Document{
						ID:            docID,
						FileReference: newRef,
					},
				},
			},
		},
	}

	ref, err := extractFileRefFromMessages(msgs, docID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ref, newRef) {
		t.Errorf("got %v, want %v", ref, newRef)
	}
}

func TestExtractFileRefFromMessages_ChannelType(t *testing.T) {
	docID := int64(222)
	newRef := []byte("channel-ref")

	msgs := &tg.MessagesChannelMessages{
		Messages: []tg.MessageClass{
			&tg.Message{
				ID: 1,
				Media: &tg.MessageMediaDocument{
					Document: &tg.Document{
						ID:            docID,
						FileReference: newRef,
					},
				},
			},
		},
	}

	ref, err := extractFileRefFromMessages(msgs, docID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ref, newRef) {
		t.Errorf("got %v, want %v", ref, newRef)
	}
}

// mockFileRefInvoker returns FILE_REFERENCE_EXPIRED on the first upload.getFile
// call, then serves file data on subsequent calls. It also serves
// messages.getMessages responses for file reference refresh.
type mockFileRefInvoker struct {
	data        []byte
	chunkSize   int32
	expiredOnce bool
	newFileRef  []byte
	docID       int64
}

func (m *mockFileRefInvoker) RPCInvoke(_ context.Context, input tg.TLObject, _ func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	switch req := input.(type) {
	case *tg.UploadGetFileRequest:
		if !m.expiredOnce {
			m.expiredOnce = true
			return nil, &tgerr.Error{
				Code:    400,
				Message: "FILE_REFERENCE_EXPIRED",
				Type:    "FILE_REFERENCE_EXPIRED",
			}
		}
		start := req.Offset
		if start >= int64(len(m.data)) {
			return &tg.UploadFile{Bytes: nil}, nil
		}
		end := start + int64(m.chunkSize)
		if end > int64(len(m.data)) {
			end = int64(len(m.data))
		}
		return &tg.UploadFile{Bytes: m.data[start:end]}, nil

	case *tg.MessagesGetMessagesRequest:
		return &tg.MessagesMessages{
			Messages: []tg.MessageClass{
				&tg.Message{
					ID: 1,
					Media: &tg.MessageMediaDocument{
						Document: &tg.Document{
							ID:            m.docID,
							FileReference: m.newFileRef,
						},
					},
				},
			},
		}, nil

	case *tg.ChannelsGetMessagesRequest:
		return &tg.MessagesChannelMessages{
			Messages: []tg.MessageClass{
				&tg.Message{
					ID: 1,
					Media: &tg.MessageMediaDocument{
						Document: &tg.Document{
							ID:            m.docID,
							FileReference: m.newFileRef,
						},
					},
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unexpected request type: %T", input)
	}
}

func (m *mockFileRefInvoker) RPCInvokeRaw(_ context.Context, _ tg.TLObject) ([]byte, error) {
	return nil, nil
}

// stubFileRefresher is a simple params.FileRefresher that returns a
// predetermined file reference.
type stubFileRefresher struct {
	ref []byte
	err error
}

func (s *stubFileRefresher) RefreshFileReference(_ context.Context, _ int64) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.ref, nil
}

func TestDownloadToFileRPC_FileRefAutoRefresh(t *testing.T) {
	data := []byte("hello world file ref refresh test data")
	mock := &mockFileRefInvoker{
		data:       data,
		chunkSize:  int32(len(data)),
		newFileRef: []byte("fresh-ref"),
		docID:      100,
	}
	rpc := tg.NewRPCClient(mock)

	location := &tg.InputDocumentFileLocation{
		ID:            100,
		AccessHash:    200,
		FileReference: []byte("expired-ref"),
	}

	refresher := &stubFileRefresher{ref: []byte("fresh-ref")}
	opts := &params.Download{FileRefresher: refresher}

	var buf bytes.Buffer
	written, _, err := downloadToFileRPC(context.Background(), rpc, location, int64(len(data)), &buf, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written != int64(len(data)) {
		t.Errorf("written = %d, want %d", written, int64(len(data)))
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Error("data mismatch")
	}
}

func TestDownloadToFileRPC_FileRefExhausted(t *testing.T) {
	// File reference keeps expiring — should fail after maxFileRefRetries.
	data := []byte("test data")
	mock := &alwaysExpiredInvoker{
		data:      data,
		chunkSize: int32(len(data)),
	}
	rpc := tg.NewRPCClient(mock)

	location := &tg.InputDocumentFileLocation{
		ID:         100,
		AccessHash: 200,
	}
	refresher := &stubFileRefresher{ref: []byte("fresh")}
	opts := &params.Download{FileRefresher: refresher}

	var buf bytes.Buffer
	_, _, err := downloadToFileRPC(context.Background(), rpc, location, int64(len(data)), &buf, opts)
	if err == nil {
		t.Fatal("expected error when file reference keeps expiring")
	}
}

// alwaysExpiredInvoker always returns FILE_REFERENCE_EXPIRED for getFile.
type alwaysExpiredInvoker struct {
	data      []byte
	chunkSize int32
}

func (m *alwaysExpiredInvoker) RPCInvoke(_ context.Context, input tg.TLObject, _ func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	switch input.(type) {
	case *tg.UploadGetFileRequest:
		return nil, &tgerr.Error{
			Code:    400,
			Message: "FILE_REFERENCE_EXPIRED",
			Type:    "FILE_REFERENCE_EXPIRED",
		}
	default:
		return nil, fmt.Errorf("unexpected request type: %T", input)
	}
}

func (m *alwaysExpiredInvoker) RPCInvokeRaw(_ context.Context, _ tg.TLObject) ([]byte, error) {
	return nil, nil
}

func TestDownloadToFileRPC_NoRefresher(t *testing.T) {
	// Without a refresher, FILE_REFERENCE_EXPIRED should propagate as-is.
	mock := &mockFileRefInvoker{
		data:       []byte("test"),
		chunkSize:  4,
		newFileRef: []byte("fresh"),
		docID:      100,
	}
	rpc := tg.NewRPCClient(mock)

	location := &tg.InputDocumentFileLocation{
		ID:         100,
		AccessHash: 200,
	}
	// No FileRefresher in opts.
	opts := &params.Download{}

	var buf bytes.Buffer
	_, _, err := downloadToFileRPC(context.Background(), rpc, location, 4, &buf, opts)
	if err == nil {
		t.Fatal("expected FILE_REFERENCE_EXPIRED error without refresher")
	}
}

func TestTryRefreshLocationFileRef(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		loc := &tg.InputDocumentFileLocation{ID: 42, AccessHash: 99}
		refresher := &stubFileRefresher{ref: []byte("new-ref")}

		updated, err := tryRefreshLocationFileRef(context.Background(), loc, refresher)
		if err != nil {
			t.Fatal(err)
		}
		doc, ok := updated.(*tg.InputDocumentFileLocation)
		if !ok {
			t.Fatalf("wrong type: %T", updated)
		}
		if doc.ID != 42 || doc.AccessHash != 99 {
			t.Error("fields not preserved")
		}
		if !bytes.Equal(doc.FileReference, []byte("new-ref")) {
			t.Error("file reference not updated")
		}
	})

	t.Run("no doc id", func(t *testing.T) {
		loc := &tg.InputPeerPhotoFileLocation{}
		refresher := &stubFileRefresher{ref: []byte("new-ref")}

		_, err := tryRefreshLocationFileRef(context.Background(), loc, refresher)
		if err == nil {
			t.Error("expected error for location without doc id")
		}
	})

	t.Run("refresh fails", func(t *testing.T) {
		loc := &tg.InputDocumentFileLocation{ID: 42}
		refresher := &stubFileRefresher{err: fmt.Errorf("network error")}

		_, err := tryRefreshLocationFileRef(context.Background(), loc, refresher)
		if err == nil {
			t.Error("expected error when refresh fails")
		}
	})
}
