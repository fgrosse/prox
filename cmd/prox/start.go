package main

import (
	"context"
	"log"

	"bitbucket.com/corvan/prox"
	"github.com/spf13/cobra"
)

func init() {
	cmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use: "start",
	Run: run,
}

func run(cmd *cobra.Command, args []string) {
	ctx := context.TODO()
	err := prox.Run(ctx, ".env", "Procfile") // TODO: use flags
	if err != nil {
		log.Fatal(err)
	}
}
