package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var anthropicCmd = &cobra.Command{
	Use:   "anthropic",
	Short: "Call Z.AI's Anthropic-compatible Messages API (/api/anthropic)",
	Long: `Call Z.AI's Anthropic-compatible endpoint — the same /v1/messages surface
the GLM Coding Plan points Claude Code at — with a typed Go client instead of
the OpenAI-style /chat/completions surface the other commands use.`,
}

var anthropicMessagesCmd = &cobra.Command{
	Use:   "messages [prompt]",
	Short: "Create a message (POST /v1/messages)",
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runAnthropicMessages),
}

func init() {
	rootCmd.AddCommand(anthropicCmd)
	anthropicCmd.AddCommand(anthropicMessagesCmd)

	f := anthropicMessagesCmd.Flags()
	f.String("model", client.DefaultModel, "Model to use")
	f.Int("max-tokens", 1024, "Maximum tokens to generate (required by the Messages API)")
	f.String("system", "", "System prompt")
	f.Float64("temperature", 0, "Sampling temperature (server default when unset)")
	f.Int("thinking-budget", 0, "Enable extended thinking with this token budget (0 = off); reasoning is printed to stderr")
	f.Bool("stream", false, "Stream the response as it is generated (JSON: one event per line)")
	addFormatFlag("text", anthropicMessagesCmd)
}

func runAnthropicMessages(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	model, _ := cmd.Flags().GetString("model")
	maxTokens, _ := cmd.Flags().GetInt("max-tokens")
	system, _ := cmd.Flags().GetString("system")
	temperature, _ := cmd.Flags().GetFloat64("temperature")
	thinkingBudget, _ := cmd.Flags().GetInt("thinking-budget")
	stream, _ := cmd.Flags().GetBool("stream")

	req := client.AnthropicMessageRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  []client.AnthropicMessage{client.AnthropicTextMessage("user", args[0])},
	}
	if cmd.Flags().Changed("temperature") {
		req.Temperature = &temperature
	}
	if thinkingBudget > 0 {
		req.Thinking = &client.AnthropicThinking{Type: "enabled", BudgetTokens: thinkingBudget}
	}

	if stream {
		return runAnthropicStream(cmd, apiClient, req)
	}

	resp, err := apiClient.Anthropic().Create(cmd.Context(), req)
	if err != nil {
		return err
	}
	return emit(cmd, resp, func() error {
		printReasoning(resp.Thinking())
		fmt.Println(resp.Text())
		return nil
	})
}

// runAnthropicStream prints a streaming Messages response as it arrives: the
// answer to stdout and reasoning to stderr, or each raw event as a JSON line.
func runAnthropicStream(cmd *cobra.Command, apiClient *client.Client, req client.AnthropicMessageRequest) error {
	asJSON := isJSONFormat(cmd)
	enc := json.NewEncoder(os.Stdout)
	for ev, err := range apiClient.Anthropic().Stream(cmd.Context(), req) {
		if err != nil {
			return err
		}
		if asJSON {
			if err := enc.Encode(ev); err != nil {
				return err
			}
			continue
		}
		switch d, _ := ev.Delta(); d.Type {
		case "thinking_delta":
			fmt.Fprint(os.Stderr, d.Thinking)
		case "text_delta":
			fmt.Print(d.Text)
		}
	}
	if !asJSON {
		fmt.Println()
	}
	return nil
}
