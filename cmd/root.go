package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	KB = 1 << 10
	MB = 1 << 20
	GB = 1 << 30
)

var execFile string
var rootCmd = &cobra.Command{
	Use:   "ktns",
	Short: "K Test N Stress is a tool to generate mock data and testing/stressing HTTP endpoints.",
	Long:  `K Test N Stress is a tool to generate custom mock data and testing/stressing HTTP endpoints with several consigurations.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&execFile, "file", "f", "execute.yaml", "ktns execution filename, to run without CLI flags.")
}

func initConfig() {
	if execFile != "" {
		viper.SetConfigFile(execFile) // Use exec file from the flag.
	} else {
		viper.SetConfigName("execute") // Expected name of execution file (without extension)
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".") // Find at current directory.
	}
}
