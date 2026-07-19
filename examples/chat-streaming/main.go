// Command chat-streaming is a minimal example of streaming a chat completion
// token-by-token with the Z.AI Go client, using the Go 1.23+ iterator API.
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

	// Stream returns an iter.Seq2[StreamChunk, error] you range over. It forces
	// Stream=true on the wire and retries transient connect-level failures per
	// Config.MaxRetries. Context cancellation tears down the in-flight SSE read.
	for chunk, err := range c.Chat().Stream(context.Background(), req) {
		if err != nil {
			log.Fatalf("stream: %v", err)
		}
		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	fmt.Println()
}
