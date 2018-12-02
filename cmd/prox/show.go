package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/fgrosse/prox"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmd.AddCommand(showCmd)

	flags := showCmd.Flags()
	flags.StringP("env", "e", ".env", "path to the env file")
	flags.StringP("procfile", "f", "", `path to the Proxfile or Procfile (default "Proxfile" or "Procfile")`)
	flags.BoolP("all", "a", false, "show run configuration of all processes (ignoring any arguments)")
}

var showCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show run configuration of a single process",
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlags(cmd.Flags())
		defer logger.Sync()

		all := viper.GetBool("all")
		verbose := viper.GetBool("verbose")

		var name string
		if len(args) != 1 && !all {
			logger.Error("prox show requires exactly one argument - the process name as written in the Procfile or Proxfile\n")
			fmt.Println(cmd.UsageString())
			os.Exit(StatusMissingArgs)
		} else if !all {
			name = args[0]
		}

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

		printRunConfiguration(all, verbose, name, env, pp)
	},
}

func printRunConfiguration(all, verbose bool, processName string, env prox.Environment, pp []prox.Process) {
	if all {
		w := tabwriter.NewWriter(os.Stdout, 8, 8, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSCRIPT")
		for _, p := range pp {
			args, err := p.CommandLine()
			if err != nil {
				logger.Error("Failed to parse command line: " + err.Error())
			}
			fmt.Fprintln(w, fmt.Sprintf("%s\t%q", p.Name, args))
		}
		w.Flush()
		return
	}

	var p prox.Process
	for i := range pp {
		if pp[i].Name == processName {
			p = pp[i]
			break
		}
	}

	if p.Name == "" {
		logger.Error(fmt.Sprintf("No such process %q. Use`prox show --all` to see a list of all available processes", processName))
	}

	if p.Output.Format == "" {
		p.Output = prox.DefaultStructuredOutput(env)
	}

	if verbose {
		out, err := json.MarshalIndent(p, "", "    ")
		if err != nil {
			log.Fatal("Failed to encode message as JSON: ", err)
		}
		fmt.Println(string(out))
	} else {
		fmt.Println(p.CommandLine())
	}
}
