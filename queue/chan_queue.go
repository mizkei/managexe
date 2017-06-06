package queue

import (
	"context"

	"github.com/mizkei/tasx"
)

type chanQueue struct {
	ch        chan tasx.Task
	handleErr func(error)
}

func (q *chanQueue) HandleErr(err error) {
	q.handleErr(err)
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

func NewChanQueue(bufferN int, errHandler func(error)) (tasx.Queue, error) {
	return &chanQueue{
		ch:        make(chan tasx.Task, bufferN),
		handleErr: errHandler,
	}, nil
}
