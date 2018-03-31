package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the version of prox set at compile time.
var Version = "0.0.0-unknown"

func init() {
	cmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of prox and then exit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}
