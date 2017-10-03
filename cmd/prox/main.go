package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Version is the version of prox set at compile time.
var Version = "0.0.0-unknown"

var cmd = &cobra.Command{
	Use:   "prox",
	Short: "A process runner for Procfile-based applications",
	Run:   run,
}

func main() {
	cmd.PersistentFlags().BoolP("verbose", "v", false, "enable detailed log output for debugging")
	cmd.Flags().AddFlagSet(startCmd.Flags())

	viper.AutomaticEnv()
	viper.SetEnvPrefix("PROX")
	viper.BindPFlags(cmd.PersistentFlags())

	cmd.AddCommand(versionCmd)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of prox and then exit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func GetStringFlag(cmd *cobra.Command, name string, logger *zap.Logger) string {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		logger.Fatal("Failed to get --" + name + " flag: " + err.Error())
	}

	return value
}

func GetBoolFlag(cmd *cobra.Command, name string, logger *zap.Logger) bool {
	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		logger.Fatal("Failed to get --" + name + " flag: " + err.Error())
	}

	return value
}
