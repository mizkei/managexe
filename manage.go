package managexe

import (
	"context"
	"fmt"
	"sync"

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
}

func (m *Manager) NumTask() int {
	return len(m.ch)
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
	m.ch <- task
}

func (m *Manager) Run(ctx context.Context) {
	eg, egctx := errgroup.WithContext(ctx)

	m.Resume()

	for i := 0; i < m.workerN; i++ {
		eg.Go(func() error {
			for task := range m.ch {
				<-m.pauseCh

				select {
				case <-egctx.Done():
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
	}
}
