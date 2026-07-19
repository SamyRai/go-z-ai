package cli

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// buildInfo is set by Execute() from main.go's ldflag-populated vars so that
// `go-z-ai --version` reports the GoReleaser build metadata.
var buildInfo = struct {
	version string
	commit  string
	date    string
}{version: "dev", commit: "none", date: "unknown"}

var rootCmd = &cobra.Command{
	Use:     "go-z-ai",
	Short:   "Z.AI API Client",
	Long:    `A comprehensive CLI client for the Z.AI (Zhipu AI) API platform.`,
	Version: "dev",
}

// SetBuildInfo configures the version/commit/date reported by --version.
// It must be called before Execute(). When not called, defaults to "dev".
func SetBuildInfo(version, commit, date string) {
	buildInfo.version = version
	buildInfo.commit = commit
	buildInfo.date = date
	rootCmd.Version = version
	if version == "dev" {
		// Show a richer string for development builds so it's clear --version
		// is wired but no release tag was applied.
		rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	}
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
	rootCmd.PersistentFlags().String("china-api-key", "", "open.bigmodel.cn API key for embeddings/moderations (can also set ZAI_CHINA_API_KEY environment variable; falls back to --api-key)")
	rootCmd.PersistentFlags().String("region", "", "Regional gateway for monitor/biz/agents/detection: 'global' (api.z.ai, default) or 'china' (open.bigmodel.cn). Aliases: cn, bigmodel, west. Does not override --base-url.")

	viper.BindPFlag("api-key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("base-url", rootCmd.PersistentFlags().Lookup("base-url"))
	viper.BindPFlag("account", rootCmd.PersistentFlags().Lookup("account"))
	viper.BindPFlag("china-api-key", rootCmd.PersistentFlags().Lookup("china-api-key"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
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

	// Try to read config, but don't fail if .env doesn't exist — the
	// application falls back to environment variables in that case.
	_ = viper.ReadInConfig()
}
