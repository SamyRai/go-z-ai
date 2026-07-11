package main

import (
	"fmt"
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
	RunE:  runWebSearch,
}

var toolsWebReaderCmd = &cobra.Command{
	Use:   "web-reader [url]",
	Short: "Read web page content",
	Long:  `Parse and extract content from a URL with structured output.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWebReader,
}

var toolsTokenizerCmd = &cobra.Command{
	Use:   "tokenizer [text]",
	Short: "Count tokens",
	Long:  `Count tokens for text using Z.AI's tokenizer.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTokenizer,
}

func init() {
	rootCmd.AddCommand(toolsCmd)
	toolsCmd.AddCommand(toolsWebSearchCmd)
	toolsCmd.AddCommand(toolsWebReaderCmd)
	toolsCmd.AddCommand(toolsTokenizerCmd)
	
	toolsWebSearchCmd.Flags().Int("top-k", 5, "Number of results to return")
	toolsWebReaderCmd.Flags().Bool("images", false, "Include images in output")
	toolsWebReaderCmd.Flags().Bool("summary", true, "Include summary in output")
	toolsTokenizerCmd.Flags().String("model", "glm-4.7", "Model to use for tokenization")
}

func runWebSearch(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	query := args[0]
	topK, _ := cmd.Flags().GetInt("top-k")

	fmt.Printf("🔍 Searching for: %s\n", query)
	fmt.Printf("Top %d results requested\n\n", topK)

	result, err := apiClient.Tools().WebSearch(query, topK)
	if err != nil {
		return fmt.Errorf("web search failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("search API error: %s (code: %d)", result.Msg, result.Code)
	}

	if len(result.Data) == 0 {
		fmt.Println("No results found")
		return nil
	}

	fmt.Printf("✅ Found %d results:\n\n", len(result.Data))
	for i, item := range result.Data {
		fmt.Printf("%d. %s\n", i+1, item.Title)
		fmt.Printf("   URL: %s\n", item.URL)
		if item.Content != "" {
			preview := item.Content
			if len(preview) > 150 {
				preview = preview[:150] + "..."
			}
			fmt.Printf("   Content: %s\n", preview)
		}
		if item.Score > 0 {
			fmt.Printf("   Score: %.2f\n", item.Score)
		}
		fmt.Println()
	}

	return nil
}

func runWebReader(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	url := args[0]
	withImages, _ := cmd.Flags().GetBool("images")
	withSummary, _ := cmd.Flags().GetBool("summary")

	fmt.Printf("📖 Reading: %s\n", url)
	fmt.Printf("Images: %t, Summary: %t\n\n", withImages, withSummary)

	result, err := apiClient.Tools().WebReader(url, withImages, withSummary)
	if err != nil {
		return fmt.Errorf("web reader failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("reader API error: %s (code: %d)", result.Msg, result.Code)
	}

	if result.Data == nil {
		fmt.Println("No data returned")
		return nil
	}

	fmt.Println("✅ Page Parsed Successfully:")
	fmt.Printf("Title: %s\n", result.Data.Title)
	fmt.Printf("URL: %s\n", result.Data.URL)
	
	if result.Data.Summary != "" {
		fmt.Printf("\n📝 Summary:\n%s\n", result.Data.Summary)
	}
	
	if result.Data.Content != "" {
		preview := result.Data.Content
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		fmt.Printf("\n📄 Content Preview:\n%s\n", preview)
	}
	
	if len(result.Data.Images) > 0 {
		fmt.Printf("\n🖼️  Images: %d found\n", len(result.Data.Images))
		for i, img := range result.Data.Images {
			fmt.Printf("   %d. %s\n", i+1, img)
		}
	}
	
	if len(result.Data.Links) > 0 {
		fmt.Printf("\n🔗 Links: %d found\n", len(result.Data.Links))
		for i, link := range result.Data.Links {
			if i < 5 { // Show first 5 links
				fmt.Printf("   %d. %s\n", i+1, link)
			}
		}
	}

	return nil
}

func runTokenizer(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	text := args[0]
	model, _ := cmd.Flags().GetString("model")

	fmt.Printf("🔢 Tokenizing text with %s...\n", model)
	fmt.Printf("Text length: %d characters\n\n", len(text))

	result, err := apiClient.Tools().Tokenize(text, model)
	if err != nil {
		return fmt.Errorf("tokenizer failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("tokenizer API error: %s (code: %d)", result.Msg, result.Code)
	}

	if result.Data != nil {
		fmt.Printf("✅ Token Count: %d tokens\n", result.Data.TokenCount)
		
		if len(result.Data.Tokens) > 0 {
			fmt.Printf("\n📝 Token Breakdown (first 20):\n")
			for i, token := range result.Data.Tokens {
				if i < 20 {
					fmt.Printf("   %d. %s\n", i+1, token)
				}
			}
			if len(result.Data.Tokens) > 20 {
				fmt.Printf("   ... and %d more tokens\n", len(result.Data.Tokens)-20)
			}
		}
		
		fmt.Printf("\n💡 Estimated cost for %s:\n", model)
		// Rough cost estimation (will vary by actual pricing)
		estimatedCost := float64(result.Data.TokenCount) / 1000000.0 * 0.5 // rough estimate
		fmt.Printf("   ~$%.6f per 1M tokens (estimate)\n", estimatedCost)
	}

	return nil
}
