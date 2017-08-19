package prox

import (
	"errors"
	"os/exec"
)

type Process interface {
	Name() string
	Run() error // TODO pass a ctx
	Interrupt() error
}

type ShellProcess struct {
	CommandLine string
	//Env         Environment
	cmd *exec.Cmd
	//output      TaskOutput

	name string
}

func NewShellProcess(name, script string) *ShellProcess {
	return &ShellProcess{
		CommandLine: script,
		name:        name,
	}
}

func (p *ShellProcess) Name() string {
	return p.name
}

func (p *ShellProcess) Run() error {
	return errors.New("NOT IMPLEMENTED")
}

func (p *ShellProcess) Interrupt() error {
	return errors.New("NOT IMPLEMENTED")
}
