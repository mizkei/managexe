package tasx

import (
	"context"
)

// Task is task interface.
type Task interface {
	Run(context.Context) error
}

// TaskInserter is the interface that implements insert task.
type TaskInserter interface {
	InsertTask(context.Context, Task) error
}

// TaskFetcher is the interface that implements fetch task.
// FetchTask should block until fetch a task.
type TaskFetcher interface {
	FetchTask(context.Context) (Task, error)
}

// Queue is the inteface that implements TaskInserter and TaskFetcher.
type Queue interface {
	TaskInserter
	TaskFetcher
}
