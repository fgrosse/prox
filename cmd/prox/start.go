package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"bitbucket.org/corvan/prox"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	StatusFailedProcess = 1
	StatusBadEnvFile    = 2
	StatusBadProcFile   = 3
)

func init() {
	cmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use: "start",
	Run: run,
}

func run(_ *cobra.Command, _ []string) {
	ctx := cliContext()
	debug := viper.GetBool("verbose")
	logger := prox.NewLogger(os.Stderr, debug)
	defer logger.Sync()

	env, err := environment(".env") // TODO: use flag
	if err != nil {
		logger.Error("Failed to parse env file: " + err.Error() + "\n")
		os.Exit(StatusBadEnvFile)
	}

	f, err := os.Open("Procfile") // TODO: use flag
	if err != nil {
		logger.Error("Failed to open Procfile: " + err.Error() + "\n")
		os.Exit(StatusBadProcFile)
	}

	pp, err := prox.ParseProcFile(f, env)
	f.Close()
	if err != nil {
		logger.Error("Failed to parse Procfile: " + err.Error() + "\n")
		os.Exit(StatusBadProcFile)
	}

	e := prox.NewExecutor(debug)
	err = e.Run(ctx, pp)
	if err != nil {
		// The error was logged by the executor already
		os.Exit(StatusFailedProcess)
	}
}

func cliContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGALRM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		cancel()
	}()

	return ctx
}

func environment(path string) (prox.Environment, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return prox.SystemEnv(), nil
	}

	if err != nil {
		return prox.Environment{}, errors.Wrap(err, "failed to open env file")
	}

	env := prox.SystemEnv()
	err = env.ParseEnvFile(f)
	f.Close()

	return env, err
}
