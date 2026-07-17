package cli

import (
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var embeddingsCmd = &cobra.Command{
	Use:   "embeddings",
	Short: "Generate text embeddings (routes to open.bigmodel.cn)",
	Long: `Generate vector embeddings via open.bigmodel.cn — the only platform
that documents this endpoint (api.z.ai's docs don't mention it at all).

Authenticates with --china-api-key / ZAI_CHINA_API_KEY, falling back to
--api-key if unset — a regular z.ai key authenticates fine here too
(confirmed live: same /models catalog, same billing errors on both
platforms). Whether you get real results depends on your account's plan
entitlement, not which key you use: a GLM Coding Plan account, for
example, returns "Unknown Model" here since embeddings aren't in that
plan's catalog.`,
}

var embeddingsCreateCmd = &cobra.Command{
	Use:   "create [text]",
	Short: "Generate an embedding for a text input",
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runEmbeddingsCreate),
}

func init() {
	rootCmd.AddCommand(embeddingsCmd)
	embeddingsCmd.AddCommand(embeddingsCreateCmd)

	embeddingsCreateCmd.Flags().String("model", client.EmbeddingModel3, "Embedding model: embedding-3 or embedding-2")
	embeddingsCreateCmd.Flags().Int("dimensions", 0, "Output vector dimensions (embedding-3 only: 256, 512, 1024, or 2048)")
	// Default json: the vector payload is machine-oriented, so JSON stays the
	// out-of-the-box output (text mode prints a summary).
	addFormatFlag("json", embeddingsCreateCmd)
}

func runEmbeddingsCreate(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	model, _ := cmd.Flags().GetString("model")
	dimensions, _ := cmd.Flags().GetInt("dimensions")

	resp, err := apiClient.Embeddings().Create(cmd.Context(), client.EmbeddingsRequest{
		Model:      model,
		Input:      args[0],
		Dimensions: dimensions,
	})
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	return emit(cmd, resp, func() error {
		dims := 0
		if len(resp.Data) > 0 {
			dims = len(resp.Data[0].Embedding)
		}
		fmt.Printf("model: %s\n", resp.Model)
		fmt.Printf("embeddings: %d (dimensions: %d)\n", len(resp.Data), dims)
		fmt.Printf("tokens: %d\n", resp.Usage.TotalTokens)
		return nil
	})
}
