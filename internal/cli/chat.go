package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/SamyRai/go-z-ai/internal/fileinput"
	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat completion operations",
	Long:  `Create chat completions with Z.AI models.`,
}

var chatCreateCmd = &cobra.Command{
	Use:   "create <message>",
	Short: "Create a chat completion",
	Long: `Create a chat completion.

Supports streaming (--stream), reasoning control (--thinking, --effort),
JSON output (--json, or --json-schema to also describe the expected shape),
function-calling tool declarations (--tool, streamed with --tool-stream),
and image/video/file attachments for multimodal models. Tool calls in the
response are printed, not executed; use the Go RunWithTools helper for an
executing loop. Sampling settings default to the model's own.`,
	Args: cobra.ExactArgs(1),
	RunE: runWithClient(runChatCreate),
}

var chatAsyncResultCmd = &cobra.Command{
	Use:   "async-result <task-id>",
	Short: "Get the result of an async chat completion",
	Long:  `Get the result of a task submitted with "chat create --async".`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWithClient(runAsyncStatus),
}

// chatOptions holds the "chat create" flags; request turns them into a
// ChatRequest without touching package state.
type chatOptions struct {
	model, system         string
	temperature, topP     float64
	maxTokens             int
	doSample              *bool // nil unless --do-sample was given
	stop                  []string
	stream, async         bool
	toolStream            bool
	thinking, effort      string
	showReasoning         bool
	jsonObject            bool
	schema, tools         string // @file or inline JSON
	images, videos, files []string
}

var chatOpts chatOptions

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.AddCommand(chatCreateCmd, chatAsyncResultCmd)
	addFormatFlag("text", chatCreateCmd, chatAsyncResultCmd)

	f := chatCreateCmd.Flags()
	f.StringVar(&chatOpts.model, "model", client.DefaultModel, "Model to use")
	f.StringVar(&chatOpts.system, "system", "", "System message")
	f.Float64Var(&chatOpts.temperature, "temperature", 0, "Sampling temperature in (0, 1]; 0 uses the model default")
	f.Float64Var(&chatOpts.topP, "top-p", 0, "Nucleus sampling in (0, 1]; 0 uses the model default")
	f.IntVar(&chatOpts.maxTokens, "max-tokens", 0, "Maximum tokens to generate; 0 uses the model default")
	f.Bool("do-sample", true, "Sample (false = greedy decoding)")
	f.StringSliceVar(&chatOpts.stop, "stop", nil, "Stop sequence (the API honors one)")
	f.BoolVar(&chatOpts.stream, "stream", false, "Stream the response token by token")
	f.BoolVar(&chatOpts.async, "async", false, "Submit without waiting; poll with 'chat async-result'")
	f.BoolVar(&chatOpts.toolStream, "tool-stream", false, "Stream tool-call arguments incrementally (GLM-4.6+)")
	f.StringVar(&chatOpts.thinking, "thinking", "", "Reasoning: enabled or disabled (GLM-5.3 models always reason)")
	f.StringVar(&chatOpts.effort, "effort", "", "Reasoning effort: "+strings.Join(client.AllEfforts, ", ")+" (GLM-5.3: low, high, max)")
	f.BoolVar(&chatOpts.showReasoning, "show-reasoning", false, "Print the reasoning (to stderr in text mode)")
	f.BoolVar(&chatOpts.jsonObject, "json", false, "Ask for a JSON object response")
	f.StringVar(&chatOpts.schema, "json-schema", "", "JSON object response matching this schema (@file.json or inline JSON); the schema is added to the system message")
	f.StringVar(&chatOpts.tools, "tool", "", "Tool definitions: @tools.json or an inline JSON array")
	f.StringArrayVar(&chatOpts.images, "image", nil, "Attach an image (repeatable): a URL or @path")
	f.StringArrayVar(&chatOpts.videos, "video", nil, "Attach a video (repeatable): a URL or @path")
	f.StringArrayVar(&chatOpts.files, "file", nil, "Attach a document (repeatable): a URL or @path")
}

func runChatCreate(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	opts := chatOpts
	if cmd.Flags().Changed("do-sample") {
		v, _ := cmd.Flags().GetBool("do-sample")
		opts.doSample = &v
	}
	req, err := opts.request(args[0])
	if err != nil {
		return err
	}

	switch {
	case opts.stream:
		return streamChat(cmd, apiClient, req, opts.showReasoning)
	case opts.async:
		task, err := apiClient.Chat().CreateAsync(cmd.Context(), req)
		if err != nil {
			return err
		}
		return emit(cmd, task, func() error {
			fmt.Printf("Task submitted: %s (poll with 'go-z-ai chat async-result %s')\n", task.ID, task.ID)
			return nil
		})
	}
	resp, err := apiClient.Chat().Create(cmd.Context(), req)
	if err != nil {
		return err
	}
	return emit(cmd, resp, func() error {
		if len(resp.Choices) > 0 {
			printReply(resp.Choices[0].Message, opts.showReasoning)
		}
		return nil
	})
}

