package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"zai-api-client/pkg/client"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat completion operations",
	Long:  `Create and manage chat completions using Z.AI models.`,
}

var (
	chatModel       string
	chatTemperature float64
	chatMaxTokens   int
	chatSystemMsg   string
	chatStream      bool
	chatFormat      string

	// Advanced completion controls (structured output, thinking, tools).
	chatTopP         float64
	chatDoSample     bool
	chatStop         []string
	chatThinking     string
	chatEffort       string
	chatShowReason   bool
	chatSchemaFile   string
	chatSchemaName   string
	chatSchemaStrict bool
	chatToolFile     string
)

var chatCreateCmd = &cobra.Command{
	Use:   "create [message]",
	Short: "Create a chat completion",
	Long: `Create a chat completion with the given message and optional parameters.

Supports streaming (--stream), deep thinking (--thinking/--effort), structured
output (--json-schema), stop sequences (--stop), and function-calling tool
declarations (--tool). Tool calls in the response are printed but not executed
by the CLI; use the Go RunWithTools helper for an auto-executing loop.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runChatCreate,
}

var chatSimpleCmd = &cobra.Command{
	Use:   "simple [model] [message]",
	Short: "Create a simple chat completion",
	Long:  `Create a simple chat completion with basic parameters.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runChatSimple,
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.AddCommand(chatCreateCmd)
	chatCmd.AddCommand(chatSimpleCmd)

	chatCreateCmd.Flags().StringVar(&chatModel, "model", "glm-5.2", "Model to use")
	chatCreateCmd.Flags().Float64Var(&chatTemperature, "temperature", 0.7, "Sampling temperature (0.0-1.0)")
	chatCreateCmd.Flags().IntVar(&chatMaxTokens, "max-tokens", 4096, "Maximum tokens to generate")
	chatCreateCmd.Flags().StringVar(&chatSystemMsg, "system", "You are a helpful AI assistant.", "System message")
	chatCreateCmd.Flags().BoolVar(&chatStream, "stream", false, "Stream the response token-by-token")
	chatCreateCmd.Flags().StringVar(&chatFormat, "format", "text", "Output format (text, json)")

	chatCreateCmd.Flags().Float64Var(&chatTopP, "top-p", 0.95, "Nucleus sampling probability (0.01-1.0)")
	chatCreateCmd.Flags().BoolVar(&chatDoSample, "do-sample", false, "Enable the sampling strategy")
	chatCreateCmd.Flags().StringSliceVar(&chatStop, "stop", nil, "Stop sequences (repeatable, max 4)")
	chatCreateCmd.Flags().StringVar(&chatThinking, "thinking", "", "Deep thinking: enabled or disabled")
	chatCreateCmd.Flags().StringVar(&chatEffort, "effort", "", "Thinking effort: max, high, medium, low, minimal, none")
	chatCreateCmd.Flags().BoolVar(&chatShowReason, "show-reasoning", false, "Print reasoning_content (to stderr in text mode)")
	chatCreateCmd.Flags().StringVar(&chatSchemaFile, "json-schema", "", "Structured output schema: @file.json or inline JSON")
	chatCreateCmd.Flags().StringVar(&chatSchemaName, "schema-name", "output", "Name for the json_schema response format")
	chatCreateCmd.Flags().BoolVar(&chatSchemaStrict, "schema-strict", false, "Require strict schema adherence")
	chatCreateCmd.Flags().StringVar(&chatToolFile, "tool", "", "Function-calling tool definitions: @tools.json or inline JSON array")

	chatSimpleCmd.Flags().StringVar(&chatFormat, "format", "text", "Output format (text, json)")
}

func runChatCreate(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	userMessage := ""
	if len(args) > 0 {
		userMessage = args[0]
	}
	if userMessage == "" {
		return fmt.Errorf("please provide a message")
	}

	req, err := buildChatRequest(userMessage)
	if err != nil {
		return err
	}

	if chatStream {
		return runChatStream(apiClient, context.Background(), req)
	}

	resp, err := apiClient.Chat().Create(*req)
	if err != nil {
		return fmt.Errorf("failed to create chat completion: %w", err)
	}
	return outputChatResponse(resp, chatFormat)
}

