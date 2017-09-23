package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"bitbucket.org/corvan/prox"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	StatusFailedProcess = 1
	StatusBadEnvFile    = 2
	StatusBadProcFile   = 3
)

func init() {
	cmd.AddCommand(startCmd)

	startCmd.Flags().StringP("env-file", "e", ".env", "path to the env file")
	startCmd.Flags().StringP("proc-file", "p", "Procfile", "path to the Procfile")
}

var startCmd = &cobra.Command{
	Use: "start",
	Run: run,
}

func run(cmd *cobra.Command, _ []string) {
	ctx := cliContext()
	flags := cmd.Flags()

	debug := viper.GetBool("verbose")
	logger := prox.NewLogger(os.Stderr, debug)
	defer logger.Sync()

	env, err := environment(flags)
	if err != nil {
		logger.Error("Failed to parse env file: " + err.Error() + "\n")
		os.Exit(StatusBadEnvFile)
	}

	pp, err := processes(flags, env)
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

func environment(flags *pflag.FlagSet) (prox.Environment, error) {
	path, err := flags.GetString("env-file")
	if err != nil {
		return prox.Environment{}, errors.New("failed to get --env-file flag: " + err.Error())
	}

	if path == "" {
		return prox.Environment{}, errors.New("env file path cannot be empty")
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return prox.SystemEnv(), nil
	}

	if err != nil {
		return prox.Environment{}, errors.New("failed to open env file: " + err.Error())
	}

	env := prox.SystemEnv()
	err = env.ParseEnvFile(f)
	f.Close()

	return env, err
}

func processes(flags *pflag.FlagSet, env prox.Environment) ([]prox.Process, error) {
	path, err := flags.GetString("proc-file")
	if err != nil {
		return nil, errors.New("failed to get --proc-file flag: " + err.Error())
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	pp, err := prox.ParseProcFile(f, env)
	f.Close()

	return pp, err
}
