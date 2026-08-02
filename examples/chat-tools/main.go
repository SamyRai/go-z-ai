// Command chat-tools demonstrates function/tool calling via RunWithTools:
// define one or more tools (functions the model can invoke), let the model
// decide which to call, dispatch the calls, and loop until the model gives
// a final answer. The simplest agent loop the client supports.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/chat-tools "What's the weather like in Berlin and Tokyo?"
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	prompt := "What's the weather like in Berlin and Tokyo?"
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}

	c, err := client.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	// Define one tool: get_weather. The model fills the arguments; the
	// ToolExecutor callback dispatches the actual call. RunWithTools loops
	// (call model → exec tools → feed results back → repeat) until the model
	// returns a non-tool finish reason or the round cap is hit.
	tools := []client.Tool{
		client.NewFunctionTool(
			"get_weather",
			"Get the current weather for a city.",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"city":    map[string]any{"type": "string", "description": "City name"},
					"country": map[string]any{"type": "string", "description": "ISO country code"},
				},
				"required": []any{"city"},
			},
		),
	}

	resp, err := c.Chat().RunWithTools(context.Background(), client.ChatRequest{
		Model:    "glm-5.2",
		Messages: []client.Message{{Role: "user", Content: prompt}},
		Tools:    tools,
	}, func(name, argsJSON string) (string, error) {
		// Tool dispatch: parse args, run the tool, return a JSON string.
		// Errors become tool-result messages with "error: ..." so the model
		// can recover (retry with different args, give up gracefully, etc.).
		if name != "get_weather" {
			return "", fmt.Errorf("unknown tool %q", name)
		}
		var args struct {
			City    string `json:"city"`
			Country string `json:"country"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("parse args: %w", err)
		}
		// Stub: return a deterministic-but-random temperature. A real impl
		// would call a weather API here.
		tempC := 5 + rand.Intn(30)
		result, _ := json.Marshal(map[string]any{
			"city":        args.City,
			"temperature": tempC,
			"unit":        "C",
			"conditions":  []string{"sunny", "cloudy", "rainy"}[rand.Intn(3)],
		})
		fmt.Fprintf(os.Stderr, "[tool] get_weather(%s) → %s\n", args.City, result)
		return string(result), nil
	})
	if err != nil {
		log.Fatalf("run: %v", err)
	}
	fmt.Println(resp.Choices[0].Message.Content)
}
