// Command quickstart-vision sends an image to a vision-capable GLM model
// (glm-4.6v) and prints the model's description. Pass an image URL — the
// client passes image URLs through verbatim. To use a local file, base64-encode
// it into a "data:image/...;base64,..." URI yourself and pass that string.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/quickstart-vision "https://example.com/photo.jpg" "Describe this image"
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	imageURL := "https://upload.wikimedia.org/wikipedia/commons/thumb/4/47/PNG_transparency_demonstration_1.png/640px-PNG_transparency_demonstration_1.png"
	prompt := "Describe this image in two sentences."
	switch {
	case len(os.Args) >= 3:
		prompt = os.Args[2]
		fallthrough
	case len(os.Args) >= 2:
		imageURL = os.Args[1]
	}

	c, err := client.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	resp, err := c.Chat().Create(context.Background(), client.ChatRequest{
		Model: "glm-4.6v",
		Messages: []client.Message{{
			Role:    "user",
			Content: prompt,
			Images:  []string{imageURL}, // URL or data: URI
		}},
	})
	if err != nil {
		log.Fatalf("vision: %v", err)
	}
	fmt.Println(resp.Choices[0].Message.Content)
}
