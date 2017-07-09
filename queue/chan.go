package queue

import (
	"context"

	"github.com/mizkei/tasx"
)

type chanQueue struct {
	ch chan tasx.Task
}

func (q *chanQueue) InsertTask(ctx context.Context, task tasx.Task) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.ch <- task:
		return nil
	}
}

func (q *chanQueue) FetchTask(ctx context.Context) (tasx.Task, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case t := <-q.ch:
		return t, nil
	}
}

// NewChanQueue create queue that implemented in channel.
func NewChanQueue(bufferN int) (tasx.Queue, error) {
	return &chanQueue{
		ch: make(chan tasx.Task, bufferN),
	}, nil
}
