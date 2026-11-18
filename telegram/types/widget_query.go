package types

import (
	"context"

	"github.com/mtgo-labs/mtgo/tg"
)

// WidgetQuery represents an incoming widget query: a widget message of the
// logged-in bot asking for data or a wallet action.
type WidgetQuery struct {
	// ID is the unique identifier of the query; required when answering.
	ID int64
	// UserID is the user interacting with the widget.
	UserID int64
	// ChatID is the chat containing the widget message.
	ChatID int64
	// MessageID is the widget message.
	MessageID int32
	// ChatInstance is the opaque per-chat instance identifier.
	ChatInstance int64
	// ChatType describes the chat kind the widget is rendered in.
	ChatType string
	// Data is the widget-supplied query payload.
	Data []byte
	// WidgetID identifies a shared widget; 0 for per-message widgets.
	WidgetID int64
	// QueryType describes the kind of widget query (data, sign-data, wallet).
	QueryType tg.WidgetQueryTypeClass
	// FromUser is the resolved user interacting with the widget.
	FromUser *User
	binder   WidgetQueryBinder
}

// SetBinder injects the binder backing the bound convenience methods. Called
// internally by the Client after constructing the query from an update.
func (q *WidgetQuery) SetBinder(b WidgetQueryBinder) {
	q.binder = b
}

// Answer delivers the widget-visible reply to the query.
func (q *WidgetQuery) Answer(result string) error {
	return q.AnswerContext(context.Background(), result)
}

func (q *WidgetQuery) AnswerContext(ctx context.Context, result string) error {
	if q.binder == nil {
		return ErrNoBinder
	}
	return q.binder.AnswerWidgetQuery(ctx, q.ID, result)
}

// ParseWidgetQuery extracts a WidgetQuery from an MTProto update. Returns nil
// if raw is nil.
func ParseWidgetQuery(raw *tg.UpdateBotWidgetQuery) *WidgetQuery {
	if raw == nil {
		return nil
	}
	q := &WidgetQuery{
		ID:           raw.QueryID,
		UserID:       raw.UserID,
		MessageID:    raw.MsgID,
		ChatInstance: raw.ChatInstance,
		ChatType:     raw.ChatType,
		Data:         raw.Data,
		QueryType:    raw.QueryType,
	}
	if raw.WidgetID != 0 {
		q.WidgetID = raw.WidgetID
	}
	if raw.Peer != nil {
		q.ChatID = getBarePeerID(raw.Peer)
	}
	return q
}
