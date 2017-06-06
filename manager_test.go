package managexe

import (
	"context"
	"testing"
	"time"
)

type testA struct {
	ch chan struct{}
	fn func()
}

func (a *testA) wait() {
	<-a.ch
}

func (a *testA) Exec(ctx context.Context) error {
	time.Sleep(1 * time.Second)
	if a.fn != nil {
		a.fn()
	}
	close(a.ch)
	return nil
}

func (a *testA) ErrHandle(_ error) {}

func TestRun(t *testing.T) {
	manager := NewManager(3, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go manager.Run(ctx)

	start := time.Now()
	list := []*testA{
		{make(chan struct{}), func() {}},
		{make(chan struct{}), func() {}},
		{make(chan struct{}), func() {}},
	}
	for _, a := range list {
		manager.AddTask(a)
	}

	if n := manager.NumTask(); n != 3 {
		t.Errorf("[error] number of task. got %d, wont %d", n, 3)
		return
	}

	for _, a := range list {
		a.wait()
	}
	end := time.Now()

	t.Logf("execution time: %s", end.Sub(start))
}

func TestPauseAndResume(t *testing.T) {
	manager := NewManager(3, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go manager.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	manager.Pause()

	ch := make(chan struct{})
	count := 0
	go func() {
		for range ch {
			count++
		}
	}()

	list := []*testA{
		{make(chan struct{}), func() { ch <- struct{}{} }},
		{make(chan struct{}), func() { ch <- struct{}{} }},
		{make(chan struct{}), func() { ch <- struct{}{} }},
	}
	for _, a := range list {
		manager.AddTask(a)
	}

	time.Sleep(2 * time.Second)

	if count != 0 {
		t.Errorf("[error] count. got %d, wont %d", count, 0)
		return
	}
	if wn, rn := manager.WorkerState(); wn != 3 || rn != 0 {
		t.Errorf("[error] (workerN, runningN) . got (%d, %d), wont (%d, %d)", wn, rn, 3, 0)
		return
	}

	manager.Resume()

	time.Sleep(2 * time.Second)

	if count != 3 {
		t.Errorf("[error] count. got %d, wont %d", count, 3)
		return
	}

	for _, a := range list {
		a.wait()
	}
}
