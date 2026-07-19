package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/SamyRai/go-z-ai/internal/coding"
	"github.com/spf13/cobra"
)

// codingCmd ports Z.AI's official @z_ai/coding-helper ("chelper") Node CLI to
// Go: it manages GLM Coding Plan credentials and loads them into supported
// coding tools (Claude Code, OpenCode, Crush, Factory Droid, Cursor) using
// each tool's native config format. The credential store at
// ~/.chelper/config.yaml is shared/compatible with the Node helper.
var codingCmd = &cobra.Command{
	Use:   "coding",
	Short: "GLM Coding Plan credentials & coding-tool configuration",
	Long: `Manage GLM Coding Plan credentials and configure coding tools to use Z.AI.

A Go port of the official @z_ai/coding-helper ("chelper"). Supported tools:
Claude Code, OpenCode, Crush, Factory Droid, Cursor. Plans:
  glm_coding_plan_global  -> https://api.z.ai
  glm_coding_plan_china   -> https://open.bigmodel.cn

Credentials are stored in ~/.chelper/config.yaml (compatible with chelper).`,
}

var codingAuthCmd = &cobra.Command{
	Use:   "auth [plan] [key] | revoke | reload <tool>",
	Short: "Store/validate/revoke the GLM Coding Plan key, or reload it into a tool",
	Long: `Manage the stored GLM Coding Plan credential.

  coding auth glm_coding_plan_global <key>   validate and store the Global plan key
  coding auth glm_coding_plan_china <key>     validate and store the China plan key
  coding auth revoke                          remove the stored key (keeps plan)
  coding auth reload <tool>                   load stored creds into a tool`,
	Args: cobra.MaximumNArgs(2),
	RunE: runCodingAuth,
}

