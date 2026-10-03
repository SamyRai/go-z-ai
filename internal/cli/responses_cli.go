package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var responsesCmd = &cobra.Command{
	Use:   "responses",
	Short: "Call Z.AI's OpenAI Responses-protocol API (/api/v1, what Codex uses)",
	Long: `Call Z.AI's OpenAI Responses-protocol endpoint — the surface Codex speaks,
documented for the GLM Coding Plan. --region selects the gateway
(api.z.ai/api/v1 or open.bigmodel.cn/api/v1).`,
}

var responsesCreateCmd = &cobra.Command{
	Use:   "create <prompt>",
	Short: "Create a response (POST /responses)",
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runResponsesCreate),
}

func init() {
	rootCmd.AddCommand(responsesCmd)
	responsesCmd.AddCommand(responsesCreateCmd)

	f := responsesCreateCmd.Flags()
	f.String("model", client.DefaultModel, "Model to use")
	f.String("instructions", "", "Instructions (system prompt)")
	f.String("effort", "", "Reasoning effort, validated against the model (GLM-5.3: low, high, max)")
	f.Int("max-output-tokens", 0, "Maximum output tokens; 0 uses the model default")
	f.Bool("stream", false, "Stream the response as it is generated (JSON: one event per line)")
	f.Bool("show-reasoning", false, "Print the reasoning to stderr")
	addFormatFlag("text", responsesCreateCmd)
}

func runResponsesCreate(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	f := cmd.Flags()
	model, _ := f.GetString("model")
	instructions, _ := f.GetString("instructions")
	effort, _ := f.GetString("effort")
	maxOutput, _ := f.GetInt("max-output-tokens")
	stream, _ := f.GetBool("stream")
	showReasoning, _ := f.GetBool("show-reasoning")

	req := client.ResponsesRequest{
		Model:           model,
		Instructions:    instructions,
		Input:           []client.ResponsesItem{client.ResponsesMessage("user", args[0])},
		MaxOutputTokens: maxOutput,
	}
	if effort != "" {
		req.Reasoning = &client.ResponsesReasoning{Effort: effort}
	}
	if stream {
		return streamResponses(cmd, apiClient, req, showReasoning)
	}

	resp, err := apiClient.Responses().Create(cmd.Context(), req)
	if err != nil {
		return err
	}
	return emit(cmd, resp, func() error {
		if showReasoning {
			printReasoning(resp.ReasoningText())
		}
		fmt.Println(resp.OutputText())
		for _, call := range resp.FunctionCalls() {
			fmt.Fprintf(os.Stderr, "tool call: %s(%s)\n", call.Name, call.Arguments)
		}
		return nil
	})
}

// streamResponses prints text deltas to stdout (reasoning deltas to stderr
// when asked), or each event as a JSON line.
func streamResponses(cmd *cobra.Command, apiClient *client.Client, req client.ResponsesRequest, showReasoning bool) error {
	asJSON := isJSONFormat(cmd)
	enc := json.NewEncoder(os.Stdout)
	for ev, err := range apiClient.Responses().Stream(cmd.Context(), req) {
		if err != nil {
			return err
		}
		switch {
		case asJSON:
			if err := enc.Encode(ev); err != nil {
				return err
			}
		case ev.Type == client.ResponsesEventOutputTextDelta:
			fmt.Print(ev.Delta)
		case showReasoning && (ev.Type == client.ResponsesEventReasoningSummaryDelta || ev.Type == client.ResponsesEventReasoningTextDelta):
			fmt.Fprint(os.Stderr, ev.Delta)
		}
	}
	if !asJSON {
		fmt.Println()
	}
	return nil
}
