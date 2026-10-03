package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/SamyRai/go-z-ai/internal/usageview"
	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Inspect the current API key's account",
	Long:  `Inspect the account behind the current API key: type and region, health, balance, and coding-plan subscriptions.`,
}

var accountDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect the key's account type and region",
	Long:  `Detect whether the key belongs to a GLM Coding Plan or pay-as-you-go account, and on which gateway (api.z.ai or open.bigmodel.cn). Free: no tokens are spent.`,
	RunE:  runWithClient(runAccountDetect),
}

var accountStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check that the key works and can spend",
	Long: `Check that the key authenticates and can currently spend. Coding-plan keys are
checked for free through the quota endpoint; pay-as-you-go keys need one
minimal billed request (Z.AI has no balance-check API for them).`,
	RunE: runWithClient(runAccountStatus),
}

var accountBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Show the pay-as-you-go wallet balance",
	RunE:  runWithClient(runAccountBalance),
}

var accountSubscriptionsCmd = &cobra.Command{
	Use:   "subscriptions",
	Short: "List GLM Coding Plan subscriptions",
	RunE:  runWithClient(runAccountSubscriptions),
}

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(accountDetectCmd, accountStatusCmd, accountBalanceCmd, accountSubscriptionsCmd)
	addFormatFlag("text", accountDetectCmd, accountStatusCmd, accountBalanceCmd, accountSubscriptionsCmd)
	accountStatusCmd.Flags().Duration("watch", 0, "Re-check at this interval (e.g. 5m) until interrupted; pay-as-you-go checks are billed each time")
}

func runAccountDetect(cmd *cobra.Command, _ []string, apiClient *client.Client) error {
	det, err := apiClient.Detection().DetectAccountType(cmd.Context())
	if err != nil {
		return err
	}
	return emit(cmd, det, func() error {
		printDetected(det)
		return nil
	})
}

func printDetected(det *client.DetectedAccount) {
	fmt.Printf("Type:     %s", det.Type)
	if det.Level != "" {
		fmt.Printf(" (%s)", det.Level)
	}
	if !det.Confirmed {
		fmt.Print(" — inferred: the coding-plan quota endpoint did not recognize the key")
	}
	fmt.Printf("\nRegion:   %s\nEndpoint: %s\n", det.Region, det.BaseURL)
}

func runAccountStatus(cmd *cobra.Command, _ []string, apiClient *client.Client) error {
	every, _ := cmd.Flags().GetDuration("watch")
	check := func(ctx context.Context) error {
		status, err := apiClient.Detection().CheckAccountStatus(ctx)
		if err != nil {
			return err
		}
		return emit(cmd, status, func() error {
			mark := "✅"
			if !status.APIAccessible || !status.HasBalance {
				mark = "⚠️ "
			}
			fmt.Printf("[%s] %s %s\n", status.LastChecked.Format(time.TimeOnly), mark, status.Message)
			return nil
		})
	}
	if err := check(cmd.Context()); err != nil || every <= 0 {
		return err
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-cmd.Context().Done():
			return nil
		case <-ticker.C:
			if err := check(cmd.Context()); err != nil {
				progressf("check failed: %v\n", err)
			}
		}
	}
}

func runAccountBalance(cmd *cobra.Command, _ []string, apiClient *client.Client) error {
	b, err := apiClient.Account().Balance(cmd.Context())
	if err != nil {
		return err
	}
	return emit(cmd, b, func() error {
		fmt.Printf("Available: %.2f\nBalance:   %.2f (frozen %.2f)\nRecharged: %.2f, granted %.2f, spent %.2f\n",
			b.AvailableBalance, b.Balance, b.FrozenBalance, b.RechargeAmount, b.GiveAmount, b.TotalSpendAmount)
		fmt.Printf("(%s; USD on api.z.ai, CNY on open.bigmodel.cn)\n", apiClient.Region().ConsoleURL())
		return nil
	})
}

func runAccountSubscriptions(cmd *cobra.Command, _ []string, apiClient *client.Client) error {
	subs, err := apiClient.Account().Subscriptions(cmd.Context())
	if err != nil {
		return err
	}
	return emit(cmd, subs, func() error {
		if len(subs) == 0 {
			fmt.Println("No GLM Coding Plan subscription.")
			return nil
		}
		for _, s := range subs {
			renew := "off"
			if s.AutoRenew == 1 {
				renew = "on, next " + s.NextRenewTime
			}
			fmt.Printf("%s — %s, %s billing, valid %s, auto-renew %s\n", s.ProductName, s.Status, s.BillingCycle, s.Valid, renew)
		}
		return nil
	})
}

// printQuota renders a coding plan's quota windows through usageview's
// shared summaries (the TUI's Usage tab uses the same ones).
func printQuota(q *client.QuotaData, now time.Time) {
	fmt.Printf("📊 GLM Coding Plan (%s)\n\n", q.Level)
	mode := usageview.BillingModeOf(q.Limits)
	for _, l := range q.Limits {
		s := usageview.SummarizeLimit(l, mode, now, time.Local)
		fmt.Printf("• %s — %.0f%% used\n", s.Title, s.Used*100)
		for _, line := range []string{s.Counts, s.Reset, s.Pace, s.Peak} {
			if line != "" {
				fmt.Printf("  %s\n", line)
			}
		}
		for _, tool := range s.Tools {
			fmt.Printf("    %s\n", tool)
		}
		fmt.Println()
	}
}
