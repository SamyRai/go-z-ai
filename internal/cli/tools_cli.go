package cli

import (
	"fmt"

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
	toolsCmd.AddCommand(toolsWebSearchCmd)
	toolsCmd.AddCommand(toolsWebReaderCmd)
	toolsCmd.AddCommand(toolsTokenizerCmd)

	toolsWebSearchCmd.Flags().String("engine", client.SearchEnginePro, "Search engine: search_std, search_pro, search_pro_sogou, search_pro_quark")
	toolsWebSearchCmd.Flags().Int("count", 10, "Number of results to return (1-50)")
	toolsWebReaderCmd.Flags().Bool("no-images", false, "Strip images from the parsed content")
	toolsTokenizerCmd.Flags().String("model", "glm-4.7", "Model to use for tokenization")
}

func runWebSearch(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	engine, _ := cmd.Flags().GetString("engine")
	count, _ := cmd.Flags().GetInt("count")

	fmt.Printf("🔍 Searching for: %s\n\n", args[0])

	result, err := apiClient.Tools().WebSearch(cmd.Context(), client.WebSearchRequest{
		SearchQuery:  args[0],
		SearchEngine: engine,
		Count:        count,
	})
	if err != nil {
		return fmt.Errorf("web search failed: %w", err)
	}

	if len(result.SearchResult) == 0 {
		fmt.Println("No results found")
		return nil
	}

	fmt.Printf("✅ Found %d results:\n\n", len(result.SearchResult))
	for i, item := range result.SearchResult {
		fmt.Printf("%d. %s\n", i+1, item.Title)
		fmt.Printf("   URL: %s\n", item.Link)
		if item.Content != "" {
			preview := item.Content
			if len(preview) > 150 {
				preview = preview[:150] + "..."
			}
			fmt.Printf("   Content: %s\n", preview)
		}
		fmt.Println()
	}

	return nil
}

func runWebReader(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	url := args[0]
	noImages, _ := cmd.Flags().GetBool("no-images")

	fmt.Printf("📖 Reading: %s\n\n", url)

	req := client.WebReaderRequest{URL: url, WithImagesSummary: true, WithLinksSummary: true}
	if noImages {
		retain := false
		req.RetainImages = &retain
	}

	result, err := apiClient.Tools().WebReader(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("web reader failed: %w", err)
	}

	if result.ReaderResult == nil {
		fmt.Println("No data returned")
		return nil
	}

	fmt.Println("✅ Page Parsed Successfully:")
	fmt.Printf("Title: %s\n", result.ReaderResult.Title)
	fmt.Printf("URL: %s\n", result.ReaderResult.URL)

	if result.ReaderResult.Description != "" {
		fmt.Printf("\n📝 Description:\n%s\n", result.ReaderResult.Description)
	}

	if result.ReaderResult.Content != "" {
		preview := result.ReaderResult.Content
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		fmt.Printf("\n📄 Content Preview:\n%s\n", preview)
	}

	return nil
}

func runTokenizer(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	text := args[0]
	model, _ := cmd.Flags().GetString("model")

	fmt.Printf("🔢 Tokenizing text with %s...\n", model)
	fmt.Printf("Text length: %d characters\n\n", len(text))

	result, err := apiClient.Tools().Tokenize(cmd.Context(), client.TokenizerRequest{
		Model:    model,
		Messages: []client.Message{{Role: "user", Content: text}},
	})
	if err != nil {
		return fmt.Errorf("tokenizer failed: %w", err)
	}

	if result.Usage != nil {
		fmt.Printf("✅ Token Count: %d tokens (prompt: %d, image: %d, video: %d)\n",
			result.Usage.TotalTokens, result.Usage.PromptTokens, result.Usage.ImageTokens, result.Usage.VideoTokens)
	}

	return nil
}
