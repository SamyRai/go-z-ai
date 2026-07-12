package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/accounts"
	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/SamyRai/go-z-ai/pkg/usageview"
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage multiple Z.AI account credentials",
	Long:  `Add, list, switch between, and check quota for multiple named Z.AI accounts, instead of hand-editing .env.`,
}

var accountsAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a Z.AI account",
	Long:  `Registers a named account. Type (coding_plan vs pay_as_you_go) is auto-detected via a free probe unless --type is given.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsAdd,
}

var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored accounts",
	RunE:  runAccountsList,
}

var accountsUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsUse,
}

var accountsRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a stored account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsRemove,
}

var accountsShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show one account's details (defaults to the active account)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAccountsShow,
}

var accountsCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the active account (shorthand for 'accounts show')",
	Args:  cobra.NoArgs,
	RunE:  runAccountsShow,
}

var accountsQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Check quota and reset times across stored accounts",
	Long:  `Fetches GLM Coding Plan quota/usage windows for stored accounts. Defaults to all accounts; pay_as_you_go accounts are skipped since the coding-plan monitor endpoint doesn't apply to them.`,
	RunE:  runAccountsQuota,
}

var accountsUsageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Show a token/tool usage heat map across stored accounts",
	Long:  `Renders per-model token usage and per-tool call counts as a terminal heat map. The API buckets hourly for windows of 8 days or less (use --today or a small --days for that detail) and daily for 9+ days. Defaults to all accounts; pay_as_you_go accounts are skipped since the coding-plan monitor endpoint doesn't apply to them.`,
	RunE:  runAccountsUsage,
}

func init() {
	rootCmd.AddCommand(accountsCmd)
	accountsCmd.AddCommand(accountsAddCmd)
	accountsCmd.AddCommand(accountsListCmd)
	accountsCmd.AddCommand(accountsUseCmd)
	accountsCmd.AddCommand(accountsRemoveCmd)
	accountsCmd.AddCommand(accountsShowCmd)
	accountsCmd.AddCommand(accountsCurrentCmd)
	accountsCmd.AddCommand(accountsQuotaCmd)
	accountsCmd.AddCommand(accountsUsageCmd)

	accountsAddCmd.Flags().String("api-key", "", "Z.AI API key for this account (required)")
	accountsAddCmd.Flags().String("type", "", "Account type: coding_plan or pay_as_you_go (auto-detected via a free probe if omitted)")
	accountsAddCmd.Flags().String("base-url-override", "", "Custom base URL, overriding the type-derived default")
	accountsAddCmd.Flags().Bool("force", false, "Overwrite an existing account with the same name")
	accountsAddCmd.MarkFlagRequired("api-key")

	accountsListCmd.Flags().String("format", "table", "Output format (table, json)")

	accountsRemoveCmd.Flags().Bool("yes", false, "Confirm removal of the active account")

	accountsShowCmd.Flags().String("format", "table", "Output format (table, json)")
	accountsCurrentCmd.Flags().String("format", "table", "Output format (table, json)")

	accountsQuotaCmd.Flags().StringArray("only", nil, "Limit to specific account names (repeatable; default: all accounts)")
	accountsQuotaCmd.Flags().String("format", "table", "Output format (table, json)")

	accountsUsageCmd.Flags().StringArray("only", nil, "Limit to specific account names (repeatable; default: all accounts)")
	accountsUsageCmd.Flags().Int("days", 14, "Number of trailing calendar days to include (the API returns hourly buckets for <=8 days, daily for >=9 — 1-8 gets noisy/wide, so the default stays above that line)")
	accountsUsageCmd.Flags().Bool("today", false, "Shorthand for --days 1 (today only, hourly detail)")
	accountsUsageCmd.Flags().String("metric", "both", "Which usage to show: model, tool, or both")
	accountsUsageCmd.Flags().String("format", "table", "Output format (table, json)")
}

func runAccountsAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	apiKey, _ := cmd.Flags().GetString("api-key")
	if apiKey == "" {
		return fmt.Errorf("--api-key is required")
	}
	typeFlag, _ := cmd.Flags().GetString("type")
	baseURLOverride, _ := cmd.Flags().GetString("base-url-override")
	force, _ := cmd.Flags().GetBool("force")

	var accountType client.AccountType
	if typeFlag != "" {
		accountType = client.AccountType(typeFlag)
		if accountType != client.AccountTypeCodingPlan && accountType != client.AccountTypePayAsYouGo {
			return fmt.Errorf("invalid --type %q (expected %q or %q)", typeFlag, client.AccountTypeCodingPlan, client.AccountTypePayAsYouGo)
		}
	} else {
		fmt.Println("🔍 Detecting account type (free probe, no tokens spent)...")
		detected, confirmed, err := accounts.ProbeType(cmd.Context(), apiKey)
		if err != nil {
			return fmt.Errorf("failed to detect account type: %w", err)
		}
		accountType = detected
		if confirmed {
			fmt.Printf("✅ Detected type: %s (confirmed via monitor endpoint)\n", accountType)
		} else {
			fmt.Printf("⚠️  Detected type: %s (inferred by elimination — the monitor endpoint didn't confirm a coding-plan subscription; run 'zai-client usage detect' for a definitive check, or pass --type explicitly)\n", accountType)
		}
	}

	store, err := accounts.Load()
	if err != nil {
		return err
	}

	acct := accounts.Account{
		Name:            name,
		APIKey:          apiKey,
		Type:            accountType,
		BaseURLOverride: baseURLOverride,
		CreatedAt:       time.Now(),
	}

	if err := store.Add(acct, force); err != nil {
		return err
	}
	if err := store.Save(); err != nil {
		return err
	}

	fmt.Printf("✅ Account %q added (%s)\n", name, accountType)
	if store.Active == name {
		fmt.Println("   Set as the active account (first account added).")
	}

	return nil
}

func runAccountsList(cmd *cobra.Command, args []string) error {
	store, err := accounts.Load()
	if err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	list := store.List()

	if format == "json" {
		return outputJSON(list)
	}

	if len(list) == 0 {
		fmt.Println("No accounts configured. Add one with: zai-client accounts add <name> --api-key <key>")
		return nil
	}

	fmt.Printf("%-16s %-14s %-38s %-14s %-8s %s\n", "NAME", "TYPE", "BASE URL", "API KEY", "ACTIVE", "LAST USED")
	for _, acct := range list {
		baseURL, err := acct.ResolvedBaseURL()
		if err != nil {
			baseURL = "(unresolved)"
		}
		active := ""
		if acct.Name == store.Active {
			active = "✅"
		}
		fmt.Printf("%-16s %-14s %-38s %-14s %-8s %s\n", acct.Name, acct.Type, baseURL, maskAPIKey(acct.APIKey), active, usageview.FormatRelativeTime(acct.LastUsedAt))
	}

	return nil
}

func runAccountsUse(cmd *cobra.Command, args []string) error {
	name := args[0]

	store, err := accounts.Load()
	if err != nil {
		return err
	}
	if err := store.SetActive(name); err != nil {
		return err
	}
	if err := store.Save(); err != nil {
		return err
	}

	fmt.Printf("✅ Active account switched to %q\n", name)
	return nil
}

func runAccountsRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	yes, _ := cmd.Flags().GetBool("yes")

	store, err := accounts.Load()
	if err != nil {
		return err
	}
	if err := store.Remove(name, yes); err != nil {
		return err
	}
	if err := store.Save(); err != nil {
		return err
	}

	fmt.Printf("🗑️  Account %q removed\n", name)
	return nil
}

