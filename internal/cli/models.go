package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/SamyRai/go-z-ai/internal/modelview"
	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Model operations",
	Long: `List and inspect Z.AI models. The /models endpoint returns bare IDs; context
sizes, prices, capabilities, and reasoning efforts come from the curated
catalog in pkg/client.`,
}

var modelsGetCmd = &cobra.Command{
	Use:   "get [model-id]",
	Short: "Get details for a specific model",
	Long:  `Get detailed information for a specific model including pricing, capabilities, and reasoning efforts.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runModelsGet),
}

// modelLister lists one selection of models.
type modelLister func(*client.ModelsService, context.Context) ([]client.ModelDetails, error)

// newModelsListCmd builds a listing subcommand; list, text, vision, and free
// differ only in which models they select.
func newModelsListCmd(use, short string, list modelLister) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: runWithClient(func(cmd *cobra.Command, _ []string, apiClient *client.Client) error {
			models, err := list(apiClient.Models(), cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to list models: %w", err)
			}
			return emit(cmd, models, func() error { return printModelsTable(models) })
		}),
	}
}

func init() {
	rootCmd.AddCommand(modelsCmd)
	listCmds := []*cobra.Command{
		newModelsListCmd("list", "List all available models", func(s *client.ModelsService, ctx context.Context) ([]client.ModelDetails, error) {
			info, err := s.List(ctx)
			if err != nil {
				return nil, err
			}
			return info.Models, nil
		}),
		newModelsListCmd("text", "List text-capable (chat) models", (*client.ModelsService).GetTextModels),
		newModelsListCmd("vision", "List models that accept image input", (*client.ModelsService).GetVisionModels),
		newModelsListCmd("free", "List free models", (*client.ModelsService).GetFreeModels),
	}
	modelsCmd.AddCommand(listCmds...)
	modelsCmd.AddCommand(modelsGetCmd)
	addFormatFlag("table", append(listCmds, modelsGetCmd)...)
}

func runModelsGet(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	model, err := apiClient.Models().Get(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}
	return emit(cmd, model, func() error {
		printModelDetails(model)
		return nil
	})
}

// printModelsTable renders the model list. Every column except the ID comes
// from the catalog enrichment.
func printModelsTable(models []client.ModelDetails) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "MODEL\tFAMILY\tCONTEXT\tMAXOUT\tIN/1M\tOUT/1M\tCACHED\tCAPS")
	for _, m := range models {
		in, out, cached := modelview.Rates(m.Pricing)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			m.ID, orUnknown(m.Family), modelview.FormatTokens(m.ContextSize), modelview.FormatTokens(m.MaxOutput),
			in, out, cached, modelview.Capabilities(m.Capabilities))
	}
	fmt.Fprintf(w, "\n(prices are USD per 1M tokens; %s = unknown)\n", modelview.Unknown)
	return w.Flush()
}

func printModelDetails(m *client.ModelDetails) {
	row := func(label, value string) {
		if value != "" && value != modelview.Unknown {
			fmt.Printf("%-14s%s\n", label+":", value)
		}
	}
	row("Model ID", m.ID)
	row("Name", m.Name)
	row("Family", m.Family)
	row("Tier", m.Tier)
	row("Description", m.Description)
	row("Context", modelview.FormatTokens(m.ContextSize))
	row("Max output", modelview.FormatTokens(m.MaxOutput))
	row("Owned by", m.OwnedBy)
	row("Released", modelview.ReleaseDate(*m))
	row("Capabilities", modelview.Capabilities(m.Capabilities))
	row("Efforts", strings.Join(m.ReasoningEfforts, ", "))

	if m.Pricing == nil {
		return
	}
	if m.IsFree() {
		fmt.Println("\nPricing:      free")
		return
	}
	in, out, cached := modelview.Rates(m.Pricing)
	fmt.Println("\nPricing (USD per 1M tokens):")
	row("  Input", in)
	row("  Output", out)
	row("  Cached", cached)
}

// orUnknown renders an empty catalog field as modelview.Unknown.
func orUnknown(s string) string {
	if s == "" {
		return modelview.Unknown
	}
	return s
}
