package types

import "github.com/mtgo-labs/mtgo/tg"

type Checklist struct {
	ID    int64
	Title string
	Tasks []*ChecklistTask
}

type ChecklistTask struct {
	ID        int32
	Text      string
	Completed bool
}

func ParseChecklist(raw *tg.MessageMediaToDo) *Checklist {
	if raw == nil || raw.Todo == nil {
		return nil
	}
	todo := raw.Todo
	out := &Checklist{}
	for _, item := range todo.List {
		if item == nil {
			continue
		}
		task := &ChecklistTask{
			ID: item.ID,
		}
		if item.Title != nil {
			task.Text = item.Title.Text
		}
		out.Tasks = append(out.Tasks, task)
	}
	if todo.Title != nil {
		out.Title = todo.Title.Text
	}
	return out
}
