package tasx

import (
	"context"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Manager manages worker.
type Manager interface {
	// WorkerState returns state of workers.
	WorkerState() (workerN, runningN int)
	// Wait blocks until all worker finish.
	Wait()
	// SetLogger sets logger.
	SetLogger(*log.Logger)
	// Run starts fetching and task execution.
	Run(context.Context)
}

type manager struct {
	wg       *sync.WaitGroup
	workerN  int
	runningN int32
	fetcher  TaskFetcher
	waitCh   chan struct{}
	logger   *log.Logger
}

func (m manager) WorkerState() (workerN, runningN int) {
	return m.workerN, int(m.runningN)
}

func (m *manager) Wait() {
	<-m.waitCh
	m.wg.Wait()
}

func (m *manager) SetLogger(l *log.Logger) {
	atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&m.logger)), unsafe.Pointer(l))
}

func (m *manager) Run(ctx context.Context) {
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
						m.logger.Println(err)
						continue
					}
					atomic.AddInt32(&m.runningN, 1)

					func() {
						defer func() {
							atomic.AddInt32(&m.runningN, -1)
							if err := recover(); err != nil {
								m.logger.Println(err)
							}
						}()
						if err := task.Run(ctx); err != nil {
							m.logger.Println(err)
						}
					}()
				}
			}
		}()
	}

	close(m.waitCh)
	m.wg.Wait()
}

// NewManager returns Manager instance.
// workerN is number of worker.
// worker fetch task from fetcher and run task.
func NewManager(workerN int, fetcher TaskFetcher) Manager {
	return &manager{
		workerN:  workerN,
		fetcher:  fetcher,
		runningN: 0,
		waitCh:   make(chan struct{}),
		logger:   log.New(os.Stderr, "", log.Ldate|log.Ltime),
	}
}
