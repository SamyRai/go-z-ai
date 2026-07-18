// Command chat-streaming is a minimal example of streaming a chat completion
// token-by-token with the Z.AI Go client.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/chat-streaming "Explain goroutines in one paragraph"
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	prompt := "Explain goroutines in one paragraph"
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}

	c, err := client.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	req := client.ChatRequest{
		Model:    "glm-5.2",
		Messages: []client.Message{{Role: "user", Content: prompt}},
		TopP:     0.95,
	}

	// CreateStream forces Stream=true on the wire and invokes onChunk for every
	// SSE event, retrying transient connect-level failures per Config.MaxRetries.
	err = c.Chat().CreateStream(context.Background(), req, func(ch client.StreamChunk) error {
		if len(ch.Choices) > 0 {
			fmt.Print(ch.Choices[0].Delta.Content)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("stream: %v", err)
	}
	fmt.Println()
}
