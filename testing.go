package prox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/fgrosse/zaptest"
	"go.uber.org/zap"
)

type TestExecutor struct {
	*Executor
	Error        error
	mu           sync.RWMutex
	executorDone bool
}

func TestNewExecutor(w io.Writer) *TestExecutor {
	e := &TestExecutor{Executor: NewExecutor(true)}
	e.log = zaptest.LoggerWriter(w)

	return e
}

func (e *TestExecutor) Run(processes ...Process) {
	e.mu.Lock()
	e.executorDone = false
	e.mu.Unlock()

	e.log.Info("Executor starting")
	ctx := context.Background()
	e.Error = e.Executor.Run(ctx, processes)
	e.log.Info("Executor finished")

	e.mu.Lock()
	e.executorDone = true
	e.mu.Unlock()
}

func (e *TestExecutor) IsDone() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.executorDone
}

type TestProcess struct {
	name        string // TODO: make settable from the outside
	mu          sync.Mutex
	started     bool
	interrupted bool

	finish chan chan bool // bool instead of struct{} for better readability of test code
	fail   chan chan bool

	interruptFinisher chan bool // to optionally block on Interrupt calls
}

func (t *TestProcess) Name() string {
	return t.name
}

func (t *TestProcess) String() string {
	return t.Name()
}

func (t *TestProcess) Run(ctx context.Context, _ io.Writer, _ *zap.Logger) error { // TODO: use ctx
	t.mu.Lock()
	if t.started {
		return errors.New("started multiple times")
	}

	t.started = true
	t.interrupted = false
	t.finish = make(chan chan bool)
	t.fail = make(chan chan bool)
	t.mu.Unlock()

	select {
	case <-ctx.Done():
		if t.interruptFinisher != nil {
			<-t.interruptFinisher
		}
		t.mu.Lock()
		t.interrupted = true
		t.mu.Unlock()
		return ctx.Err()
	case c := <-t.finish:
		c <- true
		return nil
	case c := <-t.fail:
		c <- true
		return fmt.Errorf("TestProcess simulated a failure")
	}
}

func (t *TestProcess) Finish() {
	t.signal(t.finish)
}

func (t *TestProcess) Fail() {
	t.signal(t.fail)
}

func (t *TestProcess) signal(c chan chan bool) {
	syncChan := make(chan bool)
	select {
	case c <- syncChan:
		// sync with the TestProcess.Run goroutine
		<-syncChan
	default:
		// process is not running
	}
}

func (t *TestProcess) HasBeenStarted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.started
}

func (t *TestProcess) HasBeenInterrupted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.interrupted
}

func (t *TestProcess) ShouldBlockOnInterrupt() {
	t.interruptFinisher = make(chan bool)
}

func (t *TestProcess) FinishInterrupt() {
	t.interruptFinisher <- true
}
