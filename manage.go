package managexe

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

type ErrPanic struct {
	Err interface{}
}

func (ep ErrPanic) Error() string {
	return fmt.Sprint(ep.Err)
}

type Execer interface {
	Exec() error
}

type Manager struct {
	ch      chan Execer
	errCh   chan error
	workerN int
}

func (m *Manager) NumTask() int {
	return len(m.ch)
}

func (m *Manager) ErrCh() <-chan error {
	return m.errCh
}

func (m *Manager) AddTask(task Execer) {
	m.ch <- task
}

func (m *Manager) Run(ctx context.Context) {
	eg, egctx := errgroup.WithContext(ctx)

	for i := 0; i < m.workerN; i++ {
		eg.Go(func() error {
			for task := range m.ch {
				select {
				case <-egctx.Done():
					return egctx.Err()
				default:
					func() {
						defer func() {
							if err := recover(); err != nil {
								m.errCh <- ErrPanic{err}
							}
						}()

						if err := task.Exec(); err != nil {
							// TODO: error
							m.errCh <- err
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
		ch:      make(chan Execer, bufferN),
		errCh:   make(chan error, bufferN),
		workerN: workerN,
	}
}
