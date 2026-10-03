package cli

import (
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the API key",
	Long:  `Validate the configured API key with a free request (listing models). For balance and plan checks, see 'account status'.`,
	RunE:  runWithClient(runValidate),
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, _ []string, apiClient *client.Client) error {
	if _, err := apiClient.Models().List(cmd.Context()); err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}
	fmt.Println("✓ API key is valid")
	return nil
}
