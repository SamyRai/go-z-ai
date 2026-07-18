// Command anthropic-messages hits the Anthropic-compatible /v1/messages
// endpoint exposed by Z.AI, using the same API key (Bearer auth, not
// x-api-key). Useful for dropping Z.AI into Anthropic-shaped code paths.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/anthropic-messages "Give me three Go tips"
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	prompt := "Give me three Go tips"
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}

	c, err := client.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	temp := 0.5
	resp, err := c.Anthropic().Create(context.Background(), client.AnthropicMessageRequest{
		Model:       "glm-4.6",
		MaxTokens:   512, // required by the Anthropic surface; must be > 0
		System:      "be concise",
		Temperature: &temp, // pointer: omit entirely to leave it unset
		Messages:    []client.AnthropicMessage{client.AnthropicTextMessage("user", prompt)},
	})
	if err != nil {
		log.Fatalf("messages: %v", err)
	}

	fmt.Println(resp.Text())
	fmt.Fprintf(os.Stderr, "\n(stop_reason=%s, in=%d tok, out=%d tok)\n",
		resp.StopReason, resp.Usage.InputTokens, resp.Usage.OutputTokens)
}
