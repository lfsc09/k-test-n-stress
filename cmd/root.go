package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var execFile string
var RootCmd = &cobra.Command{
	Use:   "ktns",
	Short: "K Test N Stress is a tool to generate mock data and testing/stressing HTTP endpoints.",
	Long:  `K Test N Stress is a tool to generate custom mock data and testing/stressing HTTP endpoints with several consigurations.`,
}

func Execute() error {
	return RootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&execFile, "file", "f", "execute.yaml", "ktns execution filename, to run without CLI flags.")
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
