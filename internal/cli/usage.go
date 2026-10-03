package cli

import (
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "GLM Coding Plan quota",
	Long:  `Show the GLM Coding Plan quota windows for the current key. For several stored accounts at once, see 'accounts quota' and 'accounts usage'.`,
}

var usageQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Show quota windows, reset times, and pace",
	RunE:  runWithClient(runUsageQuota),
}

func init() {
	rootCmd.AddCommand(usageCmd)
	usageCmd.AddCommand(usageQuotaCmd)
	addFormatFlag("text", usageQuotaCmd)
}

func runUsageQuota(cmd *cobra.Command, _ []string, apiClient *client.Client) error {
	quota, err := apiClient.Quota().GetQuotaLimit(cmd.Context())
	if err != nil {
		return err
	}
	return emit(cmd, quota, func() error {
		printQuota(&quota.Data, time.Now())
		return nil
	})
}
