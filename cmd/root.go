package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	ip          string
	port        int
	model       string
	backend     string
	user        string
	prompt      int
	temperature float32
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "perf",
	Short: "llm perf analyzer by nullxjx",
	Long:  `llm perf analyzer by nullxjx`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
