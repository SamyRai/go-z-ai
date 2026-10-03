package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/SamyRai/go-z-ai/internal/accounts"
	"github.com/SamyRai/go-z-ai/internal/usageview"
	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

// Reports across stored accounts: coding-plan quota and usage heat maps.

var accountsQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Check quota and reset times across stored accounts",
	Long:  `Fetches GLM Coding Plan quota windows for stored accounts (default: all). Pay-as-you-go accounts are skipped: the monitor endpoints are coding-plan only.`,
	RunE:  runAccountsQuota,
}

var accountsUsageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show a token/tool usage heat map across stored accounts",
	Long:  `Renders per-model token usage and per-tool call counts as a terminal heat map. The API buckets hourly for windows of 8 days or less (use --today or a small --days) and daily for 9+ days. Defaults to all accounts; pay-as-you-go accounts are skipped.`,
	RunE:  runAccountsUsage,
}

func init() {
	accountsCmd.AddCommand(accountsQuotaCmd, accountsUsageCmd)
	addFormatFlag("text", accountsQuotaCmd, accountsUsageCmd)
	for _, c := range []*cobra.Command{accountsQuotaCmd, accountsUsageCmd} {
		c.Flags().StringArray("only", nil, "Limit to specific account names (repeatable; default: all accounts)")
	}
	accountsUsageCmd.Flags().Int("days", 14, "Trailing calendar days to include (hourly buckets for ≤8 days, daily for ≥9)")
	accountsUsageCmd.Flags().Bool("today", false, "Shorthand for --days 1 (today only, hourly detail)")
	accountsUsageCmd.Flags().String("metric", "both", "Which usage to show: model, tool, or both")
}

// accountReport is one account's slice of a cross-account report. Skipped
// explains a skipped or failed account; notApplicable marks a pay-as-you-go
// account (shown as skipped rather than failed).
type accountReport struct {
	Name          string                     `json:"name"`
	Type          client.AccountType         `json:"type"`
	Skipped       string                     `json:"skipped,omitempty"`
	Quota         *client.QuotaLimitResponse `json:"quota,omitempty"`
	Models        *client.ModelUsageResponse `json:"models,omitempty"`
	Tools         *client.ToolUsageResponse  `json:"tools,omitempty"`
	serverTZ      *time.Location
	notApplicable bool
}

// reportClient builds acct's client for a monitor report, or explains why
// the account is skipped.
func reportClient(acct accounts.Account) (*client.Client, accountReport, error) {
	r := accountReport{Name: acct.Name, Type: acct.Type}
	if !acct.SupportsMonitorEndpoints() {
		r.Skipped, r.notApplicable = fmt.Sprintf("the monitor endpoints don't apply to %s accounts", acct.Type), true
		return nil, r, nil
	}
	cfg, err := acct.ClientConfig()
	if err != nil {
		return nil, r, fmt.Errorf("account %q: %w", acct.Name, err)
	}
	c, err := client.NewClient(cfg)
	if err != nil {
		return nil, r, fmt.Errorf("account %q: %w", acct.Name, err)
	}
	r.serverTZ = c.MonitorTimezone()
	return c, r, nil
}

// runReport fetches and prints a report for each targeted account.
func runReport(cmd *cobra.Command, fetch func(context.Context, *client.Client, *accountReport) error, print func(accountReport)) error {
	store, err := accounts.Load()
	if err != nil {
		return err
	}
	only, _ := cmd.Flags().GetStringArray("only")
	targets, err := resolveTargets(store, only)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		progressf("No accounts configured. Add one with: go-z-ai accounts add <name> --api-key <key>\n")
		return nil
	}
	reports := make([]accountReport, 0, len(targets))
	for _, acct := range targets {
		c, r, err := reportClient(acct)
		if err != nil {
			return err
		}
		if c != nil {
			if err := fetch(cmd.Context(), c, &r); err != nil {
				r.Skipped = err.Error()
			}
		}
		reports = append(reports, r)
	}
	return emit(cmd, reports, func() error {
		for _, r := range reports {
			fmt.Printf("=== %s ===\n", r.Name)
			switch {
			case r.notApplicable:
				fmt.Printf("⏭️  Skipped: %s\n\n", r.Skipped)
			case r.Skipped != "":
				fmt.Printf("❌ %s\n\n", r.Skipped)
			default:
				print(r)
			}
		}
		return nil
	})
}

