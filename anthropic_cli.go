package main

import (
	"context"
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
	RunE:  runAnthropicMessages,
}

func init() {
	rootCmd.AddCommand(anthropicCmd)
	anthropicCmd.AddCommand(anthropicMessagesCmd)

	f := anthropicMessagesCmd.Flags()
	f.String("model", "glm-4.6", "Model to use")
	f.Int("max-tokens", 1024, "Maximum tokens to generate (required by the Messages API)")
	f.String("system", "", "System prompt")
	f.Float64("temperature", -1, "Sampling temperature (omitted when negative)")
	f.Int("thinking-budget", 0, "Enable extended thinking with this token budget (0 = off); reasoning is printed to stderr")
	f.Bool("stream", false, "Stream the response as it is generated")
}

func runAnthropicMessages(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

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
	if temperature >= 0 {
		req.Temperature = &temperature
	}
	if thinkingBudget > 0 {
		req.Thinking = &client.AnthropicThinking{Type: "enabled", BudgetTokens: thinkingBudget}
	}

	if stream {
		return runAnthropicStream(cmd.Context(), apiClient, req)
	}

	resp, err := apiClient.Anthropic().Create(cmd.Context(), req)
	if err != nil {
		return err // already descriptive ("failed to create anthropic message: …")
	}
	if reasoning := resp.Thinking(); reasoning != "" {
		fmt.Fprintln(os.Stderr, "--- reasoning ---")
		fmt.Fprintln(os.Stderr, reasoning)
		fmt.Fprintln(os.Stderr, "------------------")
	}
	fmt.Println(resp.Text())
	return nil
}

// runAnthropicStream prints text deltas from a streaming Messages response to
// stdout as they arrive.
func runAnthropicStream(ctx context.Context, apiClient *client.Client, req client.AnthropicMessageRequest) error {
	err := apiClient.Anthropic().CreateStream(ctx, req, func(ev client.AnthropicStreamEvent) error {
		if ev.Type != "content_block_delta" {
			return nil
		}
		var d struct {
			Delta struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				Thinking string `json:"thinking"`
			} `json:"delta"`
		}
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return nil // non-text delta (e.g. tool input JSON); ignore for plain output
		}
		switch d.Delta.Type {
		case "thinking_delta":
			fmt.Fprint(os.Stderr, d.Delta.Thinking) // reasoning to stderr, answer to stdout
		case "text_delta":
			fmt.Print(d.Delta.Text)
		}
		return nil
	})
	if err != nil {
		return err // the APIError / stream error is already self-describing
	}
	fmt.Fprintln(os.Stdout)
	return nil
}
