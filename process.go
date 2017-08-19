package prox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"time"

	"syscall"

	"go.uber.org/zap"
)

type Process interface {
	Name() string
	Run(context.Context) error
}

type shellProcess struct {
	name   string
	script string
	env    Environment
	logger *zap.Logger
	writer io.Writer

	mu  sync.Mutex
	cmd *exec.Cmd
}

func NewShellProcess(name, script string) Process {
	return &shellProcess{
		script: script,
		name:   name,
		env:    SystemEnv(),
	}
}

func (p *shellProcess) Name() string {
	return p.name
}

func (p *shellProcess) Run(ctx context.Context) error {
	p.mu.Lock()

	if p.logger == nil {
		p.logger = zap.NewNop()
	}

	commandLine := p.buildCommandLine()
	p.logger.Debug("Starting process",
		zap.String("script", commandLine),
		zap.Strings("env", p.env.List()),
	)

	cmdParts := strings.Fields(commandLine)
	p.cmd = exec.Command(cmdParts[0], cmdParts[1:]...)

	p.cmd.Stdout = p.writer
	p.cmd.Stderr = p.writer
	p.cmd.Env = p.env.List()

	err := p.cmd.Start()
	p.mu.Unlock()

	if err != nil {
		return fmt.Errorf("could not start shell task: %s", err)
	}

	return p.wait(ctx)
}

func (p *shellProcess) wait(ctx context.Context) error {
	done := make(chan error)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		p.logger.Info("Sending interrupt signal")
		err := p.cmd.Process.Signal(syscall.SIGINT)
		if err != nil {
			p.logger.Error("Failed to send SIGINT to process", zap.Error(err))
			p.cmd.Process.Kill()
			return ctx.Err()
		}

		select {
		case <-done:
			p.logger.Info("Process interrupted successfully", zap.Error(err))
		case <-time.After(time.Second): // TODO: make configurable
			err := p.cmd.Process.Kill()
			if err != nil {
				p.logger.Error("Failed to kill process", zap.Error(err))
			}
		}

		return ctx.Err()
	}
}

func (p *shellProcess) buildCommandLine() string {
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

func (p *shellProcess) Interrupt() error {
	return errors.New("NOT IMPLEMENTED")
}
