package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

type CommandOptions struct {
	Out io.Writer
}

func NewRootCmd(opts *CommandOptions) *cobra.Command {
	if opts == nil {
		opts = &CommandOptions{
			Out: os.Stdout,
		}
	}

	rootCmd := &cobra.Command{
		Use:   "ktns",
		Short: "K Test N Stress is a tool to generate mock data, make HTTP requests and stress HTTP endpoints.",
		Long:  `K Test N Stress is a tool to generate mock data, make HTTP requests and stress HTTP endpoints.`,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		Version: Version,
	}

	// Configure cobra ouput streams to use the custom 'Out'
	rootCmd.SetOut(opts.Out)

	rootCmd.AddCommand(NewMockCmd(opts))
	rootCmd.AddCommand(NewRequestCmd(opts))

	// Disable automatic call of `--help` during errors
	rootCmd.SilenceUsage = true

	return rootCmd
}

func Execute() {
	// Use default options for real application
	rootCmd := NewRootCmd(nil)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