func runAccountsShow(cmd *cobra.Command, args []string) error {
	store, err := accounts.Load()
	if err != nil {
		return err
	}

	var acct accounts.Account
	var found bool
	if len(args) == 1 {
		acct, found = store.Get(args[0])
		if !found {
			return fmt.Errorf("account %q not found", args[0])
		}
	} else {
		acct, found = store.ActiveAccount()
		if !found {
			return fmt.Errorf("no active account set (run 'zai-client accounts use <name>' or 'zai-client accounts list')")
		}
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "json" {
		return outputJSON(acct)
	}

	baseURL, err := acct.ResolvedBaseURL()
	if err != nil {
		baseURL = fmt.Sprintf("(unresolved: %v)", err)
	}

	fmt.Printf("👤 Account: %s\n", acct.Name)
	fmt.Printf("   Type: %s\n", acct.Type)
	fmt.Printf("   Base URL: %s\n", baseURL)
	fmt.Printf("   API Key: %s\n", maskAPIKey(acct.APIKey))
	fmt.Printf("   Active: %t\n", acct.Name == store.Active)
	fmt.Printf("   Added: %s\n", acct.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Last Used: %s\n", usageview.FormatRelativeTime(acct.LastUsedAt))

	return nil
}

// resolveTargets returns the accounts to operate on: the ones named in only
// (erroring on any unknown name), or all stored accounts if only is empty.
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

type accountQuotaResult struct {
	Name    string                     `json:"name"`
	Type    client.AccountType         `json:"type"`
	Skipped string                     `json:"skipped,omitempty"`
	Quota   *client.QuotaLimitResponse `json:"quota,omitempty"`
	// notApplicable mirrors accountUsageResult's field of the same name —
	// see its doc comment.
	notApplicable bool
}

// fetchAccountQuota fetches quota for one account, isolating the
// per-account branching (monitor-endpoint support, client construction,
// the fetch itself) out of runAccountsQuota's loop. A non-nil error means
// client construction failed, which aborts the whole command (matching the
// pre-refactor behavior); a populated result.Skipped with a nil error is a
// per-account skip that does not abort the batch.
func fetchAccountQuota(ctx context.Context, acct accounts.Account) (accountQuotaResult, error) {
	result := accountQuotaResult{Name: acct.Name, Type: acct.Type}

	if !acct.SupportsMonitorEndpoints() {
		result.Skipped = fmt.Sprintf("quota endpoint doesn't apply to %s accounts", acct.Type)
		result.notApplicable = true
		return result, nil
	}

	apiClient, err := client.NewClient(client.Config{APIKey: acct.APIKey})
	if err != nil {
		return accountQuotaResult{}, fmt.Errorf("account %q: %w", acct.Name, err)
	}

	quota, err := apiClient.Quota().GetQuotaLimit(ctx)
	if err != nil {
		result.Skipped = err.Error()
		return result, nil
	}
	result.Quota = quota
	return result, nil
}

// printAccountQuotaResult renders one account's quota result in text mode.
func printAccountQuotaResult(result accountQuotaResult) error {
	if result.notApplicable {
		fmt.Printf("=== %s ===\n⏭️  Skipped: %s\n\n", result.Name, result.Skipped)
		return nil
	}
	fmt.Printf("=== %s ===\n", result.Name)
	if result.Quota == nil {
		fmt.Printf("❌ Failed to fetch quota: %s\n\n", result.Skipped)
		return nil
	}
	return outputQuotaLimit(result.Quota)
}

func runAccountsQuota(cmd *cobra.Command, args []string) error {
	store, err := accounts.Load()
	if err != nil {
		return err
	}

	only, _ := cmd.Flags().GetStringArray("only")
	format, _ := cmd.Flags().GetString("format")

	targets, err := resolveTargets(store, only)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fmt.Println("No accounts configured. Add one with: zai-client accounts add <name> --api-key <key>")
		return nil
	}

	results := make([]accountQuotaResult, 0, len(targets))
	for _, acct := range targets {
		result, err := fetchAccountQuota(cmd.Context(), acct)
		if err != nil {
			return err
		}
		results = append(results, result)
		if format != "json" {
			if err := printAccountQuotaResult(result); err != nil {
				return err
			}
		}
	}

	if format == "json" {
		return outputJSON(results)
	}

	return nil
}

type accountUsageResult struct {
	Name    string                     `json:"name"`
	Type    client.AccountType         `json:"type"`
	Skipped string                     `json:"skipped,omitempty"`
	Models  *client.ModelUsageResponse `json:"models,omitempty"`
	Tools   *client.ToolUsageResponse  `json:"tools,omitempty"`
	// notApplicable distinguishes "this account type doesn't support usage
	// endpoints at all" from "the fetch failed" for printAccountUsageResult's
	// benefit — both set Skipped, but read differently in text mode (⏭️ vs
	// ❌). Excluded from JSON: the Skipped message text already carries the
	// distinction there.
	notApplicable bool
}

// fetchAccountUsage fetches model/tool usage for one account, isolating the
// per-account fetch branching (monitor-endpoint support, client
// construction, per-metric requests) out of runAccountsUsage's loop. A
// non-nil error means client construction itself failed, which — matching
// the pre-refactor behavior exactly — aborts the whole command rather than
// being recorded as a per-account skip; a populated result.Skipped with a
// nil error is a per-account skip (unsupported account type or a fetch
// failure), which does not abort the batch.
func fetchAccountUsage(ctx context.Context, acct accounts.Account, metric string, start, end time.Time) (accountUsageResult, error) {
	result := accountUsageResult{Name: acct.Name, Type: acct.Type}

	if !acct.SupportsMonitorEndpoints() {
		result.Skipped = fmt.Sprintf("usage endpoint doesn't apply to %s accounts", acct.Type)
		result.notApplicable = true
		return result, nil
	}

	apiClient, err := client.NewClient(client.Config{APIKey: acct.APIKey})
	if err != nil {
		return accountUsageResult{}, fmt.Errorf("account %q: %w", acct.Name, err)
	}

	if metric == "model" || metric == "both" {
		models, err := apiClient.Quota().GetModelUsage(ctx, start, end)
		if err != nil {
			result.Skipped = fmt.Sprintf("failed to fetch model usage: %v", err)
			return result, nil
		}
		result.Models = models
	}
	if metric == "tool" || metric == "both" {
		tools, err := apiClient.Quota().GetToolUsage(ctx, start, end)
		if err != nil {
			result.Skipped = fmt.Sprintf("failed to fetch tool usage: %v", err)
			return result, nil
		}
		result.Tools = tools
	}
	return result, nil
}

// printAccountUsageResult renders one account's usage result in text mode.
func printAccountUsageResult(result accountUsageResult) {
	if result.notApplicable {
		fmt.Printf("=== %s ===\n⏭️  Skipped: %s\n\n", result.Name, result.Skipped)
		return
	}
	fmt.Printf("=== %s ===\n", result.Name)
	if result.Skipped != "" {
		fmt.Printf("❌ %s\n\n", result.Skipped)
		return
	}
	if result.Models != nil {
		printModelHeatmap(result.Models)
	}
	if result.Tools != nil {
		printToolHeatmap(result.Tools)
	}
}

func runAccountsUsage(cmd *cobra.Command, args []string) error {
	store, err := accounts.Load()
	if err != nil {
		return err
	}

	only, _ := cmd.Flags().GetStringArray("only")
	days, _ := cmd.Flags().GetInt("days")
	today, _ := cmd.Flags().GetBool("today")
	metric, _ := cmd.Flags().GetString("metric")
	format, _ := cmd.Flags().GetString("format")

	if metric != "model" && metric != "tool" && metric != "both" {
		return fmt.Errorf("invalid --metric %q (expected \"model\", \"tool\", or \"both\")", metric)
	}
	if days < 1 {
		days = 1
	}
	start, end := usageview.Window(days, today)

	targets, err := resolveTargets(store, only)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		fmt.Println("No accounts configured. Add one with: zai-client accounts add <name> --api-key <key>")
		return nil
	}

	if format != "json" {
		fmt.Println("Legend: (blank)=0  ░▒▓█=low→peak, scaled per row against that row's own max")
		fmt.Println()
	}

	results := make([]accountUsageResult, 0, len(targets))
	for _, acct := range targets {
		result, err := fetchAccountUsage(cmd.Context(), acct, metric, start, end)
		if err != nil {
			return err
		}
		results = append(results, result)
		if format != "json" {
			printAccountUsageResult(result)
		}
	}

	if format == "json" {
		return outputJSON(results)
	}
	return nil
}

