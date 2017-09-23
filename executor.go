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
		debug: debug,
	}
}

// Run starts all processes and blocks until all processes have finished or the
// context is done (e.g. canceled). If a process crashes or the context is
// canceled early, all running processes receive an interrupt signal.
func (e *Executor) Run(ctx context.Context, processes []Process) error {
	output := newOutput(processes)

	if e.log == nil {
		out := output.nextColored("prox", colorWhite)
		e.log = logger(out, e.debug)
	}

	// make sure all log output is flushed before we leave this function
	defer e.log.Sync()

	go e.monitorContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	e.startAll(ctx, processes, output)
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
func (e *Executor) startAll(ctx context.Context, pp []Process, output *output) {
	e.log.Info("Starting processes", zap.Int("amount", len(pp)))

	e.running = map[string]Process{}
	e.messages = make(chan message)

	for _, p := range pp {
		name := p.Name()
		e.running[name] = p
		output := output.next(name)
		go e.run(ctx, p, output)
	}
}

// Run starts a single process and blocks until it has completed or failed.
func (e *Executor) run(ctx context.Context, p Process, output *processOutput) {
	name := p.Name()
	e.log.Info("Starting process", zap.String("process_name", name))

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
	var firstErrProcess string
	for len(e.running) > 0 {
		e.log.Debug("Waiting for processes to complete", zap.Int("amount", len(e.running)))

		message := <-e.messages
		name := message.p.Name() // TODO what if names collide?
		delete(e.running, name)

		switch message.status {
		case statusSuccess:
			e.log.Info("Process finished successfully", zap.String("process_name", message.p.Name()))
		case statusInterrupted:
			e.log.Info("Process was interrupted", zap.String("process_name", message.p.Name()))
		case statusError:
			e.log.Error("Process error", zap.String("process_name", message.p.Name()), zap.Error(message.err))
			if firstErr == nil {
				firstErr = message.err
				firstErrProcess = message.p.Name()
			}
			interruptAll()
		}
	}

	if firstErr != nil {
		e.log.Error("Stopped due to error in process",
			zap.String("process_name", firstErrProcess),
			zap.Error(firstErr),
		)
	}

	return errors.Wrap(firstErr, "first error")
}
