package managexe

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type ErrPanic struct {
	Err interface{}
}

func (ep ErrPanic) Error() string {
	return fmt.Sprint(ep.Err)
}

type Execer interface {
	Exec(context.Context) error
	ErrHandle(error)
}

type Manager struct {
	mutex    sync.Mutex
	ch       chan Execer
	pauseCh  chan struct{}
	isPaused bool
	workerN  int
	taskN    int32
	runningN int32
}

func (m Manager) IsPaused() bool {
	return m.isPaused
}

func (m Manager) NumTask() int {
	return int(m.taskN)
}

func (m Manager) WorkerState() (workerN, runningN int) {
	return m.workerN, int(m.runningN)
}

func (m *Manager) Pause() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.isPaused {
		return
	}

	m.isPaused = true
	m.pauseCh = make(chan struct{})
}

func (m *Manager) Resume() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if !m.isPaused {
		return
	}

	m.isPaused = false
	close(m.pauseCh)
}

func (m *Manager) AddTask(task Execer) {
	atomic.AddInt32(&m.taskN, 1)
	m.ch <- task
}

func (m *Manager) Run(ctx context.Context) {
	eg, egctx := errgroup.WithContext(ctx)

	m.Resume()

	for i := 0; i < m.workerN; i++ {
		eg.Go(func() error {
			for task := range m.ch {
				<-m.pauseCh
				atomic.AddInt32(&m.runningN, 1)

				select {
				case <-egctx.Done():
					atomic.AddInt32(&m.runningN, -1)
					atomic.AddInt32(&m.taskN, -1)
					return egctx.Err()
				default:
					func() {
						defer func() {
							if err := recover(); err != nil {
								task.ErrHandle(ErrPanic{err})
							}
						}()

						if err := task.Exec(ctx); err != nil {
							task.ErrHandle(err)
						}
					}()
				}

				atomic.AddInt32(&m.runningN, -1)
				atomic.AddInt32(&m.taskN, -1)
			}

			return nil
		})
	}

	eg.Wait()
}

func NewManager(workerN, bufferN int) *Manager {
	return &Manager{
		ch:       make(chan Execer, bufferN),
		pauseCh:  make(chan struct{}),
		isPaused: true,
		workerN:  workerN,
		taskN:    0,
		runningN: 0,
	}
}
