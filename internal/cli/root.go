package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "observeall",
	Short: "Universal OTel-powered observability platform",
	Long: color.CyanString(`
  ___  _                         _   _ _ 
 / _ \| |__  ___  ___ _ ____   ___| | | |
| | | | '_ \/ __|/ _ \ '__\ \ / / _' | | |
| |_| | |_) \__ \  __/ |   \ V / (_| | | |
 \___/|_.__/|___/\___|_|    \_/ \__,_|_|_|

`) + `One agent. Any stack. Any backend.
Powered by OpenTelemetry | https://github.com/observeall/observeall`,
}

// Execute runs the root CLI command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(generateCmd)

	rootCmd.PersistentFlags().String("config", "observeall.yaml", "ObserveAll project config file")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
}

func successf(format string, a ...any) {
	color.Green("✔ "+format, a...)
}

func infof(format string, a ...any) {
	color.Cyan("ℹ "+format, a...)
}

func errorf(format string, a ...any) {
	fmt.Printf(color.RedString("✖ "+format+"\n", a...))
}

