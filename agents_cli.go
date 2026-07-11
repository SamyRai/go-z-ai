package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"zai-api-client/pkg/client"
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
	RunE:  runAgentsInvoke,
}

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsInvokeCmd)

	agentsInvokeCmd.Flags().String("source-lang", "", "Source language (translation agents, e.g. 'auto')")
	agentsInvokeCmd.Flags().String("target-lang", "", "Target language (translation agents, e.g. 'zh-CN')")
}

func runAgentsInvoke(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	agentID, message := args[0], args[1]

	req := client.AgentInvokeRequest{
		AgentID:  agentID,
		Messages: []client.AgentMessage{client.NewAgentTextMessage("user", message)},
	}
	sourceLang, _ := cmd.Flags().GetString("source-lang")
	targetLang, _ := cmd.Flags().GetString("target-lang")
	if sourceLang != "" || targetLang != "" {
		req.CustomVariables = map[string]any{}
		if sourceLang != "" {
			req.CustomVariables["source_lang"] = sourceLang
		}
		if targetLang != "" {
			req.CustomVariables["target_lang"] = targetLang
		}
	}

	resp, err := apiClient.Agents().Invoke(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to invoke agent: %w", err)
	}

	if resp.Failed() {
		msg := "unknown error"
		if resp.Error != nil {
			msg = resp.Error.Message
		}
		return fmt.Errorf("agent invocation failed: %s", msg)
	}

	for _, choice := range resp.Choices {
		fmt.Println(choice.Messages.Content.Text)
	}
	return nil
}
