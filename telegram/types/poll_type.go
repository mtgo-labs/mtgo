package types

// PollType indicates whether a poll is a quiz (exactly one correct answer) or a regular poll.
type PollType string

// Poll type constants.
const (
	PollTypeQuiz    PollType = "quiz"
	PollTypeRegular PollType = "regular"
)

// String returns the string representation of the PollType.
func (p PollType) String() string { return string(p) }
