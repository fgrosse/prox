package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

func run(_ *cobra.Command, _ []string) {
	log.SetFlags(0)
	ctx := cliContext()
	status, err := prox.Run(ctx, debug, ".env", "Procfile") // TODO: use flags
	if err != nil {
		log.Printf("%s\tERROR\tprox\t%s", time.Now().Format("15:04:05"), err.Error()) // TODO: uniform logging
	}
	os.Exit(status)
}

func cliContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGALRM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		s := <-sigs
		fmt.Println("received signal", s) // TODO: use synchronized output
		cancel()
	}()

	return ctx
}
