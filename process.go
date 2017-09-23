package prox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// A Process is an abstraction of a child process which is started by the
// Executor.
type Process interface {
	Name() string
	Run(context.Context, io.Writer, *zap.Logger) error
}

// process is a Process implementation that uses os/exec to start system
// processes.
type process struct {
	name   string
	script string
	env    Environment

	interruptTimeout time.Duration

	mu  sync.Mutex
	cmd *exec.Cmd
}

// NewProcess creates a new Process that executes the given script as a new
// system process (using os/exec).
func NewProcess(name, script string, env Environment) Process {
	return &process{
		script:           script,
		name:             name,
		interruptTimeout: 5 * time.Second,
		env:              env,
	}
}

// Name returns the human readable name of p that can be used to identify a
// specific process.
func (p *process) Name() string {
	return p.name
}

// Run starts the shell process and blocks until it finishes or the context is
// done. The given io.Writer receives all output (both stdout and stderr) of the
// process.
func (p *process) Run(ctx context.Context, output io.Writer, logger *zap.Logger) error {
	p.mu.Lock()

	if logger == nil {
		logger = zap.NewNop()
	}

	commandLine := p.buildCommandLine()
	logger.Debug("Starting new shell process", zap.String("script", commandLine))

	cmdParts := strings.Fields(commandLine)
	p.cmd = exec.Command(cmdParts[0], cmdParts[1:]...)

	// TODO: add current PWD to PATH

	p.cmd.Stdout = output
	p.cmd.Stderr = output
	p.cmd.Env = p.env.List()

	err := p.cmd.Start()
	p.mu.Unlock()

	if err != nil {
		return fmt.Errorf("could not start shell task: %s", err)
	}

	return p.wait(ctx, logger)
}

func (p *process) wait(ctx context.Context, logger *zap.Logger) error {
	done := make(chan error)
	go func() {
		done <- p.cmd.Wait()
	}()

	// n.b. By default child processes are often started in the same
	// process group as the parent. Under these circumstances the shell
	// will send the signal to all processes, causing them to terminate on
	// their own. We cannot rely on this behavior but we should not report
	// an error if the process has already finished before we asked it to.

	select {
	case err := <-done:
		if err != nil && strings.HasPrefix(err.Error(), "signal: ") {
			// see note from above...
			err = nil
		}
		return err
	case <-ctx.Done():
		if p.cmd.ProcessState != nil && p.cmd.ProcessState.Exited() {
			// There is nothing to do anymore so we can return early.
			return ctx.Err()
		}

		logger.Info("Sending interrupt signal", zap.Duration("timeout", p.interruptTimeout))

		/*
			TODO: to kill all child processes as well try this:
			group, err := os.FindProcess(-1 * p.Process.Pid)
			if err == nil {
				err = group.Signal(signal)
			}
		*/

		// TODO: this results in our child processes to receive SIGINT twice, due to the process group issue (e.g. visible in redis)
		err := p.cmd.Process.Signal(syscall.SIGINT)
		if err != nil && err.Error() != "os: process already finished" {
			logger.Error("Failed to send SIGINT to process", zap.Error(err))
			p.cmd.Process.Kill()
			return ctx.Err()
		}

		select {
		case <-done:
			logger.Info("Process interrupted successfully", zap.Error(err))
		case <-time.After(p.interruptTimeout):
			err := p.cmd.Process.Kill()
			if err != nil {
				logger.Error("Failed to kill process", zap.Error(err))
			}
		}

		return ctx.Err()
	}
}

func (p *process) buildCommandLine() string {
	script := p.env.Expand(p.script)

	r := regexp.MustCompile(`[a-zA-Z_]+=\S+`)

	b := new(bytes.Buffer)
	parts := strings.Fields(script) // TODO breaks if we have quotes spaces

	var done bool
	for _, part := range parts {
		match := r.FindString(part)
		if done == false && match != "" {
			p.env.Set(match)
		} else {
			done = true
		}

		if done {
			b.WriteString(part)
			b.WriteString(" ")
		}
	}

	return strings.TrimSpace(b.String())
}
