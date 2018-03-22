package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"bitbucket.org/corvan/prox"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	StatusFailedProcess = 1
	StatusBadEnvFile    = 2
	StatusBadProcFile   = 3

	DefaultSocketPath = ".prox.sock" // hidden file in current PWD
)

func init() {
	cmd.AddCommand(startCmd)

	startCmd.Flags().Bool("no-color", false, "disable colored output")
	startCmd.Flags().StringP("env", "e", ".env", "path to the env file")
	startCmd.Flags().StringP("procfile", "f", "", `path to the Proxfile or Procfile (Default "Proxfile" or "Procfile")`)
	startCmd.Flags().StringP("socket", "s", DefaultSocketPath, "path of the temporary unix socket file that clients can use to establish a connection")
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Run all processes",
	Run:   run,
}

func run(cmd *cobra.Command, _ []string) {
	viper.BindPFlags(cmd.Flags())

	ctx := cliContext()

	debug := viper.GetBool("verbose")
	logger := prox.NewLogger(os.Stderr, debug)
	defer logger.Sync()

	socketPath := GetStringFlag(cmd, "socket", logger)
	disableColors := GetBoolFlag(cmd, "no-color", logger)

	env, err := environment()
	if err != nil {
		logger.Error("Failed to parse env file: " + err.Error() + "\n")
		os.Exit(StatusBadEnvFile)
	}

	pp, err := processes(env)
	if err != nil {
		logger.Error("Failed to parse Procfile: " + err.Error() + "\n")
		os.Exit(StatusBadProcFile)
	}

	// TODO: implement opt out for socket feature
	e := prox.NewExecutorServer(socketPath, debug)
	if disableColors {
		e.DisableColoredOutput()
	}

	err = e.Run(ctx, pp)
	e.Close() // always close the executor/server regardless of any error

	if err != nil {
		// The error was logged by the executor already
		// TODO: change signature of Run to return boolean
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

func environment() (prox.Environment, error) {
	path := viper.GetString("env")
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

	defer f.Close()
	env := prox.SystemEnv()
	err = env.ParseEnvFile(f)
	return env, err
}

func processes(env prox.Environment) ([]prox.Process, error) {
	var (
		f     *os.File
		err   error
		parse = prox.ParseProxFile
	)

	if path := viper.GetString("procfile"); path != "" {
		// user has specified a path
		f, err = os.Open(path)
	} else {
		// default to "Proxfile"
		f, err = os.Open("Proxfile")
		if os.IsNotExist(err) {
			// no "Proxfile" but maybe we can open a "Procfile"
			parse = prox.ParseProcFile
			f, err = os.Open("Procfile")
		}
	}

	if err != nil {
		// well fuck it..
		return nil, err
	}

	defer f.Close()
	return parse(f, env)
}
