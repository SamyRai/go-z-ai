package cli

import (
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Invoke specialized Z.AI agents",
	Long: `Invoke Z.AI's specialized agents (e.g. general_translation, GLM Slide/Poster,
Video Effect Template), each identified by an agent_id.

Note: the Agents API returns HTTP 200 even when the invocation fails at the
business level (e.g. insufficient account balance) — this command reports
that failure from the response body, not from a transport error.`,
}

var agentsInvokeCmd = &cobra.Command{
	Use:   "invoke [agent-id] [message]",
	Short: "Invoke an agent with a text message",
	Args:  cobra.ExactArgs(2),
	RunE:  runWithClient(runAgentsInvoke),
}

var agentsAsyncResultCmd = &cobra.Command{
	Use:   "async-result [agent-id] [async-id]",
	Short: "Poll the result of an async agent task",
	Long:  `Poll the result of a long-running async agent task (e.g. intelligent_education_correction_polling).`,
	Args:  cobra.ExactArgs(2),
	RunE:  runWithClient(runAgentsAsyncResult),
}

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsInvokeCmd, agentsAsyncResultCmd)

	for _, c := range []*cobra.Command{agentsInvokeCmd, agentsAsyncResultCmd} {
		c.Flags().StringToString("var", nil, "Agent custom variable key=value (repeatable)")
	}
	agentsInvokeCmd.Flags().String("source-lang", "", "Source language (translation agents, e.g. 'auto')")
	agentsInvokeCmd.Flags().String("target-lang", "", "Target language (translation agents, e.g. 'zh-CN')")
	agentsAsyncResultCmd.Flags().String("conversation-id", "", "Conversation ID returned by the invocation")
	addFormatFlag("text", agentsInvokeCmd, agentsAsyncResultCmd)
}

// translationVars maps the translation convenience flags to the custom
// variables they set.
var translationVars = map[string]string{"source-lang": "source_lang", "target-lang": "target_lang"}

// customVariables merges --var with the non-empty convenience flags in
// flagVars; nil when none are set.
func customVariables(cmd *cobra.Command, flagVars map[string]string) map[string]any {
	vars, _ := cmd.Flags().GetStringToString("var")
	out := make(map[string]any, len(vars))
	for k, v := range vars {
		out[k] = v
	}
	for flag, name := range flagVars {
		if v, _ := cmd.Flags().GetString(flag); v != "" {
			out[name] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func runAgentsInvoke(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	resp, err := apiClient.Agents().Invoke(cmd.Context(), client.AgentInvokeRequest{
		AgentID:         args[0],
		Messages:        []client.AgentMessage{client.NewAgentTextMessage("user", args[1])},
		CustomVariables: customVariables(cmd, translationVars),
	})
	if err != nil {
		return fmt.Errorf("failed to invoke agent: %w", err)
	}
	if resp.Failed() {
		return fmt.Errorf("agent invocation failed: %w", resp.Error)
	}
	return emit(cmd, resp, func() error {
		for _, choice := range resp.Choices {
			fmt.Println(choice.Messages.Content.Text)
		}
		if resp.ConversationID != "" {
			progressf("conversation: %s\n", resp.ConversationID)
		}
		return nil
	})
}

func runAgentsAsyncResult(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	conversationID, _ := cmd.Flags().GetString("conversation-id")
	resp, err := apiClient.Agents().AsyncResult(cmd.Context(), client.AgentAsyncResultRequest{
		AgentID:         args[0],
		AsyncID:         args[1],
		ConversationID:  conversationID,
		CustomVariables: customVariables(cmd, nil),
	})
	if err != nil {
		return fmt.Errorf("failed to get async result: %w", err)
	}
	if resp.Failed() {
		return fmt.Errorf("agent task failed: %w", resp.Error)
	}
	return emit(cmd, resp, func() error {
		if !resp.Done() {
			progressf("status: %s (try again shortly)\n", resp.Status)
			return nil
		}
		for _, choice := range resp.Choices {
			for _, msg := range choice.Messages {
				for _, part := range msg.Content {
					if part.FileURL != "" {
						fmt.Printf("%s: %s\n", part.TagEN, part.FileURL)
					}
				}
			}
		}
		if resp.Usage != nil {
			progressf("total tokens: %d\n", resp.Usage.TotalTokens)
		}
		return nil
	})
}
