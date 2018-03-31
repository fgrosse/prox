package main

import (
	"context"
	"os"

	"bitbucket.org/corvan/prox"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(startCmd)

	flags := startCmd.Flags()
	flags.Bool("no-color", false, "disable colored output")
	flags.StringP("env", "e", ".env", "path to the env file")
	flags.StringP("procfile", "f", "", `path to the Proxfile or Procfile (Default "Proxfile" or "Procfile")`)
	flags.StringP("socket", "s", DefaultSocketPath, "path of the temporary unix socket file that clients can use to establish a connection")
	flags.Bool("no-socket", false, "do not create a unix socket for prox clients")
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Run all processes (default if no command is given)",
	Run:   run,
}

func run(cmd *cobra.Command, _ []string) {
	viper.BindPFlags(cmd.Flags())
	defer logger.Sync()

	ctx := cliContext()
	debug := viper.GetBool("verbose")

	env, err := environment(viper.GetString("env"))
	if err != nil {
		logger.Error("Failed to parse env file: " + err.Error())
		os.Exit(StatusBadEnvFile)
	}

	pp, err := processes(env, viper.GetString("procfile"))
	if err != nil {
		logger.Error("Failed to parse Procfile: " + err.Error())
		os.Exit(StatusBadProcFile)
	}

	var done func() error
	var executor interface {
		Run(context.Context, []prox.Process) error
		DisableColoredOutput()
	}

	if viper.GetBool("no-socket") {
		logger.Debug("Skipping prox socket creation (--no-socket)")
		executor = prox.NewExecutor(debug)
		done = func() error { return nil } // noop
	} else {
		socketPath := viper.GetString("socket")
		es := prox.NewExecutorServer(socketPath, debug)
		done = es.Close
		executor = es
	}

	if viper.GetBool("no-color") {
		executor.DisableColoredOutput()
	}

	err = executor.Run(ctx, pp)
	done() // always close the executor/server regardless of any error

	if err != nil {
		// the error was logged by the executor already
		os.Exit(StatusFailedProcess)
	}
}
