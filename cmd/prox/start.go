package main

import (
	"context"
	"log"
	"os"
	"os/signal"

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
	ctx := cliContext()
	err := prox.Run(ctx, ".env", "Procfile") // TODO: use flags
	if err != nil {
		log.Fatal(err)
	}
}

func cliContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Reset(os.Interrupt, os.Kill)
	signal.Notify(sigs, os.Interrupt, os.Kill)

	go func() {
		<-sigs
		cancel()
	}()

	return ctx
}
