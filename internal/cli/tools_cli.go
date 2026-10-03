package cli

import (
	"fmt"
	"strings"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Tool capabilities",
	Long:  `Z.AI tool capabilities including web search, web reader, and tokenizer.`,
}

var toolsWebSearchCmd = &cobra.Command{
	Use:   "web-search [query]",
	Short: "Search the web",
	Long:  `Use Z.AI's specialized web search for LLM-optimized results.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runWebSearch),
}

var toolsWebReaderCmd = &cobra.Command{
	Use:   "web-reader [url]",
	Short: "Read web page content",
	Long:  `Parse and extract content from a URL with structured output.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runWebReader),
}

var toolsTokenizerCmd = &cobra.Command{
	Use:   "tokenizer [text]",
	Short: "Count tokens",
	Long:  `Count tokens for a single-message chat request using Z.AI's tokenizer.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runTokenizer),
}

func init() {
	rootCmd.AddCommand(toolsCmd)
	toolsCmd.AddCommand(toolsWebSearchCmd, toolsWebReaderCmd, toolsTokenizerCmd)

	f := toolsWebSearchCmd.Flags()
	f.String("engine", client.SearchEnginePrime, "Search engine: "+strings.Join(client.SearchEngines, ", "))
	f.Int("count", 10, "Number of results to return (1-50)")
	f.String("recency", "", "Recency filter: oneDay, oneWeek, oneMonth, oneYear, noLimit")
	f.String("domain", "", "Restrict results to this domain")
	f.String("content-size", "", "Result content size: medium or high")
	toolsWebReaderCmd.Flags().Bool("no-images", false, "Strip images from the parsed content")
	toolsTokenizerCmd.Flags().String("model", client.DefaultModel, "Model to use for tokenization")
	addFormatFlag("text", toolsWebSearchCmd, toolsWebReaderCmd, toolsTokenizerCmd)
}

func runWebSearch(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	f := cmd.Flags()
	engine, _ := f.GetString("engine")
	count, _ := f.GetInt("count")
	recency, _ := f.GetString("recency")
	domain, _ := f.GetString("domain")
	contentSize, _ := f.GetString("content-size")

	result, err := apiClient.Tools().WebSearch(cmd.Context(), client.WebSearchRequest{
		SearchQuery:         args[0],
		SearchEngine:        engine,
		Count:               count,
		SearchRecencyFilter: recency,
		SearchDomainFilter:  domain,
		ContentSize:         contentSize,
	})
	if err != nil {
		return fmt.Errorf("web search failed: %w", err)
	}
	return emit(cmd, result, func() error {
		if len(result.SearchResult) == 0 {
			fmt.Println("No results found")
			return nil
		}
		for i, item := range result.SearchResult {
			fmt.Printf("%d. %s\n   %s\n", i+1, item.Title, item.Link)
			if item.Content != "" {
				fmt.Printf("   %s\n", preview(item.Content, 150))
			}
			fmt.Println()
		}
		return nil
	})
}

func runWebReader(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	noImages, _ := cmd.Flags().GetBool("no-images")
	req := client.WebReaderRequest{URL: args[0], WithImagesSummary: true, WithLinksSummary: true}
	if noImages {
		retain := false
		req.RetainImages = &retain
	}

	result, err := apiClient.Tools().WebReader(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("web reader failed: %w", err)
	}
	return emit(cmd, result, func() error {
		r := result.ReaderResult
		if r == nil {
			fmt.Println("No data returned")
			return nil
		}
		fmt.Printf("Title: %s\nURL:   %s\n", r.Title, r.URL)
		if r.Description != "" {
			fmt.Printf("\n%s\n", r.Description)
		}
		if r.Content != "" {
			fmt.Printf("\n%s\n", r.Content)
		}
		return nil
	})
}

func runTokenizer(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	model, _ := cmd.Flags().GetString("model")
	result, err := apiClient.Tools().Tokenize(cmd.Context(), client.TokenizerRequest{
		Model:    model,
		Messages: []client.Message{{Role: "user", Content: args[0]}},
	})
	if err != nil {
		return fmt.Errorf("tokenizer failed: %w", err)
	}
	return emit(cmd, result, func() error {
		if u := result.Usage; u != nil {
			fmt.Printf("%s: %d tokens (prompt: %d, image: %d, video: %d)\n",
				model, u.TotalTokens, u.PromptTokens, u.ImageTokens, u.VideoTokens)
		}
		return nil
	})
}

// preview shortens s to at most n runes, marking the cut with an ellipsis.
func preview(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
