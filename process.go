package prox

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
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

// Validate checks if all given processes are valid and no process name is used
// multiple times. If an error is returned it will be a multierror.
func Validate(pp []Process) error {
	errs := newMultiError()
	seen := map[string]struct{}{}
	for i, p := range pp {
		if _, ok := seen[p.Name]; ok {
			errs = multierror.Append(errs, errors.Errorf("process %d: name %q is already used", i+1, p.Name))
		}
		seen[p.Name] = struct{}{}

		err := p.Validate()
		if err == nil {
			continue
		}

		id := fmt.Sprintf("%q", strings.TrimSpace(p.Name))
		if id == `""` {
			id = fmt.Sprint(i + 1)
		}

		for _, err := range err.(*multierror.Error).Errors {
			errs = multierror.Append(errs, errors.Wrap(err, "process "+id))
		}
	}

	return errs.ErrorOrNil()
}

// Validate checks that the Process configuration is complete and without errors.
// If an error is returned it will be a multierror.
func (p Process) Validate() error {
	errs := newMultiError()

	if strings.TrimSpace(p.Name) == "" {
		errs = multierror.Append(errs, errors.New("missing name"))
	}

	if strings.TrimSpace(p.Script) == "" {
		errs = multierror.Append(errs, errors.New("missing script"))
	}

	switch p.Output.Format {
	case "", "auto":
		// using default values, nothing to check
	case "json":
		if p.Output.MessageField == "" {
			errs = multierror.Append(errs, errors.New(`missing log output "message" field`))
		}
		if p.Output.LevelField == "" {
			errs = multierror.Append(errs, errors.New(`missing log output "level" field`))
		}
	default:
		errs = multierror.Append(errs, errors.Errorf("unknown log output format %q", p.Output.Format))
	}

	return errs.ErrorOrNil()
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

	args, err := p.parseCommandLine()
	if err != nil {
		return errors.Wrap(err, "failed to parse command line")
	}

	p.logger.Debug("Starting new shell process", zap.Strings("script", args))
	p.cmd = exec.Command("env", args...)

	p.cmd.Stdout = p.output
	p.cmd.Stderr = p.output
	p.cmd.Env = p.env.List()

	p.startedAt = time.Now()
	err = p.cmd.Start()
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
			// TODO: this seems fishy since it also hides issues with broken output such as "signal: broken pipe"
			// also matching errors based on string prefixes is pretty much an anti-pattern
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

func (p *systemProcess) parseCommandLine() ([]string, error) {
	var (
		args         []string
		buf          string
		escaped      bool
		doubleQuoted bool
		singleQuoted bool
	)

	for _, r := range p.script {
		switch {
		case escaped:
			buf += string(r)
			escaped = false

		case r == '\\' && !singleQuoted:
			escaped = true

		case unicode.IsSpace(r) && (singleQuoted || doubleQuoted):
			buf += string(r)

		case unicode.IsSpace(r) && buf != "":
			args = append(args, buf)
			buf = ""

		case unicode.IsSpace(r) && buf == "":
			// collapse (i.e. ignore) multiple spaces between arguments
			continue

		case r == '"' && !singleQuoted:
			doubleQuoted = !doubleQuoted

		case r == '\'' && !doubleQuoted:
			singleQuoted = !singleQuoted

		case r == ';', r == '&', r == '|', r == '<', r == '>':
			if !(escaped || singleQuoted || doubleQuoted) {
				return nil, errors.Errorf("command redirection or piping is not supported(got %v)", r)
			} else {
				buf += string(r)
			}

		default:
			buf += string(r)
		}
	}

	switch {
	case escaped:
		return nil, errors.New("bad escape at the end")
	case singleQuoted:
		return nil, errors.New("unclosed single quote")
	case doubleQuoted:
		return nil, errors.New("unclosed double quote")
	}

	if buf != "" {
		args = append(args, buf)
	}

	envRe := regexp.MustCompile(`\$({[a-zA-Z0-9_]+}|[a-zA-Z0-9_]+)`) // TODO: use os.Expand

	for i := range args {
		args[i] = envRe.ReplaceAllStringFunc(args[i], func(s string) string {
			s = s[1:]
			if s[0] == '{' {
				s = s[1 : len(s)-1]
			}
			return p.env.Get(s, "")
		})
	}

	return args, nil
}

// CommandLine returns the shell command line that would be executed when the
// given Process is started.
func (p Process) CommandLine() ([]string, error) {
	sp := newSystemProcess(p.Name, p.Script, p.Env, nil, nil)
	return sp.parseCommandLine()
}
