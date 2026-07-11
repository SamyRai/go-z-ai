package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "zai-client",
	Short: "Z.AI API Client",
	Long:  `A comprehensive CLI client for the Z.AI (Zhipu AI) API platform.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .env)")
	rootCmd.PersistentFlags().String("api-key", "", "Z.AI API key (can also set ZAI_API_KEY environment variable)")
	rootCmd.PersistentFlags().String("base-url", "", "API base URL (default: https://api.z.ai/api/paas/v4)")
	rootCmd.PersistentFlags().String("account", "", "Use a stored account by name for this command (see 'accounts list')")

	viper.BindPFlag("api-key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("base-url", rootCmd.PersistentFlags().Lookup("base-url"))
	viper.BindPFlag("account", rootCmd.PersistentFlags().Lookup("account"))
}

func initConfig() {
	viper.AutomaticEnv()

	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("env")
		viper.SetConfigName(".env")
	}

	// Try to read config, but don't fail if .env doesn't exist
	if err := viper.ReadInConfig(); err != nil {
		// .env file not found or error reading it - this is OK
		// The application will fall back to environment variables
	}
}

func main() {
	Execute()
}