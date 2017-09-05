package prox

import (
	"context"
	"os"

	"github.com/pkg/errors"
)

const (
	StatusFailedProcess = 1
	StatusBadEnvFile    = 2
	StatusBadProcFile   = 3
)

func Run(ctx context.Context, debug bool, envFilePath, procFilePath string) (statusCode int, err error) {
	env, err := environment(envFilePath)
	if err != nil {
		return StatusBadEnvFile, err
	}

	f, err := os.Open(procFilePath)
	if err != nil {
		return StatusBadProcFile, errors.Wrap(err, "failed to open Procfile")
	}

	pp, err := ParseProcFile(f, env)
	f.Close()
	if err != nil {
		return StatusBadProcFile, err
	}

	e := NewExecutor(debug)
	return StatusFailedProcess, e.Run(ctx, pp)
}

func environment(path string) (Environment, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return SystemEnv(), nil
	}

	if err != nil {
		return Environment{}, errors.Wrap(err, "failed to open env file")
	}

	env, err := ParseEnvFile(f)
	f.Close()
	if err != nil {
		return Environment{}, err
	}

	return env.Merge(SystemEnv()), nil
}
