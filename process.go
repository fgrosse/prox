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

// Process holds all information about a process that is executed by prox.
type Process struct {
	Name   string
	Script string
	Env    Environment
	Output StructuredOutput // optional
}

// ProcessInfo contains information about a running process.
type ProcessInfo struct {
	Name   string
	PID    int
	Uptime time.Duration
}

// A process is an abstraction of a child process which is started by the
// Executor.
type process interface {
	Name() string
	Info() ProcessInfo
	Run(context.Context) error
}

// a systemProcess is a Process implementation that uses os/exec to start shell
// processes.
type systemProcess struct {
	name   string
	script string
	env    Environment
	output io.Writer
	logger *zap.Logger

	startedAt        time.Time
	interruptTimeout time.Duration

	mu  sync.Mutex
	cmd *exec.Cmd
}

// newSystemProcess creates a new process that executes the given script as a
// new system process (using os/exec).
func newSystemProcess(name, script string, env Environment, output io.Writer, logger *zap.Logger) *systemProcess {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &systemProcess{
		script:           script,
		name:             name,
		interruptTimeout: 5 * time.Second,
		env:              env,
		output:           output,
		logger:           logger,
	}
}

// Name returns the human readable name of p that can be used to identify a
// specific process.
func (p *systemProcess) Name() string {
	return p.name
}

func (p *systemProcess) Info() ProcessInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return ProcessInfo{PID: -1}
	}

	return ProcessInfo{
		PID:    p.cmd.Process.Pid,
		Uptime: time.Since(p.startedAt),
	}
}

// Run starts the shell process and blocks until it finishes or the context is
// done. The systemProcess.output receives both the stdout and stderr output
// of the process.
func (p *systemProcess) Run(ctx context.Context) error {
	p.mu.Lock()

	commandLine := p.buildCommandLine()
	p.logger.Debug("Starting new shell process", zap.String("script", commandLine))

	cmdParts := strings.Fields(commandLine)
	p.cmd = exec.Command(cmdParts[0], cmdParts[1:]...)

	p.cmd.Stdout = p.output
	p.cmd.Stderr = p.output
	p.cmd.Env = p.env.List()

	p.startedAt = time.Now()
	err := p.cmd.Start()
	p.mu.Unlock()

	if err != nil {
		return fmt.Errorf("could not start shell task: %s", err)
	}

	return p.wait(ctx)
}

func (p *systemProcess) wait(ctx context.Context) error {
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

		p.logger.Info("Sending interrupt signal", zap.Duration("timeout", p.interruptTimeout))

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
			p.logger.Error("Failed to send SIGINT to process", zap.Error(err))
			p.cmd.Process.Kill()
			return ctx.Err()
		}

		select {
		case <-done:
			p.logger.Debug("Process interrupted successfully", zap.Error(err))
		case <-time.After(p.interruptTimeout):
			err := p.cmd.Process.Kill()
			if err != nil {
				p.logger.Error("Failed to kill process", zap.Error(err))
			}
		}

		return ctx.Err()
	}
}

func (p *systemProcess) buildCommandLine() string {
	script := p.env.Expand(p.script)

	r := regexp.MustCompile(`[a-zA-Z_]+=\S+`)

	b := new(bytes.Buffer)
	parts := strings.Fields(script) // TODO breaks if we have quoted spaces

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

// CommandLine returns the shell command line that would be executed when the
// given Process is started.
func (p Process) CommandLine() string {
	sp := newSystemProcess(p.Name, p.Script, p.Env, nil, nil)
	return sp.buildCommandLine()
}
