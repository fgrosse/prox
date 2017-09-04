package prox

import (
	"context"
	"os"

	"github.com/pkg/errors"
)

func Run(ctx context.Context, envFilePath, procFilePath string) error {
	f, err := os.Open(envFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to open env file")
	}

	env, err := ParseEnvFile(f)
	f.Close()
	if err != nil {
		return err
	}

	env = env.Merge(SystemEnv())

	f, err = os.Open(procFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to open Procfile")
	}

	pp, err := ParseProcFile(f, env)
	f.Close()
	if err != nil {
		return err
	}

	e := NewExecutor()
	return e.Run(ctx, pp)
}
