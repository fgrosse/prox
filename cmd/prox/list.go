package main

import (
	"context"
	"os"

	"bitbucket.org/corvan/prox"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(lsCmd)

	flags := lsCmd.Flags()
	flags.StringP("socket", "s", DefaultSocketPath, "path of unix socket file to connect to")
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List information about currently running processes",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cliContext()
		debug := viper.GetBool("verbose")
		logger := prox.NewLogger(os.Stderr, debug)

		socketPath, err := cmd.Flags().GetString("socket")
		if err != nil {
			logger.Fatal("Failed to get --socket flag: " + err.Error())
		}

		c, err := prox.NewClient(socketPath, debug)
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer c.Close()

		err = c.List(ctx, os.Stdout)
		if err != nil && err != context.Canceled {
			logger.Fatal(err.Error())
		}
	},
}
