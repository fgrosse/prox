package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/corvan/prox"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(showCmd)

	showCmd.Flags().StringP("env", "e", ".env", "path to the env file")
	showCmd.Flags().StringP("procfile", "f", "", `path to the Proxfile or Procfile (Default "Proxfile" or "Procfile")`)
}

var showCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show run configuration of a single process",
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlags(cmd.Flags())

		debug := viper.GetBool("verbose")
		logger := prox.NewLogger(os.Stderr, debug)

		if len(args) != 1 {
			logger.Error("prox show requires exactly one argument - the process name as written in the Procfile or Proxfile\n\n")
			fmt.Println(cmd.UsageString())
			os.Exit(StatusMissingArgs)
		}

		name := args[0]

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

		var p prox.Process
		for i := range pp {
			if pp[i].Name == name {
				p = pp[i]
				break
			}
		}

		if viper.GetBool("verbose") {
			out, err := json.MarshalIndent(p, "", "    ")
			if err != nil {
				log.Fatal("Failed to encode message as JSON: ", err)
			}
			fmt.Println(string(out))
		} else {
			fmt.Println(p.CommandLine())
		}
	},
}
