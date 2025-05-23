package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRootCmd(opts *CommandOptions) *cobra.Command {
	if opts == nil {
		opts = &CommandOptions{
			Out:   os.Stdout,
			Viper: viper.GetViper(),
		}
	}

	rootCmd := &cobra.Command{
		Use:   "ktns",
		Short: "K Test N Stress is a tool to generate mock data, make HTTP requests, stress HTTP endpoints and seed databases.",
		Long:  `K Test N Stress is a tool to generate mock data, make HTTP requests, stress HTTP endpoints and seed databases with several configurations.`,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// If --file flag is set, set it and parse it
			if cmd.Flags().Changed("file") {
				execFile, _ := cmd.Flags().GetString("file")

				opts.Viper.SetConfigFile(execFile)

				if err := opts.Viper.ReadInConfig(); err != nil {
					return fmt.Errorf("%w", err)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// If --file flag is set, use it to run the command
			if cmd.Flags().Changed("file") {
				// Look for the command name in the execution file
				subCmdName := opts.Viper.GetString("command")
				if subCmdName == "" {
					return fmt.Errorf("no command specified in execution file")
				}
				subCmd, _, err := cmd.Find([]string{subCmdName})
				if err != nil {
					return fmt.Errorf("command '%s' does not exist: %w", subCmdName, err)
				}

				// Get command flags
				var cmdArgs []string
				prefix := subCmdName + "."
				for _, key := range opts.Viper.AllKeys() {
					// Get only keys in the 'yaml' file that are nested under the command name
					if strings.HasPrefix(key, prefix) {
						flagName := strings.TrimPrefix(key, prefix)
						// Handle different flag types (Booleans, Slices and Strings)
						if opts.Viper.IsSet(key) {
							if val := opts.Viper.GetBool(key); opts.Viper.GetString(key) == "true" || opts.Viper.GetString(key) == "false" {
								if val {
									cmdArgs = append(cmdArgs, fmt.Sprintf("--%s", flagName))
								}
							} else if opts.Viper.Get(key) != nil && reflect.TypeOf(opts.Viper.Get(key)).Kind() == reflect.Slice {
								for _, item := range opts.Viper.GetStringSlice(key) {
									cmdArgs = append(cmdArgs, fmt.Sprintf("--%s", flagName), item)
								}
							} else {
								cmdArgs = append(cmdArgs, fmt.Sprintf("--%s", flagName), opts.Viper.GetString(key))
							}
						}
					}
				}

				if len(cmdArgs) > 0 {
					subCmd.SetArgs(cmdArgs)
				}
				return subCmd.Execute()
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().StringP("file", "f", "execute.yaml", "ktns execution filename, to run without CLI flags.")

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