var codingLoadCmd = &cobra.Command{
	Use:   "load <tool>",
	Short: "Load stored credentials into a coding tool",
	Long:  `Load the stored GLM Coding Plan into a tool's native config. Overrides: --plan, --key.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCodingLoad,
}

var codingUnloadCmd = &cobra.Command{
	Use:   "unload <tool>",
	Short: "Remove Z.AI configuration from a coding tool",
	Args:  cobra.ExactArgs(1),
	RunE:  runCodingUnload,
}

var codingStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show stored credentials and per-tool configuration status",
	RunE:  runCodingStatus,
}

var codingToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List supported coding tools, install status, and config paths",
	RunE:  runCodingTools,
}

var codingDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Health check: credentials and installed tools",
	RunE:  runCodingDoctor,
}

var codingMcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage Z.AI's Vision MCP server in a coding tool",
	Long: `Register or remove Z.AI's official Vision MCP Server (@z_ai/mcp-server) —
screenshot OCR, error-screenshot diagnosis, diagram/chart understanding, and
image/video analysis via GLM-4.6V — in a supported coding tool. Requires
Node.js (npx) to actually run the server; this only writes the config.`,
}

var codingMcpAddCmd = &cobra.Command{
	Use:   "add <tool>",
	Short: "Register the Vision MCP server for a tool",
	Args:  cobra.ExactArgs(1),
	RunE:  runCodingMcpAdd,
}

var codingMcpRemoveCmd = &cobra.Command{
	Use:   "remove <tool>",
	Short: "Remove the Vision MCP server entry from a tool",
	Args:  cobra.ExactArgs(1),
	RunE:  runCodingMcpRemove,
}

var codingMcpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show which tools have the Vision MCP server registered",
	RunE:  runCodingMcpStatus,
}

var (
	codingNoValidate  bool
	codingPlanFlag    string
	codingKeyFlag     string
	codingMcpKeyFlag  string
	codingNoModelMap  bool
	codingHaikuModel  string
	codingSonnetModel string
	codingOpusModel   string
	codingAutoCompact int
	codingMaxThinking int
	codingMaxOutput   int
)

func init() {
	rootCmd.AddCommand(codingCmd)
	codingCmd.AddCommand(codingAuthCmd, codingLoadCmd, codingUnloadCmd, codingStatusCmd, codingToolsCmd, codingDoctorCmd, codingMcpCmd)
	codingMcpCmd.AddCommand(codingMcpAddCmd, codingMcpRemoveCmd, codingMcpStatusCmd)

	codingAuthCmd.Flags().BoolVar(&codingNoValidate, "no-validate", false, "Store the key without validating against the API")
	codingLoadCmd.Flags().StringVar(&codingPlanFlag, "plan", "", "Plan override (glm_coding_plan_global | glm_coding_plan_china)")
	codingLoadCmd.Flags().StringVar(&codingKeyFlag, "key", "", "API key override (uses stored creds if omitted)")
	codingMcpAddCmd.Flags().StringVar(&codingMcpKeyFlag, "key", "", "API key override (uses stored creds if omitted)")

	// Claude Code tuning — persistent so 'auth reload' shares the same defaults.
	// Defaults match Z.AI's recommended integration; flags override / disable.
	pf := codingCmd.PersistentFlags()
	pf.BoolVar(&codingNoModelMap, "no-model-mapping", false, "Omit ANTHROPIC_DEFAULT_*_MODEL tier mapping (match @z_ai/coding-helper exactly)")
	pf.StringVar(&codingHaikuModel, "haiku", "", "Override the Claude 'haiku' tier model id")
	pf.StringVar(&codingSonnetModel, "sonnet", "", "Override the Claude 'sonnet' tier model id")
	pf.StringVar(&codingOpusModel, "opus", "", "Override the Claude 'opus' tier model id")
	pf.IntVar(&codingAutoCompact, "auto-compact-window", 1000000, "CLAUDE_CODE_AUTO_COMPACT_WINDOW in tokens (0 to omit; use 128000 for 128K-context models)")
	pf.IntVar(&codingMaxThinking, "max-thinking-tokens", 0, "MAX_THINKING_TOKENS extended-thinking budget (0 to omit)")
	pf.IntVar(&codingMaxOutput, "max-output-tokens", 0, "CLAUDE_CODE_MAX_OUTPUT_TOKENS (0 to omit)")
}

func runCodingAuth(cmd *cobra.Command, args []string) error {
	store, err := coding.NewStore()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	switch args[0] {
	case "revoke":
		if err := store.RevokeAPIKey(); err != nil {
			return err
		}
		fmt.Println("✓ API key revoked (plan choice retained).")
		return nil
	case "reload":
		if len(args) < 2 {
			return fmt.Errorf("usage: go-z-ai coding auth reload <tool>")
		}
		return loadToolInto(store, args[1])
	}

	// Otherwise: auth <plan> <key>
	if len(args) < 2 {
		return fmt.Errorf("usage: go-z-ai coding auth <plan> <key>  |  revoke  |  reload <tool>")
	}
	plan, key := args[0], args[1]
	if !coding.IsValidPlan(plan) {
		return fmt.Errorf("invalid plan %q (want glm_coding_plan_global or glm_coding_plan_china)", plan)
	}

	if !codingNoValidate {
		fmt.Println("Validating API key…")
		if err := coding.ValidateAPIKey(cmd.Context(), plan, key); err != nil {
			if errors.Is(err, coding.ErrInvalidAPIKey) {
				return fmt.Errorf("Z.AI rejected the key (401); pass --no-validate to store it anyway")
			}
			return fmt.Errorf("validation failed: %w (pass --no-validate to store offline)", err)
		}
	}

	if err := store.SetPlan(plan); err != nil {
		return err
	}
	if err := store.SetAPIKey(key); err != nil {
		return err
	}
	fmt.Printf("✓ Stored %s (%s)\n", coding.DisplayName(plan), maskAPIKey(key))
	return nil
}

func runCodingLoad(cmd *cobra.Command, args []string) error {
	store, err := coding.NewStore()
	if err != nil {
		return err
	}
	return loadToolInto(store, args[0])
}

func runCodingUnload(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	tool, err := coding.FindTool(args[0])
	if err != nil {
		return err
	}
	if err := coding.Unload(home, args[0]); err != nil {
		return err
	}
	fmt.Printf("✓ Removed Z.AI configuration from %s (%s)\n", tool.DisplayName, tool.ConfigPath(home))
	return nil
}

func runCodingStatus(cmd *cobra.Command, args []string) error {
	store, err := coding.NewStore()
	if err != nil {
		return err
	}
	c, _ := store.Load()
	home, _ := os.UserHomeDir()

	fmt.Println("Stored credentials")
	fmt.Println("==================")
	if c.Plan != "" {
		fmt.Printf("  Plan: %s\n", coding.DisplayName(c.Plan))
	} else {
		fmt.Println("  Plan: (none)")
	}
	if c.APIKey != "" {
		fmt.Printf("  Key:  %s\n", maskAPIKey(c.APIKey))
	} else {
		fmt.Println("  Key:  (none — run 'go-z-ai coding auth <plan> <key>')")
	}

	fmt.Println("\nCoding tools")
	fmt.Println("============")
	for _, t := range coding.Tools {
		d, _ := coding.Detect(home, t.ID)
		installed := "not installed"
		if t.IsInstalled() {
			installed = "installed"
		}
		status := "native"
		if d.Configured {
			status = "Z.AI"
			if d.Plan != "" {
				status = "Z.AI · " + coding.DisplayName(d.Plan)
			}
		}
		if mcpConfigured, err := coding.DetectMCPConfigured(home, t.ID); err == nil && mcpConfigured {
			status += " · vision-mcp"
		}
		fmt.Printf("  %-14s %-15s %s\n", t.DisplayName, installed, status)
		if d.ModelMap != nil {
			fmt.Printf("                  models: haiku=%s sonnet=%s opus=%s\n", d.ModelMap.Haiku, d.ModelMap.Sonnet, d.ModelMap.Opus)
		}
	}
	return nil
}

func runCodingTools(cmd *cobra.Command, args []string) error {
	home, _ := os.UserHomeDir()
	fmt.Printf("%-14s %-8s %-10s %s\n", "TOOL", "CMD", "INSTALLED", "CONFIG PATH")
	for _, t := range coding.Tools {
		inst := "no"
		if t.IsInstalled() {
			inst = "yes"
		}
		fmt.Printf("%-14s %-8s %-10s %s\n", t.DisplayName, t.Command, inst, t.ConfigPath(home))
	}
	return nil
}

func runCodingDoctor(cmd *cobra.Command, args []string) error {
	allGood := true
	store, err := coding.NewStore()
	if err != nil {
		return err
	}
	c, _ := store.Load()
	if c.Plan == "" || c.APIKey == "" {
		fmt.Println("⚠  No credentials stored (run 'go-z-ai coding auth <plan> <key>')")
		allGood = false
	} else {
		fmt.Printf("✓ Credentials: %s / %s\n", coding.DisplayName(c.Plan), maskAPIKey(c.APIKey))
	}

	home, _ := os.UserHomeDir()
	anyInstalled := false
	for _, t := range coding.Tools {
		if t.IsInstalled() {
			anyInstalled = true
			d, _ := coding.Detect(home, t.ID)
			tag := "native config"
			if d.Configured {
				tag = "configured for Z.AI"
			}
			if mcpConfigured, err := coding.DetectMCPConfigured(home, t.ID); err == nil && mcpConfigured {
				tag += ", vision-mcp configured"
			}
			fmt.Printf("✓ %s installed (%s)\n", t.DisplayName, tag)
		}
	}
	if !anyInstalled {
		fmt.Println("⚠  No supported coding tools detected on PATH")
		allGood = false
	}
	if !coding.HasNPX() {
		fmt.Println("⚠  npx not found on PATH — Vision MCP registration would write config, but the server itself needs Node.js to run")
	}
	if allGood {
		fmt.Println("\nAll good.")
	}
	return nil
}

func runCodingMcpAdd(cmd *cobra.Command, args []string) error {
	store, err := coding.NewStore()
	if err != nil {
		return err
	}
	key, err := resolveMcpKey(store)
	if err != nil {
		return err
	}
	tool, err := coding.FindTool(args[0])
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := coding.LoadMCP(home, args[0], key); err != nil {
		return err
	}
	fmt.Printf("✓ Registered Z.AI Vision MCP server for %s\n", tool.DisplayName)
	if !coding.HasNPX() {
		fmt.Println("⚠  npx not found on PATH — install Node.js before the server can actually run")
	}
	return nil
}

func runCodingMcpRemove(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	tool, err := coding.FindTool(args[0])
	if err != nil {
		return err
	}
	if err := coding.UnloadMCP(home, args[0]); err != nil {
		return err
	}
	fmt.Printf("✓ Removed the Vision MCP server entry from %s\n", tool.DisplayName)
	return nil
}

func runCodingMcpStatus(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	fmt.Printf("%-14s %s\n", "TOOL", "VISION MCP")
	for _, t := range coding.Tools {
		configured, err := coding.DetectMCPConfigured(home, t.ID)
		if err != nil {
			return err
		}
		status := "not configured"
		if configured {
			status = "configured"
		}
		fmt.Printf("%-14s %s\n", t.DisplayName, status)
	}
	if !coding.HasNPX() {
		fmt.Println("\n⚠  npx not found on PATH — install Node.js to actually run the server")
	}
	return nil
}

// resolveMcpKey returns the API key for MCP registration: --key override,
// else the stored chelper credential. Unlike resolveCodingCreds, this
// doesn't require a plan — the Vision MCP server isn't plan-routed.
func resolveMcpKey(store *coding.Store) (string, error) {
	if codingMcpKeyFlag != "" {
		return codingMcpKeyFlag, nil
	}
	c, err := store.Load()
	if err != nil {
		return "", err
	}
	if c.APIKey == "" {
		return "", fmt.Errorf("no API key configured (run 'go-z-ai coding auth <plan> <key>' or pass --key)")
	}
	return c.APIKey, nil
}

// loadToolInto resolves credentials (flags override, else stored) and writes them
// into the named tool's config. For Claude Code it applies the tuning flags
// (defaulting to Z.AI's recommended model mapping + auto-compact window).
func loadToolInto(store *coding.Store, toolID string) error {
	plan, key, err := resolveCodingCreds(store)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	tool, err := coding.FindTool(toolID)
	if err != nil {
		return err
	}

	if tool.ID == "claude-code" {
		opts := resolvedClaudeOptions()
		if err := coding.LoadClaudeCodeOpts(home, plan, key, opts); err != nil {
			return err
		}
	} else {
		if err := coding.Load(home, toolID, plan, key); err != nil {
			return err
		}
	}

	fmt.Printf("✓ Loaded %s into %s\n   %s\n", coding.DisplayName(plan), tool.DisplayName, tool.ConfigPath(home))
	if tool.ID == "claude-code" {
		summarizeClaudeOpts(resolvedClaudeOptions())
	}
	return nil
}

// resolvedClaudeOptions starts from Z.AI's recommended Claude Code tuning and
// applies the CLI flag overrides (per-tier model ids, auto-compact window, and
// the thinking/output token budgets).
func resolvedClaudeOptions() coding.ClaudeOptions {
	opts := coding.DefaultClaudeOptions()
	if codingNoModelMap {
		opts.ModelMap = nil
	} else if codingHaikuModel != "" || codingSonnetModel != "" || codingOpusModel != "" {
		if opts.ModelMap == nil {
			opts.ModelMap = &coding.ModelMap{}
		}
		if codingHaikuModel != "" {
			opts.ModelMap.Haiku = codingHaikuModel
		}
		if codingSonnetModel != "" {
			opts.ModelMap.Sonnet = codingSonnetModel
		}
		if codingOpusModel != "" {
			opts.ModelMap.Opus = codingOpusModel
		}
	}
	opts.AutoCompactWindow = codingAutoCompact
	opts.MaxThinkingTokens = codingMaxThinking
	opts.MaxOutputTokens = codingMaxOutput
	return opts
}

// summarizeClaudeOpts prints the Claude Code tuning that was applied.
func summarizeClaudeOpts(opts coding.ClaudeOptions) {
	if opts.ModelMap != nil {
		fmt.Printf("   models: haiku=%s sonnet=%s opus=%s\n", opts.ModelMap.Haiku, opts.ModelMap.Sonnet, opts.ModelMap.Opus)
	}
	if opts.AutoCompactWindow > 0 {
		fmt.Printf("   auto-compact-window: %d\n", opts.AutoCompactWindow)
	}
	if opts.MaxThinkingTokens > 0 {
		fmt.Printf("   max-thinking-tokens: %d\n", opts.MaxThinkingTokens)
	}
	if opts.MaxOutputTokens > 0 {
		fmt.Printf("   max-output-tokens: %d\n", opts.MaxOutputTokens)
	}
}

// resolveCodingCreds returns plan+key from flags, falling back to the store.
func resolveCodingCreds(store *coding.Store) (plan, key string, err error) {
	plan = codingPlanFlag
	key = codingKeyFlag
	if plan == "" || key == "" {
		c, e := store.Load()
		if e != nil {
			return "", "", e
		}
		if plan == "" {
			plan = c.Plan
		}
		if key == "" {
			key = c.APIKey
		}
	}
	if plan == "" || key == "" {
		return "", "", fmt.Errorf("no credentials configured (run 'go-z-ai coding auth <plan> <key>' or pass --plan/--key)")
	}
	if !coding.IsValidPlan(plan) {
		return "", "", fmt.Errorf("invalid plan %q", plan)
	}
	return plan, key, nil
}