// rangeLabel summarizes a bucket list's span for display.
func rangeLabel(xTime []string) string {
	switch len(xTime) {
	case 0:
		return "no data"
	case 1:
		return xTime[0]
	default:
		return fmt.Sprintf("%s → %s", xTime[0], xTime[len(xTime)-1])
	}
}

func printModelHeatmap(u *client.ModelUsageResponse) {
	d := u.Data
	fmt.Printf("📈 Model usage (%s, %s)\n", rangeLabel(d.XTime), d.Granularity)

	if len(d.ModelDataList) == 0 {
		fmt.Println("  No model usage in this window.")
		fmt.Println()
		return
	}

	series := append([]client.ModelUsageSeries(nil), d.ModelDataList...)
	sort.Slice(series, func(i, j int) bool { return series[i].SortOrder < series[j].SortOrder })

	nameWidth := 0
	for _, m := range series {
		if len(m.ModelName) > nameWidth {
			nameWidth = len(m.ModelName)
		}
	}

	for _, m := range series {
		fmt.Printf("  %-*s %s  %s tokens\n", nameWidth, m.ModelName, usageview.HeatmapBlocks(m.TokensUsage), usageview.FormatCount(m.TotalTokens))
	}
	fmt.Printf("  Total: %s calls, %s tokens\n\n", usageview.FormatCount(d.TotalUsage.TotalModelCallCount), usageview.FormatCount(d.TotalUsage.TotalTokensUsage))
}

func printToolHeatmap(u *client.ToolUsageResponse) {
	d := u.Data
	fmt.Printf("🔧 Tool usage (%s, %s)\n", rangeLabel(d.XTime), d.Granularity)

	if len(d.ToolDataList) == 0 {
		fmt.Println("  No tool usage in this window.")
		fmt.Println()
		return
	}

	series := append([]client.ToolUsageSeries(nil), d.ToolDataList...)
	sort.Slice(series, func(i, j int) bool { return series[i].SortOrder < series[j].SortOrder })

	// ToolUsageSeries only carries the Chinese ToolName; the English label
	// lives on ToolSummaryList, keyed by ToolCode.
	englishNames := make(map[string]string, len(d.ToolSummaryList))
	for _, s := range d.ToolSummaryList {
		englishNames[s.ToolCode] = s.ToolNameI18n
	}
	label := func(t client.ToolUsageSeries) string {
		if name := englishNames[t.ToolCode]; name != "" {
			return name
		}
		return t.ToolCode
	}

	nameWidth := 0
	for _, t := range series {
		if l := len(label(t)); l > nameWidth {
			nameWidth = l
		}
	}

	for _, t := range series {
		fmt.Printf("  %-*s %s  %s calls\n", nameWidth, label(t), usageview.HeatmapBlocks(t.UsageCount), usageview.FormatCount(t.TotalUsageCount))
	}
	fmt.Println()
}
