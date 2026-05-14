package types

// PollUpdate represents a state change for a poll, such as new votes or closure.
// Delivered as an update when a poll's state changes.
type PollUpdate struct {
	// PollID is the unique identifier of the poll that was updated.
	PollID int64
	// Question is the poll question text.
	Question string
	// Closed indicates whether the poll has been closed and no longer accepts votes.
	Closed bool
}
