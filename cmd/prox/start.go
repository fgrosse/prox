package main

import (
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
	socketPath := viper.GetString("socket")
	disableColors := viper.GetBool("no-color")
	envPath := viper.GetString("env")
	procfilePath := viper.GetString("procfile")

	env, err := environment(envPath)
	if err != nil {
		logger.Error("Failed to parse env file: " + err.Error())
		os.Exit(StatusBadEnvFile)
	}

	pp, err := processes(env, procfilePath)
	if err != nil {
		logger.Error("Failed to parse Procfile: " + err.Error())
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
