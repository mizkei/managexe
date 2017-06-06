package tasx

import (
	"context"
)

type Task interface {
	Run(context.Context) error
	HandleErr(error)
}

type TaskInserter interface {
	InsertTask(context.Context, Task) error
}

type TaskFetcher interface {
	FetchTask(context.Context) (Task, error)
	HandleErr(error)
}

type Queue interface {
	TaskInserter
	TaskFetcher
}
