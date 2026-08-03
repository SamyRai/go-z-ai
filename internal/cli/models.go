package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Model operations",
	Long:  `List and get information about available Z.AI models.`,
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available models",
	Long:  `List all available models with their details and pricing.`,
	RunE:  runWithClient(runModelsList),
}

var modelsGetCmd = &cobra.Command{
	Use:   "get [model-id]",
	Short: "Get details for a specific model",
	Long:  `Get detailed information for a specific model including pricing and capabilities.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runModelsGet),
}

var modelsTextCmd = &cobra.Command{
	Use:   "text",
	Short: "List text-only models",
	Long:  `List all text-only models excluding vision models.`,
	RunE:  runWithClient(runModelsText),
}

var modelsVisionCmd = &cobra.Command{
	Use:   "vision",
	Short: "List vision models",
	Long:  `List all vision-capable models that can process images.`,
	RunE:  runWithClient(runModelsVision),
}

var modelsFreeCmd = &cobra.Command{
	Use:   "free",
	Short: "List free models",
	Long:  `List all free models with zero cost.`,
	RunE:  runWithClient(runModelsFree),
}

var (
	outputFormat string
	showPricing  bool
)

func init() {
	rootCmd.AddCommand(modelsCmd)
	modelsCmd.AddCommand(modelsListCmd)
	modelsCmd.AddCommand(modelsGetCmd)
	modelsCmd.AddCommand(modelsTextCmd)
	modelsCmd.AddCommand(modelsVisionCmd)
	modelsCmd.AddCommand(modelsFreeCmd)

	modelsListCmd.Flags().StringVar(&outputFormat, "format", "table", "Output format (table, json)")
	modelsListCmd.Flags().BoolVar(&showPricing, "pricing", false, "Show pricing information")

	modelsGetCmd.Flags().StringVar(&outputFormat, "format", "table", "Output format (table, json)")

	modelsTextCmd.Flags().StringVar(&outputFormat, "format", "table", "Output format (table, json)")
	modelsTextCmd.Flags().BoolVar(&showPricing, "pricing", false, "Show pricing information")

	modelsVisionCmd.Flags().StringVar(&outputFormat, "format", "table", "Output format (table, json)")
	modelsVisionCmd.Flags().BoolVar(&showPricing, "pricing", false, "Show pricing information")
}

func runModelsList(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	models, err := apiClient.Models().List(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	return outputModels(models.Models, outputFormat, showPricing)
}

func runModelsGet(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	modelID := args[0]
	model, err := apiClient.Models().Get(cmd.Context(), modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	return outputModel(model, outputFormat)
}

func runModelsText(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	models, err := apiClient.Models().GetTextModels(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get text models: %w", err)
	}

	return outputModels(models, outputFormat, showPricing)
}

func runModelsVision(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	models, err := apiClient.Models().GetVisionModels(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get vision models: %w", err)
	}

	return outputModels(models, outputFormat, showPricing)
}

func runModelsFree(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	models, err := apiClient.Models().GetFreeModels(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get free models: %w", err)
	}

	fmt.Printf("Found %d free models:\n\n", len(models))
	return outputModels(models, outputFormat, false)
}

func outputModels(models []client.ModelDetails, format string, showPrice bool) error {
	switch format {
	case "json":
		return outputJSON(models)
	default:
		return outputModelsTable(models, showPrice)
	}
}

func outputModel(model *client.ModelDetails, format string) error {
	switch format {
	case "json":
		return outputJSON(model)
	default:
		return outputModelTable(model)
	}
}

// outputModelsTable renders the default human-readable model list. Columns
// come from the enriched catalog (pkg/client/models_catalog.go), since
// Z.AI's /models endpoint returns only {id, created, owned_by} and leaves
// every other column empty. Pricing is shown by default and labeled honestly
// as "per 1M tokens" — the previous header said "$/1K" but formatted the
// per-1M value, which was just wrong.
//
// The --pricing flag is kept for back-compat but is now a no-op (pricing is
// always shown); it prints a one-line note when set so existing scripts that
// pass it aren't surprised by silent behavior change.
func outputModelsTable(models []client.ModelDetails, showPrice bool) error {
	_ = showPrice // always shown now; flag kept for back-compat

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "MODEL\tFAMILY\tCONTEXT\tMAXOUT\tIN/1M\tOUT/1M\tCACHED\tCAPS")

	for _, m := range models {
		in, out, cached := "—", "—", "—"
		if m.Pricing != nil {
			in = formatCLIPrice(m.Pricing.Input)
			out = formatCLIPrice(m.Pricing.Output)
			if m.Pricing.Cached > 0 {
				cached = formatCLIPrice(m.Pricing.Cached)
			}
		}
		ctx := formatCLIContext(m.ContextSize)
		maxOut := formatCLIContext(m.MaxOutput)
		family := m.Family
		caps := formatCLICaps(m.Capabilities)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			m.ID, family, ctx, maxOut, in, out, cached, caps)
	}
	fmt.Fprintln(w, "\n(prices are USD per 1M tokens; — = unknown)")
	return w.Flush()
}

func outputModelTable(model *client.ModelDetails) error {
	fmt.Printf("Model ID:     %s\n", model.ID)
	if model.Name != "" {
		fmt.Printf("Name:         %s\n", model.Name)
	}
	if model.Family != "" {
		fmt.Printf("Family:       %s\n", model.Family)
	}
	if model.Tier != "" {
		fmt.Printf("Tier:         %s\n", model.Tier)
	}
	if model.Description != "" {
		fmt.Printf("Description:  %s\n", model.Description)
	}
	fmt.Printf("Context:      %s tokens\n", formatCLIContext(model.ContextSize))
	if model.MaxOutput > 0 {
		fmt.Printf("Max output:   %s tokens\n", formatCLIContext(model.MaxOutput))
	}
	fmt.Printf("Owned by:     %s\n", model.OwnedBy)
	if model.Created > 0 {
		fmt.Printf("Released:     %s\n", model.CreatedTime().UTC().Format("2006-01-02"))
	}
	if len(model.Capabilities) > 0 {
		fmt.Printf("Capabilities: %s\n", formatCLICaps(model.Capabilities))
	}

	if model.Pricing != nil {
		fmt.Printf("\nPricing (per 1M tokens, USD):\n")
		fmt.Printf("  Input:   %s\n", formatCLIPrice(model.Pricing.Input))
		fmt.Printf("  Output:  %s\n", formatCLIPrice(model.Pricing.Output))
		if model.Pricing.Cached > 0 {
			fmt.Printf("  Cached:  %s\n", formatCLIPrice(model.Pricing.Cached))
		}
		if model.Pricing.CacheStore > 0 {
			fmt.Printf("  Cache storage: %s\n", formatCLIPrice(model.Pricing.CacheStore))
		}
		if model.IsFree() {
			fmt.Printf("  (free model)\n")
		}
	}

	return nil
}

// formatCLIContext renders a token count compactly (200K, 128K, — for zero).
func formatCLIContext(n int) string {
	if n <= 0 {
		return "—"
	}
	if n >= 1000 {
		k := float64(n) / 1000
		if k >= 100 {
			return fmt.Sprintf("%dK", int(k))
		}
		return fmt.Sprintf("%.0fK", k)
	}
	return fmt.Sprintf("%d", n)
}

// formatCLIPrice renders a per-1M USD rate with two decimals, or — for zero.
func formatCLIPrice(v float64) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("$%.2f", v)
}

// formatCLICaps joins capability codes with the canonical names.
func formatCLICaps(caps []string) string {
	if len(caps) == 0 {
		return "—"
	}
	labels := map[string]string{
		client.CapText:     "text",
		client.CapVision:   "vision",
		client.CapThinking: "thinking",
		client.CapTools:    "tools",
		client.CapCode:     "code",
		client.CapOCR:      "ocr",
	}
	out := make([]string, 0, len(caps))
	for _, c := range caps {
		if l, ok := labels[c]; ok {
			out = append(out, l)
		} else {
			out = append(out, c)
		}
	}
	return strings.Join(out, ",")
}
