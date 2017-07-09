package tasx

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// Manager manages worker.
type Manager interface {
	// WorkerState returns state of workers.
	WorkerState() (workerN, runningN int)
	// Wait blocks until all worker finish.
	Wait()
	// Run starts fetching and task execution.
	Run(context.Context)
}

type manager struct {
	wg         *sync.WaitGroup
	workerN    int
	runningN   int32
	fetcher    TaskFetcher
	waitCh     chan struct{}
	errHandler func(error)
}

func (m manager) WorkerState() (workerN, runningN int) {
	return m.workerN, int(m.runningN)
}

func (m *manager) Wait() {
	<-m.waitCh
	m.wg.Wait()
}

func (m *manager) runWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			task, err := m.fetcher.FetchTask(ctx)
			if err != nil {
				m.errHandler(err)
				continue
			}
			atomic.AddInt32(&m.runningN, 1)

			func() {
				defer atomic.AddInt32(&m.runningN, -1)
				if err := task.Run(ctx); err != nil {
					m.errHandler(err)
				}
			}()
		}
	}
}

func (m *manager) Run(ctx context.Context) {
	runCh := make(chan struct{})
	go func() {
		for {
			select {
			case <-runCh:
				go func() {
					defer func() {
						if r := recover(); r != nil {
							err, ok := r.(error)
							if !ok {
								err = fmt.Errorf("tasx: %v", r)
							}
							m.errHandler(err)
							runCh <- struct{}{}
						}
					}()
					err := m.runWorker(ctx)
					if err == context.Canceled || err == context.DeadlineExceeded {
						m.wg.Done()
					}
				}()
			}
		}
	}()

	for i := 0; i < m.workerN; i++ {
		m.wg.Add(1)
		runCh <- struct{}{}
	}

	close(m.waitCh)
	m.wg.Wait()
}

// NewManager returns Manager instance.
// workerN is number of worker.
// worker fetch task from fetcher and run task.
func NewManager(workerN int, fetcher TaskFetcher, errHandler func(error)) (Manager, error) {
	if workerN < 1 {
		return nil, errors.New("number of workers should be greater than 0")
	}
	if fetcher == nil {
		return nil, errors.New("featcher should not be nil")
	}
	if errHandler == nil {
		return nil, errors.New("error handler should not be nil")
	}

	return &manager{
		workerN:    workerN,
		fetcher:    fetcher,
		runningN:   0,
		waitCh:     make(chan struct{}),
		errHandler: errHandler,
	}, nil
}
