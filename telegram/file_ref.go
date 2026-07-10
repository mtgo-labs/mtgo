package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/mtgo-labs/mtgo/tg"
)

// FileRefOriginKind identifies the source of a file reference.
type FileRefOriginKind int

const (
	// FileRefOriginUnknown means no origin has been recorded.
	FileRefOriginUnknown FileRefOriginKind = iota
	// FileRefOriginMessage means the file reference comes from a message.
	FileRefOriginMessage
)

// FileRefOrigin records where a file reference came from so it can be
// refreshed when FILE_REFERENCE_EXPIRED is received from the server.
type FileRefOrigin struct {
	Kind FileRefOriginKind

	// Message-origin fields (Kind == FileRefOriginMessage).
	ChatID     int64 // peer ID of the chat containing the message
	MessageID  int32 // message ID
	AccessHash int64 // access hash for channel messages (0 for basic groups/DMs)
	IsChannel  bool  // true if the chat is a supergroup/channel
}

// FileRefresher caches file reference origins and refreshes expired references
// by re-fetching the source object. It implements the params.FileRefresher
// interface.
//
// The MVP supports message-based origins: when a file reference embedded in a
// document or photo expires, the refresher re-fetches the containing message
// (via messages.getMessages or channels.getMessages) and extracts a fresh
// file reference.
type FileRefresher struct {
	client  *Client
	mu      sync.Mutex
	origins map[int64]FileRefOrigin // doc_id/photo_id → origin
}

// NewFileRefresher creates a FileRefresher bound to the given client.
func NewFileRefresher(client *Client) *FileRefresher {
	return &FileRefresher{
		client:  client,
		origins: make(map[int64]FileRefOrigin),
	}
}

// Record stores the origin for a document or photo ID. Call this when you
// first obtain a file from a message so that a later FILE_REFERENCE_EXPIRED
// can be handled automatically.
func (f *FileRefresher) Record(id int64, origin FileRefOrigin) {
	if id == 0 {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.origins[id] = origin
}

// GetOrigin returns the stored origin for the given ID, or false if no origin
// has been recorded.
func (f *FileRefresher) GetOrigin(id int64) (FileRefOrigin, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	origin, ok := f.origins[id]
	return origin, ok
}

// RefreshFileReference re-fetches the source object for the given document or
// photo ID and returns a fresh file reference. Returns an error if no origin
// has been recorded or the refresh fails.
func (f *FileRefresher) RefreshFileReference(ctx context.Context, id int64) ([]byte, error) {
	origin, ok := f.GetOrigin(id)
	if !ok {
		return nil, fmt.Errorf("file_ref: no origin recorded for id %d", id)
	}

	switch origin.Kind {
	case FileRefOriginMessage:
		return f.refreshFromMessage(ctx, id, origin)
	default:
		return nil, fmt.Errorf("file_ref: unsupported origin kind %d for id %d", origin.Kind, id)
	}
}

// refreshFromMessage re-fetches the message containing the file and extracts
// the fresh file reference for the target document/photo ID.
func (f *FileRefresher) refreshFromMessage(ctx context.Context, id int64, origin FileRefOrigin) ([]byte, error) {
	inputMsg := &tg.InputMessageID{ID: origin.MessageID}

	var result tg.MessagesClass
	var err error

	if origin.IsChannel {
		result, err = f.client.Raw().ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
			Channel: &tg.InputChannel{
				ChannelID:  origin.ChatID,
				AccessHash: origin.AccessHash,
			},
			ID: []tg.InputMessageClass{inputMsg},
		})
		if err != nil {
			return nil, fmt.Errorf("file_ref: channels.getMessages: %w", err)
		}
	} else {
		result, err = f.client.Raw().MessagesGetMessages(ctx, &tg.MessagesGetMessagesRequest{
			ID: []tg.InputMessageClass{inputMsg},
		})
		if err != nil {
			return nil, fmt.Errorf("file_ref: messages.getMessages: %w", err)
		}
	}

	return extractFileRefFromMessages(result, id)
}

// extractFileRefFromMessages searches the messages in a messages.MessagesClass
// result for the document or photo with the given ID and returns its file
// reference.
func extractFileRefFromMessages(result tg.MessagesClass, id int64) ([]byte, error) {
	var msgs []tg.MessageClass
	switch r := result.(type) {
	case *tg.MessagesMessages:
		msgs = r.Messages
	case *tg.MessagesMessagesSlice:
		msgs = r.Messages
	case *tg.MessagesChannelMessages:
		msgs = r.Messages
	default:
		return nil, fmt.Errorf("file_ref: unexpected messages result type %T", result)
	}

	for _, msg := range msgs {
		m, ok := msg.(*tg.Message)
		if !ok || m.Media == nil {
			continue
		}

		switch media := m.Media.(type) {
		case *tg.MessageMediaDocument:
			if doc, ok := media.Document.(*tg.Document); ok && doc.ID == id {
				return doc.FileReference, nil
			}
		case *tg.MessageMediaPhoto:
			if photo, ok := media.Photo.(*tg.Photo); ok && photo.ID == id {
				return photo.FileReference, nil
			}
		}
	}

	return nil, fmt.Errorf("file_ref: document/photo id %d not found in refreshed messages", id)
}
