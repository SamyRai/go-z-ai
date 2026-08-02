// Command quickstart-chat is the minimum hello-world for the go-z-ai client:
// one shot, non-streaming, prints the assistant's reply. Fits in a README
// code block.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/quickstart-chat "Explain goroutines in one paragraph"
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

	resp, err := c.Chat().Create(context.Background(), client.ChatRequest{
		Model:    "glm-5.2",
		Messages: []client.Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		log.Fatalf("chat: %v", err)
	}
	fmt.Println(resp.Choices[0].Message.Content)
}
