package types

// DeletedMessages represents a batch of messages deleted from a chat.
// Delivered as an update when messages are removed by a user or admin.
type DeletedMessages struct {
	// Messages is the list of message IDs that were deleted.
	Messages []int32
	// ChatID is the chat or channel from which the messages were removed.
	ChatID int64
}
