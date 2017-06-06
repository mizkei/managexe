package tasx

import (
	"context"
	"sync"
	"sync/atomic"
)

type Manager struct {
	workerN     int
	runningN    int32
	fetcher     TaskFetcher
	handlePanic func(interface{})
}

func (m Manager) WorkerState() (workerN, runningN int) {
	return m.workerN, int(m.runningN)
}

func (m *Manager) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < m.workerN; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					task, err := m.fetcher.FetchTask(ctx)
					if err != nil {
						m.fetcher.HandleErr(err)
						continue
					}
					atomic.AddInt32(&m.runningN, 1)

					func() {
						defer func() {
							atomic.AddInt32(&m.runningN, -1)
							if err := recover(); err != nil {
								m.handlePanic(err)
							}
						}()
						if err := task.Run(ctx); err != nil {
							task.HandleErr(err)
						}
					}()
				}
			}
		}()
	}

	wg.Wait()
}

func NewManager(workerN int, fetcher TaskFetcher, panicHandler func(interface{})) *Manager {
	return &Manager{
		workerN:     workerN,
		fetcher:     fetcher,
		runningN:    0,
		handlePanic: panicHandler,
	}
}
