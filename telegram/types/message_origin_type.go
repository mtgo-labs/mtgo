package types

// MessageOriginType describes where a forwarded or imported message originally came from.
type MessageOriginType string

const (
	// MessageOriginTypeChannel indicates the message was forwarded from a channel post.
	MessageOriginTypeChannel MessageOriginType = "channel"
	// MessageOriginTypeChat indicates the message was forwarded from a group chat.
	MessageOriginTypeChat MessageOriginType = "chat"
	// MessageOriginTypeHiddenUser indicates the original sender chose to remain hidden.
	MessageOriginTypeHiddenUser MessageOriginType = "hidden_user"
	// MessageOriginTypeImport indicates the message was imported from an external service.
	MessageOriginTypeImport MessageOriginType = "import"
	// MessageOriginTypeUser indicates the message was originally sent by a specific user.
	MessageOriginTypeUser MessageOriginType = "user"
)

// String returns the string representation of the MessageOriginType.
func (m MessageOriginType) String() string { return string(m) }
