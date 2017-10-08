package prox

import (
	"context"
	"io"
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// An Executor manages a set of processes. It is responsible for running them
// concurrently and waits until they have finished or an error occurs.
type Executor struct {
	output       io.Writer
	debug        bool
	noColors     bool
	proxLogColor color
	running      map[string]process
	outputs      map[string]*processOutput
	messages     chan message
}

type Process struct {
	Name       string
	Script     string
	Env        Environment
	JSONOutput bool
}

// messages are passed to signal that a specific process has finished along with
// its reason for termination (i.e. status). For each started process we expect
// a single message to eventually be sent to the Executor.
type message struct {
	p      process
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
		output:       os.Stdout,
		debug:        debug,
		proxLogColor: colorWhite,
		running:      map[string]process{},
		outputs:      map[string]*processOutput{},
		messages:     make(chan message),
	}
}

// DisableColoredOutput disables colored prefixes in the output.
func (e *Executor) DisableColoredOutput() {
	e.noColors = true
	e.proxLogColor = colorNone
}

// Run starts all processes and blocks until all processes have finished or the
// context is done (e.g. canceled). If a process crashes or the context is
// canceled early, all running processes receive an interrupt signal.
func (e *Executor) Run(ctx context.Context, processes []Process) error {
	output := newOutput(processes, e.noColors, e.output)
	out := output.nextColored("prox", e.proxLogColor)
	logger := NewLogger(out, e.debug)

	// make sure all log output is flushed before we leave this function
	defer logger.Sync()
	go e.monitorContext(ctx, logger)

	pp := make([]process, len(processes))
	for i, p := range processes {
		po := output.next(p.Name)
		e.outputs[p.Name] = po
		log := logger.With(zap.String("process", p.Name))
		pp[i] = newSystemProcess(p.Name, p.Script, p.Env, po, log)
	}

	return e.run(ctx, pp, logger)
}

func (e *Executor) monitorContext(ctx context.Context, log *zap.Logger) {
	<-ctx.Done()
	if ctx.Err() == context.Canceled {
		log.Info("Received interrupt signal")
	}
}

func (e *Executor) run(ctx context.Context, processes []process, logger *zap.Logger) error {
	ctx, cancel := context.WithCancel(ctx)
	e.startAll(ctx, processes, logger)
	return e.waitForAll(cancel, logger)
}

// StartAll starts all processes in a separate goroutine and then returns
// immediately.
func (e *Executor) startAll(ctx context.Context, pp []process, logger *zap.Logger) {
	logger.Info("Starting processes", zap.Int("amount", len(pp)))
	for _, p := range pp {
		name := p.Name()
		e.running[name] = p

		go func(p process) {
			logger.Info("Starting process", zap.String("process_name", p.Name()))
			e.runProcess(ctx, p)
		}(p)
	}
}

// runProcess starts a single process and blocks until it has completed or failed.
func (e *Executor) runProcess(ctx context.Context, p process) {
	var result status
	err := p.Run(ctx)

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

func (e *Executor) waitForAll(interruptAll func(), logger *zap.Logger) error {
	var firstErr error
	var firstErrProcess string
	for len(e.running) > 0 {
		logger.Debug("Waiting for processes to complete", zap.Int("amount", len(e.running)))

		message := <-e.messages
		name := message.p.Name() // TODO what if names collide?
		delete(e.running, name)

		switch message.status {
		case statusSuccess:
			logger.Info("Process finished successfully", zap.String("process_name", message.p.Name()))
		case statusInterrupted:
			logger.Info("Process was interrupted", zap.String("process_name", message.p.Name()))
		case statusError:
			logger.Error("Process error", zap.String("process_name", message.p.Name()), zap.Error(message.err))
			if firstErr == nil {
				firstErr = message.err
				firstErrProcess = message.p.Name()
			}
			interruptAll()
		}
	}

	if firstErr != nil {
		logger.Error("Stopped due to error in process",
			zap.String("process_name", firstErrProcess),
			zap.Error(firstErr),
		)
	}

	return errors.Wrap(firstErr, "first error")
}

// Info returns information about a running process. If there is no such process
// running process a ProcessInfo with a PID of -1 is returned.
func (e *Executor) Info(processName string) ProcessInfo {
	p, ok := e.running[processName]
	if !ok {
		return ProcessInfo{PID: -1}
	}

	inf := p.Info()
	inf.Name = processName
	return inf
}
