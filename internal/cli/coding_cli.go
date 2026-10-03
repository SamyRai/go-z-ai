package cli

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/SamyRai/go-z-ai/internal/coding"
	"github.com/spf13/cobra"
)

// codingCmd is a Go port of Z.AI's official @z_ai/coding-helper ("chelper"):
// it stores GLM Coding Plan credentials and writes them into supported coding
// tools in each tool's native config format. The credential store at
// ~/.chelper/config.yaml is shared with the Node helper.
var codingCmd = &cobra.Command{
	Use:   "coding",
	Short: "GLM Coding Plan credentials & coding-tool configuration",
	Long:  codingLong(),
}

// codingLong renders the command's help from the registries, so the tool and
// plan lists can't drift from what the coding package supports.
func codingLong() string {
	var b strings.Builder
	b.WriteString("Manage GLM Coding Plan credentials and configure coding tools to use Z.AI.\n\n")
	b.WriteString("A Go port of the official @z_ai/coding-helper (\"chelper\"). Supported tools:\n")
	for _, t := range coding.Tools {
		fmt.Fprintf(&b, "  %-14s %s\n", t.ID, t.DisplayName)
	}
	b.WriteString("\nPlans:\n")
	for _, p := range coding.Plans {
		fmt.Fprintf(&b, "  %-24s %s\n", p, coding.Region(p).CodingBaseURL())
	}
	b.WriteString("\nCredentials are stored in ~/.chelper/config.yaml (compatible with chelper).")
	return b.String()
}

var codingAuthCmd = &cobra.Command{
	Use:   "auth [plan] [key] | revoke | reload <tool>",
	Short: "Store/validate/revoke the GLM Coding Plan key, or reload it into a tool",
	Long: `Manage the stored GLM Coding Plan credential.

  coding auth glm_coding_plan_global <key>   validate and store the Global plan key
  coding auth glm_coding_plan_china <key>    validate and store the China plan key
  coding auth revoke                         remove the stored key (keeps plan)
  coding auth reload <tool>                  load stored creds into a tool`,
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

// codingFlags holds the coding command tree's flag values.
var codingFlags struct {
	noValidate bool
	plan, key  string
	servers    []string
	claude     claudeFlags
}

func init() {
	rootCmd.AddCommand(codingCmd)
	codingCmd.AddCommand(codingAuthCmd, codingLoadCmd, codingUnloadCmd)

	codingAuthCmd.Flags().BoolVar(&codingFlags.noValidate, "no-validate", false, "Store the key without validating against the API")
	addCredentialFlags(codingLoadCmd)
	codingFlags.claude.register(codingLoadCmd, codingAuthCmd)
}

// addCredentialFlags registers --plan/--key, which override the stored
// credentials (see codingCredentials).
func addCredentialFlags(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().StringVar(&codingFlags.plan, "plan", "", "Plan override ("+strings.Join(coding.Plans, " | ")+")")
		c.Flags().StringVar(&codingFlags.key, "key", "", "API key override (uses stored creds if omitted)")
	}
}

// claudeFlags tunes the Claude Code config written by 'load' and 'auth
// reload'; unset flags keep the catalog-derived defaults of
// coding.DefaultClaudeOptions.
type claudeFlags struct {
	noModelMap          bool
	haiku, sonnet, opus string
	autoCompact         int
	maxThinking         int
	maxOutput           int
}

func (f *claudeFlags) register(cmds ...*cobra.Command) {
	for _, cmd := range cmds {
		fs := cmd.Flags()
		fs.BoolVar(&f.noModelMap, "no-model-mapping", false, "Claude Code: omit the ANTHROPIC_DEFAULT_*_MODEL tier mapping (match @z_ai/coding-helper exactly)")
		fs.StringVar(&f.haiku, "haiku", "", "Claude Code: override the 'haiku' tier model id")
		fs.StringVar(&f.sonnet, "sonnet", "", "Claude Code: override the 'sonnet' tier model id")
		fs.StringVar(&f.opus, "opus", "", "Claude Code: override the 'opus' tier model id")
		fs.IntVar(&f.autoCompact, "auto-compact-window", 0, "Claude Code: CLAUDE_CODE_AUTO_COMPACT_WINDOW in tokens (default: the main model's context; 0 to omit)")
		fs.IntVar(&f.maxThinking, "max-thinking-tokens", 0, "Claude Code: MAX_THINKING_TOKENS extended-thinking budget (0 to omit)")
		fs.IntVar(&f.maxOutput, "max-output-tokens", 0, "Claude Code: CLAUDE_CODE_MAX_OUTPUT_TOKENS (0 to omit)")
	}
}

