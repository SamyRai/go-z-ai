package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Usage and quota operations",
	Long:  `Monitor and manage your API usage, quotas, and limits.`,
}

var usageQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Get current quota and usage",
	Long:  `Get current usage and quota information including remaining tokens and reset time.`,
	RunE:  runUsageQuota,
}

var usageSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Get usage summary",
	Long:  `Get a comprehensive summary of your usage including quota, account info, and statistics.`,
	RunE:  runUsageSummary,
}

var usageAccountCmd = &cobra.Command{
	Use:   "account",
	Short: "Get account information",
	Long:  `Get detailed account information including plan type and status.`,
	RunE:  runUsageAccount,
}

var usageBillingCmd = &cobra.Command{
	Use:   "billing",
	Short: "Get billing information",
	Long:  `Get billing information including cycle, next bill date, and last bill amount.`,
	RunE:  runUsageBilling,
}

var usageCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if quota is low",
	Long:  `Check if your quota is running low (below 20% remaining).`,
	RunE:  runUsageCheck,
}

var usageDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect account type",
	Long:  `Detect your Z.AI account type and working endpoint automatically.`,
	RunE:  runUsageDetect,
}

var (
	usageFormat string
	watchMode   bool
)

func init() {
	rootCmd.AddCommand(usageCmd)
	usageCmd.AddCommand(usageQuotaCmd)
	usageCmd.AddCommand(usageSummaryCmd)
	usageCmd.AddCommand(usageAccountCmd)
	usageCmd.AddCommand(usageBillingCmd)
	usageCmd.AddCommand(usageCheckCmd)
	usageCmd.AddCommand(usageDetectCmd)

	usageQuotaCmd.Flags().StringVar(&usageFormat, "format", "table", "Output format (table, json)")
	usageSummaryCmd.Flags().StringVar(&usageFormat, "format", "table", "Output format (table, json)")
	usageAccountCmd.Flags().StringVar(&usageFormat, "format", "table", "Output format (table, json)")
	usageBillingCmd.Flags().StringVar(&usageFormat, "format", "table", "Output format (table, json)")
	usageCheckCmd.Flags().BoolVar(&watchMode, "watch", false, "Watch mode - check every minute")
	usageDetectCmd.Flags().StringVar(&usageFormat, "format", "table", "Output format (table, json)")
}

func runUsageQuota(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	quota, err := apiClient.Quota().GetQuotaLimit(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get quota limit: %w", err)
	}

	if !quota.Success {
		return fmt.Errorf("API returned error: %s (code: %d)", quota.Msg, quota.Code)
	}

	if usageFormat == "json" {
		return outputJSON(quota)
	}

	return outputQuotaLimit(quota)
}

// outputQuotaLimit displays quota limit information in a human-readable format.
// For each quota window, it shows the current usage, remaining quota, reset time,
// and tool-specific breakdown when available.
//
// The output format:
//   - [Window Description]
//     Usage: [current/total (percentage)] — [remaining] remaining
//     Resets: [reset time] (in [countdown])
//     By tool: [breakdown for MCP tools quotas]
func outputQuotaLimit(quota *client.QuotaLimitResponse) error {
	fmt.Printf("📊 GLM Coding Plan Usage (%s tier)\n\n", strings.ToUpper(quota.Data.Level))

	now := time.Now()
	for _, limit := range quota.Data.Limits {
		// Display window type with clear description instead of cryptic unit codes
		fmt.Printf("• %s\n", limit.WindowDescription())

		// Display usage information - format varies based on whether limits are provided
		if limit.Usage > 0 {
			fmt.Printf("  Usage: %d/%d (%.0f%%) — %d remaining\n", limit.CurrentValue, limit.Usage, limit.Percentage, limit.Remaining)
		} else {
			fmt.Printf("  Usage: %.0f%%\n", limit.Percentage)
		}

		// Display reset time with countdown
		if limit.NextResetTime == 0 {
			fmt.Println("  Resets: — (window not started yet, no usage recorded)")
		} else {
			reset := time.UnixMilli(limit.NextResetTime)
			fmt.Printf("  Resets: %s (in %s)\n", reset.Format("2006-01-02 15:04:05 MST"), formatDuration(reset.Sub(now)))
		}

		// Display tool-specific breakdown for MCP tools usage (TIME_LIMIT)
		if len(limit.UsageDetails) > 0 {
			fmt.Println("  By tool:")
			for _, d := range limit.UsageDetails {
				fmt.Printf("    - %s: %d\n", d.ModelCode, d.Usage)
			}
		}
		fmt.Println()
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "now"
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}

func runUsageSummary(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	status, err := apiClient.Usage().GetAccountStatus(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get account status: %w", err)
	}

	return outputAccountStatus(status, usageFormat)
}

func runUsageAccount(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	status, err := apiClient.Usage().GetAccountStatus(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}

	return outputAccountStatus(status, usageFormat)
}

func runUsageBilling(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	fmt.Println("💳 Billing Information")
	fmt.Println("\nZ.AI doesn't provide billing API endpoints.")
	fmt.Println("Please manage billing at:", apiClient.Usage().GetWebDashboardURL())
	fmt.Println("\n💡 Tip: Use 'zai-client usage status' to check account accessibility")

	return nil
}

func runUsageCheck(cmd *cobra.Command, args []string) error {
	if watchMode {
		return runUsageWatch(cmd.Context())
	}

	apiClient, err := getClient()
	if err != nil {
		return err
	}

	status, err := apiClient.Usage().GetAccountStatus(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to check status: %w", err)
	}

	if status.HasBalance {
		fmt.Println("✅ Account is healthy and has balance")
	} else {
		fmt.Println("⚠️  Account needs attention:")
		fmt.Printf("   %s\n", status.Message)
		if !status.HasBalance {
			fmt.Println("   💡 Recharge at:", status.WebDashboard)
		}
	}

	return nil
}

func runUsageWatch(ctx context.Context) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	fmt.Println("Watching account status (checking every minute)...")
	fmt.Println("Press Ctrl+C to stop")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := apiClient.Usage().GetAccountStatus(ctx)
			if err != nil {
				fmt.Printf("Error checking status: %v\n", err)
				continue
			}

			currentTime := time.Now().Format("2006-01-02 15:04:05")
			if status.APIAccessible && status.HasBalance {
				fmt.Printf("[%s] ✅ OK: %s\n", currentTime, status.Message)
			} else if status.APIAccessible && !status.HasBalance {
				fmt.Printf("[%s] ⚠️  LOW: %s - Recharge at %s\n", currentTime, status.Message, status.WebDashboard)
			} else {
				fmt.Printf("[%s] ❌ ERROR: %s\n", currentTime, status.Message)
			}
		}
	}
}