// resolveTargets returns the accounts named in only (erroring on an unknown
// name), or every stored account when only is empty.
func resolveTargets(store *accounts.Store, only []string) ([]accounts.Account, error) {
	if len(only) == 0 {
		return store.List(), nil
	}
	targets := make([]accounts.Account, 0, len(only))
	for _, name := range only {
		acct, found := store.Get(name)
		if !found {
			return nil, fmt.Errorf("account %q not found", name)
		}
		targets = append(targets, acct)
	}
	return targets, nil
}

func runAccountsQuota(cmd *cobra.Command, _ []string) error {
	now := time.Now()
	return runReport(cmd,
		func(ctx context.Context, c *client.Client, r *accountReport) (err error) {
			r.Quota, err = c.Quota().GetQuotaLimit(ctx)
			return err
		},
		func(r accountReport) { printQuota(&r.Quota.Data, now) },
	)
}

func runAccountsUsage(cmd *cobra.Command, _ []string) error {
	days, _ := cmd.Flags().GetInt("days")
	today, _ := cmd.Flags().GetBool("today")
	metric, _ := cmd.Flags().GetString("metric")
	if metric != "model" && metric != "tool" && metric != "both" {
		return fmt.Errorf("invalid --metric %q (expected model, tool, or both)", metric)
	}
	start, end := usageview.Window(max(days, 1), today)
	if !isJSONFormat(cmd) {
		fmt.Print("Legend: (blank)=0  ░▒▓█=low→peak, scaled per row against that row's own max\n\n")
	}
	return runReport(cmd,
		func(ctx context.Context, c *client.Client, r *accountReport) error {
			var err error
			if metric != "tool" {
				if r.Models, err = c.Quota().GetModelUsage(ctx, start, end); err != nil {
					return fmt.Errorf("failed to fetch model usage: %w", err)
				}
			}
			if metric != "model" {
				if r.Tools, err = c.Quota().GetToolUsage(ctx, start, end); err != nil {
					return fmt.Errorf("failed to fetch tool usage: %w", err)
				}
			}
			return nil
		},
		func(r accountReport) {
			if note := usageview.ZoneNote(r.serverTZ); note != "" {
				fmt.Printf("🕑 %s\n", note)
			}
			if r.Models != nil {
				d := r.Models.Data
				printHeatmap("📈 Model usage", d.XTime, d.Granularity, usageview.ModelRows(d), "tokens", r.serverTZ)
				fmt.Printf("  Total: %s calls, %s tokens\n\n", usageview.FormatCount(d.TotalUsage.TotalModelCallCount), usageview.FormatCount(d.TotalUsage.TotalTokensUsage))
			}
			if r.Tools != nil {
				d := r.Tools.Data
				printHeatmap("🔧 Tool usage", d.XTime, d.Granularity, usageview.ToolRows(d), "calls", r.serverTZ)
				fmt.Println()
			}
		},
	)
}

// printHeatmap prints one heat-map section: a titled span, then a row of
// density blocks per series.
func printHeatmap(title string, xTime []string, granularity string, rows []usageview.UsageRow, unit string, serverTZ *time.Location) {
	fmt.Printf("%s (%s, %s)\n", title, rangeLabel(xTime, serverTZ), granularity)
	if len(rows) == 0 {
		fmt.Println("  No usage in this window.")
		return
	}
	width := 0
	for _, r := range rows {
		width = max(width, len(r.Label))
	}
	for _, r := range rows {
		fmt.Printf("  %-*s %s  %s %s\n", width, r.Label, usageview.HeatmapBlocks(r.Values), usageview.FormatCount(r.Total), unit)
	}
}

// rangeLabel summarizes a bucket list's span in the viewer's local time.
func rangeLabel(xTime []string, serverTZ *time.Location) string {
	local := usageview.LocalizeXTime(xTime, serverTZ)
	switch len(local) {
	case 0:
		return "no data"
	case 1:
		return local[0]
	}
	return fmt.Sprintf("%s → %s", local[0], local[len(local)-1])
}
