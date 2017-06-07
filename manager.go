package tasx

import (
	"context"
	"sync"
	"sync/atomic"
)

type Manager struct {
	wg          sync.WaitGroup
	workerN     int
	runningN    int32
	fetcher     TaskFetcher
	waitCh      chan struct{}
	handlePanic func(interface{})
}

func (m Manager) WorkerState() (workerN, runningN int) {
	return m.workerN, int(m.runningN)
}

func (m *Manager) Wait() {
	<-m.waitCh
	m.wg.Wait()
}

func (m *Manager) Run(ctx context.Context) {
	for i := 0; i < m.workerN; i++ {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()

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

	close(m.waitCh)
	m.wg.Wait()
}

func NewManager(workerN int, fetcher TaskFetcher, panicHandler func(interface{})) *Manager {
	return &Manager{
		workerN:     workerN,
		fetcher:     fetcher,
		runningN:    0,
		waitCh:      make(chan struct{}),
		handlePanic: panicHandler,
	}
}
