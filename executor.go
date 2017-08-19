package prox

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	statusSuccess status = iota
	statusError
	statusTimeout
)

type status int

type Executor struct {
	TaskInterruptTimeout time.Duration

	log      *zap.Logger
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
		log:                  logger(""),
	}
}

// Start starts all processes and blocks all tasks processes finish.
func (e *Executor) Run(processes []Process) error {
	e.startAll(processes)
	e.waitForAll()
	return nil
}

func (e *Executor) startAll(pp []Process) {
	e.log.Info("Starting processes", zap.Int("amount", len(pp)))

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

	e.log.Info("Starting process", zap.String("name", p.Name()))
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
		e.log.Debug("Waiting for processes to complete", zap.Int("amount", len(e.running)))

		message := <-e.messages
		name := message.p.Name() // TODO what if names collide?
		delete(e.running, name)

		switch message.status {
		case statusSuccess:
			e.log.Info("Task finished successfully", zap.String("name", message.p.Name()))
		case statusTimeout:
			e.log.Error("Task timeout", zap.String("name", message.p.Name()), zap.Error(message.err))
		case statusError:
			e.log.Error("Task error", zap.String("name", message.p.Name()), zap.Error(message.err))
			e.interruptAll(e.running)
		}
	}
}

func (e *Executor) interruptAll(pp map[string]Process) {
	e.log.Info("Interrupting all processes")
	for _, p := range pp {
		go e.interrupt(p)
	}
}

func (e *Executor) interrupt(p Process) {
	e.log.Info("Interrupting process", zap.String("name", p.Name()))
	done := make(chan struct{})

	go func() {
		e.log.Debug("Sending interrupt request to process", zap.String("name", p.Name()))
		err := p.Interrupt()
		e.log.Debug("Interrupt response from", zap.String("name", p.Name()), zap.Error(err))
		if err != nil {
			e.log.Error("Error while interrupting process", zap.String("name", p.Name()), zap.Error(err))
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
		e.log.Info("Process was interrupted successfully", zap.String("name", p.Name()))
		e.messages <- message{p: p, status: statusSuccess}
	case <-time.After(e.TaskInterruptTimeout):
		e.messages <- message{p: p, status: statusError,
			err: fmt.Errorf("did not respond to interrupt in time (waited %s)", e.TaskInterruptTimeout),
		}
	}
}
