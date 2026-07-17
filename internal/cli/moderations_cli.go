package cli

import (
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var moderationsCmd = &cobra.Command{
	Use:   "moderations",
	Short: "Check content against Z.AI's moderation policy (routes to open.bigmodel.cn)",
	Long: `Screen text/image/video/audio content via open.bigmodel.cn — the only
platform that documents this endpoint (api.z.ai's docs don't mention it
at all).

Authenticates with --china-api-key / ZAI_CHINA_API_KEY, falling back to
--api-key if unset — a regular z.ai key authenticates fine here too
(confirmed live: same /models catalog, same billing errors on both
platforms). Whether you get real results depends on your account's plan
entitlement, not which key you use: a GLM Coding Plan account, for
example, returns "Unknown Model" here since moderation isn't in that
plan's catalog.`,
}

var moderationsCheckCmd = &cobra.Command{
	Use:   "check [text]",
	Short: "Check a text input for policy violations",
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runModerationsCheck),
}

func init() {
	rootCmd.AddCommand(moderationsCmd)
	moderationsCmd.AddCommand(moderationsCheckCmd)

	addFormatFlag("json", moderationsCheckCmd)
}

func runModerationsCheck(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	resp, err := apiClient.Moderations().Create(cmd.Context(), client.ModerationRequest{
		Input: args[0],
	})
	if err != nil {
		return fmt.Errorf("failed to check content: %w", err)
	}

	return emit(cmd, resp, func() error {
		for _, r := range resp.ResultList {
			if len(r.RiskType) > 0 {
				fmt.Printf("%s: %s (%v)\n", r.ContentType, r.RiskLevel, r.RiskType)
			} else {
				fmt.Printf("%s: %s\n", r.ContentType, r.RiskLevel)
			}
		}
		return nil
	})
}
