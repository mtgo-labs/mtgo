package types

// ChecklistTasksAdded represents tasks added to a checklist.
type ChecklistTasksAdded struct {
	Tasks []*ChecklistTask
}

// ChecklistTasksDone represents tasks marked as done in a checklist.
type ChecklistTasksDone struct {
	Tasks []*ChecklistTask
}

// TextQuote represents a quoted portion of message text.
type TextQuote struct {
	Text   string
	Offset int32
	Limit  int32
}
