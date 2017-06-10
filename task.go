package tasx

import (
	"context"
)

type Task interface {
	Run(context.Context) error
}

type TaskInserter interface {
	InsertTask(context.Context, Task) error
}

type TaskFetcher interface {
	FetchTask(context.Context) (Task, error)
}

type Queue interface {
	TaskInserter
	TaskFetcher
}
