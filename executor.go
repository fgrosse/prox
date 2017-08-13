package prox

import (
	"fmt"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

const (
	statusSuccess status = iota
	statusError
	statusTimeout
)

type status int

type Executor struct {
	TaskInterruptTimeout time.Duration

	log      *logrus.Logger
	running  map[string]Process
	messages chan message
}

type message struct {
	p      Process
	status status
	err    error
}

func NewExecutor() *Executor {
	return &Executor{
		TaskInterruptTimeout: 5 * time.Second,
		log:                  logrus.New(),
	}
}

// Start starts all processes and blocks all tasks processes finish.
func (e *Executor) Run(processes []Process) error {
	e.startAll(processes)
	e.waitForAll()
	return nil
}

func (e *Executor) startAll(pp []Process) {
	e.log.Debug("Starting %d processes", len(pp))

	e.running = map[string]Process{}
	e.messages = make(chan message)

	startUp := new(sync.WaitGroup)
	for _, p := range pp {
		e.running[p.Name()] = p
		startUp.Add(1)
		go e.run(p, startUp)
	}

	startUp.Wait()
}

func (e *Executor) run(p Process, startUp *sync.WaitGroup) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("%v", e)
			}
			e.messages <- message{p, statusError, err}
		}
	}()

	e.log.Printf("Starting process %q", p)
	startUp.Done()

	var result status
	err := p.Run()
	if err != nil {
		result = statusError
	}

	e.messages <- message{p: p, status: result, err: err}
}

func (e *Executor) waitForAll() {
	for len(e.running) > 0 {
		e.log.Printf("Waiting for %d processes to complete", len(e.running))

		message := <-e.messages
		name := message.p.Name() // TODO what if names collide?
		delete(e.running, name)

		switch message.status {
		case statusSuccess:
			e.log.Printf("Task %q finished successfully", message.p)
		case statusTimeout:
			e.log.Printf("Task %q: %v", message.p, message.err)
		case statusError:
			e.log.Printf("Task %q: %v", message.p, message.err)
			e.interruptAll(e.running)
		}
	}
}

func (e *Executor) interruptAll(pp map[string]Process) {
	e.log.Println("Interrupting all processes")
	for _, p := range pp {
		go e.interrupt(p)
	}
}

func (e *Executor) interrupt(p Process) {
	e.log.Printf("Interrupting process %q", p)
	done := make(chan struct{})

	go func() {
		e.log.Printf("Sending interrupt request to %q", p)
		err := p.Interrupt()
		e.log.Printf("Interrupt response from %q: %v", p, err)
		if err != nil {
			e.log.Printf("Error while interrupting %q: %s", p, err)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
		e.log.Printf("Process %q was interrupted successfully", p)
		e.messages <- message{p: p, status: statusSuccess}
	case <-time.After(e.TaskInterruptTimeout):
		e.messages <- message{p: p, status: statusError,
			err: fmt.Errorf("did not respond to interrupt in time (waited %s)", e.TaskInterruptTimeout),
		}
	}
}
