package prox

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// An Executor manages a set of processes. It is responsible for running them
// concurrently and waits until they have finished or an error occurs.
type Executor struct {
	debug    bool
	log      *zap.Logger
	running  map[string]Process
	messages chan message
	output   *output
}

// messages are passed to signal that a specific process has finished along with
// its reason for termination (i.e. status). For each started process we expect
// a single message to eventually be sent to the Executor.
type message struct {
	p      Process
	status status
	err    error
}

// A status indicates why a process has finished.
type status int

const (
	statusSuccess     status = iota // process finished with error code 0
	statusError                     // process failed with some error
	statusInterrupted               // process was cancelled because the context interrupted
)

// NewExecutor creates a new Executor. The debug flag controls whether debug
// logging should be activated. If debug is false then only warnings and errors
// will be logged.
func NewExecutor(debug bool) *Executor {
	return &Executor{
		debug:  debug,
		output: newOutput(),
	}
}

// Run starts all processes and blocks until all processes have finished or the
// context is done (e.g. canceled). If a process crashes or the context is
// canceled early, all running processes receive an interrupt signal.
func (e *Executor) Run(ctx context.Context, processes []Process) error {
	n := longestName(processes)

	if e.log == nil {
		out := newProcessOutput("prox", n, colorWhite, e.output)
		e.log = logger(out, e.debug)
	}

	go e.monitorContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	e.startAll(ctx, processes, n)
	return e.waitForAll(cancel)
}

func (e *Executor) monitorContext(ctx context.Context) {
	<-ctx.Done()
	if ctx.Err() == context.Canceled {
		e.log.Info("Received interrupt signal")
	}
}

// StartAll starts all processes in a separate goroutine and then returns
// immediately.
func (e *Executor) startAll(ctx context.Context, pp []Process, longestName int) {
	e.log.Info("Starting processes", zap.Int("amount", len(pp)))

	e.running = map[string]Process{}
	e.messages = make(chan message)

	for _, p := range pp {
		name := p.Name()
		e.running[name] = p
		output := e.output.next(name, longestName)
		go e.run(ctx, p, output)
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

// Run starts a single process and blocks until it has completed or failed.
func (e *Executor) run(ctx context.Context, p Process, output *processOutput) {
	name := p.Name()
	e.log.Info("Starting process", zap.String("name", name))

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
