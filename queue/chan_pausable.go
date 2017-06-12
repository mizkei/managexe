package queue

import (
	"context"
	"sync"

	"github.com/mizkei/tasx"
)

type PausableChanQueue struct {
	mux      sync.Mutex
	cond     *sync.Cond
	que      tasx.Queue
	isPaused bool
}

func (pq *PausableChanQueue) Pause() {
	pq.mux.Lock()
	pq.isPaused = true
	pq.mux.Unlock()
}

func (pq *PausableChanQueue) Resume() {
	pq.mux.Lock()
	defer pq.mux.Unlock()

	if !pq.isPaused {
		return
	}

	pq.cond.Broadcast()
	pq.isPaused = false
}

func (pq *PausableChanQueue) wait() chan struct{} {
	ch := make(chan struct{})

	pq.mux.Lock()
	defer pq.mux.Unlock()

	if !pq.isPaused {
		close(ch)
		return ch
	}

	go func() {
		pq.cond.L.Lock()
		pq.cond.Wait()
		close(ch)
		pq.cond.L.Unlock()
	}()

	return ch
}

func (pq *PausableChanQueue) InsertTask(ctx context.Context, task tasx.Task) error {
	return pq.que.InsertTask(ctx, task)
}

func (pq *PausableChanQueue) FetchTask(ctx context.Context) (tasx.Task, error) {
	if pq.isPaused {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-pq.wait():
		}
	}

	return pq.que.FetchTask(ctx)
}

func NewPausableChanQueue(bufferN int) (*PausableChanQueue, error) {
	que, err := NewChanQueue(bufferN)
	if err != nil {
		return nil, err
	}

	return &PausableChanQueue{
		cond:     sync.NewCond(&sync.Mutex{}),
		que:      que,
		isPaused: false,
	}, nil
}
