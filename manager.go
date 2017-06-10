package tasx

import (
	"context"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

var (
	Logger = log.New(os.Stderr, "", log.Ldate|log.Ltime)
)

type Manager struct {
	wg       sync.WaitGroup
	workerN  int
	runningN int32
	fetcher  TaskFetcher
	waitCh   chan struct{}
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
						Logger.Println(err)
						continue
					}
					atomic.AddInt32(&m.runningN, 1)

					func() {
						defer func() {
							atomic.AddInt32(&m.runningN, -1)
							if err := recover(); err != nil {
								Logger.Println(err)
							}
						}()
						if err := task.Run(ctx); err != nil {
							Logger.Println(err)
						}
					}()
				}
			}
		}()
	}

	close(m.waitCh)
	m.wg.Wait()
}

func NewManager(workerN int, fetcher TaskFetcher) *Manager {
	return &Manager{
		workerN:  workerN,
		fetcher:  fetcher,
		runningN: 0,
		waitCh:   make(chan struct{}),
	}
}
