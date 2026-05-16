package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// Checklist represents a to-do checklist message with a title, task list,
// and permissions controlling who can add or complete tasks.
//
// Example:
//
//	cl := types.ParseChecklist(rawTodo)
//	fmt.Printf("Checklist: %s (%d tasks)\n", cl.Title, len(cl.Tasks))
type Checklist struct {
	Title                    string
	Entities                 []*MessageEntity
	Tasks                    []*ChecklistTask
	OthersCanAddTasks        bool
	CanAddTasks              bool
	OthersCanMarkTasksAsDone bool
	CanMarkTasksAsDone       bool
}

// ParseChecklist converts a TL MessageMediaToDo into a Checklist with resolved
// completion data for each task. Returns nil if raw or raw.Todo is nil.
//
// Example:
//
//	cl := types.ParseChecklist(rawMedia)
//	for _, task := range cl.Tasks {
//	    fmt.Printf("- %s (done by: %v)\n", task.Text, task.CompletedBy)
//	}
func ParseChecklist(raw *tg.MessageMediaToDo) *Checklist {
	if raw == nil || raw.Todo == nil {
		return nil
	}
	todo := raw.Todo
	out := &Checklist{
		OthersCanAddTasks:        todo.OthersCanAppend,
		OthersCanMarkTasksAsDone: todo.OthersCanComplete,
	}
	if todo.Title != nil {
		out.Title = todo.Title.Text
		out.Entities = ParseMessageEntities(todo.Title.Entities)
	}
	completionMap := make(map[int32]*tg.TodoCompletion)
	for _, comp := range raw.Completions {
		if comp != nil {
			completionMap[comp.ID] = comp
		}
	}
	for _, item := range todo.List {
		if item == nil {
			continue
		}
		task := &ChecklistTask{
			ID: item.ID,
		}
		if item.Title != nil {
			task.Text = item.Title.Text
			task.Entities = ParseMessageEntities(item.Title.Entities)
		}
		if comp, ok := completionMap[item.ID]; ok {
			if comp.CompletedBy != nil {
				task.CompletedBy = ParseChatFromPeer(comp.CompletedBy, nil)
			}
			if comp.Date != 0 {
				task.CompletionDate = time.Unix(int64(comp.Date), 0)
			}
		}
		out.Tasks = append(out.Tasks, task)
	}
	return out
}

// ChecklistTask represents a single task within a checklist, including its text,
// formatting entities, and completion information.
type ChecklistTask struct {
	ID             int32
	Text           string
	Entities       []*MessageEntity
	CompletedBy    *Chat
	CompletionDate time.Time
}

// ChecklistTasksAdded represents the event of new tasks being added to an
// existing checklist message.
type ChecklistTasksAdded struct {
	ChecklistMessageID int32
	Tasks              []*ChecklistTask
}

// ChecklistTasksDone represents the event of tasks being marked as done or
// undone within a checklist.
type ChecklistTasksDone struct {
	ChecklistMessageID     int32
	MarkedAsDoneTaskIDs    []int32
	MarkedAsNotDoneTaskIDs []int32
}

// TextQuote represents a quoted portion of a message, used in reply contexts
// to highlight a specific passage.
type TextQuote struct {
	Text     string
	Entities []*MessageEntity
	Position int32
	Offset   int32
	Limit    int32
	IsManual bool
}