// buildChatRequest assembles a ChatRequest from the chat create flags, loading
// any structured-output schema and tool definitions from file or inline JSON.
func buildChatRequest(userMessage string) (*client.ChatRequest, error) {
	req := &client.ChatRequest{
		Model:       chatModel,
		Temperature: chatTemperature,
		TopP:        chatTopP,
		MaxTokens:   chatMaxTokens,
		DoSample:    chatDoSample,
		Stop:        chatStop,
		Messages: []client.Message{
			{Role: "system", Content: chatSystemMsg},
			{Role: "user", Content: userMessage},
		},
	}

	if chatThinking != "" || chatEffort != "" {
		tc := &client.ThinkingConfig{}
		if chatThinking != "" {
			tc.Type = chatThinking
		}
		if chatEffort != "" {
			tc.Effort = chatEffort
		}
		req.Thinking = tc
	}

	if chatSchemaFile != "" {
		raw, err := loadJSONArg(chatSchemaFile)
		if err != nil {
			return nil, fmt.Errorf("read --json-schema: %w", err)
		}
		req.ResponseFormat = client.NewJSONSchemaFormat(chatSchemaName, raw, chatSchemaStrict)
	}

	if chatToolFile != "" {
		raw, err := loadJSONArg(chatToolFile)
		if err != nil {
			return nil, fmt.Errorf("read --tool: %w", err)
		}
		var tools []client.Tool
		if err := json.Unmarshal(raw, &tools); err != nil {
			return nil, fmt.Errorf("parse --tool JSON: %w", err)
		}
		req.Tools = tools
	}

	return req, nil
}

// loadJSONArg resolves a "@path" file reference or returns the literal bytes.
func loadJSONArg(arg string) ([]byte, error) {
	if strings.HasPrefix(arg, "@") {
		return os.ReadFile(arg[1:])
	}
	return []byte(arg), nil
}

// runChatStream drives a streaming completion, printing content deltas to stdout
// (or JSONL chunks in json mode) and reasoning to stderr when requested.
func runChatStream(apiClient *client.Client, ctx context.Context, req *client.ChatRequest) error {
	jsonEnc := json.NewEncoder(os.Stdout)

	err := apiClient.Chat().CreateStream(ctx, *req, func(ch client.StreamChunk) error {
		if chatFormat == "json" {
			return jsonEnc.Encode(ch)
		}
		if len(ch.Choices) == 0 {
			return nil
		}
		d := ch.Choices[0].Delta
		if d.Content != "" {
			fmt.Print(d.Content)
		}
		if chatShowReason && d.ReasoningContent != "" {
			fmt.Fprint(os.Stderr, d.ReasoningContent)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to stream chat completion: %w", err)
	}
	if chatFormat != "json" {
		fmt.Println()
	}
	return nil
}

func runChatSimple(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	model := args[0]
	message := args[1]

	messages := []client.Message{
		{Role: "user", Content: message},
	}

	response, err := apiClient.Chat().CreateSimple(model, message, messages)
	if err != nil {
		return fmt.Errorf("failed to create simple chat: %w", err)
	}

	return outputChatResponse(response, chatFormat)
}

func outputChatResponse(response *client.ChatResponse, format string) error {
	switch format {
	case "json":
		return outputJSON(response)
	default:
		if len(response.Choices) == 0 {
			return nil
		}
		msg := response.Choices[0].Message
		if chatShowReason && msg.ReasoningContent != "" {
			fmt.Fprintln(os.Stderr, "--- reasoning ---")
			fmt.Fprintln(os.Stderr, msg.ReasoningContent)
			fmt.Fprintln(os.Stderr, "------------------")
		}
		fmt.Println(msg.Content)
		if len(msg.ToolCalls) > 0 {
			fmt.Fprintln(os.Stderr, "--- tool calls ---")
			for _, tc := range msg.ToolCalls {
				if tc.Function != nil {
					fmt.Fprintf(os.Stderr, "%s(%s)\n", tc.Function.Name, tc.Function.Arguments)
				}
			}
		}
		return nil
	}
}
