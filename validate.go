package main

import (
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate API configuration",
	Long:  `Validate your API key and configuration by making a test request to the Z.AI API.`,
	RunE:  validateAPIKey,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
