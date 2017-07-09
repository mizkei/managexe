package queue

import (
	"context"
	"sync"

	"github.com/mizkei/tasx"
)

// PausableChanQueue is queue that can pause FetchTask.
type PausableChanQueue struct {
	mux      sync.Mutex
	wg       *sync.WaitGroup
	que      tasx.Queue
	isPaused bool
}

// Pause pause FetchTask.
func (pq *PausableChanQueue) Pause() {
	pq.mux.Lock()
	defer pq.mux.Unlock()
	if pq.isPaused {
		return
	}

	pq.isPaused = true
	pq.wg.Add(1)
}

// Resume resume FetchTask.
func (pq *PausableChanQueue) Resume() {
	pq.mux.Lock()
	defer pq.mux.Unlock()
	if !pq.isPaused {
		return
	}

	pq.isPaused = false
	pq.wg.Done()
}

// InsertTask insert a task to queue.
func (pq *PausableChanQueue) InsertTask(ctx context.Context, task tasx.Task) error {
	return pq.que.InsertTask(ctx, task)
}

// FetchTask fetch a task from channel.
// if queue is paused, FetchTask blocks until Resume called.
func (pq *PausableChanQueue) FetchTask(ctx context.Context) (tasx.Task, error) {
	pq.wg.Wait()
	return pq.que.FetchTask(ctx)
}

// NewPausableChanQueue create a queue that is pausable.
func NewPausableChanQueue(bufferN int) (*PausableChanQueue, error) {
	que, err := NewChanQueue(bufferN)
	if err != nil {
		return nil, err
	}

	return &PausableChanQueue{
		wg:       &sync.WaitGroup{},
		que:      que,
		isPaused: false,
	}, nil
}
