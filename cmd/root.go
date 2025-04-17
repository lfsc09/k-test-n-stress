package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
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

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "file", "f", "", "config file (default is ./config.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile) // Use config file from the flag.
	} else {
		viper.SetConfigName("config") // Expected name of config file (without extension)
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".") // Find at current directory.
	}
}