// options applies the flags the user set over the recommended defaults.
func (f claudeFlags) options(cmd *cobra.Command) coding.ClaudeOptions {
	opts := coding.DefaultClaudeOptions()
	switch {
	case f.noModelMap:
		opts.ModelMap = nil
	case opts.ModelMap != nil:
		m := opts.ModelMap
		m.Haiku, m.Sonnet, m.Opus = cmp.Or(f.haiku, m.Haiku), cmp.Or(f.sonnet, m.Sonnet), cmp.Or(f.opus, m.Opus)
	}
	if cmd.Flags().Changed("auto-compact-window") {
		opts.AutoCompactWindow = f.autoCompact
	}
	opts.MaxThinkingTokens = f.maxThinking
	opts.MaxOutputTokens = f.maxOutput
	return opts
}

func runCodingAuth(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	store, err := coding.NewStore()
	if err != nil {
		return err
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
			return errors.New("usage: go-z-ai coding auth reload <tool>")
		}
		return loadTool(cmd, store, args[1])
	}

	if len(args) < 2 {
		return errors.New("usage: go-z-ai coding auth <plan> <key>  |  revoke  |  reload <tool>")
	}
	plan, key := args[0], args[1]
	if !coding.IsValidPlan(plan) {
		return fmt.Errorf("invalid plan %q (want %s)", plan, strings.Join(coding.Plans, " or "))
	}
	if !codingFlags.noValidate {
		progressf("Validating API key…\n")
		if err := coding.ValidateAPIKey(cmd.Context(), plan, key); err != nil {
			if errors.Is(err, coding.ErrInvalidAPIKey) {
				return errors.New("Z.AI rejected the key (401); pass --no-validate to store it anyway")
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
	return loadTool(cmd, store, args[0])
}

// loadTool writes the resolved credentials into the named tool's config. For
// Claude Code it also applies the tuning flags.
func loadTool(cmd *cobra.Command, store *coding.Store, toolID string) error {
	tool, err := coding.FindTool(toolID)
	if err != nil {
		return err
	}
	plan, key, err := codingCredentials(store)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfg := coding.NewLoadConfig(plan, key)
	cfg.Claude = codingFlags.claude.options(cmd)
	if err := tool.Load(home, cfg); err != nil {
		return err
	}
	fmt.Printf("✓ Loaded %s into %s\n   %s\n", coding.DisplayName(plan), tool.DisplayName, tool.ConfigPath(home))
	if tool.ID == coding.ClaudeCodeID {
		printClaudeOptions(cfg.Claude)
	}
	return nil
}

func runCodingUnload(cmd *cobra.Command, args []string) error {
	tool, err := coding.FindTool(args[0])
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	switch err := tool.Unload(home); {
	case errors.Is(err, coding.ErrNotConfigured):
		fmt.Printf("%s is not configured for a Z.AI plan; nothing to remove.\n", tool.DisplayName)
		return nil
	case err != nil:
		return err
	}
	fmt.Printf("✓ Removed Z.AI configuration from %s (%s)\n", tool.DisplayName, tool.ConfigPath(home))
	return nil
}

// printClaudeOptions prints the Claude Code tuning that was applied.
func printClaudeOptions(opts coding.ClaudeOptions) {
	if m := opts.ModelMap; m != nil {
		fmt.Printf("   models: haiku=%s sonnet=%s opus=%s\n", m.Haiku, m.Sonnet, m.Opus)
	}
	for _, v := range []struct {
		name  string
		value int
	}{
		{"auto-compact-window", opts.AutoCompactWindow},
		{"max-thinking-tokens", opts.MaxThinkingTokens},
		{"max-output-tokens", opts.MaxOutputTokens},
	} {
		if v.value > 0 {
			fmt.Printf("   %s: %d\n", v.name, v.value)
		}
	}
}

// codingCredentials returns the plan and key from --plan/--key, falling back
// to the stored credentials.
func codingCredentials(store *coding.Store) (plan, key string, err error) {
	plan, key = codingFlags.plan, codingFlags.key
	if plan == "" || key == "" {
		stored, err := store.Load()
		if err != nil {
			return "", "", err
		}
		plan = cmp.Or(plan, stored.Plan)
		key = cmp.Or(key, stored.APIKey)
	}
	if plan == "" || key == "" {
		return "", "", errors.New("no credentials configured (run 'go-z-ai coding auth <plan> <key>' or pass --plan/--key)")
	}
	if !coding.IsValidPlan(plan) {
		return "", "", fmt.Errorf("invalid plan %q (want %s)", plan, strings.Join(coding.Plans, " or "))
	}
	return plan, key, nil
}
