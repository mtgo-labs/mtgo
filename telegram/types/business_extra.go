package types

// BusinessMessage represents a message sent through a business connection.
type BusinessMessage struct {
	ConnectionID string
	MsgID        int32
}

// MessageContent is the interface for typed message content payloads.
type MessageContent interface {
	ContentType() string
}
