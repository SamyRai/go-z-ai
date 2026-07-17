package cli

import (
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var rerankCmd = &cobra.Command{
	Use:   "rerank [query] [documents...]",
	Short: "Score candidate documents against a query for relevance",
	Long:  `Score candidate documents against a query for relevance (useful for reordering RAG search results).`,
	Args:  cobra.MinimumNArgs(2),
	RunE:  runWithClient(runRerank),
}

func init() {
	rootCmd.AddCommand(rerankCmd)

	rerankCmd.Flags().Int("top-n", 0, "Return only the top N results by score (0 = all)")
}

func runRerank(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	topN, _ := cmd.Flags().GetInt("top-n")

	resp, err := apiClient.Rerank().Create(cmd.Context(), client.RerankRequest{
		Query:           args[0],
		Documents:       args[1:],
		TopN:            topN,
		ReturnDocuments: true,
	})
	if err != nil {
		return fmt.Errorf("rerank failed: %w", err)
	}

	for i, r := range resp.Results {
		fmt.Printf("%d. [%.4f] (doc #%d) %s\n", i+1, r.RelevanceScore, r.Index, r.Document)
	}
	return nil
}
