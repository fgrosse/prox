package prox

import (
	"errors"
	"fmt"
)

type TestProcess struct {
	name        string // TODO: make settable from the outside
	started     bool
	interrupted bool

	finish    chan chan bool // bool instead of struct{} for better readability of test code
	interrupt chan chan bool
	fail      chan chan bool
	panic     chan chan bool

	interruptFinisher chan bool // to optionally block on Interrupt calls
}

func (t *TestProcess) Name() string {
	return t.name
}

func (t *TestProcess) String() string {
	return t.Name()
}

func (t *TestProcess) Run() error {
	if t.started {
		return errors.New("started multiple times")
	}

	t.started = true
	t.interrupted = false
	t.finish = make(chan chan bool)
	t.fail = make(chan chan bool)
	t.panic = make(chan chan bool)

	select {
	case c := <-t.finish:
		c <- true
		return nil
	case c := <-t.interrupt:
		c <- true
		return nil
	case c := <-t.fail:
		c <- true
		return fmt.Errorf("TestProcess simulated a failure")
	case c := <-t.panic:
		c <- true
		panic(fmt.Errorf("TestProcess simulated a panic"))
	}
}

func (t *TestProcess) Interrupt() error {
	if t.interruptFinisher != nil {
		<-t.interruptFinisher
	}

	t.signal(t.interrupt)
	t.interrupted = true

	return nil
}

func (t *TestProcess) Finish() {
	t.signal(t.finish)
}

func (t *TestProcess) Fail() {
	t.signal(t.fail)
}

func (t *TestProcess) Panic() {
	t.signal(t.panic)
}

func (t *TestProcess) signal(c chan chan bool) {
	sync := make(chan bool)
	select {
	case c <- sync:
		// sync with the TestProcess.Run goroutine
		<-sync
	default:
		// process is not running
	}
}

func (t *TestProcess) HasBeenStarted() bool {
	return t.started
}

func (t *TestProcess) HasBeenInterrupted() bool {
	return t.interrupted
}

func (t *TestProcess) ShouldBlockOnInterrupt() {
	t.interruptFinisher = make(chan bool)
}

func (t *TestProcess) FinishInterrupt() {
	t.interruptFinisher <- true
}
