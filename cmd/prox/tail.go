package main

import (
	"context"
	"os"

	"bitbucket.org/corvan/prox"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(tailCmd)

	flags := tailCmd.Flags()
	flags.StringP("socket", "s", DefaultSocketPath, "path of unix socket file to connect to")
	// TODO: flag to omit prefix (useful if connecting to a single command and piping JSON output into jq)
}

var tailCmd = &cobra.Command{
	Use:   "tail <process> [process-2] … [process-N]",
	Short: "Follow the log output of one or many running processes",
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlags(cmd.Flags())
		defer logger.Sync()

		debug := viper.GetBool("verbose")
		socketPath := viper.GetString("socket")

		if len(args) == 0 {
			logger.Fatal("prox tail requires at least one argument")
		}

		c, err := prox.NewClient(socketPath, debug)
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer c.Close()

		ctx := cliContext()
		err = c.Tail(ctx, args, os.Stdout)
		if err != nil && err != context.Canceled {
			logger.Fatal(err.Error())
		}
	},
}