// request builds the ChatRequest for message.
func (o chatOptions) request(message string) (client.ChatRequest, error) {
	req := client.ChatRequest{
		Model:           o.model,
		Temperature:     o.temperature,
		TopP:            o.topP,
		MaxTokens:       o.maxTokens,
		DoSample:        o.doSample,
		Stop:            o.stop,
		ToolStream:      o.toolStream,
		ReasoningEffort: o.effort,
	}
	if o.thinking != "" {
		req.Thinking = &client.ThinkingConfig{Type: o.thinking}
	}

	system := o.system
	if o.jsonObject || o.schema != "" {
		req.ResponseFormat = client.JSONObjectFormat()
	}
	if o.schema != "" {
		schema, err := loadJSONArg(o.schema)
		if err != nil {
			return req, fmt.Errorf("read --json-schema: %w", err)
		}
		instruction, err := client.JSONSchemaPrompt(schema)
		if err != nil {
			return req, fmt.Errorf("--json-schema: %w", err)
		}
		system = strings.TrimSpace(system + "\n\n" + instruction)
	}
	if system != "" {
		req.Messages = append(req.Messages, client.Message{Role: "system", Content: system})
	}

	user := client.Message{Role: "user", Content: message}
	for _, a := range []struct {
		flag, fallback string
		args           []string
		dst            *[]string
	}{
		{"--image", "image/jpeg", o.images, &user.Images},
		{"--video", "video/mp4", o.videos, &user.Videos},
		{"--file", "application/pdf", o.files, &user.Files},
	} {
		for _, arg := range a.args {
			ref, err := fileinput.URLOrDataURI(arg, a.fallback)
			if err != nil {
				return req, fmt.Errorf("%s %q: %w", a.flag, arg, err)
			}
			*a.dst = append(*a.dst, ref)
		}
	}
	req.Messages = append(req.Messages, user)

	if o.tools != "" {
		raw, err := loadJSONArg(o.tools)
		if err != nil {
			return req, fmt.Errorf("read --tool: %w", err)
		}
		if err := json.Unmarshal(raw, &req.Tools); err != nil {
			return req, fmt.Errorf("parse --tool JSON: %w", err)
		}
	}
	return req, nil
}

// loadJSONArg resolves an "@path" file reference or returns the literal bytes.
func loadJSONArg(arg string) ([]byte, error) {
	if path, ok := strings.CutPrefix(arg, "@"); ok {
		return os.ReadFile(path)
	}
	return []byte(arg), nil
}

// streamChat prints content deltas to stdout (JSON lines with --format json)
// and, when asked, reasoning to stderr.
func streamChat(cmd *cobra.Command, apiClient *client.Client, req client.ChatRequest, showReasoning bool) error {
	asJSON := isJSONFormat(cmd)
	enc := json.NewEncoder(os.Stdout)
	for chunk, err := range apiClient.Chat().Stream(cmd.Context(), req) {
		if err != nil {
			return err
		}
		if asJSON {
			if err := enc.Encode(chunk); err != nil {
				return err
			}
			continue
		}
		for _, c := range chunk.Choices {
			if showReasoning && c.Delta.ReasoningContent != "" {
				fmt.Fprint(os.Stderr, c.Delta.ReasoningContent)
			}
			fmt.Print(c.Delta.Content)
		}
	}
	if !asJSON {
		fmt.Println()
	}
	return nil
}

// printReply prints an assistant message: reasoning (when asked) and tool
// calls go to stderr so stdout carries only the content.
func printReply(msg client.ResponseMsg, showReasoning bool) {
	if showReasoning {
		printReasoning(msg.ReasoningContent)
	}
	fmt.Println(msg.Content)
	for _, tc := range msg.ToolCalls {
		if tc.Function != nil {
			fmt.Fprintf(os.Stderr, "tool call: %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
		}
	}
}

// printReasoning prints a model's reasoning to stderr, framed, so stdout
// carries only the answer.
func printReasoning(reasoning string) {
	if reasoning != "" {
		fmt.Fprintf(os.Stderr, "--- reasoning ---\n%s\n-----------------\n", reasoning)
	}
}
