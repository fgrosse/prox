package prox

import (
	"context"
	"fmt"
	"io"
	"os"

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
	output   io.Writer
	colors   *colorProvider
}

type message struct {
	p      Process
	status status
	err    error
}

type status int

// NewExecutor creates a new Executor.
func NewExecutor() *Executor {
	return &Executor{
		log:    logger(""),
		output: os.Stdout,
		colors: newColorProvider(),
	}
}

// Start starts all processes and blocks until all processes have finished or
// the context is done (e.g. canceled). If a process crashes or the context is
// canceled early, all running processes receive an interrupt signal.
func (e *Executor) Run(ctx context.Context, processes []Process) error {
	ctx, cancel := context.WithCancel(ctx)
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

	name := p.Name()
	e.log.Info("Starting process", zap.String("name", name))

	output := &processOutput{
		Writer: e.output,
		name:   name,
		color:  e.colors.next(),
	}

	logger := e.log.Named(name)
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
			e.log.Info("Task was interrupted", zap.String("name", message.p.Name()))
		case statusError:
			e.log.Error("Task error", zap.String("name", message.p.Name()), zap.Error(message.err))
			interruptAll()
		}
	}
}

type processOutput struct {
	io.Writer
	name  string
	color color
}

func (o *processOutput) Write(p []byte) (int, error) {
	fmt.Fprint(o.Writer, o.color)
	n, err := o.Writer.Write(p)
	fmt.Fprint(o.Writer, colorDefault)

	return n, err
}
