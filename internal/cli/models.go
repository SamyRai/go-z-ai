package cli

import (
	"fmt"
	"os"
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

func outputModelsTable(models []client.ModelDetails, showPrice bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "MODEL ID\tNAME\tCONTEXT")

	for _, model := range models {
		fmt.Fprintf(w, "%s\t%s\t%d\n", model.ID, model.Name, model.ContextSize)
	}

	if showPrice {
		fmt.Fprintln(w, "\nPRICING (per 1M tokens)")
		fmt.Fprintln(w, "MODEL\tINPUT\tOUTPUT\tCACHED")
		for _, model := range models {
			if model.Pricing != nil {
				fmt.Fprintf(w, "%s\t$%.2f\t$%.2f\t$%.2f\n",
					model.ID,
					model.Pricing.Input,
					model.Pricing.Output,
					model.Pricing.Cached)
			}
		}
	}

	return w.Flush()
}

func outputModelTable(model *client.ModelDetails) error {
	fmt.Printf("Model ID: %s\n", model.ID)
	fmt.Printf("Name: %s\n", model.Name)
	fmt.Printf("Description: %s\n", model.Description)
	fmt.Printf("Context Size: %d tokens\n", model.ContextSize)
	fmt.Printf("Owned By: %s\n", model.OwnedBy)

	if model.Pricing != nil {
		fmt.Printf("\nPricing (per 1M tokens):\n")
		fmt.Printf("  Input: $%.2f\n", model.Pricing.Input)
		fmt.Printf("  Output: $%.2f\n", model.Pricing.Output)
		if model.Pricing.Cached > 0 {
			fmt.Printf("  Cached: $%.2f\n", model.Pricing.Cached)
		}
		if model.Pricing.CacheStore > 0 {
			fmt.Printf("  Cache Storage: $%.2f\n", model.Pricing.CacheStore)
		}
	}

	return nil
}
