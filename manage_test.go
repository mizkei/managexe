package managexe

import (
	"context"
	"testing"
	"time"
)

type testA struct {
	ch chan struct{}
}

func (a *testA) wait() {
	<-a.ch
}

func (a *testA) Exec(ctx context.Context) error {
	time.Sleep(3 * time.Second)
	close(a.ch)
	return nil
}

func TestRun(t *testing.T) {
	manager := NewManager(3, 100)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go manager.Run(ctx)

	start := time.Now()
	list := []*testA{{make(chan struct{})}, {make(chan struct{})}, {make(chan struct{})}}
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