func outputQuota(quota interface{}, format string) error {
	switch format {
	case "json":
		return outputJSON(quota)
	default:
		fmt.Println("📊 Client-Side Usage Tracking")
		fmt.Printf("%v\n", quota)
		return nil
	}
}

func outputUsageSummary(summary interface{}, format string) error {
	switch format {
	case "json":
		return outputJSON(summary)
	default:
		fmt.Println("📊 Usage Summary")
		fmt.Printf("%v\n", summary)
		return nil
	}
}

func outputAccountStatus(status *client.AccountStatus, format string) error {
	switch format {
	case "json":
		return outputJSON(status)
	default:
		fmt.Printf("📊 Account Status\n\n")
		fmt.Printf("API Accessible: %t\n", status.APIAccessible)
		fmt.Printf("Has Balance: %t\n", status.HasBalance)
		fmt.Printf("Status: %s\n", status.Message)
		fmt.Printf("Last Checked: %s\n", status.LastChecked.Format("2006-01-02 15:04:05"))
		fmt.Printf("Web Dashboard: %s\n", status.WebDashboard)
		return nil
	}
}

func outputAccount(account interface{}, format string) error {
	switch format {
	case "json":
		return outputJSON(account)
	default:
		fmt.Printf("👤 Account Information\n\n")
		fmt.Printf("%v\n", account)
		return nil
	}
}

func outputBilling(billing *client.BillingInfo, format string) error {
	switch format {
	case "json":
		return outputJSON(billing)
	default:
		return outputBillingTable(billing)
	}
}

func outputQuotaTable(quota interface{}) error {
	fmt.Printf("📊 Usage Information\n\n")
	fmt.Printf("%v\n", quota)
	return nil
}

func outputUsageSummaryTable(summary interface{}) error {
	fmt.Printf("📊 Usage Summary\n\n")
	fmt.Printf("%v\n", summary)
	return nil
}

func outputAccountTable(account interface{}) error {
	fmt.Printf("👤 Account Information\n\n")
	fmt.Printf("%v\n", account)
	return nil
}

func outputBillingTable(billing interface{}) error {
	fmt.Printf("💳 Billing Information\n\n")
	fmt.Printf("%v\n", billing)
	return nil
}
func runUsageDetect(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	fmt.Println("🔍 Detecting Account Type...")
	fmt.Print("Testing both pay-as-you-go and coding plan endpoints...\n\n")

	account, err := apiClient.Detection().GetAccountInfo(cmd.Context())
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	fmt.Printf("✅ Account Type: %s\n", account.Type)
	fmt.Printf("📍 Working Endpoint: %s\n", account.BaseURL)
	fmt.Printf("🔧 API Status: %v\n", account.Working)

	if account.UsageLimits != nil {
		fmt.Println("\n📊 Usage Limits:")
		if account.UsageLimits.HourlyPromptLimit == 0 {
			fmt.Println("   Hourly Prompt Limit: Unknown (API doesn't provide limits)")
		} else {
			fmt.Printf("   Hourly Prompt Limit: %d\n", account.UsageLimits.HourlyPromptLimit)
		}
		if account.UsageLimits.WeeklyPromptLimit == 0 {
			fmt.Println("   Weekly Prompt Limit: Unknown (API doesn't provide limits)")
		} else {
			fmt.Printf("   Weekly Prompt Limit: %d\n", account.UsageLimits.WeeklyPromptLimit)
		}
		fmt.Printf("   Hourly Window Reset: %s\n", account.UsageLimits.HourlyWindowReset)
		fmt.Printf("   Weekly Reset: %s\n", account.UsageLimits.WeeklyReset)
		fmt.Println("\n   💡 Z.AI doesn't provide usage limit endpoints.")
		fmt.Println("   Check your actual limits at: https://z.ai")
	}
	return nil
}
