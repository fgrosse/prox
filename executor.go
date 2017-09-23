package prox

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	statusSuccess status = iota
	statusError
	statusInterrupted
)

// An Executor manages a set of processes. It is responsible for running them
// concurrently and waits until they have finished or an error occurs.
type Executor struct {
	log      *zap.Logger
	running  map[string]Process
	messages chan message
	output   *output
}

type message struct {
	p      Process
	status status
	err    error
}

type status int

// NewExecutor creates a new Executor. The debug flag controls whether debug
// logging should be activated. If debug is false then only warnings and errors
// will be logged.
func NewExecutor(debug bool) *Executor {
	return &Executor{
		log:    logger(debug),
		output: newOutput(),
	}
}

// Run starts all processes and blocks until all processes have finished or the
// context is done (e.g. canceled). If a process crashes or the context is
// canceled early, all running processes receive an interrupt signal.
func (e *Executor) Run(ctx context.Context, processes []Process) error {

	ctx, cancel := context.WithCancel(ctx)
	e.startAll(ctx, processes)
	return e.waitForAll(cancel)
}

func (e *Executor) monitorContext(ctx context.Context) {
	<-ctx.Done()
	if ctx.Err() == context.Canceled {
		e.output.Write([]byte("Received interrupt signal"))
	}
}

func (e *Executor) startAll(ctx context.Context, pp []Process) {
	e.log.Info("Starting processes", zap.Int("amount", len(pp)))

	e.running = map[string]Process{}
	e.messages = make(chan message)

	n := longestName(pp)
	for _, p := range pp {
		e.running[p.Name()] = p
		go e.run(ctx, p, n)
	}
}

func longestName(pp []Process) int {
	var longest string
	for _, p := range pp {
		if n := p.Name(); len(n) > len(longest) {
			longest = n
		}
	}

	n := len(longest)
	if n < 8 {
		n = 8
	}

	return n
}

func (e *Executor) run(ctx context.Context, p Process, longestName int) {
	name := p.Name()
	e.log.Info("Starting process", zap.String("name", name))

	output := e.output.next(name, longestName)
	logger := e.log.With(zap.String("process", name))
	err := p.Run(ctx, output, logger)

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

func (e *Executor) waitForAll(interruptAll func()) error {
	var firstErr error
	for len(e.running) > 0 {
		e.log.Debug("Waiting for processes to complete", zap.Int("amount", len(e.running)))

		message := <-e.messages
		name := message.p.Name() // TODO what if names collide?
		delete(e.running, name)

		switch message.status {
		case statusSuccess:
			e.log.Info("Task finished successfully", zap.String("name", message.p.Name()))
		case statusInterrupted:
			e.log.Info("Task was interrupted", zap.String("name", message.p.Name()))
		case statusError:
			e.log.Error("Task error", zap.String("name", message.p.Name()), zap.Error(message.err))
			if firstErr == nil {
				firstErr = message.err
			}
			interruptAll()
		}
	}

	return errors.Wrap(firstErr, "first error")
}
