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

	tailCmd.Flags().StringP("socket", "s", DefaultSocketPath, "path of unix socket file to connect to")
	// TODO: flag to omit prefix (useful if connecting to a single command and piping JSON output into jq)
}

var tailCmd = &cobra.Command{
	Use:   "tail <process>",
	Short: "Follow the log output of running processes",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cliContext()
		debug := viper.GetBool("verbose")
		logger := prox.NewLogger(os.Stderr, debug)

		if len(args) == 0 {
			logger.Fatal("tail requires at least one argument")
		}

		socketPath, err := cmd.Flags().GetString("socket")
		if err != nil {
			logger.Fatal("Failed to get --socket flag: " + err.Error())
		}

		c, err := prox.NewClient(socketPath, debug)
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer c.Close()

		err = c.Tail(ctx, args, os.Stdout)
		if err != nil && err != context.Canceled {
			logger.Fatal(err.Error())
		}
	},
}
