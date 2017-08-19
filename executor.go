package prox

import (
	"fmt"
	"time"

	"context"

	"go.uber.org/zap"
)

const (
	statusSuccess status = iota
	statusError
	statusInterrupted
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
func (e *Executor) Run(processes []Process) error { // TODO: pass context
	ctx, cancel := context.WithCancel(context.TODO())
	e.startAll(ctx, processes)
	e.waitForAll(cancel)
	return nil
}

func (e *Executor) startAll(ctx context.Context, pp []Process) {
	e.log.Info("Starting processes", zap.Int("amount", len(pp)))

	e.running = map[string]Process{}
	e.messages = make(chan message)

	for _, p := range pp {
		e.running[p.Name()] = p
		go e.run(ctx, p)
	}
}

func (e *Executor) run(ctx context.Context, p Process) {
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
	err := p.Run(ctx)

	var result status
	switch {
	case err == context.Canceled:
		result = statusInterrupted
	case err != nil:
		result = statusError
	default:
		result = statusSuccess
	}

	e.messages <- message{p: p, status: result, err: err}
}

func (e *Executor) waitForAll(interruptAll func()) {
	for len(e.running) > 0 {
		e.log.Debug("Waiting for processes to complete", zap.Int("amount", len(e.running)))

		message := <-e.messages
		name := message.p.Name() // TODO what if names collide?
		delete(e.running, name)

		switch message.status {
		case statusSuccess:
			e.log.Info("Task finished successfully", zap.String("name", message.p.Name()))
		case statusInterrupted:
			e.log.Error("Task was interrupted", zap.String("name", message.p.Name()), zap.Error(message.err))
		case statusError:
			e.log.Error("Task error", zap.String("name", message.p.Name()), zap.Error(message.err))
			interruptAll()
		}
	}
}
